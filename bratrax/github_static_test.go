package bratrax

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// The behaviour worth pinning down is the incident behaviour: what happens when
// GitHub starts failing. A cache that only works on the happy path would have
// been no help on 2026-08-17.

func TestGithubStaticCacheServesFromCacheWithinTTL(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		_, _ = w.Write([]byte("page"))
	}))
	defer srv.Close()

	c := newGithubStaticCache(time.Minute, 0, 2*time.Second)
	for i := 0; i < 5; i++ {
		body, stale, err := c.Get(srv.URL)
		if err != nil || stale || string(body) != "page" {
			t.Fatalf("call %d: got (%q, stale=%v, err=%v)", i, body, stale, err)
		}
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Fatalf("expected 1 upstream fetch, got %d", got)
	}
}

// The incident case: content was fetched once, then GitHub starts returning 429.
// Visitors must keep seeing the page.
func TestGithubStaticCacheServesStaleWhenUpstreamFails(t *testing.T) {
	var fail atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fail.Load() {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte("good"))
	}))
	defer srv.Close()

	// Zero TTL so every call is a refresh attempt — the worst case.
	c := newGithubStaticCache(0, 0, 2*time.Second)

	if _, _, err := c.Get(srv.URL); err != nil {
		t.Fatalf("warm-up fetch failed: %v", err)
	}

	fail.Store(true)
	body, stale, err := c.Get(srv.URL)
	if err != nil {
		t.Fatalf("expected stale success, got error: %v", err)
	}
	if !stale {
		t.Fatal("expected stale=true so the caller can log it")
	}
	if string(body) != "good" {
		t.Fatalf("expected last known-good body, got %q", body)
	}
}

// Nothing cached and GitHub failing is the one case that must still error —
// serving an empty page would be worse than a 502.
func TestGithubStaticCacheErrorsWhenColdAndUpstreamFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	if _, _, err := newGithubStaticCache(time.Minute, 0, 2*time.Second).Get(srv.URL); err == nil {
		t.Fatal("expected an error when cold and upstream fails")
	}
}

// A 404 must not be cached as a valid page — the /changelog/{slug} and
// /vs/{slug} routes accept any well-formed slug, so an unknown one has to keep
// failing rather than being remembered as an empty document.
func TestGithubStaticCacheTreatsNotFoundAsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	c := newGithubStaticCache(time.Minute, 0, 2*time.Second)
	if _, _, err := c.Get(srv.URL); err == nil {
		t.Fatal("expected 404 to surface as an error")
	}
	if body, _, err := c.Get(srv.URL); err == nil {
		t.Fatalf("404 must not be cached as a page, got %q", body)
	}
}

// Single-flight: a burst on a cold URL is exactly what trips a rate limit, so it
// must collapse to one upstream request.
func TestGithubStaticCacheSingleFlightsConcurrentColdRequests(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		time.Sleep(50 * time.Millisecond) // widen the race window
		_, _ = w.Write([]byte("page"))
	}))
	defer srv.Close()

	c := newGithubStaticCache(time.Minute, 0, 2*time.Second)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, _, err := c.Get(srv.URL); err != nil {
				t.Errorf("concurrent Get failed: %v", err)
			}
		}()
	}
	wg.Wait()

	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Fatalf("expected 20 concurrent cold requests to collapse to 1 fetch, got %d", got)
	}
}

// The raw -> Contents API conversion. The escaping is the subtle part: PathEscape
// would turn "faq/index.html" into "faq%2Findex.html" and the API would 404,
// silently disabling the fallback exactly when it is needed.
func TestGithubContentsAPIURL(t *testing.T) {
	const base = "https://raw.githubusercontent.com/yuolel/bratrax-wip/refs/heads/bratrax-com-static/"

	got, ok := githubContentsAPIURL(base + "faq/index.html")
	if !ok {
		t.Fatal("expected the standard raw URL shape to convert")
	}
	want := "https://api.github.com/repos/yuolel/bratrax-wip/contents/faq/index.html?ref=bratrax-com-static"
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}

	// Nested path (the /changelog/{slug} and /vs/{slug} shape).
	got, _ = githubContentsAPIURL(base + "vs/hyros/index.html")
	if want := "https://api.github.com/repos/yuolel/bratrax-wip/contents/vs/hyros/index.html?ref=bratrax-com-static"; got != want {
		t.Fatalf("nested path:\ngot  %s\nwant %s", got, want)
	}

	// Anything not matching the expected shape must opt out rather than
	// produce a nonsense request.
	for _, bad := range []string{
		"https://example.com/foo/bar",
		"https://raw.githubusercontent.com/owner/repo/main/file.html", // no refs/heads
		"not a url",
	} {
		if _, ok := githubContentsAPIURL(bad); ok {
			t.Fatalf("expected %q to be rejected", bad)
		}
	}
}

