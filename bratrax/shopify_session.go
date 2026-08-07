package bratrax

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/golang-jwt/jwt/v4"
)

// Shopify session tokens authenticate a merchant inside the Shopify admin
// iframe, where our own bratrax_auth cookie is a third-party cookie and is
// therefore blocked outright by Safari's ITP (and increasingly by Chrome).
//
// App Bridge mints one of these per request on the frontend and sends it as
// `Authorization: Bearer <jwt>`. They are HS256 signed with the app's CLIENT
// SECRET — not our RS256 keypair — and live for about a minute.
//
// This only extracts and verifies the shop. Mapping shop -> client -> user and
// injecting the X-Bratrax-* headers happens in AuthMapper.Middleware, so the
// downstream Flask API sees exactly the same headers it gets from the cookie
// path and needs no changes at all.
//
// Reference claims:
//
//	iss   https://<shop>.myshopify.com/admin
//	dest  https://<shop>.myshopify.com
//	aud   <the app's client_id / API key>
//	sub   the Shopify user id
//	exp   ~60s out
//
// Note jti and nbf are also present; we check nbf via the standard claim
// validation and deliberately do not track jti. These tokens are short-lived
// bearer credentials scoped to a single shop, and a replay-cache would need to
// be shared across processes to be worth anything.

var (
	errShopifySessionDisabled = errors.New("bratrax: shopify session tokens not configured")
	errShopifySessionClaims   = errors.New("bratrax: shopify session token claims invalid")
)

// shopifySessionClaims covers the fields we actually validate. Embedding
// RegisteredClaims gets exp/nbf/aud checking from the jwt library.
type shopifySessionClaims struct {
	Dest string `json:"dest"`
	jwt.RegisteredClaims
}

// verifyShopifySessionToken validates a Shopify App Bridge session token and
// returns the shop domain it was issued for (e.g. "mystore.myshopify.com").
//
// Returns errShopifySessionDisabled when the app credentials are unset, so a
// deployment without Shopify configured fails closed rather than accepting
// unverifiable tokens.
func verifyShopifySessionToken(tokenStr, appClientID, appClientSecret string) (string, error) {
	if appClientID == "" || appClientSecret == "" {
		return "", errShopifySessionDisabled
	}

	claims := &shopifySessionClaims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(appClientSecret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return "", fmt.Errorf("bratrax: shopify session token rejected: %w", err)
	}

	// aud must be this app. Without it, a session token minted for a different
	// Shopify app that happens to share our secret would be accepted — and more
	// practically, it catches a misconfigured client_id early.
	if !claims.VerifyAudience(appClientID, true) {
		return "", fmt.Errorf("%w: audience", errShopifySessionClaims)
	}

	// dest identifies the shop. iss is the same origin with /admin appended;
	// requiring them to agree stops a token issued for one shop carrying a dest
	// pointing at another.
	shop, err := shopHostFromURL(claims.Dest)
	if err != nil {
		return "", fmt.Errorf("%w: dest %q", errShopifySessionClaims, claims.Dest)
	}
	issuerShop, err := shopHostFromURL(claims.Issuer)
	if err != nil {
		return "", fmt.Errorf("%w: iss %q", errShopifySessionClaims, claims.Issuer)
	}
	if shop != issuerShop {
		return "", fmt.Errorf("%w: iss/dest mismatch", errShopifySessionClaims)
	}

	return shop, nil
}

// shopHostFromURL pulls the host out of a Shopify-issued URL claim and checks
// it is a myshopify domain. Everything downstream keys off this string, so it
// is normalised to lower case and validated rather than trusted.
func shopHostFromURL(raw string) (string, error) {
	if raw == "" {
		return "", errors.New("empty")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	host := strings.ToLower(u.Hostname())
	if host == "" || !strings.HasSuffix(host, ".myshopify.com") {
		return "", fmt.Errorf("not a myshopify host: %q", host)
	}
	return host, nil
}
