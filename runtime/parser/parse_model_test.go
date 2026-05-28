package parser

import (
	"context"
	"testing"

	runtimev1 "github.com/rilldata/rill/proto/gen/rill/runtime/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestModelOutput(t *testing.T) {
	files := map[string]string{
		`rill.yaml`: ``,
		`m1.sql`: `
SELECT 1
`,
		`m2.yaml`: `
type: model
sql: SELECT 1
`,
		`m3.yaml`: `
type: model
connector: bigquery
sql: SELECT 1
`,
		`m4.yaml`: `
type: model
connector: bigquery
sql: SELECT 1
output:
  table: foobar
`,
		`m5.yaml`: `
type: model
connector: bigquery
sql: SELECT 1
output: clickhouse
`,
		`m6.yaml`: `
type: model
connector: bigquery
sql: SELECT 1
output:
  connector: clickhouse
`,
	}
	resources := []*Resource{
		// model m1
		{
			Name:  ResourceName{Kind: ResourceKindModel, Name: "m1"},
			Paths: []string{"/m1.sql"},
			ModelSpec: &runtimev1.ModelSpec{
				RefreshSchedule: &runtimev1.Schedule{RefUpdate: true},
				InputConnector:  "duckdb",
				InputProperties: must(structpb.NewStruct(map[string]any{"sql": "SELECT 1"})),
				OutputConnector: "duckdb",
				ChangeMode:      runtimev1.ModelChangeMode_MODEL_CHANGE_MODE_RESET,
			},
		},
		// model m2
		{
			Name:  ResourceName{Kind: ResourceKindModel, Name: "m2"},
			Paths: []string{"/m2.yaml"},
			ModelSpec: &runtimev1.ModelSpec{
				RefreshSchedule: &runtimev1.Schedule{RefUpdate: true},
				InputConnector:  "duckdb",
				InputProperties: must(structpb.NewStruct(map[string]any{"sql": "SELECT 1"})),
				OutputConnector: "duckdb",
				ChangeMode:      runtimev1.ModelChangeMode_MODEL_CHANGE_MODE_RESET,
			},
		},
		// model m3
		{
			Name:  ResourceName{Kind: ResourceKindModel, Name: "m3"},
			Paths: []string{"/m3.yaml"},
			ModelSpec: &runtimev1.ModelSpec{
				RefreshSchedule: &runtimev1.Schedule{RefUpdate: true},
				InputConnector:  "bigquery",
				InputProperties: must(structpb.NewStruct(map[string]any{"sql": "SELECT 1"})),
				OutputConnector: "duckdb",
				ChangeMode:      runtimev1.ModelChangeMode_MODEL_CHANGE_MODE_RESET,
			},
		},
		// model m4
		{
			Name:  ResourceName{Kind: ResourceKindModel, Name: "m4"},
			Paths: []string{"/m4.yaml"},
			ModelSpec: &runtimev1.ModelSpec{
				RefreshSchedule: &runtimev1.Schedule{RefUpdate: true},
				InputConnector:  "bigquery",
				InputProperties: must(structpb.NewStruct(map[string]any{"sql": "SELECT 1"})),
				OutputConnector: "duckdb",
				OutputProperties: must(structpb.NewStruct(map[string]any{
					"table": "foobar",
				})),
				ChangeMode: runtimev1.ModelChangeMode_MODEL_CHANGE_MODE_RESET,
			},
		},
		// model m5
		{
			Name:  ResourceName{Kind: ResourceKindModel, Name: "m5"},
			Paths: []string{"/m5.yaml"},
			ModelSpec: &runtimev1.ModelSpec{
				RefreshSchedule: &runtimev1.Schedule{RefUpdate: true},
				InputConnector:  "bigquery",
				InputProperties: must(structpb.NewStruct(map[string]any{"sql": "SELECT 1"})),
				OutputConnector: "clickhouse",
				ChangeMode:      runtimev1.ModelChangeMode_MODEL_CHANGE_MODE_RESET,
			},
		},
		// model m6
		{
			Name:  ResourceName{Kind: ResourceKindModel, Name: "m6"},
			Paths: []string{"/m6.yaml"},
			ModelSpec: &runtimev1.ModelSpec{
				RefreshSchedule: &runtimev1.Schedule{RefUpdate: true},
				InputConnector:  "bigquery",
				InputProperties: must(structpb.NewStruct(map[string]any{"sql": "SELECT 1"})),
				OutputConnector: "clickhouse",
				ChangeMode:      runtimev1.ModelChangeMode_MODEL_CHANGE_MODE_RESET,
			},
		},
	}

	ctx := context.Background()
	repo := makeRepo(t, files)
	p, err := Parse(ctx, repo, "", "", "duckdb")
	require.NoError(t, err)
	requireResourcesAndErrors(t, p, resources, nil)
}

