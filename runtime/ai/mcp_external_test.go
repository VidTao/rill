package ai_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rilldata/rill/runtime"
	"github.com/rilldata/rill/runtime/ai"
	"github.com/rilldata/rill/runtime/pkg/activity"
	"github.com/rilldata/rill/runtime/testruntime"
	"github.com/stretchr/testify/require"
)

// chartTestFiles is a project with a daily time series and a categorical
// dimension, enough to exercise both show_chart modes.
var chartTestFiles = map[string]string{
	"orders.sql": `
SELECT * FROM (VALUES
  (TIMESTAMP '2026-09-01 10:00:00', 'Meta',   120.0, 3),
  (TIMESTAMP '2026-09-01 12:00:00', 'Google',  80.0, 2),
  (TIMESTAMP '2026-09-02 09:00:00', 'Meta',    60.0, 1),
  (TIMESTAMP '2026-09-03 15:00:00', 'TikTok',  45.0, 1)
) AS t(ts, channel, revenue, orders)`,
	"sales.yaml": `
type: metrics_view
version: 1
model: orders
timeseries: ts
dimensions:
- column: channel
  display_name: Channel
measures:
- name: revenue
  display_name: Revenue
  expression: SUM(revenue)
  format_preset: currency_usd
- name: order_count
  expression: SUM(orders)
explore:
  skip: true
`,
}

// newExternalSession is a session as an MCP client like Claude sees it: a
// non-"rill" user agent and a merchant's permissions.
func newExternalSession(t *testing.T, rt *runtime.Runtime, instanceID string) *ai.Session {
	t.Helper()
	claims := &runtime.SecurityClaims{UserID: uuid.NewString(), SkipChecks: true}
	s, err := ai.NewRunner(rt, activity.NewNoopClient()).Session(t.Context(), &ai.SessionOptions{
		InstanceID: instanceID,
		Claims:     claims,
		UserAgent:  "claude-ai/0.1.0",
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, s.Flush(t.Context())) })
	return s
}

