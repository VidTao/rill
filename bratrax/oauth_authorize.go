package bratrax

import (
	"context"
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"go.uber.org/zap"
)

// The authorize leg, and the consent page's API.
//
// GET /bratrax/oauth/authorize never shows anything itself. It validates the
// request, parks it as a pending authorization, and redirects the browser into
// the SvelteKit app: to /oauth/consent when there is a session, otherwise to
// /login (which links to /signup) with the consent page as the destination.
//
// The consent page owns the rest. It asks GET /bratrax/oauth/pending what to
// show; that call also claims the parked request for the signed-in user, which
// is what lets a brand-new merchant go through signup, payment and every
// connector redirect in the onboarding funnel and still come back to it. When
// the workspace isn't far enough through onboarding the page sends them to
// their current step and remembers the request; the root layout guard brings
// them back once onboarding reaches `activating`.

var pkceChallengeRe = regexp.MustCompile(`^[A-Za-z0-9_-]{43,128}$`)

// handleAuthorize handles GET /bratrax/oauth/authorize.
func (s *OAuthService) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	ctx := r.Context()

	// Until client and redirect_uri are verified, errors are shown to the user
	// instead of being redirected: redirecting to an unverified URI is exactly
	// the open redirect OAuth forbids (RFC 6749 §4.1.2.1).
	client, err := s.resolveClient(ctx, q.Get("client_id"))
	if err != nil {
		s.logger.Info("oauth: authorize with unusable client", zap.String("client_id", q.Get("client_id")), zap.Error(err))
		renderOAuthErrorPage(w, http.StatusBadRequest, "This app isn't set up to connect to Bratrax.",
			"Try removing the connector and adding it again. If it keeps happening, contact support@bratrax.com.")
		return
	}
	redirectURI := q.Get("redirect_uri")
	if redirectURI == "" && len(client.RedirectURIs) == 1 {
		redirectURI = client.RedirectURIs[0]
	}
	if !redirectURIMatches(client.RedirectURIs, redirectURI) {
		s.logger.Info("oauth: authorize with unregistered redirect_uri",
			zap.String("client_id", client.ClientID), zap.String("redirect_uri", redirectURI))
		renderOAuthErrorPage(w, http.StatusBadRequest, "This app's sign-in link is invalid.",
			"Try removing the connector and adding it again. If it keeps happening, contact support@bratrax.com.")
		return
	}

	state := q.Get("state")
	fail := func(code, description string) {
		http.Redirect(w, r, withQuery(redirectURI, map[string]string{
			"error": code, "error_description": description, "state": state,
		}), http.StatusFound)
	}
	if q.Get("response_type") != "code" {
		fail("unsupported_response_type", "only response_type=code is supported")
		return
	}
	challenge := q.Get("code_challenge")
	if q.Get("code_challenge_method") != "S256" || !pkceChallengeRe.MatchString(challenge) {
		fail("invalid_request", "PKCE with code_challenge_method=S256 is required")
		return
	}
	if !s.resourceMatches(q.Get("resource")) {
		fail("invalid_target", "unknown resource")
		return
	}

	user, _, err := s.authMapper.ResolveClientFromCookie(r)
	if err != nil {
		// A database blip while resolving the session: treat the visitor as
		// signed out. /login bounces an actually-signed-in user straight on to
		// the consent page, so nothing is lost.
		s.logger.Warn("oauth: session lookup failed during authorize", zap.Error(err))
		user = nil
	}

	p := &pendingAuthorization{
		ResumeID:      randomToken(24),
		OAuthClientID: client.ClientID,
		RedirectURI:   redirectURI,
		CodeChallenge: challenge,
		State:         state,
		Scope:         q.Get("scope"),
		Resource:      q.Get("resource"),
		ExpiresAt:     s.now().Add(oauthPendingTTL),
	}
	if user != nil {
		p.UserID = &user.ID
	}
	if err := s.store.CreatePending(ctx, p); err != nil {
		s.logger.Error("oauth: pending insert failed", zap.Error(err))
		fail("server_error", "could not start authorization")
		return
	}
	if err := s.store.PruneStalePending(ctx); err != nil {
		s.logger.Warn("oauth: pending prune failed", zap.Error(err))
	}

	consent := oauthConsentPath + "?resume=" + url.QueryEscape(p.ResumeID)
	target := consent
	if user == nil {
		target = "/login?redirect=" + url.QueryEscape(consent)
	}
	s.logger.Info("oauth: authorization parked",
		zap.String("client_id", client.ClientID), zap.Bool("signed_in", user != nil))
	w.Header().Set("Cache-Control", "no-store")
	http.Redirect(w, r, target, http.StatusFound)
}

