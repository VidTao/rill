package bratrax

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rilldata/rill/runtime"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// mockUserStore is a test double implementing UserStoreInterface.
type mockUserStore struct {
	users []User
	nextID int
}

func newMockStore() *mockUserStore {
	hash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.MinCost)
	return &mockUserStore{
		users: []User{
			{
				ID:           1,
				Email:        "admin@bratrax.com",
				PasswordHash: string(hash),
				Name:         "Admin",
				Role:         "admin",
			},
		},
		nextID: 2,
	}
}

func (m *mockUserStore) Authenticate(_ context.Context, email, password string) (*User, error) {
	for i := range m.users {
		if m.users[i].Email == email {
			if err := bcrypt.CompareHashAndPassword([]byte(m.users[i].PasswordHash), []byte(password)); err != nil {
				return nil, nil
			}
			return &m.users[i], nil
		}
	}
	return nil, nil
}

func (m *mockUserStore) GetByID(_ context.Context, id int) (*User, error) {
	for i := range m.users {
		if m.users[i].ID == id {
			return &m.users[i], nil
		}
	}
	return nil, nil
}

func (m *mockUserStore) CreateUser(_ context.Context, email, password, name, role string, projectID *string) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		return nil, err
	}
	u := User{
		ID:           m.nextID,
		Email:        email,
		PasswordHash: string(hash),
		Name:         name,
		Role:         role,
		ProjectID:    projectID,
	}
	m.nextID++
	m.users = append(m.users, u)
	return &u, nil
}

func (m *mockUserStore) ListUsers(_ context.Context) ([]User, error) {
	return m.users, nil
}

func (m *mockUserStore) LinkUserToClient(_ context.Context, userID int, clientID string) error {
	for i := range m.users {
		if m.users[i].ID == userID {
			cid := clientID
			m.users[i].ClientID = &cid
			return nil
		}
	}
	return nil
}

func (m *mockUserStore) SetLastClientID(_ context.Context, userID int, clientID string) error {
	for i := range m.users {
		if m.users[i].ID == userID {
			cid := clientID
			m.users[i].LastClientID = &cid
			return nil
		}
	}
	return nil
}

const (
	testIssuerURL    = "http://localhost:9009/bratrax"
	testAudienceURL  = "http://localhost:9009"
	testSecureCookie = false
)

func setupAuthService(t *testing.T) (*AuthService, *mockUserStore) {
	t.Helper()
	store := newMockStore()
	svc, err := NewAuthService(store, zap.NewNop(), testIssuerURL, testAudienceURL, testSecureCookie, false)
	require.NoError(t, err)
	return svc, store
}

// loginAndGetCookie performs a login and returns the auth cookie.
func loginAndGetCookie(t *testing.T, svc *AuthService) *http.Cookie {
	t.Helper()
	body := `{"email":"admin@bratrax.com","password":"admin123"}`
	req := httptest.NewRequest(http.MethodPost, "/bratrax/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	svc.HandleLogin(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	for _, c := range rec.Result().Cookies() {
		if c.Name == bratraxCookieName {
			return c
		}
	}
	t.Fatal("auth cookie not found in login response")
	return nil
}

