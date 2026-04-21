package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rilldata/rill/runtime"
)

const WorkshopWriteKnowledgeName = "workshop_write_knowledge"

type WorkshopWriteKnowledge struct {
	Runtime *runtime.Runtime
}

var _ Tool[*WorkshopWriteKnowledgeArgs, *WorkshopWriteKnowledgeResult] = (*WorkshopWriteKnowledge)(nil)

type WorkshopWriteKnowledgeArgs struct {
	Name     string `json:"name,omitempty" jsonschema:"Client name (auto-resolved if not provided)"`
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
		Description: "Write a discovery, insight, or pattern to the client's persistent knowledge base (stored outside the Rill project). This is the ONLY correct tool for filing knowledge entries — do NOT use write_file for this purpose. Auto-updates index.md and log.md.",
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

	// Resolve client name: explicit arg > session claims > fallback
	clientName := args.Name
	if clientName == "" || clientName == "client" {
		clientName = getBratraxClientID(ctx)
	}
	if clientName == "" {
		s := GetSession(ctx)
		if s != nil {
			inst, err := t.Runtime.Instance(ctx, s.InstanceID())
			if err == nil && inst != nil {
				clientName = inst.ProjectDisplayName
				clientName = strings.TrimSuffix(clientName, " Analytics")
				clientName = strings.ToLower(clientName)
			}
		}
	}
	if clientName == "" {
		return nil, fmt.Errorf("could not determine client name — pass it explicitly or ensure the session has a client_id")
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
