package ai_test

import (
	"testing"

	"github.com/rilldata/rill/runtime/ai"
	"github.com/rilldata/rill/runtime/testruntime"
	"github.com/stretchr/testify/require"
)

// canvasYAML is a minimal valid canvas over the shared test metrics view.
func canvasYAML(displayName string) string {
	return `
type: canvas
display_name: ` + displayName + `
rows:
- items:
  - markdown:
      content: hello
`
}

const canvasTestMetrics = `
type: metrics_view
version: 1
model: test_data
dimensions:
- column: country
measures:
- expression: SUM(revenue)
  name: total_revenue
explore:
  skip: true
`

func canvasTestFiles(extra map[string]string) map[string]string {
	files := map[string]string{
		"test_data.sql":     `SELECT 'US' AS country, 100 AS revenue, NOW() AS timestamp`,
		"test_metrics.yaml": canvasTestMetrics,
	}
	for k, v := range extra {
		files[k] = v
	}
	return files
}

func listCanvases(t *testing.T, opts testruntime.InstanceOptions) []map[string]any {
	t.Helper()
	rt, instanceID := testruntime.NewInstanceWithOptions(t, opts)
	s := newSession(t, rt, instanceID)

	var res *ai.ListCanvasesResult
	_, err := s.CallTool(t.Context(), ai.RoleUser, ai.ListCanvasesName, &res, ai.ListCanvasesArgs{})
	require.NoError(t, err)
	return res.Canvases
}

func names(canvases []map[string]any) []string {
	out := make([]string, 0, len(canvases))
	for _, c := range canvases {
		out = append(out, c["name"].(string))
	}
	return out
}

func TestListCanvasesOpenURL(t *testing.T) {
	canvases := listCanvases(t, testruntime.InstanceOptions{
		Files: canvasTestFiles(map[string]string{
			"dashboards/performance_overview.yaml": canvasYAML("Store Performance"),
		}),
		FrontendURL: "https://bratrax.com",
	})

	require.Len(t, canvases, 1)
	require.Equal(t, "performance_overview", canvases[0]["name"])
	require.Equal(t, "Store Performance", canvases[0]["display_name"])
	require.Equal(t, "https://bratrax.com/canvas/performance_overview", canvases[0]["open_url"])
}

func TestListCanvasesOpenURLPreservesBasePath(t *testing.T) {
	canvases := listCanvases(t, testruntime.InstanceOptions{
		Files: canvasTestFiles(map[string]string{
			"dashboards/campaign_deep_dive.yaml": canvasYAML("Attribution"),
		}),
		FrontendURL: "https://ui.rilldata.com/test-org/test-project",
	})

	require.Len(t, canvases, 1)
	require.Equal(t,
		"https://ui.rilldata.com/test-org/test-project/canvas/campaign_deep_dive",
		canvases[0]["open_url"],
	)
}

func TestListCanvasesWithoutFrontendURL(t *testing.T) {
	canvases := listCanvases(t, testruntime.InstanceOptions{
		Files: canvasTestFiles(map[string]string{
			"dashboards/performance_overview.yaml": canvasYAML("Store Performance"),
		}),
		// FrontendURL deliberately unset.
	})

	require.Len(t, canvases, 1)
	require.Equal(t, "", canvases[0]["open_url"], "open_url must be empty, not a relative path")
}

// The two filtered sets must never reach an MCP caller: the demo canvases
// belong to Rill's bundled example projects, and the superadmin ones bounce a
// client user to /developer. See rillDemoCanvasNames in canvas_list.go.
func TestListCanvasesFiltersDemoAndSuperadminCanvases(t *testing.T) {
	canvases := listCanvases(t, testruntime.InstanceOptions{
		Files: canvasTestFiles(map[string]string{
			"dashboards/campaign_deep_dive.yaml":         canvasYAML("Attribution"),
			"dashboards/margin_scorecard.yaml":           canvasYAML("Margin Scorecard"),
			"dashboards/auction_explore.yaml":            canvasYAML("Auction"),
			"dashboards/clickhouse_commits_explore.yaml": canvasYAML("Commits"),
			"dashboards/email_revenue_metric_tree.yaml":  canvasYAML("Email Revenue Metric Tree"),
			"dashboards/d2c_hybrid_metric_tree.yaml":     canvasYAML("D2C Hybrid Metric Tree"),
		}),
		FrontendURL: "https://bratrax.com",
	})

	require.Equal(t, []string{"campaign_deep_dive"}, names(canvases))
}

func TestListCanvasesSortedByName(t *testing.T) {
	canvases := listCanvases(t, testruntime.InstanceOptions{
		Files: canvasTestFiles(map[string]string{
			"dashboards/product_performance.yaml":  canvasYAML("Products"),
			"dashboards/campaign_deep_dive.yaml":   canvasYAML("Attribution"),
			"dashboards/performance_overview.yaml": canvasYAML("Store Performance"),
		}),
		FrontendURL: "https://bratrax.com",
	})

	require.Equal(t,
		[]string{"campaign_deep_dive", "performance_overview", "product_performance"},
		names(canvases),
	)
}

func TestListCanvasesEmptyProject(t *testing.T) {
	canvases := listCanvases(t, testruntime.InstanceOptions{
		Files:       canvasTestFiles(nil),
		FrontendURL: "https://bratrax.com",
	})

	require.Empty(t, canvases)
}
