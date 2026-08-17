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

	c := newGithubStaticCache(time.Minute, 2*time.Second)
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
	c := newGithubStaticCache(0, 2*time.Second)

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

	if _, _, err := newGithubStaticCache(time.Minute, 2*time.Second).Get(srv.URL); err == nil {
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

	c := newGithubStaticCache(time.Minute, 2*time.Second)
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

	c := newGithubStaticCache(time.Minute, 2*time.Second)

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

// Distinct URLs must not block each other — the per-URL lock is held across the
// HTTP call, so a shared lock would serialise every marketing page behind one
// slow fetch.
func TestGithubStaticCacheDoesNotSerialiseDistinctURLs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		_, _ = w.Write([]byte("page"))
	}))
	defer srv.Close()

	c := newGithubStaticCache(time.Minute, 2*time.Second)

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
