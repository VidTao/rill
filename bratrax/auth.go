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
}

// NewAuthMapper creates an AuthMapper with all required dependencies.
func NewAuthMapper(userStore UserStoreInterface, clientStore ClientStoreInterface, jwks *keyfunc.JWKS, logger *zap.Logger) *AuthMapper {
	return &AuthMapper{
		userStore:   userStore,
		clientStore: clientStore,
		jwks:        jwks,
		logger:      logger,
	}
}

// Middleware wraps an http.Handler with JWT authentication and header injection.
func (a *AuthMapper) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Extract cookie
		cookie, err := r.Cookie(bratraxCookieName)
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "authentication required")
			return
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

		if !claims.VerifyIssuer(bratraxIssuerURL, true) {
			writeJSONError(w, http.StatusUnauthorized, "invalid token issuer")
			return
		}
		if !claims.VerifyAudience(bratraxAudienceURL, true) {
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

		// 4. Resolve client based on role
		var client *Client
		switch user.Role {
		case "admin":
			client, err = a.clientStore.GetDefault(r.Context())
		case "viewer":
			if user.ProjectID == nil {
				writeJSONError(w, http.StatusUnauthorized, "viewer has no project assigned")
				return
			}
			client, err = a.clientStore.GetByRillProjectID(r.Context(), *user.ProjectID)
		default:
			writeJSONError(w, http.StatusForbidden, fmt.Sprintf("unsupported role: %s", user.Role))
			return
		}
		if err != nil {
			a.logger.Error("client lookup failed", zap.Error(err))
			writeJSONError(w, http.StatusInternalServerError, "internal error")
			return
		}
		if client == nil {
			writeJSONError(w, http.StatusForbidden, "no client associated with user")
			return
		}

		// 5. Strip any incoming identity headers to prevent spoofing
		// Uses case-insensitive matching to block all variations.
		stripBratraxHeaders(r.Header)

		// 6. Set identity headers for Flask
		r.Header.Set("user-id", strconv.Itoa(user.ID))
		r.Header.Set("X-Bratrax-User-Id", strconv.Itoa(user.ID))
		r.Header.Set("X-Bratrax-Client-Id", client.ClientID)
		r.Header.Set("X-Bratrax-Org-Id", strconv.Itoa(client.OrganizationID))
		r.Header.Set("X-Bratrax-User-Email", user.Email)
		r.Header.Set("X-Bratrax-User-Role", user.Role)

		// 7. Forward to next handler
		next.ServeHTTP(w, r)
	})
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
