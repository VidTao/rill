package ai

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rilldata/rill/runtime"
)

const WorkshopListClientsName = "workshop_list_clients"

type WorkshopListClients struct {
	Runtime *runtime.Runtime
}

var _ Tool[*WorkshopListClientsArgs, *WorkshopListClientsResult] = (*WorkshopListClients)(nil)

type WorkshopListClientsArgs struct{}

type WorkshopListClientsResult struct {
	Clients map[string]any `json:"clients"`
}

func (t *WorkshopListClients) Spec() *mcp.Tool {
	return &mcp.Tool{
		Name:        WorkshopListClientsName,
		Title:       "Workshop: List Clients",
		Description: "List all workshop clients and their configuration status (which YAML files exist).",
		Meta: map[string]any{
			"openai/toolInvocation/invoking": "Listing clients...",
			"openai/toolInvocation/invoked":  "Listed clients",
		},
	}
}

func (t *WorkshopListClients) CheckAccess(ctx context.Context) (bool, error) {
	return checkBratraxWriteAccess(ctx)
}

func (t *WorkshopListClients) Handler(ctx context.Context, _ *WorkshopListClientsArgs) (*WorkshopListClientsResult, error) {
	var raw map[string]any
	if err := bratraxGet(ctx, "/clients", nil, &raw); err != nil {
		return nil, err
	}
	return &WorkshopListClientsResult{Clients: raw}, nil
}
