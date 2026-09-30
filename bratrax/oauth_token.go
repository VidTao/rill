package bratrax

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/golang-jwt/jwt/v4"
	"github.com/rilldata/rill/runtime/server/auth"
	"go.uber.org/zap"
)

// The token endpoint, and the access tokens it issues.

// mcpAccessTokenType marks our OAuth access tokens so nothing else signed by
// the same issuer (session cookies, runtime tokens) can pass as one, on top of
// the audience check.
const mcpAccessTokenType = "mcp_access"

// mcpAccessClaims is how /bratrax/mcp reads an access token.
type mcpAccessClaims struct {
	jwt.RegisteredClaims
	Attrs map[string]any `json:"attr,omitempty"`
}

// MCPGrant is a validated access token: who, which workspace, which grant.
type MCPGrant struct {
	UserID       int
	ClientID     string
	ClickhouseDB string
	Role         string
	GrantID      string
}

// handleToken handles POST /bratrax/oauth/token.
func (s *OAuthService) handleToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		oauthError(w, http.StatusMethodNotAllowed, "invalid_request", "POST required")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, oauthMaxBodyBytes)
	if err := r.ParseForm(); err != nil {
		oauthError(w, http.StatusBadRequest, "invalid_request", "body must be form-encoded")
		return
	}
	// Parameters come from the body only (RFC 6749 §3.2); a code or refresh
	// token in the query string would end up in access logs.
	form := r.PostForm
	if !s.resourceMatches(form.Get("resource")) {
		oauthError(w, http.StatusBadRequest, "invalid_target", "unknown resource")
		return
	}
	switch form.Get("grant_type") {
	case "authorization_code":
		s.exchangeCode(w, r, form.Get("code"), form.Get("code_verifier"), form.Get("redirect_uri"), form.Get("client_id"))
	case "refresh_token":
		s.exchangeRefresh(w, r, form.Get("refresh_token"), form.Get("client_id"))
	default:
		oauthError(w, http.StatusBadRequest, "unsupported_grant_type", "grant_type must be authorization_code or refresh_token")
	}
}

func (s *OAuthService) exchangeCode(w http.ResponseWriter, r *http.Request, code, verifier, redirectURI, clientID string) {
	ctx := r.Context()
	if code == "" || verifier == "" {
		oauthError(w, http.StatusBadRequest, "invalid_request", "code and code_verifier are required")
		return
	}
	codeHash := hashToken(code)
	p, err := s.store.ConsumeAuthCode(ctx, codeHash)
	if err != nil {
		s.logger.Error("oauth: code consume failed", zap.Error(err))
		oauthError(w, http.StatusInternalServerError, "server_error", "token exchange failed")
		return
	}
	if p == nil {
		// Unknown, expired, or already used. For the last case, OAuth 2.1
		// §4.1.3: a replayed code means it leaked, so kill what it produced.
		if prior, lookupErr := s.store.GetPendingByCode(ctx, codeHash); lookupErr == nil && prior != nil && prior.GrantID != nil {
			s.logger.Warn("oauth: authorization code replayed; revoking its grant", zap.String("grant_id", *prior.GrantID))
			if revokeErr := s.store.RevokeGrant(ctx, *prior.GrantID); revokeErr != nil {
				s.logger.Error("oauth: revoke after code replay failed", zap.Error(revokeErr))
			}
		}
		oauthError(w, http.StatusBadRequest, "invalid_grant", "authorization code is invalid or expired")
		return
	}
	if clientID != "" && clientID != p.OAuthClientID {
		oauthError(w, http.StatusBadRequest, "invalid_grant", "code was issued to another client")
		return
	}
	if redirectURI != "" && redirectURI != p.RedirectURI {
		oauthError(w, http.StatusBadRequest, "invalid_grant", "redirect_uri does not match the authorization request")
		return
	}
	if !pkceMatches(verifier, p.CodeChallenge) {
		oauthError(w, http.StatusBadRequest, "invalid_grant", "code_verifier does not match")
		return
	}
	if p.UserID == nil || p.SelectedClientID == nil {
		// SetAuthCode sets both, so this is corruption rather than user error.
		s.logger.Error("oauth: consumed code without user or workspace", zap.String("resume_id", p.ResumeID))
		oauthError(w, http.StatusBadRequest, "invalid_grant", "authorization is incomplete")
		return
	}

	grant, err := s.grantFor(ctx, *p.UserID, *p.SelectedClientID, p.OAuthClientID, p.Scope)
	if err != nil {
		s.logger.Warn("oauth: code exchange refused", zap.Int("user_id", *p.UserID), zap.Error(err))
		oauthError(w, http.StatusBadRequest, "invalid_grant", err.Error())
		return
	}
	grant.GrantID = randomToken(16)
	if err := s.store.SetPendingGrant(ctx, p.ResumeID, grant.GrantID); err != nil {
		s.logger.Warn("oauth: recording grant on pending row failed", zap.Error(err))
	}
	if err := s.store.TouchClient(ctx, p.OAuthClientID); err != nil {
		s.logger.Warn("oauth: client touch failed", zap.Error(err))
	}
	s.logger.Info("oauth: grant issued",
		zap.Int("user_id", grant.UserID), zap.String("client_id", grant.ClientID),
		zap.String("oauth_client_id", grant.OAuthClientID), zap.String("grant_id", grant.GrantID))
	s.writeTokens(r.Context(), w, grant)
}

