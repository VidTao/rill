package bratrax

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/MicahParks/keyfunc"
	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
)

// bratraxClaims is an alias for RegisteredClaims used during JWT validation.
// The token also contains an "attr" field (set by AuthService.issueToken),
// but the middleware re-fetches user data from the store for freshness.
type bratraxClaims = jwt.RegisteredClaims

// headersToStrip lists request headers that the middleware must remove
// before forwarding, to prevent spoofing from untrusted clients.
var headersToStrip = []string{
	"X-Bratrax-User-Id",
	"X-Bratrax-Client-Id",
	"X-Bratrax-Org-Id",
	"X-Bratrax-User-Email",
	"X-Bratrax-User-Role",
	"user-id",
}

// AuthMapper validates the bratrax_auth JWT cookie, resolves the user and
// client, then sets identity headers for the downstream Flask API.
type AuthMapper struct {
	userStore   UserStoreInterface
	clientStore ClientStoreInterface
	jwks        *keyfunc.JWKS
	logger      *zap.Logger
	issuerURL   string
	audienceURL string
}

// NewAuthMapper creates an AuthMapper with all required dependencies.
func NewAuthMapper(userStore UserStoreInterface, clientStore ClientStoreInterface, jwks *keyfunc.JWKS, logger *zap.Logger, issuerURL, audienceURL string) *AuthMapper {
	return &AuthMapper{
		userStore:   userStore,
		clientStore: clientStore,
		jwks:        jwks,
		logger:      logger,
		issuerURL:   issuerURL,
		audienceURL: audienceURL,
	}
}

// Middleware wraps an http.Handler with JWT authentication and header injection.
func (a *AuthMapper) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Public paths bypass JWT auth entirely. The accept-invite flow uses
		// the invitation token as the auth credential — invitee has no JWT yet.
		if strings.HasPrefix(r.URL.Path, "/bratrax/invitations/") {
			stripBratraxHeaders(r.Header)
			next.ServeHTTP(w, r)
			return
		}

		// 1. Extract cookie
		cookie, err := r.Cookie(bratraxCookieName)
		if err != nil {
			// Also check Authorization header as fallback (for popup OAuth flows
			// where cookies may not be sent due to browser restrictions)
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				cookie = &http.Cookie{Value: strings.TrimPrefix(authHeader, "Bearer ")}
			} else {
				writeJSONError(w, http.StatusUnauthorized, "authentication required")
				return
			}
		}

		// 2. Parse and validate JWT
		claims := &bratraxClaims{}
		_, err = jwt.ParseWithClaims(cookie.Value, claims, a.jwks.Keyfunc,
			jwt.WithValidMethods([]string{"RS256"}),
		)
		if err != nil {
			a.logger.Debug("jwt validation failed", zap.Error(err))
			writeJSONError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		if !claims.VerifyIssuer(a.issuerURL, true) {
			writeJSONError(w, http.StatusUnauthorized, "invalid token issuer")
			return
		}
		if !claims.VerifyAudience(a.audienceURL, true) {
			writeJSONError(w, http.StatusUnauthorized, "invalid token audience")
			return
		}

		// 3. Resolve user from subject claim
		userID, err := strconv.Atoi(claims.Subject)
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "invalid token subject")
			return
		}

		user, err := a.userStore.GetByID(r.Context(), userID)
		if err != nil {
			a.logger.Error("user lookup failed", zap.Error(err))
			writeJSONError(w, http.StatusInternalServerError, "internal error")
			return
		}
		if user == nil {
			writeJSONError(w, http.StatusUnauthorized, "user not found")
			return
		}

		// 4. Resolve client via rill_users.client_id FK (admin & viewer alike).
		//    Users without a linked client are allowed through for onboarding routes
		//    (their client is created by /onboard/start).
		isOnboardRoute := strings.HasPrefix(r.URL.Path, "/bratrax/onboard/") ||
			strings.HasPrefix(r.URL.Path, "/onboard/") ||
			strings.HasPrefix(r.URL.Path, "/bratrax/connectors/")
		if user.Role != "super_admin" && user.Role != "admin" && user.Role != "viewer" {
			writeJSONError(w, http.StatusForbidden, fmt.Sprintf("unsupported role: %s", user.Role))
			return
		}
		client, err := a.clientStore.GetByUserID(r.Context(), user.ID)
		if err != nil {
			a.logger.Error("client lookup failed", zap.Error(err))
			writeJSONError(w, http.StatusInternalServerError, "internal error")
			return
		}
		if client == nil && !isOnboardRoute {
			writeJSONError(w, http.StatusForbidden, "no client provisioned for this user")
			return
		}

		// 5. Strip any incoming identity headers to prevent spoofing
		// Uses case-insensitive matching to block all variations.
		stripBratraxHeaders(r.Header)

		// 6. Set identity headers for Flask.
		// X-Bratrax-Org-Id is retained for backwards compatibility with legacy Flask
		// code (credentials.py, connectors/*.py) that reads it via helpers/auth.get_user_organization_id().
		// It now always mirrors user.id — which is what it semantically was in the old schema.
		r.Header.Set("user-id", strconv.Itoa(user.ID))
		r.Header.Set("X-Bratrax-User-Id", strconv.Itoa(user.ID))
		r.Header.Set("X-Bratrax-Org-Id", strconv.Itoa(user.ID))
		if client != nil {
			r.Header.Set("X-Bratrax-Client-Id", client.ClientID)
		}
		r.Header.Set("X-Bratrax-User-Email", user.Email)
		r.Header.Set("X-Bratrax-User-Role", user.Role)

		// 7. Forward to next handler
		next.ServeHTTP(w, r)
	})
}

// ResolveClientFromCookie validates the bratrax_auth JWT cookie on the request and returns
// the authenticated user and their associated client. Returns (nil, nil, nil) if no cookie
// or invalid token. Returns (user, nil, nil) if the user has no provisioned client yet.
// Reusable from other middleware (e.g. the per-user instance router).
func (a *AuthMapper) ResolveClientFromCookie(r *http.Request) (*User, *Client, error) {
	cookie, err := r.Cookie(bratraxCookieName)
	if err != nil {
		// Authorization header fallback (popup OAuth flows)
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			cookie = &http.Cookie{Value: strings.TrimPrefix(authHeader, "Bearer ")}
		} else {
			return nil, nil, nil
		}
	}

	claims := &bratraxClaims{}
	_, err = jwt.ParseWithClaims(cookie.Value, claims, a.jwks.Keyfunc, jwt.WithValidMethods([]string{"RS256"}))
	if err != nil {
		return nil, nil, nil
	}
	if !claims.VerifyIssuer(a.issuerURL, true) || !claims.VerifyAudience(a.audienceURL, true) {
		return nil, nil, nil
	}

	userID, err := strconv.Atoi(claims.Subject)
	if err != nil {
		return nil, nil, nil
	}
	user, err := a.userStore.GetByID(r.Context(), userID)
	if err != nil {
		return nil, nil, fmt.Errorf("user lookup: %w", err)
	}
	if user == nil {
		return nil, nil, nil
	}
	client, err := a.clientStore.GetByUserID(r.Context(), user.ID)
	if err != nil {
		return user, nil, fmt.Errorf("client lookup: %w", err)
	}
	return user, client, nil
}

// stripBratraxHeaders removes all Bratrax identity headers from a request
// using case-insensitive matching to prevent spoofing via non-canonical keys.
func stripBratraxHeaders(h http.Header) {
	for _, name := range headersToStrip {
		// Also handle case-insensitive variations
		for key := range h {
			if strings.EqualFold(key, name) {
				delete(h, key)
			}
		}
	}
}
