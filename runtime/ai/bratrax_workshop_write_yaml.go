package ai

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rilldata/rill/runtime"
)

const WorkshopWriteYamlName = "workshop_write_yaml"

type WorkshopWriteYaml struct {
	Runtime *runtime.Runtime
}

var _ Tool[*WorkshopWriteYamlArgs, *WorkshopWriteYamlResult] = (*WorkshopWriteYaml)(nil)

type WorkshopWriteYamlArgs struct {
	Name    string `json:"name" jsonschema:"Client name"`
	File    string `json:"file" jsonschema:"File key: config, sources, ontology, or tracking_plan"`
	Content string `json:"content" jsonschema:"Full YAML content to write"`
}

type WorkshopWriteYamlResult struct {
	Saved bool   `json:"saved"`
	File  string `json:"file"`
}

func (t *WorkshopWriteYaml) Spec() *mcp.Tool {
	return &mcp.Tool{
		Name:        WorkshopWriteYamlName,
		Title:       "Workshop: Write YAML",
		Description: "Write a YAML file (config, sources, ontology, or tracking_plan) for a workshop client.",
		Meta: map[string]any{
			"openai/toolInvocation/invoking": "Writing YAML...",
			"openai/toolInvocation/invoked":  "Wrote YAML",
		},
	}
}

func (t *WorkshopWriteYaml) CheckAccess(ctx context.Context) (bool, error) {
	return checkBratraxWriteAccess(ctx)
}

func (t *WorkshopWriteYaml) Handler(ctx context.Context, args *WorkshopWriteYamlArgs) (*WorkshopWriteYamlResult, error) {
	body := map[string]string{"content": args.Content}
	path := fmt.Sprintf("/clients/%s/files/%s", args.Name, args.File)

	var result struct {
		Saved bool `json:"saved"`
	}
	if err := bratraxPut(ctx, path, body, &result); err != nil {
		return nil, err
	}
	return &WorkshopWriteYamlResult{Saved: result.Saved, File: args.File}, nil
}
