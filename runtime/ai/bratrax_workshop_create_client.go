package ai

import (
	"context"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rilldata/rill/runtime"
)

const WorkshopCreateClientName = "workshop_create_client"

type WorkshopCreateClient struct {
	Runtime *runtime.Runtime
}

var _ Tool[*WorkshopCreateClientArgs, *WorkshopCreateClientResult] = (*WorkshopCreateClient)(nil)

type WorkshopCreateClientArgs struct {
	Name     string `json:"name" jsonschema:"Client name to create"`
	Template string `json:"template,omitempty" jsonschema:"Optional template name (e.g. shopify-paid-media)"`
}

type WorkshopCreateClientResult struct {
	Result map[string]any `json:"result"`
}

func (t *WorkshopCreateClient) Spec() *mcp.Tool {
	return &mcp.Tool{
		Name:        WorkshopCreateClientName,
		Title:       "Workshop: Create Client",
		Description: "Create a new workshop client, optionally from a template. Templates provide pre-configured YAML files for common setups.",
		Meta: map[string]any{
			"openai/toolInvocation/invoking": "Creating client...",
			"openai/toolInvocation/invoked":  "Created client",
		},
	}
}

func (t *WorkshopCreateClient) CheckAccess(ctx context.Context) (bool, error) {
	return checkBratraxWriteAccess(ctx)
}

func (t *WorkshopCreateClient) Handler(ctx context.Context, args *WorkshopCreateClientArgs) (*WorkshopCreateClientResult, error) {
	body := map[string]string{"name": args.Name}
	if args.Template != "" {
		body["template"] = args.Template
	}

	var raw map[string]any
	if err := bratraxPost(ctx, "/clients", body, &raw, 30*time.Second); err != nil {
		return nil, err
	}
	return &WorkshopCreateClientResult{Result: raw}, nil
}
