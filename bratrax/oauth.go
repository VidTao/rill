package bratrax

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/rilldata/rill/runtime/pkg/observability"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// The MCP authorization server.
//
// Claude's and ChatGPT's connector UIs can only authenticate to a remote MCP
// server with OAuth 2.1 (authorization code + PKCE), discovered through RFC 9728
// protected-resource metadata and RFC 8414 authorization-server metadata. The
// static brx_mcp_ token accepted by /bratrax/mcp has no place in that flow, so
// this file and its oauth_*.go siblings add one, entirely inside the Go proxy:
//
//	GET  /.well-known/oauth-protected-resource[/bratrax/mcp]   RFC 9728
//	GET  /.well-known/oauth-authorization-server               RFC 8414
//	POST /bratrax/oauth/register                               RFC 7591 (DCR)
//	GET  /bratrax/oauth/authorize                              parks the request
//	GET  /bratrax/oauth/pending                                consent page data
//	GET  /bratrax/oauth/pending/info                           login/signup branding
//	POST /bratrax/oauth/decision                               approve / deny
//	POST /bratrax/oauth/token                                  code + refresh grants
//
// The part that is specific to us is that the person clicking Connect may not
// have an account yet. Authorize parks the request in Postgres and sends them
// through the ordinary /login or /signup and the whole onboarding funnel; the
// consent page is where they come back to, and the only place a code is issued.
// A code is only ever issued for a workspace whose onboarding has reached
// `activating` (the merchant's own part is done), so reconnecting from Claude
// always recovers: an expired or abandoned attempt just starts a new one.
//
// Access tokens are 1h RS256 JWTs from the same issuer as the session cookie,
// with the MCP resource URL as audience so neither can stand in for the other.
// Refresh tokens are opaque, stored hashed, and rotated on every use.
//
// The whole thing is off unless BRATRAX_MCP_OAUTH_ENABLED is set; with it off,
// /bratrax/mcp behaves exactly as before.

const (
	oauthAccessTokenTTL   = time.Hour
	oauthRefreshTokenTTL  = 30 * 24 * time.Hour
	oauthAuthCodeTTL      = 5 * time.Minute
	oauthPendingTTL       = 24 * time.Hour
	oauthRefreshGrace     = 30 * time.Second
	oauthDCRPruneAfter    = 7 * 24 * time.Hour
	oauthCIMDCacheTTL     = 24 * time.Hour
	oauthRefreshTokenPref = "brx_rt_"
	oauthDCRClientPrefix  = "brx_oc_"
	oauthMaxBodyBytes     = 64 << 10
	oauthMCPPath          = "/bratrax/mcp"
	oauthConsentPath      = "/oauth/consent"
)

// defaultOAuthRedirectHosts are the hosts an HTTPS redirect_uri may point at.
// Registration is open (DCR), so without an allowlist anyone could register a
// client that redirects codes to their own server and send a merchant a
// convincing consent link. Loopback http redirects are always allowed on top of
// these, for native clients such as Claude Code and MCP Inspector.
var defaultOAuthRedirectHosts = []string{"claude.ai", "claude.com", "chatgpt.com"}

// oauthReadySteps are the onboarding steps at which a workspace may be bound to
// a grant: the merchant has done everything they need to do, and the rest is
// our activation pipeline. "" is a client with no onboarding row at all, which
// only happens for workspaces that predate the Lite funnel.
var oauthReadySteps = map[string]bool{
	"activating": true, "compiling": true, "deploying": true, "extracting": true,
	"ready": true, "error": true, "": true,
}

// OAuthService is the MCP authorization server. Nil means OAuth is disabled.
type OAuthService struct {
	store         oauthStore
	auth          *AuthService
	authMapper    *AuthMapper
	clientStore   ClientStoreInterface
	logger        *zap.Logger
	publicURL     string
	redirectHosts []string
	cimdClient    *http.Client
	limiter       *ipRateLimiter
	now           func() time.Time
}

