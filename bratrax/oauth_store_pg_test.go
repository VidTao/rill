package bratrax

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestPGOAuthStore runs pgOAuthStore's SQL against a real Postgres carrying the
// Flask migrations (server/migrations.py). It is skipped unless
// BRATRAX_OAUTH_TEST_DSN points at a disposable database: it inserts and
// deletes rows, so never aim it at a shared one.
func TestPGOAuthStore(t *testing.T) {
	dsn := os.Getenv("BRATRAX_OAUTH_TEST_DSN")
	if dsn == "" {
		t.Skip("BRATRAX_OAUTH_TEST_DSN not set")
	}
	db, err := sqlx.Connect("pgx", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()

	// Fixtures: a multi-store parent with two stores, one mid-onboarding.
	for _, q := range []string{
		`DELETE FROM rill_oauth_refresh_tokens`, `DELETE FROM rill_oauth_pending_authorizations`,
		`DELETE FROM rill_oauth_clients`, `DELETE FROM rill_users WHERE email LIKE 'oauth-test%'`,
		`DELETE FROM rill_clients WHERE client_id LIKE 'oauth-test%'`,
		`INSERT INTO rill_multi_clients (multi_client_id, display_name) VALUES ('oauth-test-parent', 'Parent') ON CONFLICT DO NOTHING`,
		`INSERT INTO rill_clients (client_id, company_name, clickhouse_db, multi_client_id) VALUES
		   ('oauth-test-a', 'Store A', 'store_a', 'oauth-test-parent'),
		   ('oauth-test-b', 'Store B', 'store_b', 'oauth-test-parent')`,
		`INSERT INTO rill_onboarding_state (client_id, step) VALUES ('oauth-test-a', 'ready'), ('oauth-test-b', 'platforms_connected')`,
	} {
		_, err := db.ExecContext(ctx, q)
		require.NoError(t, err, q)
	}
	var userID int
	require.NoError(t, db.GetContext(ctx, &userID,
		`INSERT INTO rill_users (email, password_hash, role, client_id, multi_client_id)
		 VALUES ('oauth-test@example.com', 'x', 'admin', 'oauth-test-a', 'oauth-test-parent') RETURNING id`))
	parent := "oauth-test-parent"
	user := &User{ID: userID, Role: "admin", MultiClientID: &parent}

	s := newPGOAuthStore(db)

	// Clients.
	now := time.Now()
	require.NoError(t, s.UpsertClient(ctx, &oauthClient{ClientID: "brx_oc_1", ClientName: "Claude", RedirectURIs: []string{testRedirect}, Kind: "dcr"}))
	require.NoError(t, s.UpsertClient(ctx, &oauthClient{ClientID: "https://claude.ai/meta.json", ClientName: "Claude", RedirectURIs: []string{testRedirect}, Kind: "cimd", FetchedAt: &now}))
	c, err := s.GetClient(ctx, "brx_oc_1")
	require.NoError(t, err)
	require.Equal(t, []string{testRedirect}, c.RedirectURIs)
	missing, err := s.GetClient(ctx, "nope")
	require.NoError(t, err)
	require.Nil(t, missing)
	require.NoError(t, s.PruneUnusedClients(ctx, time.Hour))
	require.NoError(t, s.TouchClient(ctx, "brx_oc_1"))

	// Workspaces and onboarding steps.
	ws, err := s.WorkspacesForUser(ctx, user)
	require.NoError(t, err)
	require.Len(t, ws, 2)
	require.Equal(t, "Store A", ws[0].CompanyName)
	require.Equal(t, "ready", *ws[0].Step)
	step, err := s.OnboardingStep(ctx, "oauth-test-b")
	require.NoError(t, err)
	require.Equal(t, "platforms_connected", step)
	step, err = s.OnboardingStep(ctx, "no-such-client")
	require.NoError(t, err)
	require.Empty(t, step)
	single, err := s.WorkspacesForUser(ctx, &User{ID: userID})
	require.NoError(t, err)
	require.Len(t, single, 1)
	one, err := s.WorkspaceStatus(ctx, "oauth-test-b")
	require.NoError(t, err)
	require.Equal(t, "store_b", one.ClickhouseDB)

	// Pending authorization lifecycle.
	p := &pendingAuthorization{ResumeID: "resume-1", OAuthClientID: "brx_oc_1", RedirectURI: testRedirect,
		CodeChallenge: "c", State: "s", ExpiresAt: time.Now().Add(time.Hour)}
	require.NoError(t, s.CreatePending(ctx, p))
	ok, err := s.ClaimPending(ctx, "resume-1", userID)
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = s.ClaimPending(ctx, "resume-1", userID+1)
	require.NoError(t, err)
	require.False(t, ok, "a claimed request can't be taken by another user")
	require.NoError(t, s.SetAuthCode(ctx, "resume-1", userID, "oauth-test-a", "codehash", time.Now().Add(time.Minute)))
	got, err := s.ConsumeAuthCode(ctx, "codehash")
	require.NoError(t, err)
	require.Equal(t, "oauth-test-a", *got.SelectedClientID)
	again, err := s.ConsumeAuthCode(ctx, "codehash")
	require.NoError(t, err)
	require.Nil(t, again, "codes are single use")
	require.NoError(t, s.SetPendingGrant(ctx, "resume-1", "grant-1"))
	byCode, err := s.GetPendingByCode(ctx, "codehash")
	require.NoError(t, err)
	require.Equal(t, "grant-1", *byCode.GrantID)
	require.ErrorIs(t, s.SetAuthCode(ctx, "resume-1", userID, "oauth-test-a", "codehash2", time.Now().Add(time.Minute)), errPendingGone)
	require.NoError(t, s.PruneStalePending(ctx))

	// Refresh tokens: rotation, grace, reuse, revocation.
	g := &oauthGrant{GrantID: "grant-1", OAuthClientID: "brx_oc_1", UserID: userID, ClientID: "oauth-test-a", ClickhouseDB: "store_a", Role: "admin"}
	require.NoError(t, s.InsertRefreshToken(ctx, "rt1", g, time.Now().Add(time.Hour)))
	active, err := s.GrantActive(ctx, "grant-1")
	require.NoError(t, err)
	require.True(t, active)
	rot, outcome, err := s.RotateRefreshToken(ctx, "rt1", 30*time.Second)
	require.NoError(t, err)
	require.Equal(t, rotateOK, outcome)
	require.Equal(t, "store_a", rot.ClickhouseDB)
	_, outcome, err = s.RotateRefreshToken(ctx, "rt1", 30*time.Second)
	require.NoError(t, err)
	require.Equal(t, rotateOK, outcome, "a retry inside the grace window is accepted")
	_, err = db.ExecContext(ctx, `UPDATE rill_oauth_refresh_tokens SET rotated_at = NOW() - INTERVAL '1 hour' WHERE token_hash = 'rt1'`)
	require.NoError(t, err)
	_, outcome, err = s.RotateRefreshToken(ctx, "rt1", 30*time.Second)
	require.NoError(t, err)
	require.Equal(t, rotateReused, outcome)
	active, err = s.GrantActive(ctx, "grant-1")
	require.NoError(t, err)
	require.False(t, active, "reuse revokes the grant")
	_, outcome, err = s.RotateRefreshToken(ctx, "unknown", 30*time.Second)
	require.NoError(t, err)
	require.Equal(t, rotateInvalid, outcome)

	// Deleting the user cascades its grants away.
	require.NoError(t, s.InsertRefreshToken(ctx, "rt2", &oauthGrant{GrantID: "grant-2", OAuthClientID: "brx_oc_1", UserID: userID,
		ClientID: "oauth-test-a", ClickhouseDB: "store_a", Role: "admin"}, time.Now().Add(time.Hour)))
	_, err = db.ExecContext(ctx, `DELETE FROM rill_users WHERE id = $1`, userID)
	require.NoError(t, err)
	active, err = s.GrantActive(ctx, "grant-2")
	require.NoError(t, err)
	require.False(t, active)
}

// TestOAuthFlowPostgres runs the whole connector flow (authorize, consent,
// token, MCP call, refresh, revoke) with the real Postgres store, so the glue
// between the handlers and the SQL is covered, not just each side. Same
// disposable-database requirement as TestPGOAuthStore.
func TestOAuthFlowPostgres(t *testing.T) {
	dsn := os.Getenv("BRATRAX_OAUTH_TEST_DSN")
	if dsn == "" {
		t.Skip("BRATRAX_OAUTH_TEST_DSN not set")
	}
	db, err := sqlx.Connect("pgx", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	for _, q := range []string{
		`DELETE FROM rill_oauth_refresh_tokens`, `DELETE FROM rill_oauth_pending_authorizations`,
		`DELETE FROM rill_oauth_clients`, `DELETE FROM rill_users WHERE email = 'admin@bratrax.com'`,
		`DELETE FROM rill_clients WHERE client_id = 'cod'`,
		`INSERT INTO rill_clients (client_id, company_name, clickhouse_db) VALUES ('cod', 'Test Corp', 'cod_db')`,
		`INSERT INTO rill_onboarding_state (client_id, step) VALUES ('cod', 'ready')`,
	} {
		_, err := db.ExecContext(ctx, q)
		require.NoError(t, err, q)
	}
	var userID int
	require.NoError(t, db.GetContext(ctx, &userID,
		`INSERT INTO rill_users (email, password_hash, role, client_id) VALUES ('admin@bratrax.com', 'x', 'admin', 'cod') RETURNING id`))

	// The mock user store backs sessions; give its admin the database's id so
	// both sides agree on who the user is.
	mapper, authSvc, userStore, clientStore := setupAuthMapper(t)
	userStore.users[0].ID = userID
	clientStore.userMap = map[int]*Client{userID: clientStore.defaultClient}

	svc := NewOAuthService(newPGOAuthStore(db), authSvc, mapper, clientStore, "https://bratrax.test", nil, zap.NewNop())
	mux := http.NewServeMux()
	svc.RegisterRoutes(mux)
	hit := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hit = true
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(upstream.Close)
	RegisterMCPHandler(mux, clientStore, authSvc, svc, upstream.URL, nil, zap.NewNop())
	e := &oauthTestEnv{mux: mux, svc: svc, upstreamHit: &hit, cookie: loginAsUser(t, authSvc, "admin@bratrax.com", "admin123")}

	tokens := e.connect(t)
	require.Equal(t, http.StatusOK, e.callMCP(tokens["access_token"].(string)).Code)
	require.True(t, hit)

	refreshed := e.exchange(t, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {tokens["refresh_token"].(string)}}, http.StatusOK)
	access := refreshed["access_token"].(string)
	require.Equal(t, http.StatusOK, e.callMCP(access).Code)

	// Mid-activation, the placeholder server answers instead of the runtime.
	_, err = db.ExecContext(ctx, `UPDATE rill_onboarding_state SET step = 'extracting' WHERE client_id = 'cod'`)
	require.NoError(t, err)
	hit = false
	rec := e.callMCP(access)
	require.Equal(t, http.StatusOK, rec.Code)
	require.False(t, hit)
	require.Contains(t, rec.Body.String(), notReadyToolName)

	// Revoking in Settings (the same UPDATE the Flask endpoint runs) cuts the
	// still-unexpired access token off at once.
	_, err = db.ExecContext(ctx, `UPDATE rill_oauth_refresh_tokens SET revoked_at = NOW() WHERE client_id = 'cod'`)
	require.NoError(t, err)
	require.Equal(t, http.StatusUnauthorized, e.callMCP(access).Code)
}
