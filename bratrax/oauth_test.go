package bratrax

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// fakeOAuthStore is an in-memory oauthStore. It mirrors the SQL semantics the
// flow depends on: single-use codes, claim-once pending rows, and refresh
// rotation with a grace window and reuse revocation.
type fakeOAuthStore struct {
	mu         sync.Mutex
	clients    map[string]*oauthClient
	pending    map[string]*pendingAuthorization
	codes      map[string]string // code hash → resume id
	refresh    map[string]*fakeRefresh
	workspaces map[int][]workspaceStatus
	steps      map[string]string
	users      map[int]bool
}

type fakeRefresh struct {
	grant     oauthGrant
	expiresAt time.Time
	rotatedAt *time.Time
	revokedAt *time.Time
}

func newFakeOAuthStore() *fakeOAuthStore {
	return &fakeOAuthStore{
		clients: map[string]*oauthClient{}, pending: map[string]*pendingAuthorization{},
		codes: map[string]string{}, refresh: map[string]*fakeRefresh{},
		workspaces: map[int][]workspaceStatus{}, steps: map[string]string{}, users: map[int]bool{},
	}
}

func (f *fakeOAuthStore) GetClient(_ context.Context, id string) (*oauthClient, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.clients[id], nil
}

func (f *fakeOAuthStore) UpsertClient(_ context.Context, c *oauthClient) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.clients[c.ClientID] = c
	return nil
}

func (f *fakeOAuthStore) TouchClient(context.Context, string) error               { return nil }
func (f *fakeOAuthStore) PruneUnusedClients(context.Context, time.Duration) error { return nil }
func (f *fakeOAuthStore) PruneStalePending(context.Context) error                 { return nil }
func (f *fakeOAuthStore) SetPendingGrant(_ context.Context, resumeID, grantID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pending[resumeID].GrantID = &grantID
	return nil
}

func (f *fakeOAuthStore) CreatePending(_ context.Context, p *pendingAuthorization) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := *p
	cp.CreatedAt = time.Now()
	f.pending[p.ResumeID] = &cp
	return nil
}

func (f *fakeOAuthStore) GetPending(_ context.Context, id string) (*pendingAuthorization, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if p, ok := f.pending[id]; ok {
		cp := *p
		return &cp, nil
	}
	return nil, nil
}

func (f *fakeOAuthStore) ClaimPending(_ context.Context, id string, userID int) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.pending[id]
	if !ok || (p.UserID != nil && *p.UserID != userID) {
		return false, nil
	}
	p.UserID = &userID
	return true, nil
}

func (f *fakeOAuthStore) SetAuthCode(_ context.Context, id string, userID int, selected, codeHash string, exp time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.pending[id]
	if !ok || p.UserID == nil || *p.UserID != userID || p.ConsumedAt != nil || p.DeniedAt != nil {
		return errPendingGone
	}
	p.SelectedClientID = &selected
	p.AuthCodeExpiresAt = &exp
	f.codes[codeHash] = id
	return nil
}

func (f *fakeOAuthStore) DenyPending(_ context.Context, id string, _ int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	now := time.Now()
	f.pending[id].DeniedAt = &now
	return nil
}

func (f *fakeOAuthStore) ConsumeAuthCode(_ context.Context, codeHash string) (*pendingAuthorization, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.pending[f.codes[codeHash]]
	if !ok || p.ConsumedAt != nil || p.AuthCodeExpiresAt == nil || time.Now().After(*p.AuthCodeExpiresAt) {
		return nil, nil
	}
	now := time.Now()
	p.ConsumedAt = &now
	cp := *p
	return &cp, nil
}

func (f *fakeOAuthStore) GetPendingByCode(_ context.Context, codeHash string) (*pendingAuthorization, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if p, ok := f.pending[f.codes[codeHash]]; ok {
		cp := *p
		return &cp, nil
	}
	return nil, nil
}

func (f *fakeOAuthStore) InsertRefreshToken(_ context.Context, hash string, g *oauthGrant, exp time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.refresh[hash] = &fakeRefresh{grant: *g, expiresAt: exp}
	return nil
}

