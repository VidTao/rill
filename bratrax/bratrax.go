package bratrax

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/rilldata/rill/runtime/pkg/observability"
	"go.uber.org/zap"
)

// Handlers carries the constructed bratrax components so that callers (e.g. the
// CLI's local app) can register additional middleware (such as the instance router)
// against the same AuthMapper used by the /bratrax/* proxy.
type Handlers struct {
	AuthMapper  *AuthMapper
	ClientStore *ClientStore
}

// RegisterHandlers registers Bratrax proxy routes on the given ServeMux.
// It wires up: observability → auth → reverse proxy, plus auth endpoints.
//
// Routes:
//   - POST /bratrax/auth/login            — email+password login
//   - POST /bratrax/auth/logout           — clear auth cookie
//   - GET  /bratrax/auth/me               — current user info
//   - POST /bratrax/auth/signup           — self-serve signup (public, creates user + sets JWT)
//   - POST /bratrax/auth/users            — create user (admin only)
//   - GET  /bratrax/auth/clients          — list every client (super_admin only)
//   - POST /bratrax/auth/switch-client    — set active client cookie (super_admin only)
//   - GET  /bratrax/.well-known/jwks.json — public JWKS
//   - GET  /bratrax/health                — local health check
//   - /bratrax/                            — catch-all proxy to Flask API
//
// Returns the constructed Handlers so the caller can install additional middleware.
func RegisterHandlers(mux *http.ServeMux, logger *zap.Logger) (*Handlers, error) {
	cfg, err := ConfigFromEnv()
	if err != nil {
		return nil, err
	}

	// User store
	store, err := NewUserStore(cfg.UsersDSN)
	if err != nil {
		return nil, fmt.Errorf("bratrax: failed to create user store: %w", err)
	}

	// Auth service (persistent JWT issuer)
	authSvc, err := NewAuthService(store, logger, cfg.IssuerURL, cfg.AudienceURL, cfg.SecureCookie, cfg.OnlyInvitationLink, cfg.AllowWoocommerce)
	if err != nil {
		store.Close()
		return nil, fmt.Errorf("bratrax: failed to create auth service: %w", err)
	}

	clientStore := NewClientStore(store.DB())

	proxy := NewProxy(cfg.TargetURL, logger)
	authMapper := NewAuthMapper(store, clientStore, authSvc.JWKS(), logger, cfg.IssuerURL, cfg.AudienceURL)

	// Local health endpoint — confirms the proxy layer is alive.
	healthHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if encErr := json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"service": "bratrax-proxy",
		}); encErr != nil {
			logger.Debug("failed to write health response", zap.Error(encErr))
		}
	})

	// Auth endpoints (not behind auth middleware — they handle auth internally)
	observability.MuxHandle(mux, "POST /bratrax/auth/login",
		observability.Middleware("bratrax", logger, http.HandlerFunc(authSvc.HandleLogin)))
	observability.MuxHandle(mux, "POST /bratrax/auth/logout",
		observability.Middleware("bratrax", logger, http.HandlerFunc(authSvc.HandleLogout)))
	observability.MuxHandle(mux, "GET /bratrax/auth/me",
		observability.Middleware("bratrax", logger, http.HandlerFunc(authSvc.HandleMe)))
	observability.MuxHandle(mux, "POST /bratrax/auth/users",
		observability.Middleware("bratrax", logger, http.HandlerFunc(authSvc.HandleCreateUser)))
	observability.MuxHandle(mux, "POST /bratrax/auth/signup",
		observability.Middleware("bratrax", logger, http.HandlerFunc(authSvc.HandleSignup)))
	// Public config endpoint — read by the /signup page on mount to decide
	// whether to render the form or the invite-only message. Public on purpose.
	observability.MuxHandle(mux, "GET /bratrax/auth/config",
		observability.Middleware("bratrax", logger, http.HandlerFunc(authSvc.HandleAuthConfig)))

	// Password-reset endpoints — Flask-side handlers, public on purpose
	// (the token is the auth). Proxied straight through WITHOUT the auth
	// mapper, otherwise the /forgot-password page would 401 on an
	// unauthenticated visitor — the entire point is they can't log in.
	observability.MuxHandle(mux, "POST /bratrax/auth/password-reset-request",
		observability.Middleware("bratrax", logger, proxy))
	observability.MuxHandle(mux, "POST /bratrax/auth/password-reset",
		observability.Middleware("bratrax", logger, proxy))

	// Lifecycle-email footer "Pause these emails" link. Sits OUTSIDE the
	// /bratrax/ prefix because the URL is baked into customer-facing emails
	// and we don't want /bratrax/ in those URLs. Public by design — the
	// HMAC token in `?t=` is the auth, no login possible (the whole point
	// is reaching customers who can't log in). The proxy's prefix-stripper
	// leaves non-/bratrax paths unchanged, so /email/pause passes through
	// to Flask's email_pause_routes blueprint verbatim.
	observability.MuxHandle(mux, "GET /email/pause",
		observability.Middleware("bratrax", logger, proxy))

	// WooCommerce wc-auth callback. The merchant's store POSTs the generated
	// REST API key pair here server-to-server (no login cookie), so it must
	// bypass the auth mapper. Public by design — the signed `user_id` state
	// token in the body is the auth. Registered explicitly so it wins over the
	// /bratrax/ auth catch-all; the prefix-stripper forwards it to Flask as
	// /onboard/woocommerce/callback verbatim.
	observability.MuxHandle(mux, "POST /bratrax/onboard/woocommerce/callback",
		observability.Middleware("bratrax", logger, proxy))

	// WooCommerce tracking-plugin handshake exchange. The store's WordPress
	// server POSTs its opaque token here server-to-server (no login cookie) to
	// receive its client_id + events endpoint, so it must bypass the auth mapper.
	// Public by design — the signed token in the body is the auth. The companion
	// /plugin-connect endpoint rides the authed /bratrax/ catch-all (the merchant
	// is logged in when they click "Connect"). The plugin.zip download is public
	// so it can be linked from docs / fetched without a session.
	observability.MuxHandle(mux, "POST /bratrax/onboard/woocommerce/plugin-exchange",
		observability.Middleware("bratrax", logger, proxy))
	observability.MuxHandle(mux, "GET /bratrax/onboard/woocommerce/plugin.zip",
		observability.Middleware("bratrax", logger, proxy))

	// Super_admin client-switcher endpoints. Both auth themselves via the
	// JWT cookie + role check; not behind the AuthMapper proxy middleware.
	switchSvc := NewClientSwitchService(authMapper, store, clientStore, logger, cfg.SecureCookie)
	observability.MuxHandle(mux, "GET /bratrax/auth/clients",
		observability.Middleware("bratrax", logger, http.HandlerFunc(switchSvc.HandleListClients)))
	observability.MuxHandle(mux, "POST /bratrax/auth/switch-client",
		observability.Middleware("bratrax", logger, http.HandlerFunc(switchSvc.HandleSwitchClient)))

	// JWKS endpoint for token validation
	observability.MuxHandle(mux, "GET /bratrax/.well-known/jwks.json",
		observability.Middleware("bratrax", logger, authSvc.Issuer().WellKnownHandler()))

	// sitemap.xml + robots.txt: proxy through to the marketing static repo on
	// GitHub instead of baking them into the binary. Editing the file in the
	// GitHub repo updates production without a rebuild. Crawlers fetch these
	// as raw bytes (not via a browser), so unlike the HTML marketing pages
	// these can't be iframed client-side — the proxy has to live server-side.
	githubFetcher := &http.Client{Timeout: 8 * time.Second}
	fetchGithubStatic := func(rawURL string) ([]byte, error) {
		resp, getErr := githubFetcher.Get(rawURL)
		if getErr != nil {
			return nil, getErr
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("github raw returned HTTP %d", resp.StatusCode)
		}
		return io.ReadAll(resp.Body)
	}
	observability.MuxHandle(mux, "GET /sitemap.xml",
		observability.Middleware("bratrax", logger, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, fetchErr := fetchGithubStatic("https://raw.githubusercontent.com/yuolel/bratrax-wip/refs/heads/bratrax-com-static/sitemap.xml")
			if fetchErr != nil {
				logger.Warn("sitemap.xml fetch failed", zap.Error(fetchErr))
				http.Error(w, "sitemap unavailable", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/xml; charset=utf-8")
			w.Header().Set("Cache-Control", "public, max-age=300")
			if _, writeErr := w.Write(body); writeErr != nil {
				logger.Debug("sitemap.xml write failed", zap.Error(writeErr))
			}
		})))
	observability.MuxHandle(mux, "GET /robots.txt",
		observability.Middleware("bratrax", logger, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, fetchErr := fetchGithubStatic("https://raw.githubusercontent.com/yuolel/bratrax-wip/refs/heads/bratrax-com-static/robots.txt")
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			if fetchErr != nil {
				// Fall back to allow-all rather than 5xx — some crawlers treat
				// a missing/erroring robots.txt as "back off completely", which
				// is the opposite of what we want for a marketing site.
				logger.Warn("robots.txt fetch failed; serving permissive fallback", zap.Error(fetchErr))
				if _, writeErr := w.Write([]byte("User-agent: *\nDisallow:\n")); writeErr != nil {
					logger.Debug("robots.txt fallback write failed", zap.Error(writeErr))
				}
				return
			}
			w.Header().Set("Cache-Control", "public, max-age=300")
			if _, writeErr := w.Write(body); writeErr != nil {
				logger.Debug("robots.txt write failed", zap.Error(writeErr))
			}
		})))

	// Marketing pages: serve the GitHub-authored HTML as the actual top-level
	// document so crawlers, social-media unfurlers, and the browser tab title
	// see per-page <meta>/<title> tags from the page itself (previously
	// hidden behind a client-side iframe sandbox). The Svelte iframe routes
	// at the same paths remain as the fallback for in-app SPA navigation;
	// direct visits hit these handlers first because the mux's explicit
	// patterns take precedence over the SPA's catch-all.
	serveGithubHTML := func(githubURL string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			body, fetchErr := fetchGithubStatic(githubURL)
			if fetchErr != nil {
				logger.Warn("marketing page fetch failed",
					zap.String("path", r.URL.Path),
					zap.String("github_url", githubURL),
					zap.Error(fetchErr))
				http.Error(w, "page temporarily unavailable", http.StatusBadGateway)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "public, max-age=300")
			if _, writeErr := w.Write(body); writeErr != nil {
				logger.Debug("marketing page write failed", zap.Error(writeErr))
			}
		}
	}

	// Apex: authed users go to /developer, everyone else gets the marketing
	// homepage as a real crawlable document. Restores the auth-aware split
	// the old apex handler enforced before being moved to +layout.ts in C25
	// — keeping it server-side now so unauthed crawlers see proper meta tags
	// without the iframe sandbox swallowing them.
	observability.MuxHandle(mux, "GET /{$}",
		observability.Middleware("bratrax", logger, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, _, authErr := authMapper.ResolveClientFromCookie(r)
			if authErr != nil {
				logger.Debug("apex auth resolution error", zap.Error(authErr))
				user = nil
			}
			if user != nil {
				w.Header().Set("Cache-Control", "no-store")
				http.Redirect(w, r, "/developer", http.StatusFound)
				return
			}
			serveGithubHTML("https://raw.githubusercontent.com/yuolel/bratrax-wip/refs/heads/bratrax-com-static/index.html")(w, r)
		})))

	observability.MuxHandle(mux, "GET /faq",
		observability.Middleware("bratrax", logger,
			serveGithubHTML("https://raw.githubusercontent.com/yuolel/bratrax-wip/refs/heads/bratrax-com-static/faq/index.html")))
	observability.MuxHandle(mux, "GET /privacy-policy",
		observability.Middleware("bratrax", logger,
			serveGithubHTML("https://raw.githubusercontent.com/yuolel/bratrax-wip/refs/heads/bratrax-com-static/privacy/index.html")))
	observability.MuxHandle(mux, "GET /terms-of-service",
		observability.Middleware("bratrax", logger,
			serveGithubHTML("https://raw.githubusercontent.com/yuolel/bratrax-wip/refs/heads/bratrax-com-static/terms/index.html")))

	// /vs/{slug}: competitor comparison pages. Whitelist known slugs so a
	// random /vs/foobar doesn't proxy a 404 from GitHub raw and confuse
	// callers. Add new entries here as pages are authored under
	// bratrax-com-static/vs/.
	observability.MuxHandle(mux, "GET /vs/{slug}",
		observability.Middleware("bratrax", logger, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			slug := r.PathValue("slug")
			var githubURL string
			switch slug {
			case "triple-whale":
				githubURL = "https://raw.githubusercontent.com/yuolel/bratrax-wip/refs/heads/bratrax-com-static/vs/triple-whale/index.html"
			case "hyros":
				githubURL = "https://raw.githubusercontent.com/yuolel/bratrax-wip/refs/heads/bratrax-com-static/vs/hyros/index.html"
			default:
				http.NotFound(w, r)
				return
			}
			serveGithubHTML(githubURL)(w, r)
		})))

	// Health and proxy (existing routes)
	observability.MuxHandle(mux, "/bratrax/health", observability.Middleware("bratrax", logger, healthHandler))

	// /bratrax/mcp — public MCP endpoint for Claude Desktop. Auths an opaque
	// per-client token and forwards into the runtime's existing per-instance
	// MCP handler. Registered before the catch-all proxy so it takes precedence.
	RegisterMCPHandler(mux, clientStore, authSvc, cfg.AudienceURL, logger)

	// Middleware chain: observability → auth → proxy (catch-all)
	proxyHandler := observability.Middleware("bratrax", logger, authMapper.Middleware(proxy))
	observability.MuxHandle(mux, "/bratrax/", proxyHandler)

	// Log DSN with credentials redacted
	redactedDSN := cfg.UsersDSN
	if parsed, parseErr := url.Parse(cfg.UsersDSN); parseErr == nil {
		parsed.User = nil
		redactedDSN = parsed.String()
	}
	logger.Info("bratrax proxy registered",
		zap.String("target", cfg.TargetURL.String()),
		zap.String("users_dsn", redactedDSN),
	)

	return &Handlers{
		AuthMapper:  authMapper,
		ClientStore: clientStore,
	}, nil
}
