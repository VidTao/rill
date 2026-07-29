package ai

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rilldata/rill/runtime"
)

const WorkshopReadKnowledgeName = "workshop_read_knowledge"

type WorkshopReadKnowledge struct {
	Runtime *runtime.Runtime
}

var _ Tool[*WorkshopReadKnowledgeArgs, *WorkshopReadKnowledgeResult] = (*WorkshopReadKnowledge)(nil)

type WorkshopReadKnowledgeArgs struct {
	Filepath string `json:"filepath" jsonschema:"File path relative to knowledge/ (e.g., index.md, profile.md, insights/roas_analysis.md)"`
}

type WorkshopReadKnowledgeResult struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func (t *WorkshopReadKnowledge) Spec() *mcp.Tool {
	return &mcp.Tool{
		Name:        WorkshopReadKnowledgeName,
		Title:       "Workshop: Read Knowledge",
		Description: "Read a file from the client knowledge base. Always read index.md first to find relevant files, then read specific files. Use before answering questions to leverage existing business context.",
		Meta: map[string]any{
			"openai/toolInvocation/invoking": "Reading knowledge...",
			"openai/toolInvocation/invoked":  "Read knowledge",
		},
	}
}

func (t *WorkshopReadKnowledge) CheckAccess(ctx context.Context) (bool, error) {
	return checkBratraxAccess(ctx)
}

func (t *WorkshopReadKnowledge) Handler(ctx context.Context, args *WorkshopReadKnowledgeArgs) (*WorkshopReadKnowledgeResult, error) {
	// Bratrax: the client is derived from the session ONLY. It is deliberately not a tool
	// argument — a caller-supplied name let one tenant read another tenant's knowledge base.
	clientName := bratraxSessionClientName(ctx, t.Runtime)
	if clientName == "" {
		return nil, fmt.Errorf("could not determine the client for this session")
	}

	// Flask route uses <path:filepath> so slashes must NOT be escaped
	path := fmt.Sprintf("/clients/%s/knowledge/%s", clientName, args.Filepath)

	var result struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := bratraxGet(ctx, path, nil, &result); err != nil {
		return nil, err
	}
	return &WorkshopReadKnowledgeResult{
		Path:    result.Path,
		Content: result.Content,
	}, nil
}
