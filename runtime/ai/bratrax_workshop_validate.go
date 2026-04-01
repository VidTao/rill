package ai

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rilldata/rill/runtime"
)

const WorkshopValidateName = "workshop_validate"

type WorkshopValidate struct {
	Runtime *runtime.Runtime
}

var _ Tool[*WorkshopValidateArgs, *WorkshopValidateResult] = (*WorkshopValidate)(nil)

type WorkshopValidateArgs struct {
	Name string `json:"name" jsonschema:"Client name to validate"`
}

type WorkshopValidateResult struct {
	Validation map[string]any `json:"validation"`
}

func (t *WorkshopValidate) Spec() *mcp.Tool {
	return &mcp.Tool{
		Name:        WorkshopValidateName,
		Title:       "Workshop: Validate",
		Description: "Validate the YAML configuration for a workshop client. Returns validation errors and warnings.",
		Meta: map[string]any{
			"openai/toolInvocation/invoking": "Validating...",
			"openai/toolInvocation/invoked":  "Validated",
		},
	}
}

func (t *WorkshopValidate) CheckAccess(ctx context.Context) (bool, error) {
	return checkBratraxWriteAccess(ctx)
}

func (t *WorkshopValidate) Handler(ctx context.Context, args *WorkshopValidateArgs) (*WorkshopValidateResult, error) {
	path := fmt.Sprintf("/clients/%s/validate", args.Name)
	var raw map[string]any
	if err := bratraxPost(ctx, path, nil, &raw, 60*time.Second); err != nil {
		return nil, err
	}
	return &WorkshopValidateResult{Validation: raw}, nil
}
