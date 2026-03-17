package ai

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rilldata/rill/runtime"
)

const CatalogGetStreamsName = "catalog_get_streams"

type CatalogGetStreams struct {
	Runtime *runtime.Runtime
}

var _ Tool[*CatalogGetStreamsArgs, *CatalogGetStreamsResult] = (*CatalogGetStreams)(nil)

type CatalogGetStreamsArgs struct {
	Tap string `json:"tap" jsonschema:"Tap/source key (e.g. tap-shopify, tap-facebook, webhook-leadbyte)"`
}

type CatalogGetStreamsResult struct {
	Streams map[string]any `json:"streams"`
}

func (t *CatalogGetStreams) Spec() *mcp.Tool {
	return &mcp.Tool{
		Name:        CatalogGetStreamsName,
		Title:       "Catalog: Get Streams",
		Description: "Get all streams in a Singer tap, with field counts, key properties, and replication methods.",
		Meta: map[string]any{
			"openai/toolInvocation/invoking": "Fetching streams...",
			"openai/toolInvocation/invoked":  "Fetched streams",
		},
	}
}

func (t *CatalogGetStreams) CheckAccess(ctx context.Context) (bool, error) {
	return checkBratraxAccess(ctx)
}

func (t *CatalogGetStreams) Handler(ctx context.Context, args *CatalogGetStreamsArgs) (*CatalogGetStreamsResult, error) {
	path := fmt.Sprintf("/catalogs/%s/streams", args.Tap)
	var raw map[string]any
	if err := bratraxGet(ctx, path, nil, &raw); err != nil {
		return nil, err
	}
	return &CatalogGetStreamsResult{Streams: raw}, nil
}