func (s *OAuthService) exchangeRefresh(w http.ResponseWriter, r *http.Request, refreshToken, clientID string) {
	ctx := r.Context()
	if refreshToken == "" {
		oauthError(w, http.StatusBadRequest, "invalid_request", "refresh_token is required")
		return
	}
	prior, outcome, err := s.store.RotateRefreshToken(ctx, hashToken(refreshToken), oauthRefreshGrace)
	if err != nil {
		s.logger.Error("oauth: refresh rotate failed", zap.Error(err))
		oauthError(w, http.StatusInternalServerError, "server_error", "token refresh failed")
		return
	}
	switch outcome {
	case rotateReused:
		s.logger.Warn("oauth: refresh token reused after rotation; grant revoked")
		oauthError(w, http.StatusBadRequest, "invalid_grant", "refresh token was already used")
		return
	case rotateInvalid:
		oauthError(w, http.StatusBadRequest, "invalid_grant", "refresh token is invalid, expired or revoked")
		return
	}
	if clientID != "" && clientID != prior.OAuthClientID {
		oauthError(w, http.StatusBadRequest, "invalid_grant", "refresh token was issued to another client")
		return
	}

	// Re-derive everything from the current state of the account, so a role
	// change or a removed teammate takes effect at the next refresh.
	grant, err := s.grantFor(ctx, prior.UserID, prior.ClientID, prior.OAuthClientID, prior.Scope)
	if err != nil {
		if revokeErr := s.store.RevokeGrant(ctx, prior.GrantID); revokeErr != nil {
			s.logger.Error("oauth: revoke of stale grant failed", zap.Error(revokeErr))
		}
		s.logger.Info("oauth: refresh refused; grant revoked", zap.String("grant_id", prior.GrantID), zap.Error(err))
		oauthError(w, http.StatusBadRequest, "invalid_grant", err.Error())
		return
	}
	grant.GrantID = prior.GrantID
	s.writeTokens(ctx, w, grant)
}

// grantFor checks that the user can still reach the workspace and builds the
// grant with their current role.
func (s *OAuthService) grantFor(ctx context.Context, userID int, clientID, oauthClientID, scope string) (*oauthGrant, error) {
	user, err := s.auth.store.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user lookup failed: %w", err)
	}
	if user == nil {
		return nil, errors.New("the account behind this connection no longer exists")
	}
	var ws *workspaceStatus
	if user.Role == "super_admin" {
		ws, err = s.store.WorkspaceStatus(ctx, clientID)
		if err != nil {
			return nil, err
		}
	} else {
		all, listErr := s.store.WorkspacesForUser(ctx, user)
		if listErr != nil {
			return nil, listErr
		}
		for i := range all {
			if all[i].ClientID == clientID {
				ws = &all[i]
				break
			}
		}
	}
	if ws == nil {
		return nil, errors.New("this account no longer has access to the connected workspace")
	}
	return &oauthGrant{
		OAuthClientID: oauthClientID,
		UserID:        user.ID,
		ClientID:      ws.ClientID,
		ClickhouseDB:  ws.ClickhouseDB,
		Role:          user.Role,
		Scope:         scope,
	}, nil
}

