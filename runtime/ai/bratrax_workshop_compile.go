package ai

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rilldata/rill/runtime"
)

const WorkshopCompileName = "workshop_compile"

type WorkshopCompile struct {
	Runtime *runtime.Runtime
}

var _ Tool[*WorkshopCompileArgs, *WorkshopCompileResult] = (*WorkshopCompile)(nil)

type WorkshopCompileArgs struct {
	Name string `json:"name" jsonschema:"Client name to compile"`
}

type WorkshopCompileResult struct {
	Compile map[string]any `json:"compile"`
}

func (t *WorkshopCompile) Spec() *mcp.Tool {
	return &mcp.Tool{
		Name:        WorkshopCompileName,
		Title:       "Workshop: Compile",
		Description: "Compile the YAML configuration for a workshop client. Generates Dataform .sqlx artifacts, Meltano config, and Rill metrics views.",
		Meta: map[string]any{
			"openai/toolInvocation/invoking": "Compiling...",
			"openai/toolInvocation/invoked":  "Compiled",
		},
	}
}

func (t *WorkshopCompile) CheckAccess(ctx context.Context) (bool, error) {
	return checkBratraxWriteAccess(ctx)
}

func (t *WorkshopCompile) Handler(ctx context.Context, args *WorkshopCompileArgs) (*WorkshopCompileResult, error) {
	path := fmt.Sprintf("/clients/%s/compile", args.Name)
	var raw map[string]any
	if err := bratraxPost(ctx, path, nil, &raw, 120*time.Second); err != nil {
		return nil, err
	}
	return &WorkshopCompileResult{Compile: raw}, nil
}
