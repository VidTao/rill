package ai

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rilldata/rill/runtime"
)

const CatalogListTapsName = "catalog_list_taps"

type CatalogListTaps struct {
	Runtime *runtime.Runtime
}

var _ Tool[*CatalogListTapsArgs, *CatalogListTapsResult] = (*CatalogListTaps)(nil)

type CatalogListTapsArgs struct{}

type CatalogListTapsResult struct {
	Taps map[string]any `json:"taps"`
}

func (t *CatalogListTaps) Spec() *mcp.Tool {
	return &mcp.Tool{
		Name:        CatalogListTapsName,
		Title:       "Catalog: List Taps",
		Description: "List all available Singer taps (data sources) with their stream counts and categories.",
		Meta: map[string]any{
			"openai/toolInvocation/invoking": "Listing taps...",
			"openai/toolInvocation/invoked":  "Listed taps",
		},
	}
}

func (t *CatalogListTaps) CheckAccess(ctx context.Context) (bool, error) {
	return checkBratraxAccess(ctx)
}

func (t *CatalogListTaps) Handler(ctx context.Context, _ *CatalogListTapsArgs) (*CatalogListTapsResult, error) {
	var raw map[string]any
	if err := bratraxGet(ctx, "/catalogs", nil, &raw); err != nil {
		return nil, err
	}
	return &CatalogListTapsResult{Taps: raw}, nil
}
