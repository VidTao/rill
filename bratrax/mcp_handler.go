package bratrax

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/rilldata/rill/runtime"
	"github.com/rilldata/rill/runtime/pkg/observability"
	"github.com/rilldata/rill/runtime/server/auth"
	"go.uber.org/zap"
)

// mcpTokenPrefix is the opaque-token prefix our /settings/mcp endpoints emit.
// All Bearer values reaching /bratrax/mcp must start with this.
const mcpTokenPrefix = "brx_mcp_"

// mcpForwardTokenTTL is the lifetime of the short-lived Rill runtime JWT we
// mint per request and forward upstream to /v1/instances/<clickhouse_db>/mcp.
// It only needs to live long enough for one MCP call to complete; we keep it
// short so a packet capture of the upstream traffic gives an attacker a tiny
// window even if our internal forwarding is somehow exposed.
const mcpForwardTokenTTL = 5 * time.Minute

// RegisterMCPHandler mounts /bratrax/mcp (and /bratrax/mcp/) — the public MCP
// endpoint clients paste into Claude Desktop. Each request:
//
//  1. Authenticates the opaque `brx_mcp_<token>` Bearer credential by
//     looking it up in rill_clients.mcp_token.
//  2. Resolves the client's `clickhouse_db` (the per-client Rill instance ID).
//  3. Mints a short-lived Rill runtime JWT (signed by our existing issuer)
//     scoped to that instance with full read+API permissions (matching
//     what an `admin`-role user gets via the cookie-based path).
//  4. Reverse-proxies the request to the in-process Rill HTTP server at
//     /v1/instances/<clickhouse_db>/mcp with the new Authorization header.
//
// This composes the existing per-client MCP server (StreamableHTTPHandler in
// runtime/server/mcp.go — already exposes all 22 MCP tools, already handles
// per-instance scoping) without re-implementing any of that machinery.
//
// Token leakage is bounded: deleting/regenerating mcp_token via /settings/mcp
// instantly invalidates the old token (next request returns 401).
func RegisterMCPHandler(mux *http.ServeMux, clientStore *ClientStore, authSvc *AuthService, runtimeAddr string, logger *zap.Logger) {
	// runtimeAddr is the local-loopback address of the Rill HTTP server (e.g.
	// "http://127.0.0.1:9009"). We resolve it once and rebuild the URL per
	// request (path varies per client).
	upstream, err := url.Parse(runtimeAddr)
	if err != nil {
		logger.Error("mcp handler: invalid runtime address; MCP route disabled",
			zap.String("addr", runtimeAddr), zap.Error(err))
		return
	}

	proxy := &httputil.ReverseProxy{
		Director: func(r *http.Request) {
			// Director sets Scheme + Host; the Path was already rewritten in
			// the handler below.
			r.URL.Scheme = upstream.Scheme
			r.URL.Host = upstream.Host
			r.Host = upstream.Host
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			logger.Warn("mcp handler: upstream proxy error",
				zap.String("path", r.URL.Path), zap.Error(err))
			writeJSONError(w, http.StatusBadGateway, "upstream MCP server unavailable")
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Extract Bearer token.
		authz := r.Header.Get("Authorization")
		if !strings.HasPrefix(authz, "Bearer ") {
			w.Header().Set("WWW-Authenticate", `Bearer realm="bratrax-mcp"`)
			writeJSONError(w, http.StatusUnauthorized, "missing or invalid Authorization header")
			return
		}
		token := strings.TrimPrefix(authz, "Bearer ")
		if !strings.HasPrefix(token, mcpTokenPrefix) {
			writeJSONError(w, http.StatusUnauthorized, "token format invalid")
			return
		}

		// 2. Resolve the client.
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		client, err := clientStore.GetByMCPToken(ctx, token)
		if err != nil {
			logger.Error("mcp handler: token lookup failed", zap.Error(err))
			writeJSONError(w, http.StatusInternalServerError, "internal error")
			return
		}
		if client == nil {
			writeJSONError(w, http.StatusUnauthorized, "token not recognized")
			return
		}
		if client.ClickhouseDB == "" {
			writeJSONError(w, http.StatusForbidden, "client has no provisioned instance")
			return
		}

		// 3. Mint a short-lived Rill runtime JWT for this instance.
		jwt, err := authSvc.Issuer().NewToken(auth.TokenOptions{
			AudienceURL:       authSvc.audienceURL,
			Subject:           "mcp:" + client.ClientID,
			TTL:               mcpForwardTokenTTL,
			SystemPermissions: runtime.AllPermissions,
			Attributes: map[string]any{
				"client_id":     client.ClientID,
				"clickhouse_db": client.ClickhouseDB,
				"source":        "bratrax_mcp",
			},
		})
		if err != nil {
			logger.Error("mcp handler: jwt mint failed",
				zap.String("client_id", client.ClientID), zap.Error(err))
			writeJSONError(w, http.StatusInternalServerError, "internal error")
			return
		}

		// 4. Rewrite the path + auth header and forward.
		// The runtime exposes both /v1/instances/{id}/mcp (full streamable-HTTP)
		// and a /v1/instances/{id}/mcp/{action} suffix path. Preserve any
		// trailing path the client sent after /bratrax/mcp.
		suffix := strings.TrimPrefix(r.URL.Path, "/bratrax/mcp")
		newPath := "/v1/instances/" + client.ClickhouseDB + "/mcp" + suffix
		r.URL.Path = newPath
		if r.URL.RawPath != "" {
			r.URL.RawPath = newPath
		}
		// Strip the bratrax bearer; replace with the freshly-minted Rill JWT.
		r.Header.Set("Authorization", "Bearer "+jwt)
		// Defense in depth: prevent any spoofed bratrax identity headers from
		// reaching the runtime via this path (it's the same trick the cookie
		// auth path uses in auth.go).
		stripBratraxHeaders(r.Header)

		proxy.ServeHTTP(w, r)
	})

	wrapped := observability.Middleware("bratrax-mcp", logger, handler)
	observability.MuxHandle(mux, "/bratrax/mcp", wrapped)
	observability.MuxHandle(mux, "/bratrax/mcp/", wrapped)

	logger.Info("bratrax /bratrax/mcp registered",
		zap.String("upstream", upstream.String()))
}
