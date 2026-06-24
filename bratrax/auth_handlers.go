package bratrax

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/MicahParks/keyfunc"
	"github.com/go-jose/go-jose/v3"
	"github.com/golang-jwt/jwt/v4"
	"github.com/rilldata/rill/runtime"
	"github.com/rilldata/rill/runtime/server/auth"
	"go.uber.org/zap"
)

const (
	bratraxCookieName  = "bratrax_auth"
	bratraxTokenTTL    = 14 * 24 * time.Hour
	maxRequestBodySize = 1 << 20 // 1 MB
	minPasswordLength  = 8
	maxPasswordLength  = 72 // bcrypt truncates beyond 72 bytes
)

// AuthService handles authentication endpoints for Bratrax.
type AuthService struct {
	store              UserStoreInterface
	issuer             *auth.Issuer
	jwks               *keyfunc.JWKS
	logger             *zap.Logger
	issuerURL          string
	audienceURL        string
	secureCookie       bool
	onlyInvitationLink bool
	allowWoocommerce   bool
}

// jwksKeyFile is the path where the dev JWT signing key is persisted.
// Tokens survive Rill restarts so users don't have to re-signup during development.
var jwksKeyFile = filepath.Join(os.Getenv("HOME"), ".bratrax", "jwt_dev_key.json")

// persistedJWKS is the on-disk format for the saved key.
type persistedJWKS struct {
	KeyID    string          `json:"key_id"`
	JwksJSON json.RawMessage `json:"jwks_json"`
}

// NewAuthService creates an AuthService with a persistent JWT issuer.
// The signing key is saved to ~/.bratrax/jwt_dev_key.json on first run
// and reused on subsequent runs so tokens survive restarts.
func NewAuthService(store UserStoreInterface, logger *zap.Logger, issuerURL, audienceURL string, secureCookie, onlyInvitationLink, allowWoocommerce bool) (*AuthService, error) {
	issuer, err := loadOrCreateIssuer(logger, issuerURL)
	if err != nil {
		return nil, err
	}

	givenKeys, err := keyfunc.NewGivenKeysFromJSON(issuer.PublicJWKS())
	if err != nil {
		return nil, err
	}
	jwks := keyfunc.NewGiven(givenKeys)

	return &AuthService{
		store:              store,
		issuer:             issuer,
		jwks:               jwks,
		logger:             logger,
		issuerURL:          issuerURL,
		audienceURL:        audienceURL,
		secureCookie:       secureCookie,
		onlyInvitationLink: onlyInvitationLink,
		allowWoocommerce:   allowWoocommerce,
	}, nil
}

