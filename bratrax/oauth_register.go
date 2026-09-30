package bratrax

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// Client registration and resolution.
//
// Two ways a client gets a client_id:
//   - Client ID Metadata Documents (CIMD): the client_id is an HTTPS URL
//     serving the client's metadata. We fetch it, check it, and cache it.
//     Claude uses this whenever the server advertises support.
//   - Dynamic Client Registration (RFC 7591): the client POSTs its metadata and
//     we mint an id. The fallback for clients without CIMD.
//
// Either way the redirect_uris go through the same allowlist, because that is
// the one thing standing between open registration and code theft.

// handleRegister handles POST /bratrax/oauth/register.
func (s *OAuthService) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		oauthError(w, http.StatusMethodNotAllowed, "invalid_request", "POST required")
		return
	}
	var req struct {
		ClientName   string   `json:"client_name"`
		RedirectURIs []string `json:"redirect_uris"`
		GrantTypes   []string `json:"grant_types"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, oauthMaxBodyBytes)).Decode(&req); err != nil {
		oauthError(w, http.StatusBadRequest, "invalid_client_metadata", "body must be JSON client metadata")
		return
	}
	uris, err := s.validateRedirectURIs(req.RedirectURIs)
	if err != nil {
		oauthError(w, http.StatusBadRequest, "invalid_redirect_uri", err.Error())
		return
	}
	for _, g := range req.GrantTypes {
		if g != "authorization_code" && g != "refresh_token" {
			oauthError(w, http.StatusBadRequest, "invalid_client_metadata", "unsupported grant_type "+g)
			return
		}
	}

	now := s.now()
	client := &oauthClient{
		ClientID:     oauthDCRClientPrefix + randomToken(18),
		ClientName:   cleanClientName(req.ClientName),
		RedirectURIs: uris,
		Kind:         "dcr",
	}
	if err := s.store.UpsertClient(r.Context(), client); err != nil {
		s.logger.Error("oauth: dcr insert failed", zap.Error(err))
		oauthError(w, http.StatusInternalServerError, "server_error", "registration failed")
		return
	}
	// Opportunistic cleanup: every registration is a good moment to drop the
	// ones that never led anywhere.
	if err := s.store.PruneUnusedClients(r.Context(), oauthDCRPruneAfter); err != nil {
		s.logger.Warn("oauth: dcr prune failed", zap.Error(err))
	}

	s.logger.Info("oauth: client registered",
		zap.String("client_id", client.ClientID), zap.String("client_name", client.ClientName),
		zap.Strings("redirect_uris", uris))
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusCreated, map[string]any{
		"client_id":                  client.ClientID,
		"client_id_issued_at":        now.Unix(),
		"client_name":                client.ClientName,
		"redirect_uris":              uris,
		"grant_types":                []string{"authorization_code", "refresh_token"},
		"response_types":             []string{"code"},
		"token_endpoint_auth_method": "none",
	})
}

// resolveClient returns the client for an authorize or token request, fetching
// its metadata document first if client_id is a CIMD URL.
func (s *OAuthService) resolveClient(ctx context.Context, clientID string) (*oauthClient, error) {
	if clientID == "" {
		return nil, errors.New("client_id is required")
	}
	if !strings.HasPrefix(clientID, "https://") {
		c, err := s.store.GetClient(ctx, clientID)
		if err != nil {
			return nil, err
		}
		if c == nil {
			return nil, errors.New("unknown client_id")
		}
		return c, nil
	}

	cached, err := s.store.GetClient(ctx, clientID)
	if err != nil {
		return nil, err
	}
	if cached != nil && cached.FetchedAt != nil && s.now().Sub(*cached.FetchedAt) < oauthCIMDCacheTTL {
		return cached, nil
	}
	fetched, err := s.fetchClientMetadata(ctx, clientID)
	if err != nil {
		if cached != nil {
			// The document host is unreachable but we trusted it before: keep
			// going on the cached copy rather than locking merchants out.
			s.logger.Warn("oauth: cimd refresh failed; using cached metadata",
				zap.String("client_id", clientID), zap.Error(err))
			return cached, nil
		}
		return nil, err
	}
	if err := s.store.UpsertClient(ctx, fetched); err != nil {
		return nil, err
	}
	return fetched, nil
}

// fetchClientMetadata downloads and validates a Client ID Metadata Document.
func (s *OAuthService) fetchClientMetadata(ctx context.Context, clientID string) (*oauthClient, error) {
	u, err := url.Parse(clientID)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.Fragment != "" || u.User != nil {
		return nil, errors.New("client_id URL is not a valid https URL")
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, clientID, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := s.cimdClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching client metadata: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("client metadata returned HTTP %d", resp.StatusCode)
	}
	var doc struct {
		ClientID     string   `json:"client_id"`
		ClientName   string   `json:"client_name"`
		RedirectURIs []string `json:"redirect_uris"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, oauthMaxBodyBytes)).Decode(&doc); err != nil {
		return nil, errors.New("client metadata is not valid JSON")
	}
	// The document must name itself: otherwise any URL serving someone else's
	// metadata could impersonate that client.
	if doc.ClientID != clientID {
		return nil, errors.New("client metadata client_id does not match its URL")
	}
	uris, err := s.validateRedirectURIs(doc.RedirectURIs)
	if err != nil {
		return nil, err
	}
	now := s.now()
	return &oauthClient{
		ClientID:     clientID,
		ClientName:   cleanClientName(doc.ClientName),
		RedirectURIs: uris,
		Kind:         "cimd",
		FetchedAt:    &now,
	}, nil
}

