package bratrax

import (
	"context"
	"errors"
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

	// Shopify app credentials for verifying App Bridge session tokens. Both
	// empty disables that path entirely; see WithShopifySessionAuth.
	shopifyClientID     string
	shopifyClientSecret string
}

// WithShopifySessionAuth enables the Shopify embedded-admin auth path, where a
// merchant is identified by an App Bridge session token instead of the
// bratrax_auth cookie (which is third-party inside the admin iframe and blocked
// by Safari). Returns the mapper for chaining.
//
// A setter rather than two more constructor arguments: the cookie path is
// unchanged whether or not this is called, and every existing NewAuthMapper
// caller keeps working untouched.
func (a *AuthMapper) WithShopifySessionAuth(clientID, clientSecret string) *AuthMapper {
	a.shopifyClientID = clientID
	a.shopifyClientSecret = clientSecret
	return a
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
		// /bratrax/demo-request is the public self-serve "Try demo" submit
		// endpoint (email -> viewer invitation on the demo client), same
		// pre-authentication rationale as access-requests.
		if strings.HasPrefix(r.URL.Path, "/bratrax/invitations/") ||
			strings.HasPrefix(r.URL.Path, "/bratrax/access-requests") ||
			strings.HasPrefix(r.URL.Path, "/bratrax/demo-request") ||
			strings.HasPrefix(r.URL.Path, "/bratrax/signup-link") ||
			r.URL.Path == "/bratrax/onboard/check-company" {
			stripBratraxHeaders(r.Header)
			next.ServeHTTP(w, r)
			return
		}

		// 1. Extract cookie
		var bearerToken string
		cookie, err := r.Cookie(bratraxCookieName)
		if err != nil {
			// Also check Authorization header as fallback (for popup OAuth flows
			// where cookies may not be sent due to browser restrictions)
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				bearerToken = strings.TrimPrefix(authHeader, "Bearer ")
				cookie = &http.Cookie{Value: bearerToken}
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
			// Not one of ours. Inside the Shopify admin iframe the bratrax_auth
			// cookie is third-party (blocked outright by Safari's ITP), so App
			// Bridge sends a Shopify session token as the bearer instead. Try
			// that before giving up. Deliberately second: the cookie path stays
			// byte-identical, and a normal expired token pays only one extra
			// HS256 parse before its 401.
			if bearerToken != "" && a.serveShopifySession(w, r, next, bearerToken) {
				return
			}
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

		// 5+6. Strip incoming identity headers (anti-spoofing) and set ours.
		applyIdentityHeaders(r, user, client)

		// 7. Forward to next handler
		next.ServeHTTP(w, r)
	})
}

// applyIdentityHeaders strips any caller-supplied identity headers and sets the
// ones Flask trusts. Shared by the cookie path and the Shopify session-token
// path so the two cannot drift — downstream must not be able to tell which
// credential authenticated the request.
//
// X-Bratrax-Org-Id is retained for backwards compatibility with legacy Flask
// code (credentials.py, connectors/*.py) that reads it via
// helpers/auth.get_user_organization_id(). It now always mirrors user.id —
// which is what it semantically was in the old schema.
func applyIdentityHeaders(r *http.Request, user *User, client *Client) {
	stripBratraxHeaders(r.Header)

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
}

// serveShopifySession authenticates a request carrying a Shopify App Bridge
// session token and, on success, forwards it with the usual identity headers.
//
// Returns true when the request has been fully handled (either served or
// answered with an error); false means "this wasn't a Shopify session token",
// and the caller should fall through to its normal 401.
//
// The token proves which SHOP is asking. Mapping shop -> client is guaranteed
// single-valued by idx_onboarding_shopify_shop_unique, and the request then
// acts as that workspace's primary admin — see UserStore.GetPrimaryUserForClient
// for why an embedded session gets an owner identity rather than a per-person one.
func (a *AuthMapper) serveShopifySession(w http.ResponseWriter, r *http.Request, next http.Handler, token string) bool {
	user, client, err := a.resolveShopifySessionDetailed(r, token)
	switch {
	case err == nil:
		applyIdentityHeaders(r, user, client)
		next.ServeHTTP(w, r)
		return true
	case errors.Is(err, errShopifySessionNotOurs):
		// Not a Shopify session token at all — let the caller fall through to
		// its normal 401 for an unrecognised credential.
		return false
	case errors.Is(err, errShopifyShopUnlinked):
		// Valid token, but nobody has claimed this shop yet — the normal state
		// for an App Store install with no Bratrax account behind it. Distinct
		// message so the embedded frontend can route to signup instead of
		// treating it as a broken session.
		writeJSONError(w, http.StatusUnauthorized, "shopify shop not linked to a bratrax account")
		return true
	case errors.Is(err, errShopifyNoAccount):
		writeJSONError(w, http.StatusForbidden, "no account provisioned for this shop")
		return true
	default:
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return true
	}
}

