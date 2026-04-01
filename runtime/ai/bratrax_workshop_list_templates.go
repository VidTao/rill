package ai

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rilldata/rill/runtime"
)

const WorkshopListTemplatesName = "workshop_list_templates"

type WorkshopListTemplates struct {
	Runtime *runtime.Runtime
}

var _ Tool[*WorkshopListTemplatesArgs, *WorkshopListTemplatesResult] = (*WorkshopListTemplates)(nil)

type WorkshopListTemplatesArgs struct{}

type WorkshopListTemplatesResult struct {
	Templates map[string]any `json:"templates"`
}

func (t *WorkshopListTemplates) Spec() *mcp.Tool {
	return &mcp.Tool{
		Name:        WorkshopListTemplatesName,
		Title:       "Workshop: List Templates",
		Description: "List available workshop templates. Templates provide pre-configured YAML files for common setups (e.g. shopify-paid-media).",
		Meta: map[string]any{
			"openai/toolInvocation/invoking": "Listing templates...",
			"openai/toolInvocation/invoked":  "Listed templates",
		},
	}
}

func (t *WorkshopListTemplates) CheckAccess(ctx context.Context) (bool, error) {
	return checkBratraxWriteAccess(ctx)
}

func (t *WorkshopListTemplates) Handler(ctx context.Context, _ *WorkshopListTemplatesArgs) (*WorkshopListTemplatesResult, error) {
	var raw map[string]any
	if err := bratraxGet(ctx, "/templates", nil, &raw); err != nil {
		return nil, err
	}
	return &WorkshopListTemplatesResult{Templates: raw}, nil
}