// loadOrCreateIssuer loads a persisted JWKS from disk, or generates and saves a new one.
func loadOrCreateIssuer(logger *zap.Logger, issuerURL string) (*auth.Issuer, error) {
	// Try to load existing key
	data, err := os.ReadFile(jwksKeyFile)
	if err == nil {
		var saved persistedJWKS
		if jsonErr := json.Unmarshal(data, &saved); jsonErr == nil {
			issuer, issErr := auth.NewIssuer(issuerURL, saved.KeyID, saved.JwksJSON)
			if issErr == nil {
				logger.Info("loaded persistent JWT key", zap.String("file", jwksKeyFile))
				return issuer, nil
			}
			logger.Warn("failed to load saved JWT key, generating new one", zap.Error(issErr))
		}
	}

	// Generate new key (same logic as auth.NewEphemeralIssuer)
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	jwk := jose.JSONWebKey{
		Key:       rsaKey,
		Algorithm: string(jose.RS256),
		Use:       "sig",
	}
	thumb, err := jwk.Thumbprint(crypto.SHA256)
	if err != nil {
		return nil, err
	}
	jwk.KeyID = base64.URLEncoding.EncodeToString(thumb)

	jwks := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{jwk}}
	jwksJSON, err := json.Marshal(jwks)
	if err != nil {
		return nil, err
	}

	// Save to disk before creating issuer
	saved := persistedJWKS{KeyID: jwk.KeyID, JwksJSON: jwksJSON}
	savedJSON, _ := json.MarshalIndent(saved, "", "  ")
	if mkErr := os.MkdirAll(filepath.Dir(jwksKeyFile), 0700); mkErr == nil {
		if writeErr := os.WriteFile(jwksKeyFile, savedJSON, 0600); writeErr == nil {
			logger.Info("saved new JWT key", zap.String("file", jwksKeyFile))
		} else {
			logger.Warn("failed to save JWT key", zap.Error(writeErr))
		}
	}

	return auth.NewIssuer(issuerURL, jwk.KeyID, jwksJSON)
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

	// Stamp last_login_at for the CRM's login-quiet workspace detection. Best
	// effort: a failure here must never block a valid login.
	if err := s.store.UpdateLastLogin(r.Context(), user.ID); err != nil {
		s.logger.Warn("update last login failed", zap.Int("user_id", user.ID), zap.Error(err))
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
		Secure:   s.secureCookie,
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
		Secure:   s.secureCookie,
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
	if caller.Role != "admin" && caller.Role != "super_admin" {
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
	// super_admin can only be minted at signup, not via admin-create.
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

	// Auto-subscribe the new user to the beehiiv newsletter via Flask
	// (fire-and-forget; Flask owns the API key + on/off flag). Must never
	// block or fail this response.
	go notifyNewsletterSignup(user.Email, s.logger)

	writeJSON(w, http.StatusCreated, map[string]any{"user": user})
}

// HandleSignup handles POST /bratrax/auth/signup.
// Public endpoint (no authentication required) for Lite self-serve signup.
// Creates user with admin role (per-client owner) and sets JWT cookie in one step.
//
// Note: super_admin is reserved for the cross-client Bratrax internal team
// (added via the SUPERADMINS tab invite flow, see Track K). Self-serve signup
// mints a per-client admin so the new user has full control of their own
// tenant but cannot reach other clients.
func (s *AuthService) HandleSignup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Kill-switch for the public signup form. When ONLY_INVITATION_LINK=true
	// the only path to a new account is /bratrax/superadmins/signup-invite →
	// /accept-invite/<token>. Returns a `code` field so the frontend can show
	// the invite-only UI (vs a generic 403).
	if s.onlyInvitationLink {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"error": "Signup is by invitation only. Contact your administrator for a signup link.",
			"code":  "signup_invite_only",
		})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	var req struct {
		Email       string `json:"email"`
		Password    string `json:"password"`
		CompanyName string `json:"company_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" || req.CompanyName == "" {
		writeJSONError(w, http.StatusBadRequest, "email, password, and company_name are required")
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

	user, err := s.store.CreateUser(r.Context(), req.Email, req.Password, req.CompanyName, "admin", nil)
	if err != nil {
		s.logger.Error("signup: create user failed", zap.Error(err))
		writeJSONError(w, http.StatusConflict, "email already registered")
		return
	}

	// Issue JWT and set cookie (same as login)
	token, err := s.issueToken(user)
	if err != nil {
		s.logger.Error("signup: token issuance failed", zap.Error(err))
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     bratraxCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(bratraxTokenTTL.Seconds()),
		HttpOnly: true,
		Secure:   s.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})

	s.logger.Info("signup: user created", zap.String("email", req.Email), zap.Int("user_id", user.ID))
	writeJSON(w, http.StatusCreated, map[string]any{
		"token": token,
		"user":  user,
	})
}

// HandleAuthConfig serves GET /bratrax/auth/config. Public, unauthenticated —
// the frontend calls this on /signup mount to decide whether to render the
// public signup form or the invite-only message. Add new public-facing config
// flags to the response shape as needed.
func (s *AuthService) HandleAuthConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"invite_only":       s.onlyInvitationLink,
		"allow_woocommerce": s.allowWoocommerce,
	})
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

	if !claims.VerifyIssuer(s.issuerURL, true) {
		return nil, jwt.ErrTokenInvalidIssuer
	}
	if !claims.VerifyAudience(s.audienceURL, true) {
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
		"admin": user.Role == "admin" || user.Role == "super_admin",
	}
	if user.ProjectID != nil {
		attrs["project_id"] = *user.ProjectID
	}

	return s.issuer.NewToken(auth.TokenOptions{
		AudienceURL:       s.audienceURL,
		Subject:           strconv.Itoa(user.ID),
		TTL:               bratraxTokenTTL,
		SystemPermissions: permissionsForRole(user.Role),
		Attributes:        attrs,
	})
}

// permissionsForRole maps a role string to Rill permissions.
func permissionsForRole(role string) []runtime.Permission {
	switch role {
	case "super_admin", "admin":
		return runtime.AllPermissions
	case "viewer":
		return []runtime.Permission{
			runtime.ReadInstance,
			runtime.ReadObjects,
			runtime.ReadOLAP,
			runtime.ReadMetrics,
			runtime.ReadAPI,
			runtime.UseAI, // Lite paid users need Claude chat access
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