var (
	errShopifySessionNotOurs = errors.New("bratrax: not a shopify session token")
	errShopifyShopUnlinked   = errors.New("bratrax: shopify shop has no client")
	errShopifyNoAccount      = errors.New("bratrax: shopify client has no admin user")
)

// resolveShopifySessionDetailed verifies a session token and maps it to a
// tenant, distinguishing the failure modes so callers can respond differently.
//
// Shared by the request middleware and ResolveClientFromCookie — the latter is
// what InstanceRouterMiddleware uses to pick the tenant's Rill instance, so
// both must agree or an embedded dashboard authenticates fine yet reads from
// the wrong (empty) instance.
func (a *AuthMapper) resolveShopifySessionDetailed(r *http.Request, token string) (*User, *Client, error) {
	shop, err := verifyShopifySessionToken(token, a.shopifyClientID, a.shopifyClientSecret)
	if err != nil {
		if !errors.Is(err, errShopifySessionDisabled) {
			a.logger.Debug("shopify session token rejected", zap.Error(err))
		}
		return nil, nil, errShopifySessionNotOurs
	}

	client, err := a.clientStore.GetByShopifyShop(r.Context(), shop)
	if err != nil {
		a.logger.Error("shopify shop lookup failed", zap.String("shop", shop), zap.Error(err))
		return nil, nil, err
	}
	if client == nil {
		a.logger.Info("shopify session for unlinked shop", zap.String("shop", shop))
		return nil, nil, errShopifyShopUnlinked
	}

	user, err := a.userStore.GetPrimaryUserForClient(r.Context(), client.ClientID)
	if err != nil {
		a.logger.Error("shopify primary user lookup failed",
			zap.String("client_id", client.ClientID), zap.Error(err))
		return nil, nil, err
	}
	if user == nil {
		a.logger.Warn("shopify session for client with no admin user",
			zap.String("shop", shop), zap.String("client_id", client.ClientID))
		return nil, nil, errShopifyNoAccount
	}

	return user, client, nil
}

// resolveShopifySession is the boolean-shaped wrapper ResolveClientFromCookie
// wants: any failure is simply "no identity", since that path has no response
// writer and its callers already handle a nil client.
func (a *AuthMapper) resolveShopifySession(r *http.Request, token string) (*User, *Client, bool) {
	user, client, err := a.resolveShopifySessionDetailed(r, token)
	if err != nil {
		return nil, nil, false
	}
	return user, client, true
}

// ResolveClientFromCookie validates the bratrax_auth JWT cookie on the request and returns
// the authenticated user and their associated client. Returns (nil, nil, nil) if no cookie
// or invalid token. Returns (user, nil, nil) if the user has no provisioned client yet.
// Reusable from other middleware (e.g. the per-user instance router).
func (a *AuthMapper) ResolveClientFromCookie(r *http.Request) (*User, *Client, error) {
	var bearerToken string
	cookie, err := r.Cookie(bratraxCookieName)
	if err != nil {
		// Authorization header fallback (popup OAuth flows)
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			bearerToken = strings.TrimPrefix(authHeader, "Bearer ")
			cookie = &http.Cookie{Value: bearerToken}
		} else {
			return nil, nil, nil
		}
	}

	claims := &bratraxClaims{}
	_, err = jwt.ParseWithClaims(cookie.Value, claims, a.jwks.Keyfunc, jwt.WithValidMethods([]string{"RS256"}))
	if err != nil {
		// Shopify embedded admin: no bratrax_auth cookie exists on this origin,
		// so the runtime client authenticates with an App Bridge session token.
		//
		// This matters more than it looks. InstanceRouterMiddleware calls this
		// function to rewrite /v1/instances/default/* to the tenant's real
		// instance. Without this branch an embedded request resolves to no
		// client, falls through to the empty "default" instance, and the
		// embedded dashboard renders with no data at all.
		if bearerToken != "" {
			if user, client, ok := a.resolveShopifySession(r, bearerToken); ok {
				return user, client, nil
			}
		}
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
