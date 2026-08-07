package bratrax

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/require"
)

const (
	testShopifyAppID  = "test-shopify-client-id"
	testShopifySecret = "test-shopify-client-secret"
	testShop          = "demo-store.myshopify.com"
)

// mintSessionToken builds a token shaped like the ones App Bridge issues.
// Overrides let each test bend exactly one field.
func mintSessionToken(t *testing.T, mutate func(claims *shopifySessionClaims), secret string) string {
	t.Helper()
	claims := &shopifySessionClaims{
		Dest: "https://" + testShop,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "https://" + testShop + "/admin",
			Audience:  jwt.ClaimStrings{testShopifyAppID},
			Subject:   "42",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	if mutate != nil {
		mutate(claims)
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	require.NoError(t, err)
	return signed
}

func TestVerifyShopifySessionToken_Valid(t *testing.T) {
	shop, err := verifyShopifySessionToken(
		mintSessionToken(t, nil, testShopifySecret), testShopifyAppID, testShopifySecret)
	require.NoError(t, err)
	require.Equal(t, testShop, shop)
}

func TestVerifyShopifySessionToken_Rejects(t *testing.T) {
	cases := []struct {
		name   string
		token  func() string
		appID  string
		secret string
	}{
		{
			name:   "wrong secret",
			token:  func() string { return mintSessionToken(t, nil, "not-the-secret") },
			appID:  testShopifyAppID,
			secret: testShopifySecret,
		},
		{
			name: "wrong audience",
			token: func() string {
				return mintSessionToken(t, func(c *shopifySessionClaims) {
					c.Audience = jwt.ClaimStrings{"some-other-app"}
				}, testShopifySecret)
			},
			appID:  testShopifyAppID,
			secret: testShopifySecret,
		},
		{
			name: "expired",
			token: func() string {
				return mintSessionToken(t, func(c *shopifySessionClaims) {
					c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Minute))
				}, testShopifySecret)
			},
			appID:  testShopifyAppID,
			secret: testShopifySecret,
		},
		{
			name: "not yet valid",
			token: func() string {
				return mintSessionToken(t, func(c *shopifySessionClaims) {
					c.NotBefore = jwt.NewNumericDate(time.Now().Add(time.Hour))
				}, testShopifySecret)
			},
			appID:  testShopifyAppID,
			secret: testShopifySecret,
		},
		{
			// The attack this specifically blocks: a token legitimately issued
			// for one shop, rewritten to point dest at another tenant's store.
			name: "iss/dest mismatch",
			token: func() string {
				return mintSessionToken(t, func(c *shopifySessionClaims) {
					c.Dest = "https://victim-store.myshopify.com"
				}, testShopifySecret)
			},
			appID:  testShopifyAppID,
			secret: testShopifySecret,
		},
		{
			name: "non-myshopify dest",
			token: func() string {
				return mintSessionToken(t, func(c *shopifySessionClaims) {
					c.Dest = "https://evil.com"
					c.Issuer = "https://evil.com/admin"
				}, testShopifySecret)
			},
			appID:  testShopifyAppID,
			secret: testShopifySecret,
		},
		{
			// Suffix match must not accept a lookalike host.
			name: "myshopify lookalike",
			token: func() string {
				return mintSessionToken(t, func(c *shopifySessionClaims) {
					c.Dest = "https://demo-store.myshopify.com.evil.com"
					c.Issuer = "https://demo-store.myshopify.com.evil.com/admin"
				}, testShopifySecret)
			},
			appID:  testShopifyAppID,
			secret: testShopifySecret,
		},
		{
			name:   "garbage",
			token:  func() string { return "not.a.jwt" },
			appID:  testShopifyAppID,
			secret: testShopifySecret,
		},
		{
			name:   "credentials unset disables the path",
			token:  func() string { return mintSessionToken(t, nil, testShopifySecret) },
			appID:  "",
			secret: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			shop, err := verifyShopifySessionToken(tc.token(), tc.appID, tc.secret)
			require.Error(t, err)
			require.Empty(t, shop)
		})
	}
}

// alg=none must never be accepted — jwt.WithValidMethods pins HS256.
func TestVerifyShopifySessionToken_RejectsAlgNone(t *testing.T) {
	claims := &shopifySessionClaims{
		Dest: "https://" + testShop,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "https://" + testShop + "/admin",
			Audience:  jwt.ClaimStrings{testShopifyAppID},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	signed, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	shop, err := verifyShopifySessionToken(signed, testShopifyAppID, testShopifySecret)
	require.Error(t, err)
	require.Empty(t, shop)
}

// --------------------------------------------------------------------------
// Middleware integration
// --------------------------------------------------------------------------

func setupShopifyMapper(t *testing.T) (*AuthMapper, *mockUserStore, *mockClientStore, *captureHandler) {
	t.Helper()
	mapper, _, userStore, clientStore := setupAuthMapper(t)
	mapper.WithShopifySessionAuth(testShopifyAppID, testShopifySecret)
	return mapper, userStore, clientStore, &captureHandler{}
}

func TestMiddleware_ShopifySession_InjectsIdentityHeaders(t *testing.T) {
	mapper, userStore, clientStore, capture := setupShopifyMapper(t)

	clientID := "cod"
	clientStore.shopMap = map[string]*Client{testShop: {ClientID: clientID, ClickhouseDB: "cod_db"}}
	// The mock's seeded admin has no client_id; bind it so the primary-user
	// lookup resolves.
	userStore.users[0].ClientID = &clientID

	req := httptest.NewRequest(http.MethodGet, "/bratrax/onboard/me", nil)
	req.Header.Set("Authorization", "Bearer "+mintSessionToken(t, nil, testShopifySecret))
	rec := httptest.NewRecorder()

	mapper.Middleware(capture).ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, capture.called)
	require.Equal(t, "1", capture.headers.Get("X-Bratrax-User-Id"))
	require.Equal(t, clientID, capture.headers.Get("X-Bratrax-Client-Id"))
	require.Equal(t, "admin@bratrax.com", capture.headers.Get("X-Bratrax-User-Email"))
	require.Equal(t, "admin", capture.headers.Get("X-Bratrax-User-Role"))
}

