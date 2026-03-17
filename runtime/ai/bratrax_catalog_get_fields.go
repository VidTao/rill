package ai

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rilldata/rill/runtime"
)

const CatalogGetFieldsName = "catalog_get_fields"

type CatalogGetFields struct {
	Runtime *runtime.Runtime
}

var _ Tool[*CatalogGetFieldsArgs, *CatalogGetFieldsResult] = (*CatalogGetFields)(nil)

type CatalogGetFieldsArgs struct {
	Tap    string `json:"tap" jsonschema:"Tap/source key (e.g. tap-shopify)"`
	Stream string `json:"stream" jsonschema:"Stream name within the tap (e.g. orders, campaigns)"`
}

type CatalogGetFieldsResult struct {
	Fields map[string]any `json:"fields"`
}

func (t *CatalogGetFields) Spec() *mcp.Tool {
	return &mcp.Tool{
		Name:        CatalogGetFieldsName,
		Title:       "Catalog: Get Fields",
		Description: "Get all fields in a specific stream of a Singer tap, with types, nullability, and metadata.",
		Meta: map[string]any{
			"openai/toolInvocation/invoking": "Fetching fields...",
			"openai/toolInvocation/invoked":  "Fetched fields",
		},
	}
}

func (t *CatalogGetFields) CheckAccess(ctx context.Context) (bool, error) {
	return checkBratraxAccess(ctx)
}

func (t *CatalogGetFields) Handler(ctx context.Context, args *CatalogGetFieldsArgs) (*CatalogGetFieldsResult, error) {
	path := fmt.Sprintf("/catalogs/%s/streams/%s", args.Tap, args.Stream)
	var raw map[string]any
	if err := bratraxGet(ctx, path, nil, &raw); err != nil {
		return nil, err
	}
	return &CatalogGetFieldsResult{Fields: raw}, nil
}
