package bratrax

import (
	"net/http"

	"go.uber.org/zap"
)

// AuthMapper resolves the Rill user/org to a Bratrax client_id.
// Currently a passthrough stub — Chunk 13 will implement:
//   - Extract JWT claims from the Rill session
//   - Look up organization_id → client_id via bratrax_clients table
//   - Set X-Bratrax-User-Id and X-Bratrax-Client-Id headers for Flask
type AuthMapper struct {
	logger *zap.Logger
}

// NewAuthMapper creates an AuthMapper.
func NewAuthMapper(logger *zap.Logger) *AuthMapper {
	return &AuthMapper{logger: logger}
}

// Middleware wraps an http.Handler with auth-mapping logic.
// Currently passes requests through unchanged.
func (a *AuthMapper) Middleware(next http.Handler) http.Handler {
	a.logger.Warn("bratrax auth middleware is a stub — all requests pass through unauthenticated")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO (Chunk 13): Extract Rill JWT, resolve org → client_id,
		// set headers for Flask API.
		next.ServeHTTP(w, r)
	})
}