// NewOAuthService builds the authorization server. publicURL is the external
// origin (e.g. https://bratrax.com), with no trailing slash.
func NewOAuthService(store oauthStore, auth *AuthService, authMapper *AuthMapper, clientStore ClientStoreInterface, publicURL string, extraRedirectHosts []string, logger *zap.Logger) *OAuthService {
	hosts := append([]string{}, defaultOAuthRedirectHosts...)
	for _, h := range extraRedirectHosts {
		if h = strings.ToLower(strings.TrimSpace(h)); h != "" {
			hosts = append(hosts, h)
		}
	}
	return &OAuthService{
		store:         store,
		auth:          auth,
		authMapper:    authMapper,
		clientStore:   clientStore,
		logger:        logger,
		publicURL:     strings.TrimSuffix(publicURL, "/"),
		redirectHosts: hosts,
		cimdClient:    newCIMDHTTPClient(),
		limiter:       newIPRateLimiter(rate.Every(2*time.Second), 30),
		now:           time.Now,
	}
}

// ResourceURL is the MCP endpoint's canonical URL: the RFC 8707 resource
// indicator clients send, and the audience of every access token we issue.
func (s *OAuthService) ResourceURL() string { return s.publicURL + oauthMCPPath }

// ResourceMetadataURL is what the 401 from /bratrax/mcp points clients at.
func (s *OAuthService) ResourceMetadataURL() string {
	return s.publicURL + "/.well-known/oauth-protected-resource"
}

// RegisterRoutes mounts every OAuth endpoint on the mux. All of them bypass the
// /bratrax/ auth catch-all: the ones that need a session resolve it themselves.
func (s *OAuthService) RegisterRoutes(mux *http.ServeMux) {
	handle := func(pattern string, h http.HandlerFunc) {
		observability.MuxHandle(mux, pattern, observability.Middleware("bratrax-oauth", s.logger, h))
	}
	// Discovery. The path-suffixed PRM variant is the RFC 9728 location for a
	// resource with a path; the bare one is where our 401 points and where
	// clients fall back to.
	handle("/.well-known/oauth-protected-resource", cors(s.handleProtectedResourceMetadata))
	handle("/.well-known/oauth-protected-resource"+oauthMCPPath, cors(s.handleProtectedResourceMetadata))
	handle("/.well-known/oauth-authorization-server", cors(s.handleAuthorizationServerMetadata))

	// Register and token are deliberately not rate-limited per IP: Claude and
	// ChatGPT call them server-to-server from a handful of addresses shared by
	// every one of their users (Anthropic's 160.79.104.0/21), so a per-IP limit
	// would throttle all merchants at once. Tokens carry 256 bits of entropy, so
	// guessing is not a threat; abandoned registrations are pruned.
	handle("/bratrax/oauth/register", cors(s.handleRegister))
	handle("GET /bratrax/oauth/authorize", s.limited(s.handleAuthorize))
	handle("/bratrax/oauth/token", cors(s.handleToken))

	handle("GET /bratrax/oauth/pending", s.handlePending)
	handle("GET /bratrax/oauth/pending/info", s.limited(s.handlePendingInfo))
	handle("POST /bratrax/oauth/decision", s.handleDecision)
}

func (s *OAuthService) handleProtectedResourceMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=300")
	writeJSON(w, http.StatusOK, map[string]any{
		"resource":                 s.ResourceURL(),
		"authorization_servers":    []string{s.publicURL},
		"bearer_methods_supported": []string{"header"},
		"resource_name":            "Bratrax",
		"resource_documentation":   s.publicURL + "/integrations",
	})
}

