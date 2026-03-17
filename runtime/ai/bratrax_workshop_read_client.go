package ai

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rilldata/rill/runtime"
)

const WorkshopReadClientName = "workshop_read_client"

type WorkshopReadClient struct {
	Runtime *runtime.Runtime
}

var _ Tool[*WorkshopReadClientArgs, *WorkshopReadClientResult] = (*WorkshopReadClient)(nil)

type WorkshopReadClientArgs struct {
	Name string `json:"name" jsonschema:"Client name to read"`
}

type WorkshopReadClientResult struct {
	Files map[string]string `json:"files"`
}

func (t *WorkshopReadClient) Spec() *mcp.Tool {
	return &mcp.Tool{
		Name:        WorkshopReadClientName,
		Title:       "Workshop: Read Client",
		Description: "Read all YAML files (config, sources, ontology, tracking_plan) for a workshop client.",
		Meta: map[string]any{
			"openai/toolInvocation/invoking": "Reading client files...",
			"openai/toolInvocation/invoked":  "Read client files",
		},
	}
}

func (t *WorkshopReadClient) CheckAccess(ctx context.Context) (bool, error) {
	return checkBratraxAccess(ctx)
}

func (t *WorkshopReadClient) Handler(ctx context.Context, args *WorkshopReadClientArgs) (*WorkshopReadClientResult, error) {
	var result struct {
		Files map[string]string `json:"files"`
	}
	path := fmt.Sprintf("/clients/%s/files", args.Name)
	if err := bratraxGet(ctx, path, nil, &result); err != nil {
		return nil, err
	}
	return &WorkshopReadClientResult{Files: result.Files}, nil
}
