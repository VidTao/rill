package bratrax

import (
	_ "embed"
	"net/http"
)

// integrationsClaudeHTML is the public docs page for the Claude connector,
// served at /integrations/claude and linked from the OAuth metadata and the
// Claude directory listing (the directory requires a documentation URL).
//
// Unlike the other marketing pages, which are fetched from the static site repo
// on GitHub (see serveGithubHTML in bratrax.go), this one is embedded: it was
// built from the /integrations/shopify page's template so it shares the site's
// styling, but it ships with the binary rather than depending on a push to that
// repo. Moving it there later only means pointing the route at serveGithubHTML.
//
//go:embed static/integrations_claude.html
var integrationsClaudeHTML []byte

func serveIntegrationsClaude(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write(integrationsClaudeHTML)
}
