package ai

import (
	"context"
	_ "embed"
	"encoding/json"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Bratrax: how tools look to external MCP clients (Claude, ChatGPT, the Slack
// assistant). Kept in one place because this is exactly what a directory
// reviewer reads.

// externalToolAnnotations is the MCP annotation block for every tool an
// external client can reach. Claude's directory rejects tools without a title
// and a read-only or destructive hint, so TestExternalMCPToolSurface fails if
// an externally reachable tool is missing here.
//
// Titles are written for merchants, who see them in Claude's tool-call UI.
var externalToolAnnotations = map[string]*mcp.ToolAnnotations{
	ListMetricsViewsName:        readOnlyAnnotations("List available metrics"),
	GetMetricsViewName:          readOnlyAnnotations("Describe a metrics view"),
	QueryMetricsViewSummaryName: readOnlyAnnotations("Summarize a metrics view"),
	QueryMetricsViewName:        readOnlyAnnotations("Query store and ad metrics"),
	ListCanvasesName:            readOnlyAnnotations("List dashboards"),
	GetCanvasName:               readOnlyAnnotations("Describe a dashboard"),
	WorkshopReadKnowledgeName:   readOnlyAnnotations("Read saved business notes"),
	ShowChartName:               readOnlyAnnotations("Show chart"),
	// Adds a note to the workspace's knowledge base. Never edits or deletes
	// existing notes, hence not destructive.
	WorkshopWriteKnowledgeName: {
		Title:           "Save a business note",
		ReadOnlyHint:    false,
		DestructiveHint: boolPtr(false),
		IdempotentHint:  false,
		OpenWorldHint:   boolPtr(false),
	},
}

func readOnlyAnnotations(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		Title:           title,
		ReadOnlyHint:    true,
		DestructiveHint: boolPtr(false),
		IdempotentHint:  true,
		OpenWorldHint:   boolPtr(false),
	}
}

func boolPtr(b bool) *bool { return &b }

// externalMCPSpec returns the tool definition an MCP client is shown: the
// annotations above, and an input schema with $ref/$defs inlined.
//
// The inlining matters. query_metrics_view's time_range, where and having are
// $refs into a $defs block, and Anthropic's tool input_schema does not resolve
// $ref: the model then never learns those parameters are objects, sends them as
// strings, and every call is rejected until it gives up. The Slack assistant
// hit exactly this (bratrax/docs/SLACK_ASSISTANT.md, "The $ref / $defs trap")
// and works around it client-side; doing it here fixes it for every client.
func externalMCPSpec(spec *mcp.Tool) *mcp.Tool {
	out := *spec
	if a, ok := externalToolAnnotations[spec.Name]; ok && out.Annotations == nil {
		out.Annotations = a
	}
	if s, ok := spec.InputSchema.(*jsonschema.Schema); ok && s != nil {
		if flat, err := inlineSchemaRefs(s, schemaRefMaxDepth); err == nil {
			out.InputSchema = flat
		}
	}
	return &out
}

// schemaRefMaxDepth bounds $ref inlining. Expression is self-referential (a
// condition's operands are expressions), so there is no finite expansion; at
// this depth the remaining reference becomes a permissive object. Depth 4 keeps
// the shapes models actually use and the schemas around 17 KB, the same
// trade-off the Slack assistant settled on.
const schemaRefMaxDepth = 4

// inlineSchemaRefs returns schema with local "#/$defs/..." references inlined.
// Schemas without $defs are returned unchanged.
func inlineSchemaRefs(s *jsonschema.Schema, maxDepth int) (*jsonschema.Schema, error) {
	raw, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, err
	}
	defs, _ := root["$defs"].(map[string]any)
	if len(defs) == 0 {
		if defs, _ = root["definitions"].(map[string]any); len(defs) == 0 {
			return s, nil
		}
	}
	flat, _ := inlineRefs(root, defs, 0, maxDepth).(map[string]any)
	delete(flat, "$defs")
	delete(flat, "definitions")

	raw, err = json.Marshal(flat)
	if err != nil {
		return nil, err
	}
	var out jsonschema.Schema
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func inlineRefs(node any, defs map[string]any, depth, maxDepth int) any {
	switch v := node.(type) {
	case map[string]any:
		if ref, ok := v["$ref"].(string); ok {
			name := strings.TrimPrefix(strings.TrimPrefix(ref, "#/$defs/"), "#/definitions/")
			target, found := defs[name].(map[string]any)
			if !found || name == ref {
				return v // not a local definition; leave it alone
			}
			if depth >= maxDepth {
				stub := map[string]any{"type": "object"}
				if d, ok := v["description"]; ok {
					stub["description"] = d
				}
				return stub
			}
			// Copy the definition, letting the referencing node's own keywords
			// (typically a description) take precedence.
			merged := make(map[string]any, len(target)+len(v))
			for k, val := range target {
				merged[k] = val
			}
			for k, val := range v {
				if k != "$ref" {
					merged[k] = val
				}
			}
			return inlineRefs(merged, defs, depth+1, maxDepth)
		}
		out := make(map[string]any, len(v))
		for k, val := range v {
			if k == "$defs" || k == "definitions" {
				continue
			}
			out[k] = inlineRefs(val, defs, depth, maxDepth)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, val := range v {
			out[i] = inlineRefs(val, defs, depth, maxDepth)
		}
		return out
	default:
		return v
	}
}

// chartWidgetHTML is the MCP App that renders show_chart results. Hand-written
// and dependency-free: it draws SVG itself and talks to the host over the MCP
// Apps postMessage protocol, so it needs no build step and no CSP allowances.
//
//go:embed mcpwidget/chart.html
var chartWidgetHTML string

// mcpAppMIMEType marks a resource as an MCP App (SEP-1865).
const mcpAppMIMEType = "text/html;profile=mcp-app"

// registerChartWidget adds the ui:// resource show_chart points at.
func registerChartWidget(srv *mcp.Server) {
	srv.AddResource(&mcp.Resource{
		URI:         showChartWidgetURI,
		Name:        "bratrax-chart",
		Title:       "Bratrax chart",
		Description: "Interactive chart for show_chart results",
		MIMEType:    mcpAppMIMEType,
	}, func(context.Context, *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{
			URI:      showChartWidgetURI,
			MIMEType: mcpAppMIMEType,
			Text:     chartWidgetHTML,
			Meta:     mcp.Meta{"ui": map[string]any{"prefersBorder": true}},
		}}}, nil
	})
}
