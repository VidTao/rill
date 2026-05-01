package bratrax

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

// ClientSwitchService handles the super_admin client-listing and -switching
// endpoints. Powers the dropdown in the top-right of the app header.
//
// Routes:
//   - GET  /bratrax/auth/clients          — list every client (super_admin only)
//   - POST /bratrax/auth/switch-client    — set bratrax_active_client cookie + last_client_id
type ClientSwitchService struct {
	authMapper   *AuthMapper
	userStore    UserStoreInterface
	clientStore  ClientStoreInterface
	logger       *zap.Logger
	secureCookie bool
}

// NewClientSwitchService wires the dependencies needed for super_admin client switching.
func NewClientSwitchService(authMapper *AuthMapper, userStore UserStoreInterface, clientStore ClientStoreInterface, logger *zap.Logger, secureCookie bool) *ClientSwitchService {
	return &ClientSwitchService{
		authMapper:   authMapper,
		userStore:    userStore,
		clientStore:  clientStore,
		logger:       logger,
		secureCookie: secureCookie,
	}
}

// HandleListClients returns every client in the system. Super_admin only.
// Used by the client-switcher dropdown in the app header.
func (s *ClientSwitchService) HandleListClients(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	user, active, err := s.authMapper.ResolveClientFromCookie(r)
	if err != nil || user == nil {
		writeJSONError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	if user.Role != "super_admin" {
		writeJSONError(w, http.StatusForbidden, "super_admin only")
		return
	}

	clients, err := s.clientStore.ListAll(r.Context())
	if err != nil {
		s.logger.Error("list clients failed", zap.Error(err))
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Trim payload to fields the dropdown actually needs.
	type listEntry struct {
		ClientID    string `json:"client_id"`
		CompanyName string `json:"company_name"`
	}
	out := make([]listEntry, 0, len(clients))
	for _, c := range clients {
		out = append(out, listEntry{ClientID: c.ClientID, CompanyName: c.CompanyName})
	}

	activeID := ""
	if active != nil {
		activeID = active.ClientID
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"clients":          out,
		"active_client_id": activeID,
	})
}

// HandleSwitchClient validates the requested client_id, sets the
// bratrax_active_client cookie, and persists last_client_id so future logins
// land on this client. Super_admin only.
//
// The frontend should hard-reload after a successful response so the Rill
// runtime stores re-init for the new instance (per-client DuckDB cache, etc.).
func (s *ClientSwitchService) HandleSwitchClient(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	user, _, err := s.authMapper.ResolveClientFromCookie(r)
	if err != nil || user == nil {
		writeJSONError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	if user.Role != "super_admin" {
		writeJSONError(w, http.StatusForbidden, "super_admin only")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	var req struct {
		ClientID string `json:"client_id"`
	}
	if decErr := json.NewDecoder(r.Body).Decode(&req); decErr != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ClientID == "" {
		writeJSONError(w, http.StatusBadRequest, "client_id is required")
		return
	}

	target, err := s.clientStore.GetByClientID(r.Context(), req.ClientID)
	if err != nil {
		s.logger.Error("switch-client: target lookup failed", zap.Error(err))
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if target == nil {
		writeJSONError(w, http.StatusNotFound, "client not found")
		return
	}

	// Persist last_client_id so the next login lands here without a switch.
	if setErr := s.userStore.SetLastClientID(r.Context(), user.ID, target.ClientID); setErr != nil {
		s.logger.Warn("switch-client: persist last_client_id failed (cookie still set)",
			zap.Error(setErr), zap.Int("user_id", user.ID), zap.String("client_id", target.ClientID))
	}

	http.SetCookie(w, &http.Cookie{
		Name:     activeClientCookieName,
		Value:    target.ClientID,
		Path:     "/",
		MaxAge:   int(bratraxTokenTTL.Seconds()),
		HttpOnly: true,
		Secure:   s.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})

	s.logger.Info("super_admin switched client",
		zap.Int("user_id", user.ID),
		zap.String("client_id", target.ClientID),
		zap.String("clickhouse_db", target.ClickhouseDB))

	writeJSON(w, http.StatusOK, map[string]any{
		"client_id":     target.ClientID,
		"company_name":  target.CompanyName,
		"clickhouse_db": target.ClickhouseDB,
	})
}
