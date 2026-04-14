package bratrax

import (
	"errors"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

// EnsureInstanceFn lazily creates (or fetches) the Rill instance for a clientDB.
// Returns the instance ID. The local.App.EnsureInstanceForClient method satisfies this.
type EnsureInstanceFn func(clientDB string) (string, error)

// ErrProjectNotProvisioned is the sentinel returned by EnsureInstanceFn when the
// requested client's project directory doesn't exist on disk. The middleware
// converts it to HTTP 503. local.App.EnsureInstanceForClient wraps this error.
var ErrProjectNotProvisioned = errors.New("project not provisioned")

// InstanceRouterMiddleware wraps the runtime HTTP mux. It rewrites occurrences of
// the placeholder instance ID "default" — found either in the URL path
// (/v1/instances/default/...) or as a query parameter (?instanceId=default,
// /v1/connectors/* endpoints) — to the per-user instance ID resolved from the
// bratrax_auth JWT cookie.
//
// Non-runtime paths and requests already targeting an explicit instance ID pass
// through unchanged. Unauthenticated requests also pass through unchanged so the
// runtime can return its own error.
func InstanceRouterMiddleware(next http.Handler, authMapper *AuthMapper, ensure EnsureInstanceFn, logger *zap.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Detect whether this request needs rewriting.
		const pathPrefix = "/v1/instances/"
		var pathInstanceSeg, pathTail string
		hasPathDefault := false
		if strings.HasPrefix(r.URL.Path, pathPrefix) {
			rest := r.URL.Path[len(pathPrefix):]
			if i := strings.IndexByte(rest, '/'); i >= 0 {
				pathInstanceSeg = rest[:i]
				pathTail = rest[i:] // keeps leading slash
			} else {
				pathInstanceSeg = rest
			}
			hasPathDefault = pathInstanceSeg == "default"
		}

		// Query-param based instance ID (used by /v1/connectors/* endpoints).
		query := r.URL.Query()
		hasQueryDefault := query.Get("instanceId") == "default"

		if !hasPathDefault && !hasQueryDefault {
			next.ServeHTTP(w, r)
			return
		}

		// Resolve the authenticated user's client.
		_, client, err := authMapper.ResolveClientFromCookie(r)
		if err != nil {
			logger.Debug("instance router: client resolution failed", zap.Error(err))
			writeJSONError(w, http.StatusInternalServerError, "client lookup failed")
			return
		}
		// No valid cookie / no client — pass through unchanged to the runtime.
		// In multi-tenant mode a real empty "default" instance exists (created at
		// startup), so the runtime returns valid responses for an uninitialized
		// project. The frontend shows the welcome/login screen. The bratrax auth
		// layer (/bratrax/auth/me) handles the actual login redirect.
		if client == nil {
			next.ServeHTTP(w, r)
			return
		}

		// Lazily create the per-client Rill instance (no-op if it already exists).
		instanceID, err := ensure(client.ClickhouseDB)
		if err != nil {
			if errors.Is(err, ErrProjectNotProvisioned) {
				// Project directory doesn't exist yet — likely mid-onboarding (before
				// /onboard/activate compiles the project). Fall through to the empty
				// "default" instance so the frontend doesn't crash. Once the project
				// is compiled, the next request will find the directory and create
				// the real instance.
				logger.Debug("instance router: project not provisioned, falling back to default",
					zap.String("client", client.ClientID), zap.String("clickhouse_db", client.ClickhouseDB))
				next.ServeHTTP(w, r)
				return
			}
			logger.Error("instance router: ensure failed", zap.String("client", client.ClientID), zap.Error(err))
			writeJSONError(w, http.StatusInternalServerError, "instance setup failed")
			return
		}

		// Rewrite path-based instance ID: /v1/instances/default/foo → /v1/instances/{instanceID}/foo
		if hasPathDefault {
			newPath := pathPrefix + instanceID + pathTail
			r.URL.Path = newPath
			if r.URL.RawPath != "" {
				r.URL.RawPath = newPath
			}
		}

		// Rewrite query-param instance ID: ?instanceId=default → ?instanceId={instanceID}
		if hasQueryDefault {
			query.Set("instanceId", instanceID)
			r.URL.RawQuery = query.Encode()
		}

		next.ServeHTTP(w, r)
	})
}