// writeTokens issues an access token and a fresh refresh token for the grant.
func (s *OAuthService) writeTokens(ctx context.Context, w http.ResponseWriter, g *oauthGrant) {
	access, err := s.auth.issuer.NewToken(auth.TokenOptions{
		AudienceURL: s.ResourceURL(),
		Subject:     strconv.Itoa(g.UserID),
		TTL:         oauthAccessTokenTTL,
		Attributes: map[string]any{
			"typ":           mcpAccessTokenType,
			"grant_id":      g.GrantID,
			"client_id":     g.ClientID,
			"clickhouse_db": g.ClickhouseDB,
			"role":          g.Role,
		},
	})
	if err != nil {
		s.logger.Error("oauth: access token mint failed", zap.Error(err))
		oauthError(w, http.StatusInternalServerError, "server_error", "token issuance failed")
		return
	}
	refresh := oauthRefreshTokenPref + randomToken(32)
	if err := s.store.InsertRefreshToken(ctx, hashToken(refresh), g, s.now().Add(oauthRefreshTokenTTL)); err != nil {
		s.logger.Error("oauth: refresh token insert failed", zap.Error(err))
		oauthError(w, http.StatusInternalServerError, "server_error", "token issuance failed")
		return
	}
	body := map[string]any{
		"access_token":  access,
		"token_type":    "Bearer",
		"expires_in":    int(oauthAccessTokenTTL.Seconds()),
		"refresh_token": refresh,
	}
	if g.Scope != "" {
		body["scope"] = g.Scope
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	writeJSON(w, http.StatusOK, body)
}

// ErrInvalidAccessToken wraps every reason an access token is refused, so the
// MCP handler can tell a 401 from a database failure.
var ErrInvalidAccessToken = errors.New("invalid access token")

// ValidateAccessToken checks an access token presented to /bratrax/mcp. The
// signature, issuer, audience and expiry are checked locally; the grant is then
// looked up so that a revoke, or a deleted user, cuts access immediately.
func (s *OAuthService) ValidateAccessToken(ctx context.Context, token string) (*MCPGrant, error) {
	invalid := func(reason string) error { return fmt.Errorf("%w: %s", ErrInvalidAccessToken, reason) }
	claims := &mcpAccessClaims{}
	if _, err := jwt.ParseWithClaims(token, claims, s.auth.jwks.Keyfunc, jwt.WithValidMethods([]string{"RS256"})); err != nil {
		return nil, invalid(err.Error())
	}
	if !claims.VerifyIssuer(s.auth.issuerURL, true) {
		return nil, invalid("wrong issuer")
	}
	if !claims.VerifyAudience(s.ResourceURL(), true) {
		return nil, invalid("audience is not this MCP server")
	}
	if typ, _ := claims.Attrs["typ"].(string); typ != mcpAccessTokenType {
		return nil, invalid("not an MCP access token")
	}
	userID, err := strconv.Atoi(claims.Subject)
	if err != nil {
		return nil, invalid("bad subject")
	}
	g := &MCPGrant{UserID: userID}
	g.GrantID, _ = claims.Attrs["grant_id"].(string)
	g.ClientID, _ = claims.Attrs["client_id"].(string)
	g.ClickhouseDB, _ = claims.Attrs["clickhouse_db"].(string)
	g.Role, _ = claims.Attrs["role"].(string)
	if g.GrantID == "" || g.ClientID == "" || g.ClickhouseDB == "" || g.Role == "" {
		return nil, invalid("missing claims")
	}
	active, err := s.store.GrantActive(ctx, g.GrantID)
	if err != nil {
		return nil, err
	}
	if !active {
		return nil, invalid("connection revoked")
	}
	return g, nil
}

// pkceMatches verifies an S256 code_verifier against the stored challenge.
func pkceMatches(verifier, challenge string) bool {
	if len(verifier) < 43 || len(verifier) > 128 {
		return false
	}
	sum := sha256.Sum256([]byte(verifier))
	computed := base64.RawURLEncoding.EncodeToString(sum[:])
	return subtle.ConstantTimeCompare([]byte(computed), []byte(challenge)) == 1
}
