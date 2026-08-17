package bratrax

import (
	"errors"
	"fmt"
	"io"
	"net/http"
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
// WHY THIS EXISTS
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
		ttl:      ttl,
		errorTTL: errorTTL,
		client:   &http.Client{Timeout: timeout},
		entries:  make(map[string]*githubStaticEntry),
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
