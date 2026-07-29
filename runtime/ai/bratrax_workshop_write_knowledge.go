package ai

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rilldata/rill/runtime"
)

const WorkshopWriteKnowledgeName = "workshop_write_knowledge"

type WorkshopWriteKnowledge struct {
	Runtime *runtime.Runtime
}

var _ Tool[*WorkshopWriteKnowledgeArgs, *WorkshopWriteKnowledgeResult] = (*WorkshopWriteKnowledge)(nil)

type WorkshopWriteKnowledgeArgs struct {
	Category string `json:"category" jsonschema:"Knowledge category: discoveries, insights, or patterns"`
	Filename string `json:"filename" jsonschema:"Filename (e.g., roas_drop_2026-04-06.md)"`
	Content  string `json:"content" jsonschema:"Markdown content to write"`
}

type WorkshopWriteKnowledgeResult struct {
	Written bool   `json:"written"`
	Path    string `json:"path"`
}

func (t *WorkshopWriteKnowledge) Spec() *mcp.Tool {
	return &mcp.Tool{
		Name:        WorkshopWriteKnowledgeName,
		Title:       "Workshop: Write Knowledge",
		Description: "Write a discovery, insight, or pattern to the client's persistent knowledge base (stored outside the Bratrax project). This is the ONLY correct tool for filing knowledge entries — do NOT use write_file for this purpose. Auto-updates index.md and log.md.",
		Meta: map[string]any{
			"openai/toolInvocation/invoking": "Writing knowledge...",
			"openai/toolInvocation/invoked":  "Wrote knowledge",
		},
	}
}

func (t *WorkshopWriteKnowledge) CheckAccess(ctx context.Context) (bool, error) {
	// Knowledge filing is a normal user action (merchants file insights),
	// not an admin-only workshop action (compile/deploy).
	return checkBratraxAccess(ctx)
}

func (t *WorkshopWriteKnowledge) Handler(ctx context.Context, args *WorkshopWriteKnowledgeArgs) (*WorkshopWriteKnowledgeResult, error) {
	// Validate category
	switch args.Category {
	case "discoveries", "insights", "patterns":
		// valid
	default:
		return nil, fmt.Errorf("invalid category %q: must be discoveries, insights, or patterns", args.Category)
	}

	// Bratrax: the client is derived from the session ONLY. A caller-supplied name would let
	// one tenant overwrite another tenant's knowledge base.
	clientName := bratraxSessionClientName(ctx, t.Runtime)
	if clientName == "" {
		return nil, fmt.Errorf("could not determine the client for this session")
	}

	body := map[string]string{"content": args.Content}
	path := fmt.Sprintf("/clients/%s/knowledge/%s/%s", clientName, args.Category, args.Filename)

	var result struct {
		Written bool   `json:"written"`
		Path    string `json:"path"`
	}
	if err := bratraxPut(ctx, path, body, &result); err != nil {
		return nil, err
	}
	return &WorkshopWriteKnowledgeResult{
		Written: result.Written,
		Path:    result.Path,
	}, nil
}