func (f *fakeOAuthStore) RotateRefreshToken(_ context.Context, hash string, grace time.Duration) (*oauthGrant, rotateOutcome, error) {
	f.mu.Lock()
	r, ok := f.refresh[hash]
	if !ok || r.revokedAt != nil || time.Now().After(r.expiresAt) {
		f.mu.Unlock()
		return nil, rotateInvalid, nil
	}
	if r.rotatedAt != nil && time.Since(*r.rotatedAt) > grace {
		grantID := r.grant.GrantID
		f.mu.Unlock()
		_ = f.RevokeGrant(context.Background(), grantID)
		return nil, rotateReused, nil
	}
	if r.rotatedAt == nil {
		now := time.Now()
		r.rotatedAt = &now
	}
	g := r.grant
	f.mu.Unlock()
	return &g, rotateOK, nil
}

func (f *fakeOAuthStore) RevokeGrant(_ context.Context, grantID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	now := time.Now()
	for _, r := range f.refresh {
		if r.grant.GrantID == grantID && r.revokedAt == nil {
			r.revokedAt = &now
		}
	}
	return nil
}

func (f *fakeOAuthStore) GrantActive(_ context.Context, grantID string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, r := range f.refresh {
		if r.grant.GrantID == grantID && r.revokedAt == nil && f.users[r.grant.UserID] {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeOAuthStore) WorkspacesForUser(_ context.Context, u *User) ([]workspaceStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := append([]workspaceStatus{}, f.workspaces[u.ID]...)
	for i := range out {
		if step, ok := f.steps[out[i].ClientID]; ok {
			out[i].Step = &step
		}
	}
	return out, nil
}

func (f *fakeOAuthStore) WorkspaceStatus(_ context.Context, clientID string) (*workspaceStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, list := range f.workspaces {
		for _, w := range list {
			if w.ClientID == clientID {
				return &w, nil
			}
		}
	}
	return nil, nil
}

func (f *fakeOAuthStore) OnboardingStep(_ context.Context, clientID string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.steps[clientID], nil
}

// oauthTestEnv is a mux with the OAuth routes and /bratrax/mcp in front of a
// fake runtime, plus a session cookie for the mock admin (user 1, client cod).
type oauthTestEnv struct {
	mux         *http.ServeMux
	store       *fakeOAuthStore
	svc         *OAuthService
	cookie      *http.Cookie
	upstreamHit *bool
}

const testRedirect = "https://claude.ai/api/mcp/auth_callback"

func newOAuthTestEnv(t *testing.T) *oauthTestEnv {
	t.Helper()
	mapper, authSvc, _, clientStore := setupAuthMapper(t)
	store := newFakeOAuthStore()
	store.users[1] = true
	store.workspaces[1] = []workspaceStatus{{ClientID: "cod", CompanyName: "Test Corp", ClickhouseDB: "cod_db"}}
	store.steps["cod"] = "ready"

	svc := NewOAuthService(store, authSvc, mapper, clientStore, "https://bratrax.test", nil, zap.NewNop())
	mux := http.NewServeMux()
	svc.RegisterRoutes(mux)

	hit := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hit = true
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(upstream.Close)
	RegisterMCPHandler(mux, clientStore, authSvc, svc, upstream.URL, nil, zap.NewNop())

	return &oauthTestEnv{
		mux: mux, store: store, svc: svc, upstreamHit: &hit,
		cookie: loginAsUser(t, authSvc, "admin@bratrax.com", "admin123"),
	}
}

func (e *oauthTestEnv) do(req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	e.mux.ServeHTTP(rec, req)
	return rec
}

// register runs DCR and returns the client_id.
func (e *oauthTestEnv) register(t *testing.T, redirect string) (string, int) {
	t.Helper()
	body := `{"client_name":"Claude","redirect_uris":["` + redirect + `"]}`
	rec := e.do(httptest.NewRequest(http.MethodPost, "/bratrax/oauth/register", strings.NewReader(body)))
	if rec.Code != http.StatusCreated {
		return "", rec.Code
	}
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	return resp["client_id"].(string), rec.Code
}

func pkcePair() (verifier, challenge string) {
	verifier = randomToken(48)
	sum := sha256.Sum256([]byte(verifier))
	return verifier, base64.RawURLEncoding.EncodeToString(sum[:])
}

func authorizeURL(clientID, challenge string) string {
	q := url.Values{
		"response_type": {"code"}, "client_id": {clientID}, "redirect_uri": {testRedirect},
		"code_challenge": {challenge}, "code_challenge_method": {"S256"}, "state": {"xyz"},
		"resource": {"https://bratrax.test/bratrax/mcp"},
	}
	return "/bratrax/oauth/authorize?" + q.Encode()
}

// connect runs authorize → pending → decision → token for the signed-in admin
// and returns the token response.
func (e *oauthTestEnv) connect(t *testing.T) map[string]any {
	t.Helper()
	clientID, _ := e.register(t, testRedirect)
	verifier, challenge := pkcePair()

	req := httptest.NewRequest(http.MethodGet, authorizeURL(clientID, challenge), nil)
	req.AddCookie(e.cookie)
	rec := e.do(req)
	require.Equal(t, http.StatusFound, rec.Code)
	loc, _ := url.Parse(rec.Header().Get("Location"))
	require.Equal(t, oauthConsentPath, loc.Path, "a signed-in user goes straight to consent")
	resume := loc.Query().Get("resume")

	req = httptest.NewRequest(http.MethodGet, "/bratrax/oauth/pending?resume="+resume, nil)
	req.AddCookie(e.cookie)
	rec = e.do(req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var view pendingView
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &view))
	require.Equal(t, "ready", view.Status)
	require.Equal(t, "claude_connector", view.Source)

	code := e.approve(t, resume, "cod")
	return e.exchange(t, url.Values{
		"grant_type": {"authorization_code"}, "code": {code}, "code_verifier": {verifier},
		"redirect_uri": {testRedirect}, "client_id": {clientID},
	}, http.StatusOK)
}

func (e *oauthTestEnv) approve(t *testing.T, resume, clientID string) string {
	t.Helper()
	body := `{"resume_id":"` + resume + `","approve":true,"client_id":"` + clientID + `"}`
	req := httptest.NewRequest(http.MethodPost, "/bratrax/oauth/decision", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(e.cookie)
	rec := e.do(req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var resp map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	to, err := url.Parse(resp["redirect_to"])
	require.NoError(t, err)
	require.Equal(t, "claude.ai", to.Host)
	require.Equal(t, "xyz", to.Query().Get("state"))
	return to.Query().Get("code")
}

func (e *oauthTestEnv) exchange(t *testing.T, form url.Values, wantStatus int) map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/bratrax/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := e.do(req)
	require.Equal(t, wantStatus, rec.Code, rec.Body.String())
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	return resp
}

func (e *oauthTestEnv) callMCP(token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/bratrax/mcp",
		strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	return e.do(req)
}

func TestOAuth_Metadata(t *testing.T) {
	e := newOAuthTestEnv(t)

	rec := e.do(httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	var prm map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &prm))
	require.Equal(t, "https://bratrax.test/bratrax/mcp", prm["resource"])
	require.Equal(t, []any{"https://bratrax.test"}, prm["authorization_servers"])

	// The RFC 9728 path-suffixed location serves the same document.
	rec = e.do(httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource/bratrax/mcp", nil))
	require.Equal(t, http.StatusOK, rec.Code)

	rec = e.do(httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	var as map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &as))
	require.Equal(t, "https://bratrax.test", as["issuer"])
	require.Equal(t, []any{"S256"}, as["code_challenge_methods_supported"])
	require.Equal(t, true, as["client_id_metadata_document_supported"])
}

// Without a credential, /bratrax/mcp must point at the metadata: it is the only
// thing that starts Claude's sign-in flow.
func TestOAuth_MCP401CarriesResourceMetadata(t *testing.T) {
	e := newOAuthTestEnv(t)
	rec := e.do(httptest.NewRequest(http.MethodPost, "/bratrax/mcp", strings.NewReader(`{}`)))
	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.Contains(t, rec.Header().Get("WWW-Authenticate"),
		`resource_metadata="https://bratrax.test/.well-known/oauth-protected-resource"`)
}

func TestOAuth_RegisterRedirectAllowlist(t *testing.T) {
	e := newOAuthTestEnv(t)
	for _, tc := range []struct {
		redirect string
		want     int
	}{
		{"https://claude.ai/api/mcp/auth_callback", http.StatusCreated},
		{"https://chatgpt.com/connector_platform_oauth_redirect", http.StatusCreated},
		{"http://localhost:33418/callback", http.StatusCreated},
		{"http://127.0.0.1:6274/oauth/callback", http.StatusCreated},
		{"https://evil.example/cb", http.StatusBadRequest},
		{"http://claude.ai/api/mcp/auth_callback", http.StatusBadRequest},
		{"https://claude.ai.evil.example/cb", http.StatusBadRequest},
		{"javascript:alert(1)", http.StatusBadRequest},
	} {
		_, code := e.register(t, tc.redirect)
		require.Equal(t, tc.want, code, tc.redirect)
	}
}

func TestOAuth_AuthorizeWithoutSessionGoesToLogin(t *testing.T) {
	e := newOAuthTestEnv(t)
	clientID, _ := e.register(t, testRedirect)
	_, challenge := pkcePair()

	rec := e.do(httptest.NewRequest(http.MethodGet, authorizeURL(clientID, challenge), nil))
	require.Equal(t, http.StatusFound, rec.Code)
	loc, _ := url.Parse(rec.Header().Get("Location"))
	require.Equal(t, "/login", loc.Path)
	next, _ := url.Parse(loc.Query().Get("redirect"))
	require.Equal(t, oauthConsentPath, next.Path)

	// The request is parked, unclaimed, for whoever signs in (or up) next.
	p, _ := e.store.GetPending(context.Background(), next.Query().Get("resume"))
	require.NotNil(t, p)
	require.Nil(t, p.UserID)

	// Branding info is public and names the client.
	rec = e.do(httptest.NewRequest(http.MethodGet, "/bratrax/oauth/pending/info?resume="+p.ResumeID, nil))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"client_name":"Claude"`)
}

func TestOAuth_AuthorizeRejectsBadRequests(t *testing.T) {
	e := newOAuthTestEnv(t)
	clientID, _ := e.register(t, testRedirect)
	_, challenge := pkcePair()

	// Unknown client: an error page, never a redirect.
	rec := e.do(httptest.NewRequest(http.MethodGet, authorizeURL("nope", challenge), nil))
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Empty(t, rec.Header().Get("Location"))

	// Redirect URI the client never registered: same.
	u := strings.Replace(authorizeURL(clientID, challenge), url.QueryEscape(testRedirect), url.QueryEscape("https://claude.ai/other"), 1)
	rec = e.do(httptest.NewRequest(http.MethodGet, u, nil))
	require.Equal(t, http.StatusBadRequest, rec.Code)

	// Missing PKCE: redirected back with an OAuth error.
	u = strings.Replace(authorizeURL(clientID, challenge), "code_challenge_method=S256", "code_challenge_method=plain", 1)
	rec = e.do(httptest.NewRequest(http.MethodGet, u, nil))
	require.Equal(t, http.StatusFound, rec.Code)
	loc, _ := url.Parse(rec.Header().Get("Location"))
	require.Equal(t, "invalid_request", loc.Query().Get("error"))
	require.Equal(t, "xyz", loc.Query().Get("state"))
}

func TestOAuth_FullFlowAndMCPAccess(t *testing.T) {
	e := newOAuthTestEnv(t)
	tokens := e.connect(t)
	access := tokens["access_token"].(string)
	require.True(t, strings.HasPrefix(tokens["refresh_token"].(string), oauthRefreshTokenPref))
	require.EqualValues(t, 3600, tokens["expires_in"])

	rec := e.callMCP(access)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.True(t, *e.upstreamHit, "a ready workspace is proxied to the runtime")
}

// An access token is not a session and a session is not an access token: the
// audiences differ, so neither is accepted in the other's place.
func TestOAuth_SessionCookieIsNotAnAccessToken(t *testing.T) {
	e := newOAuthTestEnv(t)
	rec := e.callMCP(e.cookie.Value)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.False(t, *e.upstreamHit)
}

func TestOAuth_CodeIsSingleUseAndReplayRevokes(t *testing.T) {
	e := newOAuthTestEnv(t)
	clientID, _ := e.register(t, testRedirect)
	verifier, challenge := pkcePair()
	req := httptest.NewRequest(http.MethodGet, authorizeURL(clientID, challenge), nil)
	req.AddCookie(e.cookie)
	loc, _ := url.Parse(e.do(req).Header().Get("Location"))
	resume := loc.Query().Get("resume")
	req = httptest.NewRequest(http.MethodGet, "/bratrax/oauth/pending?resume="+resume, nil)
	req.AddCookie(e.cookie)
	e.do(req)
	code := e.approve(t, resume, "cod")

	form := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "code_verifier": {verifier}, "redirect_uri": {testRedirect}}
	tokens := e.exchange(t, form, http.StatusOK)

	// A replay fails and takes the grant it produced down with it.
	e.exchange(t, form, http.StatusBadRequest)
	require.Equal(t, http.StatusUnauthorized, e.callMCP(tokens["access_token"].(string)).Code)
}

func TestOAuth_WrongVerifierRejected(t *testing.T) {
	e := newOAuthTestEnv(t)
	clientID, _ := e.register(t, testRedirect)
	_, challenge := pkcePair()
	req := httptest.NewRequest(http.MethodGet, authorizeURL(clientID, challenge), nil)
	req.AddCookie(e.cookie)
	loc, _ := url.Parse(e.do(req).Header().Get("Location"))
	resume := loc.Query().Get("resume")
	req = httptest.NewRequest(http.MethodGet, "/bratrax/oauth/pending?resume="+resume, nil)
	req.AddCookie(e.cookie)
	e.do(req)
	code := e.approve(t, resume, "cod")

	other, _ := pkcePair()
	e.exchange(t, url.Values{"grant_type": {"authorization_code"}, "code": {code}, "code_verifier": {other}}, http.StatusBadRequest)
}

func TestOAuth_RefreshRotationAndReuse(t *testing.T) {
	e := newOAuthTestEnv(t)
	first := e.connect(t)
	rt1 := first["refresh_token"].(string)

	second := e.exchange(t, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {rt1}}, http.StatusOK)
	rt2 := second["refresh_token"].(string)
	require.NotEqual(t, rt1, rt2)

	// Within the grace window a retried refresh still works (lost response).
	e.exchange(t, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {rt1}}, http.StatusOK)

	// Past it, reuse is theft: the whole grant goes.
	e.svc.store.(*fakeOAuthStore).mu.Lock()
	past := time.Now().Add(-time.Hour)
	e.svc.store.(*fakeOAuthStore).refresh[hashToken(rt1)].rotatedAt = &past
	e.svc.store.(*fakeOAuthStore).mu.Unlock()
	e.exchange(t, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {rt1}}, http.StatusBadRequest)
	e.exchange(t, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {rt2}}, http.StatusBadRequest)
	require.Equal(t, http.StatusUnauthorized, e.callMCP(second["access_token"].(string)).Code)
}

// The server-side half of "connect only after onboarding": an unready
// workspace gets no code, however the consent page is driven.
func TestOAuth_UnreadyWorkspaceCannotBeApproved(t *testing.T) {
	e := newOAuthTestEnv(t)
	e.store.steps["cod"] = "platforms_connected"
	clientID, _ := e.register(t, testRedirect)
	_, challenge := pkcePair()
	req := httptest.NewRequest(http.MethodGet, authorizeURL(clientID, challenge), nil)
	req.AddCookie(e.cookie)
	loc, _ := url.Parse(e.do(req).Header().Get("Location"))
	resume := loc.Query().Get("resume")

	req = httptest.NewRequest(http.MethodGet, "/bratrax/oauth/pending?resume="+resume, nil)
	req.AddCookie(e.cookie)
	rec := e.do(req)
	var view pendingView
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &view))
	require.Equal(t, "needs_onboarding", view.Status)
	require.Equal(t, "platforms_connected", *view.Step)

	body := `{"resume_id":"` + resume + `","approve":true,"client_id":"cod"}`
	req = httptest.NewRequest(http.MethodPost, "/bratrax/oauth/decision", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(e.cookie)
	require.Equal(t, http.StatusConflict, e.do(req).Code)
}

// While activation runs, a connected Claude gets an explanation, not a 503.
func TestOAuth_NotReadyWorkspaceGetsStatusServer(t *testing.T) {
	e := newOAuthTestEnv(t)
	tokens := e.connect(t)
	e.store.steps["cod"] = "extracting"

	rec := e.callMCP(tokens["access_token"].(string))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.False(t, *e.upstreamHit, "an unready workspace is never proxied")
	require.Contains(t, rec.Body.String(), notReadyToolName)
}

func TestOAuth_PendingBelongsToFirstClaimant(t *testing.T) {
	e := newOAuthTestEnv(t)
	other := 99
	require.NoError(t, e.store.CreatePending(context.Background(), &pendingAuthorization{
		ResumeID: "r1", OAuthClientID: "c", RedirectURI: testRedirect, UserID: &other,
		ExpiresAt: time.Now().Add(time.Hour),
	}))
	req := httptest.NewRequest(http.MethodGet, "/bratrax/oauth/pending?resume=r1", nil)
	req.AddCookie(e.cookie)
	require.Equal(t, http.StatusForbidden, e.do(req).Code)
}

func TestOAuth_DecisionRequiresJSON(t *testing.T) {
	e := newOAuthTestEnv(t)
	req := httptest.NewRequest(http.MethodPost, "/bratrax/oauth/decision", strings.NewReader("resume_id=x&approve=true"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(e.cookie)
	require.Equal(t, http.StatusUnsupportedMediaType, e.do(req).Code)
}

func TestRedirectURIMatches(t *testing.T) {
	reg := []string{"https://claude.ai/api/mcp/auth_callback", "http://127.0.0.1:5000/cb"}
	require.True(t, redirectURIMatches(reg, "https://claude.ai/api/mcp/auth_callback"))
	require.False(t, redirectURIMatches(reg, "https://claude.ai/api/mcp/auth_callback/x"))
	require.True(t, redirectURIMatches(reg, "http://127.0.0.1:61234/cb"), "loopback matches on any port")
	require.False(t, redirectURIMatches(reg, "http://127.0.0.1:61234/other"))
}

func TestIsPublicIP(t *testing.T) {
	for ip, want := range map[string]bool{
		"8.8.8.8": true, "160.79.104.1": true,
		"127.0.0.1": false, "10.0.0.5": false, "192.168.1.1": false, "169.254.169.254": false,
		"100.100.1.1": false, "::1": false, "fd00::1": false,
	} {
		parsed := net.ParseIP(ip)
		require.NotNil(t, parsed, ip)
		require.Equal(t, want, isPublicIP(parsed), ip)
	}
}

// The OAuth routes share the CLI's ServeMux with the SPA catch-all and the
// proxy's own routes. Go's mux panics at registration on an ambiguous pattern,
// which in production would take Rill down at startup, so register them next
// to the same shapes the real mux carries.
func TestOAuthRoutesCoexistWithAppMux(t *testing.T) {
	e := newOAuthTestEnv(t)
	mux := http.NewServeMux()
	ok := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	mux.Handle("/", ok)         // web.StaticHandler (cli/pkg/local/server.go)
	mux.Handle("/auth", ok)     // local server auth
	mux.Handle("GET /{$}", ok)  // apex (bratrax.go)
	mux.Handle("/bratrax/", ok) // auth catch-all proxy
	mux.Handle("GET /bratrax/.well-known/jwks.json", ok)
	require.NotPanics(t, func() { e.svc.RegisterRoutes(mux) })

	// And the SPA catch-all must not shadow the discovery documents.
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil))
	require.Contains(t, rec.Body.String(), `"issuer"`)
}
