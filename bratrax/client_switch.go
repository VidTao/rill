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

// HandleListClients returns the clients available to the caller's switcher.
//
//   - super_admin: every client in the system (existing behavior).
//   - multi-store admin/viewer (multi_client_id != NULL): only siblings
//     under the same parent multi_client.
//   - single-store users: 403 — the switcher should not be rendered for them.
//
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

	// Trim payload to fields the dropdown actually needs. AdminEmail is
	// surfaced so the super_admin can identify each client by its owner.
	type listEntry struct {
		ClientID    string  `json:"client_id"`
		CompanyName string  `json:"company_name"`
		AdminEmail  *string `json:"admin_email,omitempty"`
	}

	var out []listEntry
	switch {
	case user.Role == "super_admin":
		clients, listErr := s.clientStore.ListAllWithAdminEmail(r.Context())
		if listErr != nil {
			s.logger.Error("list clients failed", zap.Error(listErr))
			writeJSONError(w, http.StatusInternalServerError, "internal error")
			return
		}
		out = make([]listEntry, 0, len(clients))
		for _, c := range clients {
			out = append(out, listEntry{
				ClientID:    c.ClientID,
				CompanyName: c.CompanyName,
				AdminEmail:  c.AdminEmail,
			})
		}
	case user.MultiClientID != nil && *user.MultiClientID != "":
		siblings, listErr := s.clientStore.ListByMultiClientID(r.Context(), *user.MultiClientID)
		if listErr != nil {
			s.logger.Error("list multi-client siblings failed", zap.Error(listErr))
			writeJSONError(w, http.StatusInternalServerError, "internal error")
			return
		}
		out = make([]listEntry, 0, len(siblings))
		for _, c := range siblings {
			out = append(out, listEntry{
				ClientID:    c.ClientID,
				CompanyName: c.CompanyName,
			})
		}
	default:
		writeJSONError(w, http.StatusForbidden, "client switcher not available")
		return
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
// land on this client.
//
// Authorization:
//   - super_admin: may switch to any client (existing behavior, unchanged).
//   - multi-store admin/viewer (multi_client_id != NULL): may switch only to
//     a client that shares the same parent multi_client_id; foreign targets
//     return 403.
//   - single-store users: 403 — they have no switcher.
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

	// Authorize: super_admins roam freely; multi-store users are scoped to
	// siblings; everyone else is rejected. Keep these branches independent so
	// future role tweaks don't accidentally widen the super_admin path.
	switch {
	case user.Role == "super_admin":
		// allowed
	case user.MultiClientID != nil && *user.MultiClientID != "":
		if target.MultiClientID == nil || *target.MultiClientID != *user.MultiClientID {
			writeJSONError(w, http.StatusForbidden, "client not in your multi-store")
			return
		}
	default:
		writeJSONError(w, http.StatusForbidden, "client switching not allowed")
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

	s.logger.Info("user switched client",
		zap.Int("user_id", user.ID),
		zap.String("role", user.Role),
		zap.String("client_id", target.ClientID),
		zap.String("clickhouse_db", target.ClickhouseDB))

	writeJSON(w, http.StatusOK, map[string]any{
		"client_id":     target.ClientID,
		"company_name":  target.CompanyName,
		"clickhouse_db": target.ClickhouseDB,
	})
}
