package clickhouse

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/rilldata/rill/runtime/drivers"
	"github.com/rilldata/rill/runtime/pkg/observability"
	"go.uber.org/zap"
)

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
		c:       c,
		sql:     srcProps.SQL,
		tempDir: tempDir,
	}, nil
}

type warehouseSourceProperties struct {
	SQL string `mapstructure:"sql"`
}

type warehouseFileIterator struct {
	c          *Connection
	sql        string
	tempDir    string
	downloaded bool
	keepFiles  bool
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

		// Convert []byte values to strings for JSON compatibility.
		// The clickhouse-go driver returns some types as []byte.
		for k, v := range row {
			if b, ok := v.([]byte); ok {
				row[k] = string(b)
			}
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
		return nil, drivers.ErrNoRows
	}

	it.c.logger.Debug("clickhouse warehouse: query exported to ndjson",
		zap.Int64("rows", rowCount),
		zap.Duration("duration", time.Since(start)),
		observability.ZapCtx(ctx),
	)

	return []string{fw.Name()}, nil
}
