package bratrax

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// mockClientStore is a test double implementing ClientStoreInterface.
type mockClientStore struct {
	defaultClient *Client
	projectMap    map[string]*Client
}

func newMockClientStore() *mockClientStore {
	return &mockClientStore{
		defaultClient: &Client{
			ClientID:       "cod",
			OrganizationID: 1,
			CompanyName:    "Test Corp",
			ClickhouseDB:   "cod_db",
		},
		projectMap: map[string]*Client{
			"micazu": {
				ClientID:       "micazu",
				OrganizationID: 2,
				CompanyName:    "Micazu BV",
				ClickhouseDB:   "micazu_db",
			},
		},
	}
}

func (m *mockClientStore) GetByRillProjectID(_ context.Context, projectID string) (*Client, error) {
	c, ok := m.projectMap[projectID]
	if !ok {
		return nil, nil
	}
	return c, nil
}

func (m *mockClientStore) GetDefault(_ context.Context) (*Client, error) {
	return m.defaultClient, nil
}

// setupAuthMapper creates an AuthMapper with mock stores and a real JWT issuer.
func setupAuthMapper(t *testing.T) (*AuthMapper, *AuthService, *mockUserStore, *mockClientStore) {
	t.Helper()
	userStore := newMockStore()
	clientStore := newMockClientStore()
	authSvc, err := NewAuthService(userStore, zap.NewNop())
	require.NoError(t, err)

	mapper := NewAuthMapper(userStore, clientStore, authSvc.JWKS(), zap.NewNop())
	return mapper, authSvc, userStore, clientStore
}

// captureHandler records the headers it receives from the middleware.
type captureHandler struct {
	called  bool
	headers http.Header
}

func (h *captureHandler) ServeHTTP(_ http.ResponseWriter, r *http.Request) {
	h.called = true
	h.headers = r.Header.Clone()
}

func TestAuthMapper_NoCookie(t *testing.T) {
	mapper, _, _, _ := setupAuthMapper(t)
	capture := &captureHandler{}

	req := httptest.NewRequest(http.MethodGet, "/bratrax/connectors/", nil)
	rec := httptest.NewRecorder()

	mapper.Middleware(capture).ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.False(t, capture.called, "next handler should not be called")
}

func TestAuthMapper_InvalidToken(t *testing.T) {
	mapper, _, _, _ := setupAuthMapper(t)
	capture := &captureHandler{}

	req := httptest.NewRequest(http.MethodGet, "/bratrax/connectors/", nil)
	req.AddCookie(&http.Cookie{Name: bratraxCookieName, Value: "not-a-valid-jwt"})
	rec := httptest.NewRecorder()

	mapper.Middleware(capture).ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.False(t, capture.called, "next handler should not be called")
}

func TestAuthMapper_ValidAdmin(t *testing.T) {
	mapper, authSvc, _, _ := setupAuthMapper(t)
	capture := &captureHandler{}

	// Login to get a valid cookie
	cookie := loginAndGetCookie(t, authSvc)

	req := httptest.NewRequest(http.MethodGet, "/bratrax/connectors/", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()

	mapper.Middleware(capture).ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, capture.called, "next handler should be called")

	// Verify headers — admin gets default client
	require.Equal(t, "1", capture.headers.Get("user-id"))
	require.Equal(t, "1", capture.headers.Get("X-Bratrax-User-Id"))
	require.Equal(t, "cod", capture.headers.Get("X-Bratrax-Client-Id"))
	require.Equal(t, "1", capture.headers.Get("X-Bratrax-Org-Id"))
	require.Equal(t, "admin@bratrax.com", capture.headers.Get("X-Bratrax-User-Email"))
	require.Equal(t, "admin", capture.headers.Get("X-Bratrax-User-Role"))
}

func TestAuthMapper_ValidViewer(t *testing.T) {
	mapper, authSvc, userStore, _ := setupAuthMapper(t)
	capture := &captureHandler{}

	// Create a viewer user with project_id matching a client
	pid := "micazu"
	_, err := userStore.CreateUser(context.Background(), "viewer@test.com", "view1234", "Viewer", "viewer", &pid)
	require.NoError(t, err)

	// Login as viewer
	viewerCookie := loginAsUser(t, authSvc, "viewer@test.com", "view1234")

	req := httptest.NewRequest(http.MethodGet, "/bratrax/connectors/", nil)
	req.AddCookie(viewerCookie)
	rec := httptest.NewRecorder()

	mapper.Middleware(capture).ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, capture.called, "next handler should be called")

	// Viewer gets client resolved by project_id
	require.Equal(t, "2", capture.headers.Get("X-Bratrax-User-Id"))
	require.Equal(t, "micazu", capture.headers.Get("X-Bratrax-Client-Id"))
	require.Equal(t, "2", capture.headers.Get("X-Bratrax-Org-Id"))
	require.Equal(t, "viewer@test.com", capture.headers.Get("X-Bratrax-User-Email"))
	require.Equal(t, "viewer", capture.headers.Get("X-Bratrax-User-Role"))
}

