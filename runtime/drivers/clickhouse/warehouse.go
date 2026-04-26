package clickhouse

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/rilldata/rill/runtime/drivers"
	"github.com/rilldata/rill/runtime/pkg/observability"
	"go.uber.org/zap"
)

// placeholderColumn is a sentinel column injected into NDJSON output when
// allow_empty is enabled. The downstream DuckDB ingest filters rows where this
// column is non-NULL and excludes the column from the final table.
const placeholderColumn = "__rill_placeholder"

var _ drivers.Warehouse = &Connection{}

// QueryAsFiles implements drivers.Warehouse.
// It executes a SQL query on ClickHouse and writes the results to a temporary
// ndjson file that DuckDB (or another OLAP engine) can ingest.
func (c *Connection) QueryAsFiles(ctx context.Context, props map[string]any) (drivers.FileIterator, error) {
	srcProps := &warehouseSourceProperties{}
	if err := mapstructure.WeakDecode(props, srcProps); err != nil {
		return nil, fmt.Errorf("failed to parse source properties: %w", err)
	}
	if srcProps.SQL == "" {
		return nil, fmt.Errorf("property 'sql' is required")
	}

	tempDir, err := c.storage.RandomTempDir("clickhouse-warehouse-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	return &warehouseFileIterator{
		c:          c,
		sql:        srcProps.SQL,
		tempDir:    tempDir,
		allowEmpty: srcProps.AllowEmpty,
	}, nil
}

type warehouseSourceProperties struct {
	SQL string `mapstructure:"sql"`
	// AllowEmpty, when true, causes a 0-row query result to produce an empty
	// target table (with correct schema) instead of returning ErrNoRows. The
	// downstream DuckDB executor must cooperate by filtering the placeholder
	// row this driver emits.
	AllowEmpty bool `mapstructure:"allow_empty"`
}

type warehouseFileIterator struct {
	c          *Connection
	sql        string
	tempDir    string
	downloaded bool
	keepFiles  bool
	allowEmpty bool
}

var _ drivers.FileIterator = &warehouseFileIterator{}

func (it *warehouseFileIterator) Close() error {
	return os.RemoveAll(it.tempDir)
}

func (it *warehouseFileIterator) Format() string {
	return ""
}

func (it *warehouseFileIterator) SetKeepFilesUntilClose() {
	it.keepFiles = true
}

func (it *warehouseFileIterator) Next(ctx context.Context) ([]string, error) {
	if it.downloaded {
		return nil, io.EOF
	}
	it.downloaded = true

	start := time.Now()

	rows, err := it.c.readDB.QueryxContext(ctx, it.sql)
	if err != nil {
		return nil, fmt.Errorf("clickhouse query failed: %w", err)
	}
	defer rows.Close()

	fw, err := os.CreateTemp(it.tempDir, "clickhouse-*.ndjson")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer fw.Close()

	enc := json.NewEncoder(fw)
	enc.SetEscapeHTML(false)

	rowCount := int64(0)
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		row := make(map[string]interface{})
		if err := rows.MapScan(row); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Convert []byte and time.Time values for JSON compatibility.
		// The clickhouse-go driver returns some types as []byte, and
		// DateTime64 as time.Time which json.Marshal encodes as RFC3339
		// ("2026-02-22T10:35:42.123Z"). DuckDB's read_json auto-detect
		// does not recognise RFC3339 with fractional seconds as TIMESTAMP,
		// so we format to "YYYY-MM-DD HH:MM:SS.ffffff" instead.
		for k, v := range row {
			switch val := v.(type) {
			case []byte:
				row[k] = string(val)
			case time.Time:
				row[k] = val.Format("2006-01-02 15:04:05.000000")
			case *time.Time:
				if val != nil {
					row[k] = val.Format("2006-01-02 15:04:05.000000")
				}
			}
		}

		if it.allowEmpty {
			row[placeholderColumn] = nil
		}

		if err := enc.Encode(row); err != nil {
			return nil, fmt.Errorf("failed to encode row as JSON: %w", err)
		}
		rowCount++
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("clickhouse row iteration error: %w", err)
	}

	if rowCount == 0 {
		if !it.allowEmpty {
			return nil, drivers.ErrNoRows
		}
		if err := it.writePlaceholderRow(ctx, enc); err != nil {
			return nil, fmt.Errorf("failed to write placeholder row: %w", err)
		}
		rowCount = 1
	}

	it.c.logger.Debug("clickhouse warehouse: query exported to ndjson",
		zap.Int64("rows", rowCount),
		zap.Duration("duration", time.Since(start)),
		observability.ZapCtx(ctx),
	)

	return []string{fw.Name()}, nil
}

