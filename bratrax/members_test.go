package bratrax

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// Directory member accounts (Nov 2026): signup tells a member from a customer,
// sessions slide, and a failed directory sign-in returns to the directory.

func signup(t *testing.T, svc *AuthService, email string) (int, map[string]any) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"email": email, "password": "longenough1", "company_name": "Acme"})
	req := httptest.NewRequest(http.MethodPost, "/bratrax/auth/signup", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	svc.HandleSignup(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func TestHandleSignup_ExistingAccounts(t *testing.T) {
	svc, store := setupAuthService(t)
	svc.WithDemoClientSlug("dummy")
	demo := "dummy"
	store.users = append(store.users, User{ID: 50, Email: "member@example.com", Role: "viewer", ClientID: &demo})
	store.nextID = 51

	code, out := signup(t, svc, "Member@Example.com ")
	require.Equal(t, http.StatusConflict, code)
	require.Equal(t, "member_exists", out["code"])

	code, out = signup(t, svc, "ADMIN@bratrax.com")
	require.Equal(t, http.StatusConflict, code)
	require.Equal(t, "email_taken", out["code"])
}

func TestHandleSignup_NewAccountIsLowerCased(t *testing.T) {
	svc, store := setupAuthService(t)
	code, _ := signup(t, svc, " New.Person@Example.com")
	require.Equal(t, http.StatusCreated, code)
	require.Equal(t, "new.person@example.com", store.users[len(store.users)-1].Email)
}

func TestAuthMapper_SlidingSession(t *testing.T) {
	mapper, authSvc, _, _ := setupAuthMapper(t)
	mapper.WithSessionRefresh(authSvc)
	cookie := loginAndGetCookie(t, authSvc)

	serve := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/bratrax/connectors/", nil)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		mapper.Middleware(&captureHandler{}).ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
		return rec
	}
	sessionCookie := func(rec *httptest.ResponseRecorder) *http.Cookie {
		for _, c := range rec.Result().Cookies() {
			if c.Name == bratraxCookieName {
				return c
			}
		}
		return nil
	}

	// a fresh session is left alone
	require.Nil(t, sessionCookie(serve()))

	// a day later, the same (still valid) cookie gets a new 14-day one
	defer func() { timeNow = time.Now }()
	timeNow = func() time.Time { return time.Now().Add(25 * time.Hour) }
	c := sessionCookie(serve())
	require.NotNil(t, c) // (same second, same claims: the token text itself may match)
	require.Equal(t, int(bratraxTokenTTL.Seconds()), c.MaxAge)
	require.True(t, c.HttpOnly)
	require.Equal(t, "/", c.Path)
}

func TestAuthMapper_SignInLinkIsPublic(t *testing.T) {
	mapper, _, _, _ := setupAuthMapper(t)
	capture := &captureHandler{}
	req := httptest.NewRequest(http.MethodPost, "/bratrax/signin-link", nil)
	req.Header.Set("X-Bratrax-User-Id", "1") // spoofed: must be stripped
	rec := httptest.NewRecorder()
	mapper.Middleware(capture).ServeHTTP(rec, req)
	require.True(t, capture.called)
	require.Empty(t, capture.headers.Get("X-Bratrax-User-Id"))

	// only that exact path
	capture = &captureHandler{}
	rec = httptest.NewRecorder()
	mapper.Middleware(capture).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/bratrax/signin-link/x", nil))
	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.False(t, capture.called)
}

func TestHandoff_FailedDirectorySignInReturnsToDirectory(t *testing.T) {
	svc, _ := setupAuthService(t)
	h := NewEmbedHandoffService(nil, svc, zap.NewNop(), false)
	for next, want := range map[string]string{
		"/stores/foo.com":      "/stores/signin?expired=1",
		"/stores":              "/stores/signin?expired=1",
		"/canvas/x":            "/login",
		"":                     "/login",
		"//evil.com/stores/":   "/login",
		"/storesevil.com/page": "/login",
	} {
		rec := httptest.NewRecorder()
		h.HandleHandoff(rec, httptest.NewRequest(http.MethodGet, "/auth/handoff?next="+next, nil))
		require.Equal(t, http.StatusTemporaryRedirect, rec.Code, next)
		require.Equal(t, want, rec.Header().Get("Location"), next)
	}
}
