package bratrax

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/rilldata/rill/runtime/pkg/observability"
	"go.uber.org/zap"
)

// Handlers carries the constructed bratrax components so that callers (e.g. the
// CLI's local app) can register additional middleware (such as the instance router)
// against the same AuthMapper used by the /bratrax/* proxy.
type Handlers struct {
	AuthMapper  *AuthMapper
	ClientStore *ClientStore
}

// RegisterHandlers registers Bratrax proxy routes on the given ServeMux.
// It wires up: observability → auth → reverse proxy, plus auth endpoints.
//
// Routes:
//   - POST /bratrax/auth/login            — email+password login
//   - POST /bratrax/auth/logout           — clear auth cookie
//   - GET  /bratrax/auth/me               — current user info
//   - POST /bratrax/auth/signup           — self-serve signup (public, creates user + sets JWT)
//   - POST /bratrax/auth/users            — create user (admin only)
//   - GET  /bratrax/auth/clients          — list every client (super_admin only)
//   - POST /bratrax/auth/switch-client    — set active client cookie (super_admin only)
//   - GET  /bratrax/.well-known/jwks.json — public JWKS
//   - GET  /bratrax/health                — local health check
//   - /bratrax/                            — catch-all proxy to Flask API
//
// Returns the constructed Handlers so the caller can install additional middleware.
func RegisterHandlers(mux *http.ServeMux, logger *zap.Logger) (*Handlers, error) {
	cfg, err := ConfigFromEnv()
	if err != nil {
		return nil, err
	}

	// User store
	store, err := NewUserStore(cfg.UsersDSN)
	if err != nil {
		return nil, fmt.Errorf("bratrax: failed to create user store: %w", err)
	}

	// Auth service (persistent JWT issuer)
	authSvc, err := NewAuthService(store, logger, cfg.IssuerURL, cfg.AudienceURL, cfg.SecureCookie, cfg.OnlyInvitationLink)
	if err != nil {
		store.Close()
		return nil, fmt.Errorf("bratrax: failed to create auth service: %w", err)
	}

	clientStore := NewClientStore(store.DB())

	proxy := NewProxy(cfg.TargetURL, logger)
	authMapper := NewAuthMapper(store, clientStore, authSvc.JWKS(), logger, cfg.IssuerURL, cfg.AudienceURL)

	// Local health endpoint — confirms the proxy layer is alive.
	healthHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if encErr := json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"service": "bratrax-proxy",
		}); encErr != nil {
			logger.Debug("failed to write health response", zap.Error(encErr))
		}
	})

	// Auth endpoints (not behind auth middleware — they handle auth internally)
	observability.MuxHandle(mux, "POST /bratrax/auth/login",
		observability.Middleware("bratrax", logger, http.HandlerFunc(authSvc.HandleLogin)))
	observability.MuxHandle(mux, "POST /bratrax/auth/logout",
		observability.Middleware("bratrax", logger, http.HandlerFunc(authSvc.HandleLogout)))
	observability.MuxHandle(mux, "GET /bratrax/auth/me",
		observability.Middleware("bratrax", logger, http.HandlerFunc(authSvc.HandleMe)))
	observability.MuxHandle(mux, "POST /bratrax/auth/users",
		observability.Middleware("bratrax", logger, http.HandlerFunc(authSvc.HandleCreateUser)))
	observability.MuxHandle(mux, "POST /bratrax/auth/signup",
		observability.Middleware("bratrax", logger, http.HandlerFunc(authSvc.HandleSignup)))
	// Public config endpoint — read by the /signup page on mount to decide
	// whether to render the form or the invite-only message. Public on purpose.
	observability.MuxHandle(mux, "GET /bratrax/auth/config",
		observability.Middleware("bratrax", logger, http.HandlerFunc(authSvc.HandleAuthConfig)))

	// Super_admin client-switcher endpoints. Both auth themselves via the
	// JWT cookie + role check; not behind the AuthMapper proxy middleware.
	switchSvc := NewClientSwitchService(authMapper, store, clientStore, logger, cfg.SecureCookie)
	observability.MuxHandle(mux, "GET /bratrax/auth/clients",
		observability.Middleware("bratrax", logger, http.HandlerFunc(switchSvc.HandleListClients)))
	observability.MuxHandle(mux, "POST /bratrax/auth/switch-client",
		observability.Middleware("bratrax", logger, http.HandlerFunc(switchSvc.HandleSwitchClient)))

	// JWKS endpoint for token validation
	observability.MuxHandle(mux, "GET /bratrax/.well-known/jwks.json",
		observability.Middleware("bratrax", logger, authSvc.Issuer().WellKnownHandler()))

	// Apex redirect — free the plain bratrax.com domain. Authenticated users
	// go to /developer (the former landing page); everyone else is sent to the
	// Lite marketing/signup subdomain. "GET /{$}" matches ONLY exact "/" — any
	// other path falls through to the SPA catch-all registered later.
	liteURL := os.Getenv("BRATRAX_LITE_URL")
	if liteURL == "" {
		liteURL = "https://lite.bratrax.com"
	}
	apexHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, _, authErr := authMapper.ResolveClientFromCookie(r)
		if authErr != nil {
			logger.Debug("apex redirect: auth resolution error", zap.Error(authErr))
			user = nil
		}
		w.Header().Set("Cache-Control", "no-store")
		if user != nil {
			http.Redirect(w, r, "/developer", http.StatusFound)
			return
		}
		http.Redirect(w, r, liteURL, http.StatusFound)
	})
	observability.MuxHandle(mux, "GET /{$}",
		observability.Middleware("bratrax", logger, apexHandler))

	// Health and proxy (existing routes)
	observability.MuxHandle(mux, "/bratrax/health", observability.Middleware("bratrax", logger, healthHandler))

	// /bratrax/mcp — public MCP endpoint for Claude Desktop. Auths an opaque
	// per-client token and forwards into the runtime's existing per-instance
	// MCP handler. Registered before the catch-all proxy so it takes precedence.
	RegisterMCPHandler(mux, clientStore, authSvc, cfg.AudienceURL, logger)

	// Middleware chain: observability → auth → proxy (catch-all)
	proxyHandler := observability.Middleware("bratrax", logger, authMapper.Middleware(proxy))
	observability.MuxHandle(mux, "/bratrax/", proxyHandler)

	// Log DSN with credentials redacted
	redactedDSN := cfg.UsersDSN
	if parsed, parseErr := url.Parse(cfg.UsersDSN); parseErr == nil {
		parsed.User = nil
		redactedDSN = parsed.String()
	}
	logger.Info("bratrax proxy registered",
		zap.String("target", cfg.TargetURL.String()),
		zap.String("users_dsn", redactedDSN),
	)

	return &Handlers{
		AuthMapper:  authMapper,
		ClientStore: clientStore,
	}, nil
}
