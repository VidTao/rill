package ai

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rilldata/rill/runtime"
)

const WorkshopGetCatalogsName = "workshop_get_catalogs"

type WorkshopGetCatalogs struct {
	Runtime *runtime.Runtime
}

var _ Tool[*WorkshopGetCatalogsArgs, *WorkshopGetCatalogsResult] = (*WorkshopGetCatalogs)(nil)

type WorkshopGetCatalogsArgs struct{}

type WorkshopGetCatalogsResult struct {
	Catalogs map[string]any `json:"catalogs"`
}

func (t *WorkshopGetCatalogs) Spec() *mcp.Tool {
	return &mcp.Tool{
		Name:        WorkshopGetCatalogsName,
		Title:       "Workshop: Get Catalogs",
		Description: "Get all source catalogs (Singer taps) available for the workshop, including streams and fields.",
		Meta: map[string]any{
			"openai/toolInvocation/invoking": "Fetching catalogs...",
			"openai/toolInvocation/invoked":  "Fetched catalogs",
		},
	}
}

func (t *WorkshopGetCatalogs) CheckAccess(ctx context.Context) (bool, error) {
	return checkBratraxWriteAccess(ctx)
}

func (t *WorkshopGetCatalogs) Handler(ctx context.Context, _ *WorkshopGetCatalogsArgs) (*WorkshopGetCatalogsResult, error) {
	var raw map[string]any
	if err := bratraxGet(ctx, "/catalogs", nil, &raw); err != nil {
		return nil, err
	}
	return &WorkshopGetCatalogsResult{Catalogs: raw}, nil
}
