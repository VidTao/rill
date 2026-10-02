package bratrax

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// Multi-store active-client resolution. rill_users.client_id is the store a
// user started in and never follows switch-client, so each of these fallbacks
// used to land a multi-store merchant on their first store: an expired cookie
// or a new browser (last_client_id was read for super_admins only), and the
// Shopify admin iframe, which can't store the cookie at all.

const (
	testParent      = "parent-1"
	testOtherParent = "parent-2"
)

func strPtr(s string) *string { return &s }

// setupMultiStoreMapper wires three clients: the user's home store, a sibling
// under the same parent, and a store under another parent.
func setupMultiStoreMapper(t *testing.T) (*AuthMapper, *mockClientStore, *User) {
	t.Helper()
	mapper, _, _, clientStore := setupAuthMapper(t)
	mapper.WithShopifySessionAuth(testShopifyAppID, testShopifySecret)

	home := &Client{ClientID: "home", ClickhouseDB: "home_db", MultiClientID: strPtr(testParent)}
	sibling := &Client{ClientID: "sibling", ClickhouseDB: "sibling_db", MultiClientID: strPtr(testParent)}
	foreign := &Client{ClientID: "foreign", ClickhouseDB: "foreign_db", MultiClientID: strPtr(testOtherParent)}
	// userMap doubles as the GetByClientID lookup table; only user 7 is real.
	clientStore.userMap = map[int]*Client{7: home, 100: sibling, 101: foreign}
	clientStore.defaultClient = nil

	user := &User{ID: 7, Role: "admin", ClientID: strPtr("home"), MultiClientID: strPtr(testParent)}
	return mapper, clientStore, user
}

func resolveFor(t *testing.T, mapper *AuthMapper, user *User, r *http.Request) string {
	t.Helper()
	client, cookieToSet, err := mapper.resolveActiveClient(context.Background(), user, r)
	require.NoError(t, err)
	require.Empty(t, cookieToSet, "non-super_admin resolution never sets the cookie")
	require.NotNil(t, client)
	return client.ClientID
}

func TestResolveActiveClient_MultiStoreFallsBackToLastSwitchedStore(t *testing.T) {
	mapper, _, user := setupMultiStoreMapper(t)
	user.LastClientID = strPtr("sibling")

	r := httptest.NewRequest(http.MethodGet, "/bratrax/settings/account", nil)
	require.Equal(t, "sibling", resolveFor(t, mapper, user, r))
}

func TestResolveActiveClient_CookieBeatsLastSwitchedStore(t *testing.T) {
	mapper, _, user := setupMultiStoreMapper(t)
	user.LastClientID = strPtr("home")

	r := httptest.NewRequest(http.MethodGet, "/bratrax/settings/account", nil)
	r.AddCookie(&http.Cookie{Name: activeClientCookieName, Value: "sibling"})
	require.Equal(t, "sibling", resolveFor(t, mapper, user, r))
}

func TestResolveActiveClient_ForeignLastClientIsIgnored(t *testing.T) {
	mapper, _, user := setupMultiStoreMapper(t)
	user.LastClientID = strPtr("foreign")

	r := httptest.NewRequest(http.MethodGet, "/bratrax/settings/account", nil)
	require.Equal(t, "home", resolveFor(t, mapper, user, r))
}

func TestResolveActiveClient_ForeignCookieIsIgnored(t *testing.T) {
	mapper, _, user := setupMultiStoreMapper(t)

	r := httptest.NewRequest(http.MethodGet, "/bratrax/settings/account", nil)
	r.AddCookie(&http.Cookie{Name: activeClientCookieName, Value: "foreign"})
	require.Equal(t, "home", resolveFor(t, mapper, user, r))
}

func TestResolveActiveClient_SingleStoreUserIgnoresLastClient(t *testing.T) {
	mapper, _, user := setupMultiStoreMapper(t)
	user.MultiClientID = nil
	user.LastClientID = strPtr("sibling")

	r := httptest.NewRequest(http.MethodGet, "/bratrax/settings/account", nil)
	require.Equal(t, "home", resolveFor(t, mapper, user, r))
}

func TestResolveActiveClient_ShopifySessionHeaderPicksTheShopsStore(t *testing.T) {
	mapper, clientStore, user := setupMultiStoreMapper(t)
	clientStore.shopMap = map[string]*Client{testShop: clientStore.userMap[100]}

	// Inside the iframe there is no cookie; the header must win even over a
	// last_client_id pointing elsewhere.
	user.LastClientID = strPtr("home")
	r := httptest.NewRequest(http.MethodGet, "/v1/instances/default/resources", nil)
	r.Header.Set(shopifySessionHeader, mintSessionToken(t, nil, testShopifySecret))
	require.Equal(t, "sibling", resolveFor(t, mapper, user, r))
}