// validateRedirectURIs normalises a client's redirect URIs and rejects any the
// allowlist doesn't cover.
func (s *OAuthService) validateRedirectURIs(raw []string) ([]string, error) {
	if len(raw) == 0 {
		return nil, errors.New("at least one redirect_uri is required")
	}
	seen := map[string]bool{}
	var out []string
	for _, r := range raw {
		r = strings.TrimSpace(r)
		if r == "" || seen[r] {
			continue
		}
		if !s.redirectURIAllowed(r) {
			return nil, fmt.Errorf("redirect_uri %q is not allowed", r)
		}
		seen[r] = true
		out = append(out, r)
	}
	if len(out) == 0 {
		return nil, errors.New("at least one redirect_uri is required")
	}
	return out, nil
}

// redirectURIAllowed reports whether a redirect URI may receive codes: https to
// an allowlisted host, or http to loopback for native clients (RFC 8252 §7.3).
func (s *OAuthService) redirectURIAllowed(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Fragment != "" || u.User != nil || u.Host == "" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	switch u.Scheme {
	case "http":
		return isLoopbackHost(host)
	case "https":
		if isLoopbackHost(host) {
			return true
		}
		for _, allowed := range s.redirectHosts {
			if host == allowed {
				return true
			}
		}
	}
	return false
}

// redirectURIMatches reports whether an authorize/token redirect_uri matches
// one the client registered. Exact match, except that a loopback URI matches
// on any port: native clients bind an ephemeral port per attempt (RFC 8252).
func redirectURIMatches(registered []string, candidate string) bool {
	c, err := url.Parse(candidate)
	if err != nil {
		return false
	}
	for _, r := range registered {
		if r == candidate {
			return true
		}
		ru, err := url.Parse(r)
		if err != nil {
			continue
		}
		if isLoopbackHost(ru.Hostname()) && ru.Scheme == c.Scheme &&
			ru.Hostname() == c.Hostname() && ru.Path == c.Path && ru.RawQuery == c.RawQuery {
			return true
		}
	}
	return false
}

func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// cleanClientName bounds what a client can make the consent screen say.
func cleanClientName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "MCP client"
	}
	if r := []rune(name); len(r) > 100 {
		name = string(r[:100])
	}
	return name
}

// newCIMDHTTPClient returns the HTTP client used to fetch client metadata
// documents. The URL comes from an unauthenticated caller, so this is an SSRF
// vector: it refuses to connect to anything but public addresses (checked
// after DNS resolution, so a hostname pointing inward is caught too), follows
// no redirects, and is tightly time-boxed. Tailscale's 100.64.0.0/10 is blocked
// explicitly; ClickHouse sits on it.
func newCIMDHTTPClient() *http.Client {
	dialer := &net.Dialer{
		Timeout: 5 * time.Second,
		Control: func(network, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			ip := net.ParseIP(host)
			if ip == nil || !isPublicIP(ip) {
				return fmt.Errorf("refusing to connect to non-public address %s", host)
			}
			return nil
		},
	}
	return &http.Client{
		Timeout: 8 * time.Second,
		Transport: &http.Transport{
			Proxy:                 nil,
			DialContext:           dialer.DialContext,
			TLSHandshakeTimeout:   5 * time.Second,
			ResponseHeaderTimeout: 5 * time.Second,
		},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

var cgnatBlock = &net.IPNet{IP: net.IPv4(100, 64, 0, 0), Mask: net.CIDRMask(10, 32)}

func isPublicIP(ip net.IP) bool {
	return !(ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() || ip.IsMulticast() || cgnatBlock.Contains(ip))
}
