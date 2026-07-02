package bratrax

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// mcpTestSetup wires a /bratrax/mcp handler in front of a fake upstream runtime.
// It returns the mux to serve requests against and a pointer to a bool that the
// fake upstream flips to true when it is actually reached (i.e. the request was
// proxied through, not short-circuited).
func mcpTestSetup(t *testing.T, ensureReady EnsureReadyFn) (*http.ServeMux, *bool) {
	t.Helper()

	_, authSvc, _, clientStore := setupAuthMapper(t)
	// A valid, provisioned MCP client for any brx_mcp_ token.
	clientStore.mcpClient = &Client{ClientID: "cod", ClickhouseDB: "cod_db"}

	upstreamHit := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		upstreamHit = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(upstream.Close)

	mux := http.NewServeMux()
	RegisterMCPHandler(mux, clientStore, authSvc, upstream.URL, ensureReady, zap.NewNop())
	return mux, &upstreamHit
}

func mcpInitRequest() *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/bratrax/mcp",
		strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`))
	req.Header.Set("Authorization", "Bearer brx_mcp_testtoken")
	req.Header.Set("Content-Type", "application/json")
	return req
}

// When the instance is ready, the request is proxied to the upstream runtime.
func TestMCPHandler_EnsureReadySucceeds_Proxies(t *testing.T) {
	ensured := false
	mux, upstreamHit := mcpTestSetup(t, func(_ context.Context, clientDB, _ string) error {
		ensured = true
		require.Equal(t, "cod_db", clientDB)
		return nil
	})

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, mcpInitRequest())

	require.True(t, ensured, "ensureReady should be invoked before proxying")
	require.True(t, *upstreamHit, "request should be proxied to the upstream runtime")
	require.Equal(t, http.StatusOK, rec.Code)
}

// When the instance can't be made ready, the handler returns a retryable 503
// with a Retry-After header instead of proxying into a hard 400.
func TestMCPHandler_EnsureReadyFails_Returns503(t *testing.T) {
	mux, upstreamHit := mcpTestSetup(t, func(_ context.Context, _, _ string) error {
		return errors.New("controller not ready")
	})

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, mcpInitRequest())

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	require.NotEmpty(t, rec.Header().Get("Retry-After"), "503 must carry Retry-After so the client backs off")
	require.False(t, *upstreamHit, "a not-ready instance must not be proxied to")
}

// A nil ensureReady (e.g. single-tenant mode) proxies as before, unchanged.
func TestMCPHandler_NilEnsureReady_Proxies(t *testing.T) {
	mux, upstreamHit := mcpTestSetup(t, nil)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, mcpInitRequest())

	require.True(t, *upstreamHit, "with nil ensureReady the handler should proxy unchanged")
	require.Equal(t, http.StatusOK, rec.Code)
}
