package bratrax

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"time"

	// The marketing handlers fall back to the SPA when GitHub is unreachable,
	// which needs the same embedded frontend the CLI serves as its catch-all.
	"github.com/rilldata/rill/cli/pkg/web"
	"github.com/rilldata/rill/runtime/pkg/observability"
	"go.uber.org/zap"
)

// Handlers carries the constructed bratrax components so that callers (e.g. the
// CLI's local app) can register additional middleware (such as the instance router)
// against the same AuthMapper used by the /bratrax/* proxy.
type Handlers struct {
	AuthMapper  *AuthMapper
	ClientStore *ClientStore
	// PromptStore meters demo users' AI-sidebar prompts. Exposed so the local
	// app can hand it to InstanceRouterMiddleware, which is the only layer that
	// sees both the user's identity and the AI request.
	PromptStore *AIPromptStore
	// Config is surfaced so the caller can read the demo-AI settings without
	// re-reading the environment.
	Config *Config
}

// changelogSlugRe constrains the slugs accepted by GET /changelog/{slug}. The
// slug is interpolated into a GitHub raw URL, so this is the path-traversal and
// injection guard; it must match before the URL is built. Kept permissive
// enough for the slugs the public-changelog routine generates (lowercase,
// digits, hyphens) and nothing else.
var changelogSlugRe = regexp.MustCompile(`^[a-z0-9-]{1,64}$`)

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
// ensureReady (may be nil) is invoked by the /bratrax/mcp handler to register
// and warm the client's instance before proxying; see EnsureReadyFn.
//
// Returns the constructed Handlers so the caller can install additional middleware.
func RegisterHandlers(mux *http.ServeMux, logger *zap.Logger, ensureReady EnsureReadyFn) (*Handlers, error) {
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
	promptStore := NewAIPromptStore(store.DB())

	proxy := NewProxy(cfg.TargetURL, logger)
	authMapper := NewAuthMapper(store, clientStore, authSvc.JWKS(), logger, cfg.IssuerURL, cfg.AudienceURL).
		WithShopifySessionAuth(cfg.ShopifyClientID, cfg.ShopifyClientSecret)
	// Let the self-authenticating handlers (auth/me, auth/users, the client
	// switcher) resolve Shopify session tokens too. Wired after construction
	// because the mapper depends on authSvc, not the other way round.
	authSvc.WithSessionResolver(authMapper)

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

	// Shopify App Store install entry. A merchant clicking "Install" on the
	// listing has no Bratrax session, so both routes bypass the auth mapper —
	// the `hmac` query param Shopify signs is the auth, verified in Flask.
	// Outside the /bratrax/ prefix so the redirect URI registered in the Partner
	// dashboard stays clean; the prefix-stripper leaves non-/bratrax paths
	// unchanged, so these reach shopify_install_routes verbatim. Registered
	// before the /bratrax/ catch-all for the same reason /email/pause is.
	observability.MuxHandle(mux, "GET /shopify/install",
		observability.Middleware("bratrax", logger, proxy))
	observability.MuxHandle(mux, "GET /shopify/install/callback",
		observability.Middleware("bratrax", logger, proxy))
	// Account creation against a parked install. Public for the same reason:
	// the merchant has no Bratrax session yet, and the install token they hold
	// (minted only after a verified OAuth round-trip with Shopify) is the
	// authorisation. /signup is waitlist-gated and /bratrax/auth/signup is
	// gated by ONLY_INVITATION_LINK — neither is appropriate for someone who
	// already found us on the App Store and granted scopes.
	observability.MuxHandle(mux, "POST /shopify/install/account",
		observability.Middleware("bratrax", logger, proxy))

	// Shopify App Pricing return landing. Shopify App Pricing sends no webhooks
	// (APP_SUBSCRIPTIONS_UPDATE was retired 2026-04-28), so this redirect —
	// carrying ?shop= and ?plan_handle= — is the only push notification the app
	// gets when a merchant approves a plan. Public because they arrive straight
	// from admin.shopify.com with no bratrax_auth cookie.
	//
	// Unlike the install routes above there is no hmac to verify: Shopify does
	// not sign this redirect. That is safe because the handler grants nothing —
	// it reads the live subscription from Shopify's Admin API using the shop's
	// own stored token and writes whatever Shopify reports. A forged call costs
	// one API read and rewrites the state that already held.
	observability.MuxHandle(mux, "GET /shopify/billing/return",
		observability.Middleware("bratrax", logger, proxy))

	// "Open in Bratrax" hand-off. A merchant inside the Shopify admin iframe is
	// authenticated by an App Bridge session token and therefore has no
	// bratrax_auth cookie on this origin, so a new tab would land on /login.
	// Flask mints a single-use token; this exchanges it for the normal cookie.
	// Unauthenticated by necessity — the token in `?t=` IS the credential — and
	// outside the /bratrax/ prefix so the auth catch-all can't swallow it.
	handoffSvc := NewEmbedHandoffService(store.DB(), authSvc, logger, cfg.SecureCookie)
	observability.MuxHandle(mux, "GET /auth/handoff",
		observability.Middleware("bratrax", logger, http.HandlerFunc(handoffSvc.HandleHandoff)))

	// NOTE: /payment-complete (the landing tab for an embedded Lemon Squeezy
	// checkout) is deliberately NOT registered here. It is a SvelteKit page,
	// not a Flask route, so it falls through to Rill's static handler like
	// /login and /forgot-password do. It only needs the isPublicRoute
	// allowlist entry in web-local/src/routes/+layout.ts. The three-place
	// public-route checklist in CLAUDE.md applies to FLASK routes outside the
	// /bratrax/ prefix — /email/pause and /auth/handoff above.

	// WooCommerce wc-auth callback. The merchant's store POSTs the generated
	// REST API key pair here server-to-server (no login cookie), so it must
	// bypass the auth mapper. Public by design — the signed `user_id` state
	// token in the body is the auth. Registered explicitly so it wins over the
	// /bratrax/ auth catch-all; the prefix-stripper forwards it to Flask as
	// /onboard/woocommerce/callback verbatim.
	observability.MuxHandle(mux, "POST /bratrax/onboard/woocommerce/callback",
		observability.Middleware("bratrax", logger, proxy))

	// WooCommerce tracking-plugin auto-link. The installed plugin POSTs its own
	// site_url here server-to-server (no login cookie) and gets linked to the
	// client that already connected this store via wc-auth, so it must bypass the
	// auth mapper. Public by design — same store->client posture as the Shopify
	// shop_domain resolver. The plugin.zip download is public too.
	observability.MuxHandle(mux, "POST /bratrax/onboard/woocommerce/plugin-autoconnect",
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
	// Cached + single-flighted, and serves the last known-good body when GitHub
	// fails. See github_static.go for why — in short, this used to fetch on
	// every page view and ordinary traffic could exhaust GitHub's per-IP rate
	// limit, taking the whole public site (homepage included) to 502.
	//
	// 5 min TTL: the static repo is edited by hand a few times a week, so this
	// is far fresher than needed while cutting GitHub requests to at most
	// 12/hour per page.
	//
	// 60s error TTL: while GitHub is failing we ask at most once a minute per
	// page instead of once per request. That is the difference between riding
	// out a rate limit and helping sustain it. Recovery is therefore detected
	// within a minute, which is fine for a marketing page.
	githubStatic := newGithubStaticCache(5*time.Minute, 60*time.Second, 8*time.Second)
	fetchGithubStatic := func(rawURL string) ([]byte, error) {
		body, stale, err := githubStatic.Get(rawURL)
		if err != nil {
			return nil, err
		}
		if stale {
			// Served from cache past its TTL because GitHub is unreachable or
			// throttling. Visitors see the right page; this is the only signal
			// that the upstream is unhealthy, so it is a warning, not a debug.
			logger.Warn("serving stale marketing content; github fetch failed",
				zap.String("github_url", rawURL))
		}
		return body, nil
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
				// Hand off to the SPA instead of 502ing. Every one of these
				// paths also has a SvelteKit route rendering the same GitHub
				// HTML via StaticHtmlPage — i.e. fetched by the VISITOR's
				// browser, from the visitor's IP, with the visitor's own rate
				// limit. So when our server IP is throttled, the client path
				// still works and the page renders.
				//
				// This is the only reason 2026-08-17's incident took the whole
				// public site down: server-side rendering was added for crawlable
				// meta tags, but with no fallback it converted an upstream
				// hiccup into a hard outage. Degrading to client-side rendering
				// costs per-page <meta>/<title> for crawlers until the cache
				// warms — a real cost, and an obviously smaller one than the
				// homepage being unreachable. Same trade the robots.txt handler
				// above already makes.
				logger.Warn("marketing page fetch failed; falling back to client-side render",
					zap.String("path", r.URL.Path),
					zap.String("github_url", githubURL),
					zap.Error(fetchErr))
				web.StaticHandler().ServeHTTP(w, r)
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
	observability.MuxHandle(mux, "GET /changelog",
		observability.Middleware("bratrax", logger,
			serveGithubHTML("https://raw.githubusercontent.com/yuolel/bratrax-wip/refs/heads/bratrax-com-static/changelog/index.html")))
	observability.MuxHandle(mux, "GET /pricing",
		observability.Middleware("bratrax", logger,
			serveGithubHTML("https://raw.githubusercontent.com/yuolel/bratrax-wip/refs/heads/bratrax-com-static/pricing/index.html")))
	observability.MuxHandle(mux, "GET /integrations",
		observability.Middleware("bratrax", logger,
			serveGithubHTML("https://raw.githubusercontent.com/yuolel/bratrax-wip/refs/heads/bratrax-com-static/integrations/index.html")))

	// Compliance pages, referenced from the Privacy Policy / DPA and by security
	// reviewers, so they need to be real crawlable documents at stable URLs.
	//
	// Note /data-processing-agreement is served from the repo's "dpa" directory —
	// the public URL is spelled out for readability while the source path stays
	// short. The two do not have to match, but the mismatch is easy to typo.
	observability.MuxHandle(mux, "GET /subprocessors",
		observability.Middleware("bratrax", logger,
			serveGithubHTML("https://raw.githubusercontent.com/yuolel/bratrax-wip/refs/heads/bratrax-com-static/subprocessors/index.html")))
	observability.MuxHandle(mux, "GET /security",
		observability.Middleware("bratrax", logger,
			serveGithubHTML("https://raw.githubusercontent.com/yuolel/bratrax-wip/refs/heads/bratrax-com-static/security/index.html")))
	observability.MuxHandle(mux, "GET /data-processing-agreement",
		observability.Middleware("bratrax", logger,
			serveGithubHTML("https://raw.githubusercontent.com/yuolel/bratrax-wip/refs/heads/bratrax-com-static/dpa/index.html")))

	// Per-integration landing page. /integrations/slack is the canonical URL, but
	// the source lives at the repo root as "slack/index.html" — same URL/source
	// mismatch as /data-processing-agreement above.
	//
	// Registered without a trailing slash, which in Go's ServeMux is an exact
	// match, so this does NOT shadow "GET /integrations" or catch any other
	// /integrations/* path.
	observability.MuxHandle(mux, "GET /integrations/slack",
		observability.Middleware("bratrax", logger,
			serveGithubHTML("https://raw.githubusercontent.com/yuolel/bratrax-wip/refs/heads/bratrax-com-static/slack/index.html")))

	// Same shape for Shopify, except the source path mirrors the public URL:
	// it lives at "integrations/shopify/index.html" in the static repo.
	observability.MuxHandle(mux, "GET /integrations/shopify",
		observability.Middleware("bratrax", logger,
			serveGithubHTML("https://raw.githubusercontent.com/yuolel/bratrax-wip/refs/heads/bratrax-com-static/integrations/shopify/index.html")))

	// /slack is the short vanity URL that appears in the Slack app listing and
	// marketing copy; /integrations/slack is canonical. A 301 consolidates link
	// equity on the canonical URL rather than leaving crawlers with two pages of
	// identical content.
	//
	// Exact match again, so /slack/events and /slack/oauth_redirect are
	// untouched. Those are Slack's webhook + OAuth endpoints and they are served
	// on api.bratrax.com, not here — but keeping this exact avoids any chance of
	// a future host-consolidation quietly redirecting Slack's callbacks.
	observability.MuxHandle(mux, "GET /slack",
		observability.Middleware("bratrax", logger, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/integrations/slack", http.StatusMovedPermanently)
		})))

	// /shopify is the same kind of vanity URL, redirecting to the canonical
	// /integrations/shopify. Exact match once more, so the Shopify App Store
	// install and billing endpoints registered above under /shopify/install and
	// /shopify/billing — plus the /shopify/connect page served by the SPA — are
	// all untouched.
	observability.MuxHandle(mux, "GET /shopify",
		observability.Middleware("bratrax", logger, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/integrations/shopify", http.StatusMovedPermanently)
		})))

	// /changelog/{slug}: per-entry pages for changelog items marked notable.
	// Unlike /vs/{slug} below, the slug set grows every time the
	// public-changelog routine runs, so this validates the slug's shape rather
	// than whitelisting each one — otherwise every new entry would need a Go
	// rebuild. A well-formed slug with no page upstream proxies GitHub's 404,
	// which fetchGithubStatic surfaces as a 502.
	//
	// Registered separately from "GET /changelog" above: a pattern without a
	// trailing slash is an exact match in Go's ServeMux, so the bare path does
	// not catch subpaths.
	observability.MuxHandle(mux, "GET /changelog/{slug}",
		observability.Middleware("bratrax", logger, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			slug := r.PathValue("slug")
			if !changelogSlugRe.MatchString(slug) {
				http.NotFound(w, r)
				return
			}
			serveGithubHTML("https://raw.githubusercontent.com/yuolel/bratrax-wip/refs/heads/bratrax-com-static/changelog/" + slug + "/index.html")(w, r)
		})))

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
	RegisterMCPHandler(mux, clientStore, authSvc, cfg.RuntimeAddr, ensureReady, logger)

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
		PromptStore: promptStore,
		Config:      cfg,
	}, nil
}
