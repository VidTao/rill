package bratrax

import (
	"context"
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
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
//
// When oauth is non-nil (BRATRAX_MCP_OAUTH_ENABLED) the endpoint also accepts
// OAuth access tokens issued by the Claude/ChatGPT connector flow (oauth.go),
// and its 401 carries the RFC 9728 resource_metadata pointer those clients need
// to discover the flow at all. The brx_mcp_ path is unchanged either way.
func RegisterMCPHandler(mux *http.ServeMux, clientStore ClientStoreInterface, authSvc *AuthService, oauth *OAuthService, runtimeAddr string, ensureReady EnsureReadyFn, logger *zap.Logger) {
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

	// unauthorized answers a request with no usable credential. With OAuth on,
	// the header names the protected-resource metadata: Claude and ChatGPT only
	// start their sign-in flow from a 401 that carries it.
	unauthorized := func(w http.ResponseWriter, message string, tokenPresent bool) {
		challenge := `Bearer realm="bratrax-mcp"`
		if oauth != nil {
			challenge += `, resource_metadata="` + oauth.ResourceMetadataURL() + `"`
			if tokenPresent {
				challenge += `, error="invalid_token"`
			}
		}
		w.Header().Set("WWW-Authenticate", challenge)
		writeJSONError(w, http.StatusUnauthorized, message)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Extract Bearer token.
		authz := r.Header.Get("Authorization")
		if !strings.HasPrefix(authz, "Bearer ") {
			unauthorized(w, "missing or invalid Authorization header", false)
			return
		}
		token := strings.TrimPrefix(authz, "Bearer ")

		// 2. Resolve the client, and what the forwarded token may do.
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		var (
			client      *Client
			permissions = runtime.AllPermissions
			subject     string
			source      = "bratrax_mcp"
			grant       *MCPGrant
		)
		switch {
		case strings.HasPrefix(token, mcpTokenPrefix):
			var err error
			client, err = clientStore.GetByMCPToken(ctx, token)
			if err != nil {
				logger.Error("mcp handler: token lookup failed", zap.Error(err))
				writeJSONError(w, http.StatusInternalServerError, "internal error")
				return
			}
			if client == nil {
				unauthorized(w, "token not recognized", true)
				return
			}
			subject = "mcp:" + client.ClientID
		case oauth != nil:
			var err error
			grant, err = oauth.ValidateAccessToken(ctx, token)
			if err != nil {
				if errors.Is(err, ErrInvalidAccessToken) {
					logger.Debug("mcp handler: oauth token rejected", zap.Error(err))
					unauthorized(w, "access token invalid or expired", true)
					return
				}
				logger.Error("mcp handler: oauth token check failed", zap.Error(err))
				writeJSONError(w, http.StatusInternalServerError, "internal error")
				return
			}
			client, err = clientStore.GetByClientID(ctx, grant.ClientID)
			if err != nil {
				logger.Error("mcp handler: client lookup failed", zap.Error(err))
				writeJSONError(w, http.StatusInternalServerError, "internal error")
				return
			}
			// The workspace was deleted, or its slug changed under the token.
			if client == nil || client.ClickhouseDB != grant.ClickhouseDB {
				unauthorized(w, "the connected workspace no longer exists", true)
				return
			}
			// The user's role bounds what the runtime lets this token do, as it
			// does in the app. No "role" attribute is forwarded, though: the
			// workshop tools gate on it (bratrax_util.go), and an MCP client has
			// no business reaching them whatever the user's role.
			permissions = permissionsForRole(grant.Role)
			subject = "mcp-oauth:" + strconv.Itoa(grant.UserID)
			source = "bratrax_mcp_oauth"
		default:
			unauthorized(w, "token format invalid", true)
			return
		}
		if client.ClickhouseDB == "" {
			writeJSONError(w, http.StatusForbidden, "client has no provisioned instance")
			return
		}

		// 2a. OAuth grants exist from the moment the merchant finishes their part
		// of onboarding, which is before the workspace has data (or, early on,
		// even a Rill project). Answer with an explanation instead of proxying.
		if grant != nil {
			step, err := oauth.store.OnboardingStep(ctx, client.ClientID)
			if err != nil {
				logger.Error("mcp handler: onboarding step lookup failed", zap.Error(err))
				writeJSONError(w, http.StatusInternalServerError, "internal error")
				return
			}
			if step != "" && step != "ready" {
				notReadyHandler(step).ServeHTTP(w, r)
				return
			}
		}

		// 2b. Ensure the client's instance is registered and its controller is
		// ready before forwarding. The MCP inner route (/v1/instances/{id}/mcp)
		// bypasses InstanceRouterMiddleware, so without this a cold instance —
		// e.g. right after a Rill restart, before any browser opened this client —
		// makes the runtime return a hard 400 "no server available". ensureReady
		// registers the instance and waits (bounded) for it to become serveable;
		// on timeout we return a retryable 503 rather than the opaque 400.
		if ensureReady != nil {
			key, model, keyErr := clientStore.GetAnthropicSettings(ctx, client.ClickhouseDB)
			if keyErr != nil {
				// Non-fatal: the instance still starts without a BYOK key.
				logger.Debug("mcp handler: anthropic key lookup failed; continuing without",
					zap.String("client_id", client.ClientID), zap.Error(keyErr))
				key, model = "", ""
			}
			// Use the raw request context so ensureReady's own bounded timeout
			// governs the wait, not the short token-lookup deadline above.
			if err := ensureReady(r.Context(), client.ClickhouseDB, key, model); err != nil {
				if grant != nil && errors.Is(err, ErrProjectNotProvisioned) {
					// Onboarding says ready but the project isn't on disk: the
					// same situation as step 2a from the merchant's view.
					notReadyHandler("error").ServeHTTP(w, r)
					return
				}
				logger.Warn("mcp handler: instance not ready",
					zap.String("client_id", client.ClientID),
					zap.String("clickhouse_db", client.ClickhouseDB), zap.Error(err))
				w.Header().Set("Retry-After", "5")
				writeJSONError(w, http.StatusServiceUnavailable, "instance warming up, retry shortly")
				return
			}
		}

		// 3. Mint a short-lived Rill runtime JWT for this instance.
		jwt, err := authSvc.Issuer().NewToken(auth.TokenOptions{
			AudienceURL:       authSvc.audienceURL,
			Subject:           subject,
			TTL:               mcpForwardTokenTTL,
			SystemPermissions: permissions,
			Attributes: map[string]any{
				"client_id":     client.ClientID,
				"clickhouse_db": client.ClickhouseDB,
				"source":        source,
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