func TestResolveActiveClient_ShopifySessionHeaderForForeignShopIsIgnored(t *testing.T) {
	mapper, clientStore, user := setupMultiStoreMapper(t)
	clientStore.shopMap = map[string]*Client{testShop: clientStore.userMap[101]}

	r := httptest.NewRequest(http.MethodGet, "/v1/instances/default/resources", nil)
	r.Header.Set(shopifySessionHeader, mintSessionToken(t, nil, testShopifySecret))
	require.Equal(t, "home", resolveFor(t, mapper, user, r))
}

func TestResolveActiveClient_ForgedShopifySessionHeaderIsIgnored(t *testing.T) {
	mapper, clientStore, user := setupMultiStoreMapper(t)
	clientStore.shopMap = map[string]*Client{testShop: clientStore.userMap[100]}

	r := httptest.NewRequest(http.MethodGet, "/v1/instances/default/resources", nil)
	r.Header.Set(shopifySessionHeader, mintSessionToken(t, nil, "not-the-app-secret"))
	require.Equal(t, "home", resolveFor(t, mapper, user, r))
}

func TestResolveActiveClient_ShopifySessionHeaderForUnlinkedShopFallsThrough(t *testing.T) {
	mapper, clientStore, user := setupMultiStoreMapper(t)
	clientStore.shopMap = map[string]*Client{}
	user.LastClientID = strPtr("sibling")

	r := httptest.NewRequest(http.MethodGet, "/bratrax/onboard/me", nil)
	r.Header.Set(shopifySessionHeader, mintSessionToken(t, nil, testShopifySecret))
	require.Equal(t, "sibling", resolveFor(t, mapper, user, r))
}

// The JWT path end to end: a merchant signed in inside the iframe, so the
// bearer is a bratrax JWT and only the header names the shop.
func TestMiddleware_JWTWithShopifySessionHeaderUsesTheShopsStore(t *testing.T) {
	mapper, authSvc, userStore, clientStore := setupAuthMapper(t)
	mapper.WithShopifySessionAuth(testShopifyAppID, testShopifySecret)

	sibling := &Client{ClientID: "sibling", ClickhouseDB: "sibling_db", MultiClientID: strPtr(testParent)}
	home := &Client{ClientID: "home", ClickhouseDB: "home_db", MultiClientID: strPtr(testParent)}
	clientStore.userMap = map[int]*Client{1: home, 100: sibling}
	clientStore.shopMap = map[string]*Client{testShop: sibling}
	userStore.users[0].ClientID = strPtr("home")
	userStore.users[0].MultiClientID = strPtr(testParent)

	cookie := loginAsUser(t, authSvc, "admin@bratrax.com", "admin123")
	req := httptest.NewRequest(http.MethodGet, "/bratrax/settings/account", nil)
	req.Header.Set("Authorization", "Bearer "+cookie.Value)
	req.Header.Set(shopifySessionHeader, mintSessionToken(t, nil, testShopifySecret))
	rec := httptest.NewRecorder()
	capture := &captureHandler{}

	mapper.Middleware(capture).ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, capture.called)
	require.Equal(t, "sibling", capture.headers.Get("X-Bratrax-Client-Id"))
	require.Equal(t, "1", capture.headers.Get("X-Bratrax-User-Id"))
}

func TestUserCanOpenClient(t *testing.T) {
	home := &Client{ClientID: "home", MultiClientID: strPtr(testParent)}
	sibling := &Client{ClientID: "sibling", MultiClientID: strPtr(testParent)}
	foreign := &Client{ClientID: "foreign", MultiClientID: strPtr(testOtherParent)}
	solo := &Client{ClientID: "solo"}

	multi := &User{Role: "admin", ClientID: strPtr("home"), MultiClientID: strPtr(testParent)}
	single := &User{Role: "admin", ClientID: strPtr("solo")}
	super := &User{Role: "super_admin"}

	require.True(t, userCanOpenClient(multi, home))
	require.True(t, userCanOpenClient(multi, sibling))
	require.False(t, userCanOpenClient(multi, foreign))
	require.False(t, userCanOpenClient(multi, solo))

	require.True(t, userCanOpenClient(single, solo))
	// Two single-store clients share a NULL parent; that must not make them
	// siblings.
	require.False(t, userCanOpenClient(single, &Client{ClientID: "other-solo"}))
	require.False(t, userCanOpenClient(single, home))

	require.True(t, userCanOpenClient(super, foreign))
}