// TestExternalModel verifies the `external: true` field on a Model resource:
//   - parses without forcing materialize
//   - propagates External=true to the ModelSpec
//   - injects external=true into OutputProperties for the executor's defensive guard
//   - validates that output.table is required and incompatible options are rejected.
func TestExternalModel(t *testing.T) {
	files := map[string]string{
		`rill.yaml`: ``,
		// Happy path: external model with table reference. Should NOT have materialize set.
		`m_ext.yaml`: `
type: model
external: true
output:
  connector: clickhouse
  table: dim_orders
`,
		// Source-style declaration that also opts into external — verifies the
		// parse_source.go gate skips the forced materialize: true override.
		`m_ext_source.yaml`: `
type: source
connector: clickhouse
external: true
output:
  table: dim_orders
`,
	}
	resources := []*Resource{
		{
			Name:  ResourceName{Kind: ResourceKindModel, Name: "m_ext"},
			Paths: []string{"/m_ext.yaml"},
			ModelSpec: &runtimev1.ModelSpec{
				RefreshSchedule: &runtimev1.Schedule{RefUpdate: true},
				InputConnector:  "duckdb",
				InputProperties: must(structpb.NewStruct(map[string]any{})),
				OutputConnector: "clickhouse",
				OutputProperties: must(structpb.NewStruct(map[string]any{
					"table":    "dim_orders",
					"external": true,
				})),
				External:   true,
				ChangeMode: runtimev1.ModelChangeMode_MODEL_CHANGE_MODE_RESET,
			},
		},
		{
			Name:  ResourceName{Kind: ResourceKindModel, Name: "m_ext_source"},
			Paths: []string{"/m_ext_source.yaml"},
			ModelSpec: &runtimev1.ModelSpec{
				RefreshSchedule: &runtimev1.Schedule{RefUpdate: true},
				InputConnector:  "clickhouse",
				InputProperties: must(structpb.NewStruct(map[string]any{})),
				OutputConnector: "duckdb",
				OutputProperties: must(structpb.NewStruct(map[string]any{
					"table":    "dim_orders",
					"external": true,
				})),
				External:        true,
				DefinedAsSource: true,
				ChangeMode:      runtimev1.ModelChangeMode_MODEL_CHANGE_MODE_RESET,
			},
		},
	}

	ctx := context.Background()
	repo := makeRepo(t, files)
	p, err := Parse(ctx, repo, "", "", "duckdb")
	require.NoError(t, err)
	requireResourcesAndErrors(t, p, resources, nil)
}

// TestExternalModelValidation checks negative paths — external: true without the
// required output.table, and incompatible combinations.
func TestExternalModelValidation(t *testing.T) {
	cases := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{
			name: "missing output.table",
			yaml: `
type: model
external: true
output:
  connector: clickhouse
`,
			wantErr: `"external: true" requires "output.table" to be set`,
		},
		{
			name: "external + incremental",
			yaml: `
type: model
external: true
incremental: true
output:
  connector: clickhouse
  table: dim_orders
`,
			wantErr: `"external: true" cannot be combined with "incremental"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := map[string]string{
				`rill.yaml`: ``,
				`m.yaml`:    tc.yaml,
			}
			ctx := context.Background()
			repo := makeRepo(t, files)
			p, err := Parse(ctx, repo, "", "", "duckdb")
			require.NoError(t, err)
			require.Len(t, p.Errors, 1, "expected parse error")
			require.Contains(t, p.Errors[0].Message, tc.wantErr)
		})
	}
}
