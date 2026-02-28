package bratrax

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/rilldata/rill/runtime/pkg/observability"
	"go.uber.org/zap"
)

// RegisterHandlers registers Bratrax proxy routes on the given ServeMux.
// It wires up: observability → auth → reverse proxy, plus auth endpoints.
//
// Routes:
//   - POST /bratrax/auth/login    — email+password login
//   - POST /bratrax/auth/logout   — clear auth cookie
//   - GET  /bratrax/auth/me       — current user info
//   - POST /bratrax/auth/users    — create user (admin only)
//   - GET  /bratrax/.well-known/jwks.json — public JWKS
//   - GET  /bratrax/health        — local health check
//   - /bratrax/                    — catch-all proxy to Flask API
func RegisterHandlers(mux *http.ServeMux, logger *zap.Logger) error {
	cfg, err := ConfigFromEnv()
	if err != nil {
		return err
	}

	// User store
	store, err := NewUserStore(cfg.UsersDSN)
	if err != nil {
		return fmt.Errorf("bratrax: failed to create user store: %w", err)
	}

	// Auth service (ephemeral JWT issuer)
	authSvc, err := NewAuthService(store, logger)
	if err != nil {
		store.Close()
		return fmt.Errorf("bratrax: failed to create auth service: %w", err)
	}

	clientStore := NewClientStore(store.DB())

	proxy := NewProxy(cfg.TargetURL, logger)
	authMapper := NewAuthMapper(store, clientStore, authSvc.JWKS(), logger)

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

	// JWKS endpoint for token validation
	observability.MuxHandle(mux, "GET /bratrax/.well-known/jwks.json",
		observability.Middleware("bratrax", logger, authSvc.Issuer().WellKnownHandler()))

	// Health and proxy (existing routes)
	observability.MuxHandle(mux, "/bratrax/health", observability.Middleware("bratrax", logger, healthHandler))

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

	return nil
}
