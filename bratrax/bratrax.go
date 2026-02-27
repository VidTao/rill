package bratrax

import (
	"encoding/json"
	"net/http"

	"github.com/rilldata/rill/runtime/pkg/observability"
	"go.uber.org/zap"
)

// RegisterHandlers registers Bratrax proxy routes on the given ServeMux.
// It wires up: observability → auth → reverse proxy.
//
// Routes:
//   - /bratrax/health — local health check (not proxied)
//   - /bratrax/       — catch-all proxy to Flask API
func RegisterHandlers(mux *http.ServeMux, logger *zap.Logger) error {
	cfg, err := ConfigFromEnv()
	if err != nil {
		return err
	}

	proxy := NewProxy(cfg.TargetURL, logger)
	auth := NewAuthMapper(logger)

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

	// Middleware chain: observability → auth → proxy
	proxyHandler := observability.Middleware("bratrax", logger, auth.Middleware(proxy))

	observability.MuxHandle(mux, "/bratrax/health", observability.Middleware("bratrax", logger, healthHandler))
	observability.MuxHandle(mux, "/bratrax/", proxyHandler)

	logger.Info("bratrax proxy registered",
		zap.String("target", cfg.TargetURL.String()),
	)

	return nil
}