func (s *OAuthService) handleAuthorizationServerMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=300")
	writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                                s.publicURL,
		"authorization_endpoint":                s.publicURL + "/bratrax/oauth/authorize",
		"token_endpoint":                        s.publicURL + "/bratrax/oauth/token",
		"registration_endpoint":                 s.publicURL + "/bratrax/oauth/register",
		"response_types_supported":              []string{"code"},
		"response_modes_supported":              []string{"query"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":      []string{"S256"},
		"token_endpoint_auth_methods_supported": []string{"none"},
		// Client ID Metadata Documents: the client_id is an HTTPS URL we fetch.
		// Preferred over DCR by both Claude and ChatGPT because it doesn't mint
		// a new registration on every connection.
		"client_id_metadata_document_supported": true,
		"service_documentation":                 s.publicURL + "/integrations",
	})
}

// limited applies the per-IP rate limit to the browser-facing endpoints, which
// are unauthenticated and each write or read a row.
func (s *OAuthService) limited(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.limiter.Allow(clientIP(r)) {
			w.Header().Set("Retry-After", "10")
			writeJSONError(w, http.StatusTooManyRequests, "too many requests")
			return
		}
		next(w, r)
	}
}

// cors lets browser-based MCP clients (MCP Inspector, web IDEs) reach the
// discovery, registration and token endpoints. None of them read cookies, so a
// wildcard origin grants nothing a server-side caller doesn't already have.
func cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, MCP-Protocol-Version")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

// oauthError writes an RFC 6749 §5.2 error body.
func oauthError(w http.ResponseWriter, status int, code, description string) {
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, status, map[string]string{"error": code, "error_description": description})
}

// randomToken returns a URL-safe random string carrying n bytes of entropy.
func randomToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand only fails if the OS entropy source is broken, at which
		// point issuing a predictable credential would be far worse than dying.
		panic("bratrax oauth: crypto/rand failed: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// hashToken is how codes and refresh tokens are stored: a database leak must
// not hand out working credentials.
func hashToken(t string) string {
	sum := sha256.Sum256([]byte(t))
	return hex.EncodeToString(sum[:])
}

// clientIP is the address the rate limiter keys on. Behind nginx every request
// arrives from loopback, so the X-Real-IP nginx sets (overwriting anything the
// client sent) is used; a request that reached us directly is keyed on its
// socket address, since there is no proxy to trust.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		if real := strings.TrimSpace(r.Header.Get("X-Real-IP")); real != "" {
			return real
		}
	}
	return host
}

// ipRateLimiter is a token bucket per client IP. Buckets idle past the sweep
// interval are dropped so the map can't grow without bound.
type ipRateLimiter struct {
	mu        sync.Mutex
	every     rate.Limit
	burst     int
	buckets   map[string]*ipBucket
	lastSweep time.Time
}

type ipBucket struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func newIPRateLimiter(every rate.Limit, burst int) *ipRateLimiter {
	return &ipRateLimiter{every: every, burst: burst, buckets: make(map[string]*ipBucket), lastSweep: time.Now()}
}

func (l *ipRateLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if now.Sub(l.lastSweep) > 10*time.Minute {
		for k, b := range l.buckets {
			if now.Sub(b.lastSeen) > 10*time.Minute {
				delete(l.buckets, k)
			}
		}
		l.lastSweep = now
	}
	b, ok := l.buckets[key]
	if !ok {
		b = &ipBucket{limiter: rate.NewLimiter(l.every, l.burst)}
		l.buckets[key] = b
	}
	b.lastSeen = now
	return b.limiter.Allow()
}

// resourceMatches accepts an RFC 8707 resource indicator naming our MCP
// endpoint. Empty is allowed: the parameter is optional and some clients omit it.
func (s *OAuthService) resourceMatches(resource string) bool {
	if resource == "" {
		return true
	}
	return strings.TrimSuffix(resource, "/") == s.ResourceURL()
}

// redirectHost is shown on the consent screen so a merchant can see where the
// code is going before they approve.
func redirectHost(redirectURI string) string {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return ""
	}
	return u.Host
}
