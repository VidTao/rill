package bratrax

import (
	"context"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// The MCP server a workspace gets before its data is ready.
//
// An OAuth grant is issued as soon as the merchant finishes their part of
// onboarding (step `activating`), but the workspace's Rill instance only exists
// once activation has compiled it, and it only has data once the first extracts
// land: 10-15 minutes typically, longer in the worst case. Proxying to the
// runtime in that window returns a 503, which Claude shows as a broken
// connector. Instead the proxy answers with this tiny server: one tool that
// says, in words the model can relay, what is happening and when to try again.

// notReadyToolName is deliberately descriptive: it is the only tool the model
// sees, and its name alone should tell it to call this first.
const notReadyToolName = "get_bratrax_setup_status"

// notReadyMessage explains a not-yet-ready workspace for the given onboarding
// step. The tool returns it verbatim.
func notReadyMessage(step string) string {
	switch step {
	case "activating", "compiling", "deploying":
		return "Bratrax is building this store's analytics workspace. This usually takes 10 to 15 minutes " +
			"after onboarding finishes. No store or ad data can be queried yet; please try again in about 10 minutes."
	case "extracting":
		return "Bratrax is importing this store's orders and ad data. The most recent 14 days arrive first, " +
			"usually within 10 to 20 minutes, and full history over the next few hours. Please try again in about 10 minutes."
	case "error":
		return "Bratrax hit a problem while setting up this store's workspace, so no data can be queried yet. " +
			"The Bratrax team has been alerted; the merchant can also contact support@bratrax.com."
	default:
		return "This Bratrax workspace hasn't finished onboarding yet. The merchant needs to complete setup at " +
			"https://bratrax.com before data can be queried."
	}
}

// notReadyHandler serves the placeholder MCP server for one request. Stateless
// with plain JSON responses, the same transport shape the real server uses.
func notReadyHandler(step string) http.Handler {
	message := notReadyMessage(step)
	srv := mcp.NewServer(
		&mcp.Implementation{Name: "bratrax", Title: "Bratrax", Version: "1"},
		&mcp.ServerOptions{
			Instructions: "Bratrax is still setting up this workspace. Call " + notReadyToolName +
				" and relay its message to the user; no analytics tools are available until setup finishes.",
		},
	)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        notReadyToolName,
		Title:       "Check Bratrax setup status",
		Description: "Reports whether this Bratrax workspace is ready to answer questions about store and ad performance, and when it will be.",
		Annotations: &mcp.ToolAnnotations{
			Title:           "Check Bratrax setup status",
			ReadOnlyHint:    true,
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(false),
		},
	}, func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, any, error) {
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: message}}}, nil, nil
	})
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return srv },
		&mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
}

func boolPtr(b bool) *bool { return &b }
