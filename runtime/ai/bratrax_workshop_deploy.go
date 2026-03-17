package ai

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rilldata/rill/runtime"
)

const WorkshopDeployName = "workshop_deploy"

type WorkshopDeploy struct {
	Runtime *runtime.Runtime
}

var _ Tool[*WorkshopDeployArgs, *WorkshopDeployResult] = (*WorkshopDeploy)(nil)

type WorkshopDeployArgs struct {
	Name     string `json:"name" jsonschema:"Client name to deploy"`
	Apply    bool   `json:"apply,omitempty" jsonschema:"If true, actually deploy. If false (default), dry-run only."`
	CreatePR bool   `json:"create_pr,omitempty" jsonschema:"If true, create a pull request after deploying."`
}

type WorkshopDeployResult struct {
	Deploy map[string]any `json:"deploy"`
}

func (t *WorkshopDeploy) Spec() *mcp.Tool {
	return &mcp.Tool{
		Name:        WorkshopDeployName,
		Title:       "Workshop: Deploy",
		Description: "Deploy compiled artifacts for a workshop client. By default performs a dry-run. Set apply=true to actually deploy, and create_pr=true to create a pull request.",
		Meta: map[string]any{
			"openai/toolInvocation/invoking": "Deploying...",
			"openai/toolInvocation/invoked":  "Deployed",
		},
	}
}

func (t *WorkshopDeploy) CheckAccess(ctx context.Context) (bool, error) {
	return checkBratraxAccess(ctx)
}

func (t *WorkshopDeploy) Handler(ctx context.Context, args *WorkshopDeployArgs) (*WorkshopDeployResult, error) {
	path := fmt.Sprintf("/clients/%s/deploy", args.Name)
	body := map[string]bool{
		"apply":     args.Apply,
		"create_pr": args.CreatePR,
	}
	var raw map[string]any
	if err := bratraxPost(ctx, path, body, &raw, 180*time.Second); err != nil {
		return nil, err
	}
	return &WorkshopDeployResult{Deploy: raw}, nil
}
