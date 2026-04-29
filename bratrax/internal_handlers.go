package bratrax

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"os"

	"github.com/rilldata/rill/runtime/pkg/observability"
	"go.uber.org/zap"
)

// RefreshInstanceFn evicts a cached instance so the next request rebuilds it
// with fresh per-client config. The CLI's local.App.RefreshInstanceForClient
// satisfies this.
type RefreshInstanceFn func(clientDB string) error

// RegisterInternalHandlers mounts /bratrax/internal/* endpoints. These are
// intended for the Bratrax Flask backend to notify the Go runtime when
// per-client config changes (e.g. the BYOK Anthropic key was updated).
//
// Auth: requires the X-Bratrax-Internal-Secret header to match the
// BRATRAX_INTERNAL_SECRET environment variable using a constant-time compare.
// If the env var is empty, every request is rejected — the endpoint is opt-in.
//
// Routes:
//   - POST /bratrax/internal/instance/refresh
//     body: {"client_db": "<id>"} → 200 on success
func RegisterInternalHandlers(mux *http.ServeMux, refresh RefreshInstanceFn, logger *zap.Logger) {
	expectedSecret := os.Getenv("BRATRAX_INTERNAL_SECRET")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if expectedSecret == "" {
			writeJSONError(w, http.StatusForbidden, "internal endpoint disabled (BRATRAX_INTERNAL_SECRET not set)")
			return
		}
		if subtle.ConstantTimeCompare(
			[]byte(r.Header.Get("X-Bratrax-Internal-Secret")),
			[]byte(expectedSecret),
		) != 1 {
			writeJSONError(w, http.StatusUnauthorized, "invalid internal secret")
			return
		}
		var body struct {
			ClientDB string `json:"client_db"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&body); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if body.ClientDB == "" {
			writeJSONError(w, http.StatusBadRequest, "client_db required")
			return
		}
		if err := refresh(body.ClientDB); err != nil {
			logger.Warn("refresh instance failed", zap.String("client_db", body.ClientDB), zap.Error(err))
			writeJSONError(w, http.StatusInternalServerError, "refresh failed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "evicted", "client_db": body.ClientDB})
	})

	observability.MuxHandle(mux, "POST /bratrax/internal/instance/refresh",
		observability.Middleware("bratrax", logger, handler))

	logger.Info("bratrax internal handlers registered (instance refresh)")
}
