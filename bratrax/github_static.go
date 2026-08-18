package bratrax

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sync"
	"time"
)

// errGithubBackoff is returned when a URL failed recently and nothing is cached
// for it, so we deliberately skipped the upstream request. Distinct from a live
// fetch error purely so the caller can log it differently — the response to the
// visitor is the same 502 either way.
var errGithubBackoff = errors.New("github fetch skipped: backing off after a recent failure")

// Cache for the marketing HTML (and sitemap/robots) authored in the
// bratrax-wip static repo and fetched from raw.githubusercontent.com.
//
// # WHY THIS EXISTS
//
// raw.githubusercontent.com rate-limits unauthenticated requests per source IP,
// and the handlers used to fetch on EVERY page view with no caching. On
// 2026-08-17 ordinary visitor traffic exhausted the limit and every public page
// — including the homepage — served 502 for roughly half an hour. Owning the
// repo does not help: the limit is GitHub's, applies to raw regardless of who
// owns the content, and cannot be turned off.
//
// Two properties matter, in this order:
//
//  1. STALE-ON-ERROR. If GitHub fails for any reason — rate limit, outage,
//     timeout, a bad deploy in the static repo — the last known-good body is
//     served regardless of age. Once an entry is warm, GitHub's availability
//     stops being a dependency of our marketing site. Slightly stale marketing
//     HTML always beats a 502. This is the property that actually fixes the
//     incident; the TTL is just about freshness.
//
//  2. SINGLE-FLIGHT PER URL. A burst of requests for a cold page would
//     otherwise fan out into N simultaneous fetches of the same URL, which is
//     the precise shape that trips a rate limit. One in-flight fetch per URL,
//     and the rest wait for it.
//
//  3. FAILURE BACKOFF. Caching only successes is not enough, and getting this
//     wrong prolongs the very outage the cache is meant to survive. Without a
//     backoff, a failing upstream means EVERY request attempts its own fetch —
//     when cold there is nothing to serve, and when warm-but-past-TTL the stale
//     fallback only kicks in after the request has already been made. Either
//     way request volume never drops, so our own traffic (plus crawlers across
//     eight pages) keeps a rate-limit bucket pinned empty and the outage
//     self-sustains. After a failure we stop asking for errorTTL, which caps
//     upstream load at one request per URL per errorTTL while degraded.
//
// Deliberately NOT a bounded/evicting cache: the key space is the fixed set of
// marketing pages plus changelog and /vs slugs. It is small, and every entry is
// worth keeping forever precisely because a kept entry is what survives an
// outage. Eviction would reintroduce the failure mode it exists to prevent.
type githubStaticCache struct {
	ttl      time.Duration
	errorTTL time.Duration
	client   *http.Client

	// Derives the Contents API URL for a raw URL. A field rather than a direct
	// call to githubContentsAPIURL so tests can point the fallback at a local
	// server; production never reassigns it.
	apiURLFor func(rawURL string) (string, bool)

	mu      sync.Mutex
	entries map[string]*githubStaticEntry
}

type githubStaticEntry struct {
	// Guards the fetch as well as the fields, so it doubles as the
	// single-flight lock. Held across the HTTP call on purpose: concurrent
	// callers for the same URL should wait and reuse the result rather than
	// each issuing their own request.
	mu        sync.Mutex
	body      []byte
	fetchedAt time.Time
	failedAt  time.Time
}

func newGithubStaticCache(ttl, errorTTL, timeout time.Duration) *githubStaticCache {
	return &githubStaticCache{
		ttl:       ttl,
		errorTTL:  errorTTL,
		client:    &http.Client{Timeout: timeout},
		apiURLFor: githubContentsAPIURL,
		entries:   make(map[string]*githubStaticEntry),
	}
}

