package bratrax

import (
	"context"
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

// activeClientCookieName carries the currently-selected client_id for
// cross-client super_admins. Set by /bratrax/auth/switch-client (and by
// the middleware on first request, falling back to last_client_id then
// to the first client in the list). Empty / missing for non-super_admins.
const activeClientCookieName = "bratrax_active_client"

// headersToStrip lists request headers that the middleware must remove
// before forwarding, to prevent spoofing from untrusted clients.
var headersToStrip = []string{
	"X-Bratrax-User-Id",
	"X-Bratrax-Client-Id",
	"X-Bratrax-Multi-Client-Id",
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
		// /bratrax/access-requests/ is the public submit-an-email-to-request
		// endpoint, also pre-authentication by design.
		// /bratrax/onboard/check-company is a single public route inside the
		// otherwise-gated /bratrax/onboard/* namespace — exact-match (not a
		// prefix) so we don't accidentally expose adjacent onboard routes.
		if strings.HasPrefix(r.URL.Path, "/bratrax/invitations/") ||
			strings.HasPrefix(r.URL.Path, "/bratrax/access-requests") ||
			r.URL.Path == "/bratrax/onboard/check-company" {
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

		// 4. Resolve active client.
		//    - admin/viewer: rill_users.client_id FK (one user → one client)
		//    - super_admin: bratrax_active_client cookie → last_client_id → first-in-list
		//    Users without a linked client are allowed through for onboarding routes
		//    (their client is created by /onboard/start).
		isOnboardRoute := strings.HasPrefix(r.URL.Path, "/bratrax/onboard/") ||
			strings.HasPrefix(r.URL.Path, "/onboard/") ||
			strings.HasPrefix(r.URL.Path, "/bratrax/connectors/")
		if user.Role != "super_admin" && user.Role != "admin" && user.Role != "viewer" {
			writeJSONError(w, http.StatusForbidden, fmt.Sprintf("unsupported role: %s", user.Role))
			return
		}
		client, cookieToSet, err := a.resolveActiveClient(r.Context(), user, r)
		if err != nil {
			a.logger.Error("client lookup failed", zap.Error(err))
			writeJSONError(w, http.StatusInternalServerError, "internal error")
			return
		}
		if client == nil && !isOnboardRoute {
			writeJSONError(w, http.StatusForbidden, "no client provisioned for this user")
			return
		}

		// If the super_admin path picked an active client different from what
		// the cookie already had (or there was no cookie), set it now so the
		// next request short-circuits straight to the cookie value.
		if cookieToSet != "" {
			http.SetCookie(w, &http.Cookie{
				Name:     activeClientCookieName,
				Value:    cookieToSet,
				Path:     "/",
				MaxAge:   int(bratraxTokenTTL.Seconds()),
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})
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
		if user.MultiClientID != nil && *user.MultiClientID != "" {
			r.Header.Set("X-Bratrax-Multi-Client-Id", *user.MultiClientID)
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
	client, _, err := a.resolveActiveClient(r.Context(), user, r)
	if err != nil {
		return user, nil, fmt.Errorf("client lookup: %w", err)
	}
	return user, client, nil
}

// resolveActiveClient picks the client to use for this request.
//
//   - admin/viewer (single-store): rill_users.client_id (existing FK).
//   - admin/viewer (multi-store, multi_client_id != NULL): bratrax_active_client
//     cookie if it points at a sibling under the same multi_client_id;
//     otherwise rill_users.client_id (the "home" sub-store).
//   - super_admin: reads the bratrax_active_client cookie; if missing or
//     stale, falls back to user.last_client_id, then to the first client
//     by company_name. The second return value is the cookie value the
//     caller should set if non-empty (used by the middleware to refresh
//     the cookie on first super_admin request after login).
//
// Returns (nil, "", nil) if no client could be resolved (e.g. mid-onboarding,
// or zero clients in the system). The caller decides whether that's an error.
func (a *AuthMapper) resolveActiveClient(ctx context.Context, user *User, r *http.Request) (*Client, string, error) {
	if user.Role != "super_admin" {
		// Multi-store user: cookie wins if it points at a sibling. Falls back
		// to the home sub-store on rill_users.client_id. Guards against
		// hijacking: even if a hostile cookie names another tenant's client,
		// it only takes effect when that client is under the user's parent.
		if user.MultiClientID != nil && *user.MultiClientID != "" {
			if cookie, cookieErr := r.Cookie(activeClientCookieName); cookieErr == nil && cookie.Value != "" {
				client, err := a.clientStore.GetByClientID(ctx, cookie.Value)
				if err != nil {
					return nil, "", err
				}
				if client != nil && client.MultiClientID != nil &&
					*client.MultiClientID == *user.MultiClientID {
					return client, "", nil
				}
				// Cookie missing / pointing at a foreign client — fall through.
			}
		}
		client, err := a.clientStore.GetByUserID(ctx, user.ID)
		return client, "", err
	}

	// 1. Cookie — the freshest signal.
	if cookie, cookieErr := r.Cookie(activeClientCookieName); cookieErr == nil && cookie.Value != "" {
		client, err := a.clientStore.GetByClientID(ctx, cookie.Value)
		if err != nil {
			return nil, "", err
		}
		if client != nil {
			return client, "", nil
		}
		// Cookie referenced a deleted client — fall through to other fallbacks.
	}

	// 2. last_client_id — where this super_admin was last active.
	if user.LastClientID != nil && *user.LastClientID != "" {
		client, err := a.clientStore.GetByClientID(ctx, *user.LastClientID)
		if err != nil {
			return nil, "", err
		}
		if client != nil {
			return client, client.ClientID, nil
		}
	}

	// 3. First client alphabetically. Deterministic; the dropdown shows the
	//    same order so the default lines up with what the user sees.
	all, err := a.clientStore.ListAll(ctx)
	if err != nil {
		return nil, "", err
	}
	if len(all) == 0 {
		return nil, "", nil
	}
	first := all[0]
	return &first, first.ClientID, nil
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