// pendingView is what the consent page renders from.
type pendingView struct {
	// Status is "ready" (show consent), "needs_onboarding" (send the user to
	// Step's page), or "no_workspace" (signed in, but no workspace at all).
	Status          string             `json:"status"`
	ClientName      string             `json:"client_name"`
	RedirectHost    string             `json:"redirect_host"`
	Source          string             `json:"source"`
	Email           string             `json:"email"`
	Role            string             `json:"role"`
	Step            *string            `json:"step"`
	DefaultClientID string             `json:"default_client_id"`
	Workspaces      []pendingWorkspace `json:"workspaces"`
}

type pendingWorkspace struct {
	workspaceStatus
	Ready bool `json:"ready"`
}

// handlePending handles GET /bratrax/oauth/pending?resume=<id>. Session auth.
func (s *OAuthService) handlePending(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, activeClient, err := s.authMapper.ResolveClientFromCookie(r)
	if err != nil {
		s.logger.Error("oauth: session lookup failed", zap.Error(err))
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if user == nil {
		writeJSONError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	p, ok := s.livePending(ctx, w, r.URL.Query().Get("resume"))
	if !ok {
		return
	}
	claimed, err := s.store.ClaimPending(ctx, p.ResumeID, user.ID)
	if err != nil {
		s.logger.Error("oauth: pending claim failed", zap.Error(err))
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if !claimed {
		// Someone else's sign-in link. Deliberately vague.
		writeJSONError(w, http.StatusForbidden, "this connection request belongs to another account")
		return
	}

	client, err := s.store.GetClient(ctx, p.OAuthClientID)
	if err != nil || client == nil {
		s.logger.Error("oauth: pending references missing client", zap.String("client_id", p.OAuthClientID), zap.Error(err))
		writeJSONError(w, http.StatusGone, "this connection request is no longer valid")
		return
	}
	workspaces, err := s.workspacesFor(ctx, user, activeClient)
	if err != nil {
		s.logger.Error("oauth: workspace lookup failed", zap.Error(err))
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}

	view := pendingView{
		ClientName:   client.ClientName,
		RedirectHost: redirectHost(p.RedirectURI),
		Source:       sourceForRedirect(p.RedirectURI),
		Email:        user.Email,
		Role:         user.Role,
		Workspaces:   []pendingWorkspace{},
	}
	for _, ws := range workspaces {
		view.Workspaces = append(view.Workspaces, pendingWorkspace{workspaceStatus: ws, Ready: oauthReadySteps[deref(ws.Step)]})
	}
	view.Status, view.DefaultClientID, view.Step = summarizeWorkspaces(view.Workspaces, activeClient)

	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, view)
}

// summarizeWorkspaces picks the consent page's state and default workspace:
// the active workspace if it can be connected, else the first that can, else
// the active (or first) one so its onboarding step can be resumed.
func summarizeWorkspaces(workspaces []pendingWorkspace, active *Client) (status, defaultID string, step *string) {
	if len(workspaces) == 0 {
		return "no_workspace", "", nil
	}
	pick := func(ready bool) *pendingWorkspace {
		if active != nil {
			for i := range workspaces {
				if workspaces[i].ClientID == active.ClientID && (!ready || workspaces[i].Ready) {
					return &workspaces[i]
				}
			}
		}
		for i := range workspaces {
			if !ready || workspaces[i].Ready {
				return &workspaces[i]
			}
		}
		return nil
	}
	if ws := pick(true); ws != nil {
		return "ready", ws.ClientID, ws.Step
	}
	ws := pick(false)
	return "needs_onboarding", ws.ClientID, ws.Step
}

// handlePendingInfo handles GET /bratrax/oauth/pending/info?resume=<id>.
// Public: /login and /signup call it before the visitor has an account, to say
// "Sign in to connect Claude" and to record where a new signup came from. It
// reveals nothing but the client's display name.
func (s *OAuthService) handlePendingInfo(w http.ResponseWriter, r *http.Request) {
	p, err := s.store.GetPending(r.Context(), r.URL.Query().Get("resume"))
	if err != nil {
		s.logger.Error("oauth: pending lookup failed", zap.Error(err))
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if p == nil || p.ConsumedAt != nil || p.DeniedAt != nil || !s.now().Before(p.ExpiresAt) {
		writeJSON(w, http.StatusOK, map[string]any{"valid": false})
		return
	}
	name := "an AI assistant"
	if c, err := s.store.GetClient(r.Context(), p.OAuthClientID); err == nil && c != nil {
		name = c.ClientName
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{
		"valid":       true,
		"client_name": name,
		"source":      sourceForRedirect(p.RedirectURI),
	})
}

// handleDecision handles POST /bratrax/oauth/decision, the consent page's
// Approve / Cancel. Returns where to send the browser; the page navigates there
// itself so it can clear its own state first.
func (s *OAuthService) handleDecision(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// JSON only. With the SameSite=Lax session cookie this makes a cross-site
	// forged approval impossible: a form can't send this content type, and a
	// fetch that does would need a CORS preflight we never grant.
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeJSONError(w, http.StatusUnsupportedMediaType, "expected application/json")
		return
	}
	user, activeClient, err := s.authMapper.ResolveClientFromCookie(r)
	if err != nil {
		s.logger.Error("oauth: session lookup failed", zap.Error(err))
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if user == nil {
		writeJSONError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	var body struct {
		ResumeID string `json:"resume_id"`
		Approve  bool   `json:"approve"`
		ClientID string `json:"client_id"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, oauthMaxBodyBytes)).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid body")
		return
	}
	p, ok := s.livePending(ctx, w, body.ResumeID)
	if !ok {
		return
	}
	if p.UserID == nil || *p.UserID != user.ID {
		writeJSONError(w, http.StatusForbidden, "this connection request belongs to another account")
		return
	}

	if !body.Approve {
		if err := s.store.DenyPending(ctx, p.ResumeID, user.ID); err != nil {
			s.logger.Error("oauth: deny failed", zap.Error(err))
		}
		writeJSON(w, http.StatusOK, map[string]string{"redirect_to": withQuery(p.RedirectURI, map[string]string{
			"error": "access_denied", "error_description": "the user declined to connect Bratrax", "state": p.State,
		})})
		return
	}

	workspaces, err := s.workspacesFor(ctx, user, activeClient)
	if err != nil {
		s.logger.Error("oauth: workspace lookup failed", zap.Error(err))
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}
	var chosen *workspaceStatus
	for i := range workspaces {
		if workspaces[i].ClientID == body.ClientID || (body.ClientID == "" && len(workspaces) == 1) {
			chosen = &workspaces[i]
			break
		}
	}
	if chosen == nil {
		writeJSONError(w, http.StatusForbidden, "that workspace isn't available to your account")
		return
	}
	// The server-side half of "connect only after onboarding": the consent page
	// never offers an unready workspace, but this is what actually enforces it.
	if !oauthReadySteps[deref(chosen.Step)] {
		writeJSONError(w, http.StatusConflict, "finish setting up this workspace before connecting it")
		return
	}

	code := randomToken(32)
	if err := s.store.SetAuthCode(ctx, p.ResumeID, user.ID, chosen.ClientID, hashToken(code), s.now().Add(oauthAuthCodeTTL)); err != nil {
		if err == errPendingGone {
			writeJSONError(w, http.StatusGone, err.Error())
			return
		}
		s.logger.Error("oauth: set auth code failed", zap.Error(err))
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return
	}
	s.logger.Info("oauth: authorization approved",
		zap.Int("user_id", user.ID), zap.String("client_id", chosen.ClientID),
		zap.String("oauth_client_id", p.OAuthClientID))
	writeJSON(w, http.StatusOK, map[string]string{"redirect_to": withQuery(p.RedirectURI, map[string]string{
		"code": code, "state": p.State,
	})})
}

// livePending loads a pending authorization and writes the error response
// itself when it is missing, expired, or already finished.
func (s *OAuthService) livePending(ctx context.Context, w http.ResponseWriter, resumeID string) (*pendingAuthorization, bool) {
	if resumeID == "" {
		writeJSONError(w, http.StatusBadRequest, "missing resume id")
		return nil, false
	}
	p, err := s.store.GetPending(ctx, resumeID)
	if err != nil {
		s.logger.Error("oauth: pending lookup failed", zap.Error(err))
		writeJSONError(w, http.StatusInternalServerError, "internal error")
		return nil, false
	}
	if p == nil || p.ConsumedAt != nil || p.DeniedAt != nil || !s.now().Before(p.ExpiresAt) {
		writeJSONError(w, http.StatusGone, "this connection request has expired; start again from your AI assistant")
		return nil, false
	}
	return p, true
}

// workspacesFor lists what the user may connect. A super_admin gets only the
// workspace they currently have selected: they can reach every tenant, and a
// 70-entry picker on a consent screen invites connecting the wrong one.
func (s *OAuthService) workspacesFor(ctx context.Context, user *User, active *Client) ([]workspaceStatus, error) {
	if user.Role == "super_admin" {
		if active == nil {
			return nil, nil
		}
		ws, err := s.store.WorkspaceStatus(ctx, active.ClientID)
		if err != nil || ws == nil {
			return nil, err
		}
		return []workspaceStatus{*ws}, nil
	}
	return s.store.WorkspacesForUser(ctx, user)
}

// sourceForRedirect names the entry point a connection came through, for
// rill_clients.signup_source and analytics.
func sourceForRedirect(redirectURI string) string {
	switch strings.ToLower(strings.Split(redirectHost(redirectURI), ":")[0]) {
	case "claude.ai", "claude.com":
		return "claude_connector"
	case "chatgpt.com":
		return "chatgpt_plugin"
	default:
		return "mcp_connector"
	}
}

// withQuery adds params to a URL, skipping empty values. Used only on redirect
// URIs that were already validated against the client's registration.
func withQuery(raw string, params map[string]string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	q := u.Query()
	for k, v := range params {
		if v != "" {
			q.Set(k, v)
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

var oauthErrorPage = template.Must(template.New("oauth-error").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>Bratrax</title>
<style>body{font-family:system-ui,sans-serif;background:#0b0b0b;color:#e8e8e8;display:flex;min-height:100vh;align-items:center;justify-content:center;margin:0;padding:16px}
main{max-width:420px;border:1px solid #2a2a2a;padding:32px;border-top:4px solid #d4ff00}h1{font-size:18px;margin:0 0 12px}p{color:#a8a8a8;line-height:1.5;margin:0}</style>
</head><body><main><h1>{{.Title}}</h1><p>{{.Detail}}</p></main></body></html>`))

// renderOAuthErrorPage shows an error the browser can't be redirected back
// with, because the redirect target itself is what failed validation.
func renderOAuthErrorPage(w http.ResponseWriter, status int, title, detail string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = oauthErrorPage.Execute(w, struct{ Title, Detail string }{title, detail})
}