// Get returns the body for rawURL.
//
// Returns (body, false, nil) when fresh or freshly fetched, (body, true, nil)
// when GitHub failed but a cached body exists — the caller should serve it and
// log that it is stale — and (nil, false, err) only when GitHub failed and
// nothing has ever been cached for this URL.
//
// A non-200 from GitHub is an error, including 404. That is intentional for the
// slug routes: a well-formed slug with no page upstream should surface as a
// failure rather than being cached as an empty page.
func (c *githubStaticCache) Get(rawURL string) ([]byte, bool, error) {
	entry := c.entryFor(rawURL)

	entry.mu.Lock()
	defer entry.mu.Unlock()

	if entry.body != nil && time.Since(entry.fetchedAt) < c.ttl {
		return entry.body, false, nil
	}

	// Recently failed — don't ask again yet. This is what keeps us from
	// hammering a throttled upstream once per request and prolonging the outage.
	if !entry.failedAt.IsZero() && time.Since(entry.failedAt) < c.errorTTL {
		if entry.body != nil {
			return entry.body, true, nil
		}
		return nil, false, errGithubBackoff
	}

	body, err := c.fetch(rawURL)
	if err != nil && entry.body == nil {
		// Cold and raw is failing — the one case nothing above can rescue, and
		// the one that takes the site down. Try the Contents API, which is a
		// SEPARATE rate-limit bucket: on 2026-08-17 it answered 200 throughout
		// the raw endpoint's 429.
		//
		// Only attempted when cold, and that restraint is the whole design.
		// Unauthenticated the API allows just 60 requests/hour — far tighter
		// than raw — so it cannot carry ongoing traffic. It doesn't need to:
		// one success makes the entry warm, after which stale-on-error covers
		// every later failure. That caps API use at roughly one call per page
		// per process, which fits the quota many times over.
		if apiURL, ok := c.apiURLFor(rawURL); ok {
			if apiBody, apiErr := c.fetchAPI(apiURL); apiErr == nil {
				entry.body = apiBody
				entry.fetchedAt = time.Now()
				entry.failedAt = time.Time{}
				return apiBody, false, nil
			}
		}
	}
	if err != nil {
		entry.failedAt = time.Now()
		if entry.body != nil {
			return entry.body, true, nil
		}
		return nil, false, err
	}

	entry.body = body
	entry.fetchedAt = time.Now()
	entry.failedAt = time.Time{}
	return body, false, nil
}

func (c *githubStaticCache) entryFor(rawURL string) *githubStaticEntry {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[rawURL]
	if !ok {
		entry = &githubStaticEntry{}
		c.entries[rawURL] = entry
	}
	return entry
}

func (c *githubStaticCache) fetch(rawURL string) ([]byte, error) {
	resp, err := c.client.Get(rawURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github raw returned HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// rawGithubURLRe matches the one URL shape every call site uses:
//
//	https://raw.githubusercontent.com/<owner>/<repo>/refs/heads/<branch>/<path>
//
// Captures owner/repo, branch and path so the same file can be requested from
// the Contents API instead.
var rawGithubURLRe = regexp.MustCompile(
	`^https://raw\.githubusercontent\.com/([^/]+/[^/]+)/refs/heads/([^/]+)/(.+)$`)

// githubContentsAPIURL converts a raw.githubusercontent.com URL into the
// equivalent Contents API URL. Returns false for anything that doesn't match the
// expected shape, so an unrecognised URL simply skips the fallback rather than
// producing a nonsense request.
func githubContentsAPIURL(rawURL string) (string, bool) {
	m := rawGithubURLRe.FindStringSubmatch(rawURL)
	if m == nil {
		return "", false
	}
	// EscapedPath, NOT PathEscape: the latter escapes the separators too, so
	// "faq/index.html" becomes "faq%2Findex.html" and the API 404s. This escapes
	// each segment while leaving the slashes intact.
	escapedPath := (&url.URL{Path: m[3]}).EscapedPath()
	return fmt.Sprintf("https://api.github.com/repos/%s/contents/%s?ref=%s",
		m[1], escapedPath, url.QueryEscape(m[2])), true
}

// fetchAPI retrieves file contents through the GitHub Contents API.
//
// The `Accept: application/vnd.github.raw` header asks for the file bytes
// directly; without it the API returns a JSON envelope with base64 content,
// which would need decoding and would silently be served as HTML.
//
// GITHUB_TOKEN is honoured if present, purely to lift the unauthenticated
// 60/hour limit to 5000. Entirely optional — the cold-start-only usage above
// fits inside 60/hour on its own.
func (c *githubStaticCache) fetchAPI(apiURL string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.raw")
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github contents api returned HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