func TestHandleLogin_Success(t *testing.T) {
	svc, _ := setupAuthService(t)

	body := `{"email":"admin@bratrax.com","password":"admin123"}`
	req := httptest.NewRequest(http.MethodPost, "/bratrax/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	svc.HandleLogin(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	// Check cookie
	var foundCookie bool
	for _, c := range rec.Result().Cookies() {
		if c.Name == bratraxCookieName {
			foundCookie = true
			require.True(t, c.HttpOnly)
			require.Equal(t, "/", c.Path)
			require.Greater(t, c.MaxAge, 0)
		}
	}
	require.True(t, foundCookie, "auth cookie should be set")

	// Check response body
	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.NotEmpty(t, resp["token"])
	user, ok := resp["user"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "admin@bratrax.com", user["email"])
}

func TestHandleLogin_InvalidCredentials(t *testing.T) {
	svc, _ := setupAuthService(t)

	body := `{"email":"admin@bratrax.com","password":"wrong"}`
	req := httptest.NewRequest(http.MethodPost, "/bratrax/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	svc.HandleLogin(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandleLogin_MissingFields(t *testing.T) {
	svc, _ := setupAuthService(t)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/bratrax/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	svc.HandleLogin(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleMe_Authenticated(t *testing.T) {
	svc, _ := setupAuthService(t)
	cookie := loginAndGetCookie(t, svc)

	req := httptest.NewRequest(http.MethodGet, "/bratrax/auth/me", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()

	svc.HandleMe(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	user, ok := resp["user"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "admin@bratrax.com", user["email"])
	require.Equal(t, "admin", user["role"])
}

func TestHandleMe_NoCookie(t *testing.T) {
	svc, _ := setupAuthService(t)

	req := httptest.NewRequest(http.MethodGet, "/bratrax/auth/me", nil)
	rec := httptest.NewRecorder()

	svc.HandleMe(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandleLogout(t *testing.T) {
	svc, _ := setupAuthService(t)

	req := httptest.NewRequest(http.MethodPost, "/bratrax/auth/logout", nil)
	rec := httptest.NewRecorder()

	svc.HandleLogout(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var foundCookie bool
	for _, c := range rec.Result().Cookies() {
		if c.Name == bratraxCookieName {
			foundCookie = true
			require.Equal(t, -1, c.MaxAge, "cookie should be expired")
		}
	}
	require.True(t, foundCookie, "expired cookie should be set")
}

func TestHandleCreateUser_AdminOnly(t *testing.T) {
	svc, store := setupAuthService(t)

	// Create a viewer user
	viewerHash, _ := bcrypt.GenerateFromPassword([]byte("view123"), bcrypt.MinCost)
	pid := "micazu"
	store.users = append(store.users, User{
		ID:           10,
		Email:        "viewer@test.com",
		PasswordHash: string(viewerHash),
		Name:         "Viewer",
		Role:         "viewer",
		ProjectID:    &pid,
	})
	store.nextID = 11

	// Login as viewer
	viewerBody := `{"email":"viewer@test.com","password":"view123"}`
	viewerReq := httptest.NewRequest(http.MethodPost, "/bratrax/auth/login", bytes.NewBufferString(viewerBody))
	viewerReq.Header.Set("Content-Type", "application/json")
	viewerRec := httptest.NewRecorder()
	svc.HandleLogin(viewerRec, viewerReq)
	require.Equal(t, http.StatusOK, viewerRec.Code)

	var viewerCookie *http.Cookie
	for _, c := range viewerRec.Result().Cookies() {
		if c.Name == bratraxCookieName {
			viewerCookie = c
		}
	}
	require.NotNil(t, viewerCookie)

	// Viewer tries to create user → 403
	createBody := `{"email":"new@test.com","password":"pass1234","name":"New","role":"viewer","project_id":"test"}`
	createReq := httptest.NewRequest(http.MethodPost, "/bratrax/auth/users", bytes.NewBufferString(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.AddCookie(viewerCookie)
	createRec := httptest.NewRecorder()
	svc.HandleCreateUser(createRec, createReq)
	require.Equal(t, http.StatusForbidden, createRec.Code)

	// Admin creates user → 201
	adminCookie := loginAndGetCookie(t, svc)
	createReq2 := httptest.NewRequest(http.MethodPost, "/bratrax/auth/users", bytes.NewBufferString(createBody))
	createReq2.Header.Set("Content-Type", "application/json")
	createReq2.AddCookie(adminCookie)
	createRec2 := httptest.NewRecorder()
	svc.HandleCreateUser(createRec2, createReq2)
	require.Equal(t, http.StatusCreated, createRec2.Code)
}

func TestHandleCreateUser_ViewerNeedsProjectID(t *testing.T) {
	svc, _ := setupAuthService(t)
	adminCookie := loginAndGetCookie(t, svc)

	// Viewer without project_id → 400
	body := `{"email":"noproject@test.com","password":"pass1234","name":"No Project","role":"viewer"}`
	req := httptest.NewRequest(http.MethodPost, "/bratrax/auth/users", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()

	svc.HandleCreateUser(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)

	var resp map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.Contains(t, resp["error"], "project_id")
}

func TestPermissionsForRole(t *testing.T) {
	tests := []struct {
		role     string
		expected []runtime.Permission
	}{
		{
			role:     "admin",
			expected: runtime.AllPermissions,
		},
		{
			role: "viewer",
			expected: []runtime.Permission{
				runtime.ReadInstance,
				runtime.ReadObjects,
				runtime.ReadOLAP,
				runtime.ReadMetrics,
				runtime.ReadAPI,
			},
		},
		{
			role:     "unknown",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			got := permissionsForRole(tt.role)
			require.Equal(t, tt.expected, got)
		})
	}
}
