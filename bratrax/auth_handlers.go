package bratrax

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"
	"strconv"
	"time"

	"github.com/MicahParks/keyfunc"
	"github.com/golang-jwt/jwt/v4"
	"github.com/rilldata/rill/runtime"
	"github.com/rilldata/rill/runtime/server/auth"
	"go.uber.org/zap"
)

const (
	bratraxCookieName   = "bratrax_auth"
	bratraxTokenTTL     = 24 * time.Hour
	bratraxIssuerURL    = "http://localhost:9009/bratrax"
	bratraxAudienceURL  = "http://localhost:9009"
	bratraxSecureCookie = false // Set true in production (HTTPS)
	maxRequestBodySize  = 1 << 20 // 1 MB
	minPasswordLength   = 8
	maxPasswordLength   = 72 // bcrypt truncates beyond 72 bytes
)

// AuthService handles authentication endpoints for Bratrax.
type AuthService struct {
	store  UserStoreInterface
	issuer *auth.Issuer
	jwks   *keyfunc.JWKS
	logger *zap.Logger
}

// NewAuthService creates an AuthService with an ephemeral JWT issuer.
func NewAuthService(store UserStoreInterface, logger *zap.Logger) (*AuthService, error) {
	issuer, err := auth.NewEphemeralIssuer(bratraxIssuerURL)
	if err != nil {
		return nil, err
	}

	givenKeys, err := keyfunc.NewGivenKeysFromJSON(issuer.PublicJWKS())
	if err != nil {
		return nil, err
	}
	jwks := keyfunc.NewGiven(givenKeys)

	return &AuthService{
		store:  store,
		issuer: issuer,
		jwks:   jwks,
		logger: logger,
	}, nil
}

// Issuer returns the underlying JWT issuer (used for JWKS endpoint).
func (s *AuthService) Issuer() *auth.Issuer {
	return s.issuer
}

// JWKS returns the JWKS keyfunc for token validation (used by AuthMapper).
func (s *AuthService) JWKS() *keyfunc.JWKS {
	return s.jwks
}

// HandleLogin handles POST /bratrax/auth/login.
func (s *AuthService) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" {
		writeJSONError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	user, err := s.store.Authenticate(r.Context(), req.Email, req.Password)
	if err != nil {
		s.logger.Error("authentication error", zap.Error(err))
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if user == nil {
		writeJSONError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := s.issueToken(user)
	if err != nil {
		s.logger.Error("token issuance failed", zap.Error(err))
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     bratraxCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(bratraxTokenTTL.Seconds()),
		HttpOnly: true,
		Secure:   bratraxSecureCookie,
		SameSite: http.SameSiteLaxMode,
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"token": token,
		"user":  user,
	})
}

// HandleLogout handles POST /bratrax/auth/logout.
func (s *AuthService) HandleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     bratraxCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   bratraxSecureCookie,
		SameSite: http.SameSiteLaxMode,
	})

	writeJSON(w, http.StatusOK, map[string]string{"status": "logged out"})
}

// HandleMe handles GET /bratrax/auth/me.
func (s *AuthService) HandleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	user, err := s.authenticateRequest(r)
	if err != nil {
		s.logger.Debug("auth/me: authentication failed", zap.Error(err))
		writeJSONError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"user": user})
}

// HandleCreateUser handles POST /bratrax/auth/users.
func (s *AuthService) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	caller, err := s.authenticateRequest(r)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	if caller.Role != "admin" {
		writeJSONError(w, http.StatusForbidden, "admin access required")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	var req struct {
		Email     string  `json:"email"`
		Password  string  `json:"password"`
		Name      string  `json:"name"`
		Role      string  `json:"role"`
		ProjectID *string `json:"project_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" || req.Name == "" {
		writeJSONError(w, http.StatusBadRequest, "email, password, and name are required")
		return
	}
	if _, err := mail.ParseAddress(req.Email); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid email format")
		return
	}
	if len(req.Password) < minPasswordLength {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("password must be at least %d characters", minPasswordLength))
		return
	}
	if len(req.Password) > maxPasswordLength {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("password must be at most %d characters", maxPasswordLength))
		return
	}
	if req.Role == "" {
		req.Role = "viewer"
	}
	if req.Role != "admin" && req.Role != "viewer" {
		writeJSONError(w, http.StatusBadRequest, "role must be 'admin' or 'viewer'")
		return
	}
	if req.Role == "viewer" && req.ProjectID == nil {
		writeJSONError(w, http.StatusBadRequest, "project_id is required for viewer role")
		return
	}

	user, err := s.store.CreateUser(r.Context(), req.Email, req.Password, req.Name, req.Role, req.ProjectID)
	if err != nil {
		s.logger.Error("create user failed", zap.Error(err))
		writeJSONError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"user": user})
}

// authenticateRequest extracts and validates the JWT from the auth cookie,
// then resolves the user from the store.
func (s *AuthService) authenticateRequest(r *http.Request) (*User, error) {
	cookie, err := r.Cookie(bratraxCookieName)
	if err != nil {
		return nil, err
	}

	claims := &jwt.RegisteredClaims{}
	_, err = jwt.ParseWithClaims(cookie.Value, claims, s.jwks.Keyfunc,
		jwt.WithValidMethods([]string{"RS256"}),
	)
	if err != nil {
		return nil, err
	}

	if !claims.VerifyIssuer(bratraxIssuerURL, true) {
		return nil, jwt.ErrTokenInvalidIssuer
	}
	if !claims.VerifyAudience(bratraxAudienceURL, true) {
		return nil, jwt.ErrTokenInvalidAudience
	}

	userID, err := strconv.Atoi(claims.Subject)
	if err != nil {
		return nil, err
	}

	user, err := s.store.GetByID(r.Context(), userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

// issueToken creates a signed JWT for the given user.
func (s *AuthService) issueToken(user *User) (string, error) {
	attrs := map[string]any{
		"email": user.Email,
		"name":  user.Name,
		"role":  user.Role,
		"admin": user.Role == "admin",
	}
	if user.ProjectID != nil {
		attrs["project_id"] = *user.ProjectID
	}

	return s.issuer.NewToken(auth.TokenOptions{
		AudienceURL:       bratraxAudienceURL,
		Subject:           strconv.Itoa(user.ID),
		TTL:               bratraxTokenTTL,
		SystemPermissions: permissionsForRole(user.Role),
		Attributes:        attrs,
	})
}

// permissionsForRole maps a role string to Rill permissions.
func permissionsForRole(role string) []runtime.Permission {
	switch role {
	case "admin":
		return runtime.AllPermissions
	case "viewer":
		return []runtime.Permission{
			runtime.ReadInstance,
			runtime.ReadObjects,
			runtime.ReadOLAP,
			runtime.ReadMetrics,
			runtime.ReadAPI,
		}
	default:
		return nil
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// Best-effort; headers already sent.
		_ = err
	}
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