// writePlaceholderRow emits one NDJSON row with typed zero-values so DuckDB
// can infer schema from an otherwise empty query result. The row is marked
// with placeholderColumn so the downstream ingest filters it out.
func (it *warehouseFileIterator) writePlaceholderRow(ctx context.Context, enc *json.Encoder) error {
	// DESCRIBE a LIMIT 0 wrapper around the user SQL to retrieve column names
	// and types without side-effects. ClickHouse supports DESCRIBE on subqueries.
	describeSQL := fmt.Sprintf("DESCRIBE (SELECT * FROM (%s) LIMIT 0)", it.sql)
	rows, err := it.c.readDB.QueryxContext(ctx, describeSQL)
	if err != nil {
		return fmt.Errorf("describe query failed: %w", err)
	}
	defer rows.Close()

	placeholder := make(map[string]interface{})
	for rows.Next() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// DESCRIBE returns columns: name, type, default_type, default_expression,
		// comment, codec_expression, ttl_expression. We only care about name+type.
		row := make(map[string]interface{})
		if err := rows.MapScan(row); err != nil {
			return fmt.Errorf("failed to scan describe row: %w", err)
		}
		name := stringFromDescribe(row["name"])
		typ := stringFromDescribe(row["type"])
		if name == "" {
			continue
		}
		placeholder[name] = zeroJSONValueForClickHouseType(typ, it.c.logger)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("describe row iteration error: %w", err)
	}
	if len(placeholder) == 0 {
		return fmt.Errorf("describe returned no columns for sql: %s", it.sql)
	}

	placeholder[placeholderColumn] = 1
	return enc.Encode(placeholder)
}

func stringFromDescribe(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	default:
		return ""
	}
}

// zeroJSONValueForClickHouseType returns a JSON-safe Go value that, when
// encoded into NDJSON and read by DuckDB's read_json_auto, will cause DuckDB
// to infer a column type compatible with the given ClickHouse type.
//
// The ClickHouse type string may carry modifiers (Nullable, LowCardinality)
// or parameters (DateTime64(3), Decimal(38, 4)). We strip modifiers and match
// on the inner base type.
func zeroJSONValueForClickHouseType(typ string, logger *zap.Logger) interface{} {
	base := strings.TrimSpace(typ)

	// Strip wrapper modifiers; these do not change the placeholder value.
	// Loop because wrappers can nest (e.g., LowCardinality(Nullable(String))).
	for {
		stripped := false
		for _, wrapper := range []string{"Nullable(", "LowCardinality("} {
			if strings.HasPrefix(base, wrapper) && strings.HasSuffix(base, ")") {
				base = strings.TrimSuffix(strings.TrimPrefix(base, wrapper), ")")
				base = strings.TrimSpace(base)
				stripped = true
			}
		}
		if !stripped {
			break
		}
	}

	// Strip parameter lists: DateTime64(3) -> DateTime64, Decimal(38, 4) -> Decimal, etc.
	stem := base
	if idx := strings.Index(stem, "("); idx >= 0 {
		stem = strings.TrimSpace(stem[:idx])
	}

	switch {
	case strings.HasPrefix(stem, "Int"), strings.HasPrefix(stem, "UInt"):
		return 0
	case stem == "Float32", stem == "Float64":
		return 0.0
	case stem == "Bool":
		return false
	case stem == "String", stem == "FixedString", stem == "UUID", stem == "IPv4", stem == "IPv6":
		return ""
	case strings.HasPrefix(stem, "Enum"):
		return ""
	case stem == "Date", stem == "Date32":
		return "1970-01-01"
	case stem == "DateTime":
		return "1970-01-01 00:00:00"
	case stem == "DateTime64":
		return "1970-01-01 00:00:00.000"
	case strings.HasPrefix(stem, "Decimal"):
		// Encode as string to preserve precision through JSON inference.
		return "0"
	}

	// Complex types (Array, Map, Tuple, Nested) and unknown types fall through
	// to null. DuckDB will typically infer VARCHAR for a column of all nulls;
	// callers relying on such types in possibly-empty sources should either
	// pre-seed a row or avoid allow_empty.
	if logger != nil {
		logger.Warn("clickhouse warehouse: unsupported type for allow_empty placeholder, using null",
			zap.String("type", typ),
		)
	}
	return nil
}
