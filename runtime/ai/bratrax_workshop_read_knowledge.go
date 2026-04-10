package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rilldata/rill/runtime"
)

const WorkshopReadKnowledgeName = "workshop_read_knowledge"

type WorkshopReadKnowledge struct {
	Runtime *runtime.Runtime
}

var _ Tool[*WorkshopReadKnowledgeArgs, *WorkshopReadKnowledgeResult] = (*WorkshopReadKnowledge)(nil)

type WorkshopReadKnowledgeArgs struct {
	Name     string `json:"name,omitempty" jsonschema:"Client name (auto-resolved if not provided)"`
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
	// Resolve client name: explicit arg > session claims > fallback
	clientName := args.Name
	if clientName == "" || clientName == "client" {
		clientName = getBratraxClientID(ctx)
	}
	if clientName == "" {
		// Fall back to instance display name (from rill.yaml)
		s := GetSession(ctx)
		if s != nil {
			inst, err := t.Runtime.Instance(ctx, s.InstanceID())
			if err == nil && inst != nil {
				clientName = inst.ProjectDisplayName
				// Strip " Analytics" suffix if present (e.g., "Ziva Analytics" → "Ziva")
				clientName = strings.TrimSuffix(clientName, " Analytics")
				clientName = strings.ToLower(clientName)
			}
		}
	}
	if clientName == "" {
		return nil, fmt.Errorf("could not determine client name — pass it explicitly or ensure the session has a client_id")
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