// connectExternal serves the session's MCP server over an in-memory transport.
func connectExternal(t *testing.T, s *ai.Session) *mcp.ClientSession {
	t.Helper()
	serverT, clientT := mcp.NewInMemoryTransports()
	_, err := s.MCPServer(t.Context()).Connect(t.Context(), serverT, nil)
	require.NoError(t, err)
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "claude-ai", Version: "0.1.0"}, nil).Connect(t.Context(), clientT, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

// The surface a Claude or ChatGPT directory reviewer sees. Every tool must be
// annotated and $ref-free, the developer tools must stay hidden, and the chart
// widget must be served as an MCP App.
func TestExternalMCPToolSurface(t *testing.T) {
	rt, instanceID := testruntime.NewInstanceWithOptions(t, testruntime.InstanceOptions{Files: chartTestFiles})
	testruntime.RequireReconcileState(t, rt, instanceID, 3, 0, 0)
	cs := connectExternal(t, newExternalSession(t, rt, instanceID))

	res, err := cs.ListTools(t.Context(), nil)
	require.NoError(t, err)
	names := map[string]*mcp.Tool{}
	for _, tool := range res.Tools {
		names[tool.Name] = tool
	}

	for _, want := range []string{ai.ListMetricsViewsName, ai.GetMetricsViewName, ai.QueryMetricsViewName, ai.ShowChartName} {
		require.Contains(t, names, want)
	}
	for _, hidden := range []string{
		ai.ListTablesName, ai.ShowTableName, ai.ListBucketsName, ai.ListBucketObjectsName, ai.ProjectStatusName,
		ai.CreateChartName, ai.QuerySQLName, "navigate", "developer_agent", "analyst_agent", "router_agent",
		"read_file", "write_file", "workshop_deploy", "workshop_list_clients",
	} {
		require.NotContains(t, names, hidden, "%s must not reach external MCP clients", hidden)
	}

	for name, tool := range names {
		require.NotNil(t, tool.Annotations, "%s has no annotations", name)
		require.NotEmpty(t, tool.Annotations.Title, "%s has no annotation title", name)
		require.True(t, tool.Annotations.ReadOnlyHint || tool.Annotations.DestructiveHint != nil,
			"%s needs readOnlyHint or destructiveHint", name)
		schema, err := json.Marshal(tool.InputSchema)
		require.NoError(t, err)
		require.NotContains(t, string(schema), `"$ref"`, "%s input schema still has $ref", name)
		require.LessOrEqual(t, len(name), 64)
	}

	chart := names[ai.ShowChartName]
	ui, _ := chart.Meta["ui"].(map[string]any)
	require.Equal(t, "ui://bratrax/chart", ui["resourceUri"])

	read, err := cs.ReadResource(t.Context(), &mcp.ReadResourceParams{URI: "ui://bratrax/chart"})
	require.NoError(t, err)
	require.Len(t, read.Contents, 1)
	require.Equal(t, "text/html;profile=mcp-app", read.Contents[0].MIMEType)
	require.Contains(t, read.Contents[0].Text, "ui/initialize")
}

// Inside the app the widget has nowhere to render, so the tool and its
// resource stay out of the in-app agents' way.
func TestShowChartHiddenInApp(t *testing.T) {
	rt, instanceID := testruntime.NewInstanceWithOptions(t, testruntime.InstanceOptions{Files: chartTestFiles})
	testruntime.RequireReconcileState(t, rt, instanceID, 3, 0, 0)
	s := newSession(t, rt, instanceID)

	var res *ai.ShowChartResult
	_, err := s.CallTool(t.Context(), ai.RoleUser, ai.ShowChartName, &res, &ai.ShowChartArgs{
		MetricsView: "sales", Measures: []string{"revenue"},
	})
	require.Error(t, err)
}

func TestShowChart(t *testing.T) {
	rt, instanceID := testruntime.NewInstanceWithOptions(t, testruntime.InstanceOptions{
		Files:       chartTestFiles,
		FrontendURL: "https://bratrax.test",
	})
	testruntime.RequireReconcileState(t, rt, instanceID, 3, 0, 0)
	s := newExternalSession(t, rt, instanceID)

	t.Run("time series", func(t *testing.T) {
		var res *ai.ShowChartResult
		_, err := s.CallTool(t.Context(), ai.RoleUser, ai.ShowChartName, &res, &ai.ShowChartArgs{
			MetricsView: "sales", Measures: []string{"revenue", "order_count"},
			Start: "2026-09-01", End: "2026-09-04",
		})
		require.NoError(t, err)
		require.Equal(t, "line", res.ChartType)
		require.Equal(t, "time", res.XType)
		require.Equal(t, "ts", res.XField)
		require.Len(t, res.Rows, 3, "one row per day")
		require.Equal(t, "Revenue", res.Series[0].Label)
		require.Equal(t, "currency_usd", res.Series[0].Format)
		require.EqualValues(t, 200, res.Rows[0]["revenue"])
		require.Equal(t, "https://bratrax.test", res.OpenURL)
		require.Equal(t, "2026-09-01 to 2026-09-03", res.PeriodLabel)
	})

	t.Run("breakdown", func(t *testing.T) {
		var res *ai.ShowChartResult
		_, err := s.CallTool(t.Context(), ai.RoleUser, ai.ShowChartName, &res, &ai.ShowChartArgs{
			MetricsView: "sales", Measures: []string{"revenue"}, Dimension: "channel",
			Start: "2026-09-01", End: "2026-09-04", Limit: 2,
		})
		require.NoError(t, err)
		require.Equal(t, "bar", res.ChartType)
		require.Len(t, res.Rows, 2)
		require.Equal(t, "Meta", res.Rows[0]["channel"], "sorted by the first measure, descending")
		require.NotEmpty(t, res.Note, "a cut-off breakdown says so")
		require.Equal(t, "Revenue by Channel", res.Title)
	})

	t.Run("helpful errors", func(t *testing.T) {
		var res *ai.ShowChartResult
		_, err := s.CallTool(t.Context(), ai.RoleUser, ai.ShowChartName, &res, &ai.ShowChartArgs{
			MetricsView: "sales", Measures: []string{"roas"},
		})
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), "available: revenue, order_count"), err.Error())
	})
}
