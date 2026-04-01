package ai

import (
	"context"
	"net/url"
	"strconv"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rilldata/rill/runtime"
)

const CatalogSearchFieldsName = "catalog_search_fields"

type CatalogSearchFields struct {
	Runtime *runtime.Runtime
}

var _ Tool[*CatalogSearchFieldsArgs, *CatalogSearchFieldsResult] = (*CatalogSearchFields)(nil)

type CatalogSearchFieldsArgs struct {
	Query    string `json:"query" jsonschema:"Field name substring to search for (e.g. spend, email, campaign)"`
	Type     string `json:"type,omitempty" jsonschema:"Optional type filter (STRING, INT64, FLOAT64, BOOLEAN, JSON)"`
	PageSize int    `json:"page_size,omitempty" jsonschema:"Results per page (default 20, max 100)"`
}

type CatalogSearchFieldsResult struct {
	Results map[string]any `json:"results"`
}

func (t *CatalogSearchFields) Spec() *mcp.Tool {
	return &mcp.Tool{
		Name:        CatalogSearchFieldsName,
		Title:       "Catalog: Search Fields",
		Description: "Search for fields across all Singer taps by name substring and optional type filter. Returns matching fields with their source references ($sources.*.*.field).",
		Meta: map[string]any{
			"openai/toolInvocation/invoking": "Searching fields...",
			"openai/toolInvocation/invoked":  "Searched fields",
		},
	}
}

func (t *CatalogSearchFields) CheckAccess(ctx context.Context) (bool, error) {
	return checkBratraxWriteAccess(ctx)
}

func (t *CatalogSearchFields) Handler(ctx context.Context, args *CatalogSearchFieldsArgs) (*CatalogSearchFieldsResult, error) {
	q := url.Values{}
	q.Set("q", args.Query)
	if args.Type != "" {
		q.Set("type", args.Type)
	}
	if args.PageSize > 0 {
		q.Set("limit", strconv.Itoa(args.PageSize))
	}

	var raw map[string]any
	if err := bratraxGet(ctx, "/catalogs/search", q, &raw); err != nil {
		return nil, err
	}
	return &CatalogSearchFieldsResult{Results: raw}, nil
}