// A forged identity header must be stripped even on the session-token path.
func TestMiddleware_ShopifySession_StripsSpoofedHeaders(t *testing.T) {
	mapper, userStore, clientStore, capture := setupShopifyMapper(t)

	clientID := "cod"
	clientStore.shopMap = map[string]*Client{testShop: {ClientID: clientID, ClickhouseDB: "cod_db"}}
	userStore.users[0].ClientID = &clientID

	req := httptest.NewRequest(http.MethodGet, "/bratrax/onboard/me", nil)
	req.Header.Set("Authorization", "Bearer "+mintSessionToken(t, nil, testShopifySecret))
	req.Header.Set("X-Bratrax-Client-Id", "victim-tenant")
	req.Header.Set("X-Bratrax-User-Role", "super_admin")
	rec := httptest.NewRecorder()

	mapper.Middleware(capture).ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, clientID, capture.headers.Get("X-Bratrax-Client-Id"))
	require.Equal(t, "admin", capture.headers.Get("X-Bratrax-User-Role"))
}

// An install nobody has claimed yet is the normal pre-signup state.
func TestMiddleware_ShopifySession_UnlinkedShop(t *testing.T) {
	mapper, _, clientStore, capture := setupShopifyMapper(t)
	clientStore.shopMap = map[string]*Client{}

	req := httptest.NewRequest(http.MethodGet, "/bratrax/onboard/me", nil)
	req.Header.Set("Authorization", "Bearer "+mintSessionToken(t, nil, testShopifySecret))
	rec := httptest.NewRecorder()

	mapper.Middleware(capture).ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.False(t, capture.called)
	require.Contains(t, rec.Body.String(), "not linked")
}

// With Shopify credentials unset, a session token is just an unknown bearer.
func TestMiddleware_ShopifySession_DisabledFallsThroughTo401(t *testing.T) {
	mapper, _, clientStore, capture := setupShopifyMapper(t)
	mapper.WithShopifySessionAuth("", "")
	clientStore.shopMap = map[string]*Client{testShop: {ClientID: "cod"}}

	req := httptest.NewRequest(http.MethodGet, "/bratrax/onboard/me", nil)
	req.Header.Set("Authorization", "Bearer "+mintSessionToken(t, nil, testShopifySecret))
	rec := httptest.NewRecorder()

	mapper.Middleware(capture).ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.False(t, capture.called)
}

// Regression guard: the cookie path must not be reachable via a Shopify token,
// and an ordinary bad bearer must still 401 the same way it always did.
func TestMiddleware_ShopifySession_DoesNotAffectBadBearer(t *testing.T) {
	mapper, _, _, capture := setupShopifyMapper(t)

	req := httptest.NewRequest(http.MethodGet, "/bratrax/onboard/me", nil)
	req.Header.Set("Authorization", "Bearer totally-invalid")
	rec := httptest.NewRecorder()

	mapper.Middleware(capture).ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.False(t, capture.called)
	require.Contains(t, rec.Body.String(), "invalid or expired token")
}

// ResolveClientFromCookie is what InstanceRouterMiddleware uses to pick which
// Rill instance a request reads from. If it can't resolve an embedded session,
// the request silently falls through to the empty "default" instance and the
// embedded dashboard renders with no data — authenticated, but blank.
func TestResolveClientFromCookie_ShopifySession(t *testing.T) {
	mapper, userStore, clientStore, _ := setupShopifyMapper(t)

	clientID := "cod"
	clientStore.shopMap = map[string]*Client{
		testShop: {ClientID: clientID, ClickhouseDB: "cod_db"},
	}
	userStore.users[0].ClientID = &clientID

	req := httptest.NewRequest(http.MethodGet, "/v1/instances/default/resources", nil)
	req.Header.Set("Authorization", "Bearer "+mintSessionToken(t, nil, testShopifySecret))

	user, client, err := mapper.ResolveClientFromCookie(req)
	require.NoError(t, err)
	require.NotNil(t, user, "embedded session must resolve a user")
	require.NotNil(t, client, "embedded session must resolve a client")
	require.Equal(t, clientID, client.ClientID)
	require.Equal(t, "cod_db", client.ClickhouseDB)
}

func TestResolveClientFromCookie_ShopifySessionUnlinkedShop(t *testing.T) {
	mapper, _, clientStore, _ := setupShopifyMapper(t)
	clientStore.shopMap = map[string]*Client{}

	req := httptest.NewRequest(http.MethodGet, "/v1/instances/default/resources", nil)
	req.Header.Set("Authorization", "Bearer "+mintSessionToken(t, nil, testShopifySecret))

	user, client, err := mapper.ResolveClientFromCookie(req)
	require.NoError(t, err)
	require.Nil(t, user)
	require.Nil(t, client)
}
