package clickhouse

import (
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/require"
)

func TestZeroJSONValueForClickHouseType(t *testing.T) {
	tests := []struct {
		typ  string
		want interface{}
	}{
		// Integers
		{"Int8", 0},
		{"Int16", 0},
		{"Int32", 0},
		{"Int64", 0},
		{"Int128", 0},
		{"Int256", 0},
		{"UInt8", 0},
		{"UInt16", 0},
		{"UInt32", 0},
		{"UInt64", 0},

		// Floats
		{"Float32", 0.0},
		{"Float64", 0.0},

		// Bool
		{"Bool", false},

		// Strings / IDs
		{"String", ""},
		{"FixedString(16)", ""},
		{"UUID", ""},
		{"IPv4", ""},
		{"IPv6", ""},
		{"Enum8('a' = 1, 'b' = 2)", ""},
		{"Enum16('x' = 10)", ""},

		// Dates
		{"Date", "1970-01-01"},
		{"Date32", "1970-01-01"},
		{"DateTime", "1970-01-01 00:00:00"},
		{"DateTime('UTC')", "1970-01-01 00:00:00"},
		{"DateTime64(3)", "1970-01-01 00:00:00"},
		{"DateTime64(6, 'UTC')", "1970-01-01 00:00:00"},

		// Decimal — encoded as string to preserve precision
		{"Decimal(38, 4)", "0"},
		{"Decimal32(2)", "0"},
		{"Decimal64(4)", "0"},
		{"Decimal128(8)", "0"},
		{"Decimal256(10)", "0"},

		// Nullable wrappers — unwrap to inner type
		{"Nullable(Int32)", 0},
		{"Nullable(String)", ""},
		{"Nullable(DateTime)", "1970-01-01 00:00:00"},
		{"Nullable(Decimal(38, 4))", "0"},

		// LowCardinality wrappers
		{"LowCardinality(String)", ""},
		{"LowCardinality(Nullable(String))", ""},

		// Unsupported complex types fall through to nil
		{"Array(String)", nil},
		{"Map(String, Int64)", nil},
		{"Tuple(Int32, String)", nil},
		{"Nested(a Int32, b String)", nil},
		{"SomeUnknownType", nil},
	}

	for _, tc := range tests {
		t.Run(tc.typ, func(t *testing.T) {
			got := zeroJSONValueForClickHouseType(tc.typ, nil)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestWarehouseSourcePropertiesAllowEmpty(t *testing.T) {
	props := &warehouseSourceProperties{}
	err := mapstructure.WeakDecode(map[string]any{
		"sql":         "SELECT * FROM x",
		"allow_empty": true,
	}, props)
	require.NoError(t, err)
	require.Equal(t, "SELECT * FROM x", props.SQL)
	require.True(t, props.AllowEmpty)

	// Default: AllowEmpty is false
	props2 := &warehouseSourceProperties{}
	err = mapstructure.WeakDecode(map[string]any{"sql": "SELECT 1"}, props2)
	require.NoError(t, err)
	require.False(t, props2.AllowEmpty)
}