func TestAuthMapper_ViewerNoProject(t *testing.T) {
	mapper, authSvc, userStore, _ := setupAuthMapper(t)
	capture := &captureHandler{}

	// Create viewer WITHOUT project_id
	_, err := userStore.CreateUser(context.Background(), "noproj@test.com", "pass1234", "NoProj", "viewer", nil)
	require.NoError(t, err)

	viewerCookie := loginAsUser(t, authSvc, "noproj@test.com", "pass1234")

	req := httptest.NewRequest(http.MethodGet, "/bratrax/connectors/", nil)
	req.AddCookie(viewerCookie)
	rec := httptest.NewRecorder()

	mapper.Middleware(capture).ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.False(t, capture.called)
}

func TestAuthMapper_ViewerNoClient(t *testing.T) {
	mapper, authSvc, userStore, _ := setupAuthMapper(t)
	capture := &captureHandler{}

	// Create viewer with project_id that doesn't map to any client
	pid := "nonexistent-project"
	_, err := userStore.CreateUser(context.Background(), "orphan@test.com", "pass1234", "Orphan", "viewer", &pid)
	require.NoError(t, err)

	viewerCookie := loginAsUser(t, authSvc, "orphan@test.com", "pass1234")

	req := httptest.NewRequest(http.MethodGet, "/bratrax/connectors/", nil)
	req.AddCookie(viewerCookie)
	rec := httptest.NewRecorder()

	mapper.Middleware(capture).ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.False(t, capture.called)
}

func TestAuthMapper_HeaderStripping(t *testing.T) {
	mapper, authSvc, _, _ := setupAuthMapper(t)
	capture := &captureHandler{}

	cookie := loginAndGetCookie(t, authSvc)

	req := httptest.NewRequest(http.MethodGet, "/bratrax/connectors/", nil)
	req.AddCookie(cookie)

	// Inject spoofed headers
	req.Header.Set("X-Bratrax-User-Id", "9999")
	req.Header.Set("X-Bratrax-Client-Id", "hacker")
	req.Header.Set("X-Bratrax-Org-Id", "9999")
	req.Header.Set("X-Bratrax-User-Email", "evil@hacker.com")
	req.Header.Set("X-Bratrax-User-Role", "admin")
	req.Header.Set("user-id", "9999")

	rec := httptest.NewRecorder()
	mapper.Middleware(capture).ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, capture.called)

	// Spoofed values must be replaced with real ones
	require.Equal(t, "1", capture.headers.Get("X-Bratrax-User-Id"))
	require.Equal(t, "cod", capture.headers.Get("X-Bratrax-Client-Id"))
	require.Equal(t, "1", capture.headers.Get("X-Bratrax-Org-Id"))
	require.Equal(t, "admin@bratrax.com", capture.headers.Get("X-Bratrax-User-Email"))
	require.Equal(t, "admin", capture.headers.Get("X-Bratrax-User-Role"))
	require.Equal(t, "1", capture.headers.Get("user-id"))
}

func TestAuthMapper_AllHeadersForwarded(t *testing.T) {
	mapper, authSvc, _, _ := setupAuthMapper(t)
	capture := &captureHandler{}

	cookie := loginAndGetCookie(t, authSvc)

	req := httptest.NewRequest(http.MethodGet, "/bratrax/connectors/", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()

	mapper.Middleware(capture).ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, capture.called)

	// All 6 headers must be present
	expectedHeaders := []string{
		"user-id",
		"X-Bratrax-User-Id",
		"X-Bratrax-Client-Id",
		"X-Bratrax-Org-Id",
		"X-Bratrax-User-Email",
		"X-Bratrax-User-Role",
	}
	for _, h := range expectedHeaders {
		require.NotEmpty(t, capture.headers.Get(h), "header %q should be set", h)
	}
}

// loginAsUser performs a login for the given credentials and returns the auth cookie.
func loginAsUser(t *testing.T, svc *AuthService, email, password string) *http.Cookie {
	t.Helper()
	payload, err := json.Marshal(map[string]string{"email": email, "password": password})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/bratrax/auth/login", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	svc.HandleLogin(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	for _, c := range rec.Result().Cookies() {
		if c.Name == bratraxCookieName {
			return c
		}
	}
	t.Fatalf("auth cookie not found in login response for %s", email)
	return nil
}