// Cold + raw failing must fall through to the Contents API. This is the path
// that actually restores the site during a raw-endpoint outage.
func TestGithubStaticCacheFallsBackToContentsAPIWhenCold(t *testing.T) {
	var apiHits int32
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&apiHits, 1)
		if r.Header.Get("Accept") != "application/vnd.github.raw" {
			t.Errorf("missing raw Accept header; would get a base64 JSON envelope")
		}
		_, _ = w.Write([]byte("from-api"))
	}))
	defer api.Close()

	raw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer raw.Close()

	c := newGithubStaticCache(time.Minute, time.Minute, 2*time.Second)
	c.apiURLFor = func(string) (string, bool) { return api.URL, true }

	body, stale, err := c.Get(raw.URL)
	if err != nil || stale || string(body) != "from-api" {
		t.Fatalf("got (%q, stale=%v, err=%v); want the API body", body, stale, err)
	}

	// Now warm: further calls must be served from cache, not re-hit the scarce API.
	for i := 0; i < 5; i++ {
		if _, _, err := c.Get(raw.URL); err != nil {
			t.Fatalf("call %d after warm: %v", i, err)
		}
	}
	if got := atomic.LoadInt32(&apiHits); got != 1 {
		t.Fatalf("API must be a cold-start rescue only; got %d hits", got)
	}
}

// The backoff is the part that stops us prolonging an upstream outage. Caching
// only successes left every request making its own doomed fetch, so our own
// traffic kept the rate-limit bucket empty. These two tests pin the fix.

func TestGithubStaticCacheBacksOffWhenColdAndFailing(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := newGithubStaticCache(time.Minute, time.Minute, 2*time.Second)
	for i := 0; i < 10; i++ {
		if _, _, err := c.Get(srv.URL); err == nil {
			t.Fatalf("call %d: expected an error while cold and failing", i)
		}
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Fatalf("expected 10 requests during an outage to make 1 upstream fetch, got %d", got)
	}
}

// Warm but past its TTL is the sneakier case: the stale fallback only engages
// after the fetch has already been attempted, so without a backoff the upstream
// still sees one request per pageview even though visitors are served fine.
func TestGithubStaticCacheBacksOffWhenWarmAndFailing(t *testing.T) {
	var hits int32
	var fail atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		if fail.Load() {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte("good"))
	}))
	defer srv.Close()

	// ttl=0 so every call wants a refresh; errorTTL should suppress them anyway.
	c := newGithubStaticCache(0, time.Minute, 2*time.Second)
	if _, _, err := c.Get(srv.URL); err != nil {
		t.Fatalf("warm-up failed: %v", err)
	}
	fail.Store(true)

	for i := 0; i < 10; i++ {
		body, stale, err := c.Get(srv.URL)
		if err != nil || !stale || string(body) != "good" {
			t.Fatalf("call %d: got (%q, stale=%v, err=%v); want stale good body", i, body, stale, err)
		}
	}
	// 1 warm-up + 1 failed refresh; the other 9 must be suppressed.
	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Fatalf("expected 2 upstream fetches total, got %d", got)
	}
}

// Recovery must not be blocked by the backoff once it expires.
func TestGithubStaticCacheRecoversAfterBackoffExpires(t *testing.T) {
	var fail atomic.Bool
	fail.Store(true)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fail.Load() {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte("recovered"))
	}))
	defer srv.Close()

	c := newGithubStaticCache(time.Minute, 20*time.Millisecond, 2*time.Second)
	if _, _, err := c.Get(srv.URL); err == nil {
		t.Fatal("expected initial failure")
	}
	fail.Store(false)
	time.Sleep(40 * time.Millisecond) // let the backoff lapse

	body, stale, err := c.Get(srv.URL)
	if err != nil || stale || string(body) != "recovered" {
		t.Fatalf("got (%q, stale=%v, err=%v); want a fresh recovered body", body, stale, err)
	}
}

// Distinct URLs must not block each other — the per-URL lock is held across the
// HTTP call, so a shared lock would serialise every marketing page behind one
// slow fetch.
func TestGithubStaticCacheDoesNotSerialiseDistinctURLs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		_, _ = w.Write([]byte("page"))
	}))
	defer srv.Close()

	c := newGithubStaticCache(time.Minute, 0, 2*time.Second)

	start := time.Now()
	var wg sync.WaitGroup
	for _, p := range []string{"/a", "/b", "/c", "/d"} {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			if _, _, err := c.Get(srv.URL + path); err != nil {
				t.Errorf("Get(%s) failed: %v", path, err)
			}
		}(p)
	}
	wg.Wait()

	// Serialised would be ~400ms; parallel ~100ms. 250ms separates them safely.
	if elapsed := time.Since(start); elapsed > 250*time.Millisecond {
		t.Fatalf("distinct URLs appear serialised: 4 fetches took %v", elapsed)
	}
}
