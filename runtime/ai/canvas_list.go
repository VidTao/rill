package ai

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	runtimev1 "github.com/rilldata/rill/proto/gen/rill/runtime/v1"
	"github.com/rilldata/rill/runtime"
)

const ListCanvasesName = "list_canvases"

// Bratrax: canvases that must never be offered to an external MCP client.
//
// KEEP IN SYNC with RILL_DEMO_CANVAS_NAMES / INTERNAL_SUPERADMIN_CANVAS_NAMES in
// web-local/src/lib/bratrax/dashboardPrefs.ts — the frontend filters the tab
// strip with its own copy of these sets.
var (
	// Shipped inside the Rill binary as bundled example projects, so they can
	// surface in a tenant's resource list without belonging to that tenant.
	rillDemoCanvasNames = []string{
		"margin_scorecard",
		"auction_explore",
		"clickhouse_commits_explore",
	}

	// Internal-only. web-local/src/routes/+layout.ts redirects a non-super_admin
	// who opens one of these to /developer, so linking a client to them is a
	// dead end. An MCP caller has no way to know the role of whoever will click
	// the link, so these are always excluded.
	internalSuperadminCanvasNames = []string{
		"email_revenue_metric_tree",
		"d2c_hybrid_metric_tree",
	}
)

func hiddenFromCanvasList(name string) bool {
	return slices.Contains(rillDemoCanvasNames, name) ||
		slices.Contains(internalSuperadminCanvasNames, name)
}

type ListCanvases struct {
	Runtime *runtime.Runtime
}

var _ Tool[*ListCanvasesArgs, *ListCanvasesResult] = (*ListCanvases)(nil)

type ListCanvasesArgs struct{}

type ListCanvasesResult struct {
	Canvases []map[string]any `json:"canvases"`
}

func (t *ListCanvases) Spec() *mcp.Tool {
	return &mcp.Tool{
		Name:  ListCanvasesName,
		Title: "List Canvases",
		Description: "List the canvas dashboards in the current project. Each entry includes an " +
			"open_url that opens that dashboard — link it verbatim rather than building a URL.",
		Meta: map[string]any{
			"openai/toolInvocation/invoking": "Listing dashboards...",
			"openai/toolInvocation/invoked":  "Listed dashboards",
		},
	}
}

func (t *ListCanvases) CheckAccess(ctx context.Context) (bool, error) {
	s := GetSession(ctx)
	// Deliberately no `rill*` user-agent check (unlike CreateChart and Navigate):
	// external MCP clients such as the Slack assistant need this tool, and a
	// plain canvas URL works for them.
	return s.Claims().Can(runtime.ReadObjects), nil
}

func (t *ListCanvases) Handler(ctx context.Context, args *ListCanvasesArgs) (*ListCanvasesResult, error) {
	session := GetSession(ctx)

	ctrl, err := t.Runtime.Controller(ctx, session.InstanceID())
	if err != nil {
		return nil, err
	}

	rs, err := ctrl.List(ctx, runtime.ResourceKindCanvas, "", false)
	if err != nil {
		return nil, err
	}

	slices.SortFunc(rs, func(a, b *runtimev1.Resource) int {
		return strings.Compare(a.Meta.Name.Name, b.Meta.Name.Name)
	})

	i := 0
	for i < len(rs) {
		r := rs[i]
		r, access, err := t.Runtime.ApplySecurityPolicy(ctx, session.InstanceID(), session.Claims(), r)
		if err != nil {
			return nil, err
		}
		if !access {
			// Remove from the slice
			rs[i] = rs[len(rs)-1]
			rs[len(rs)-1] = nil
			rs = rs[:len(rs)-1]
			continue
		}
		rs[i] = r
		i++
	}

	instance, err := t.Runtime.Instance(ctx, session.InstanceID())
	if err != nil {
		return nil, fmt.Errorf("failed to get instance %q: %w", session.InstanceID(), err)
	}

	var canvases []map[string]any
	for _, r := range rs {
		c := r.GetCanvas()
		if c == nil || c.State.ValidSpec == nil {
			continue
		}
		name := r.Meta.Name.Name
		if hiddenFromCanvasList(name) {
			continue
		}

		canvases = append(canvases, map[string]any{
			"name":         name,
			"display_name": c.State.ValidSpec.DisplayName,
			"open_url":     canvasOpenURL(instance.FrontendURL, name),
		})
	}

	return &ListCanvasesResult{Canvases: canvases}, nil
}

// canvasOpenURL builds the frontend URL for a canvas dashboard. Returns "" when
// the instance has no frontend URL configured, matching generateOpenURL in
// metrics_view_query.go.
func canvasOpenURL(frontendURL, name string) string {
	if frontendURL == "" {
		return ""
	}
	u, err := url.Parse(frontendURL)
	if err != nil {
		return ""
	}
	u.Path, err = url.JoinPath(u.Path, "canvas", name)
	if err != nil {
		return ""
	}
	return u.String()
}
