package server

import (
	"net/http"
)

// mcpJSONHandler injects the Accept header that Rill's Streamable HTTP handler
// requires, then passes the SSE response through unchanged. Claude Code's
// type: "http" implements MCP Streamable HTTP and expects SSE-framed responses.
func mcpJSONHandler(inner http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			r.Header.Set("Accept", "application/json, text/event-stream")
		}
		inner.ServeHTTP(w, r)
	})
}
