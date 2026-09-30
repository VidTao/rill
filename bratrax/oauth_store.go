package bratrax

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// Storage for the MCP OAuth authorization server (see oauth.go).
//
// Flask owns the schema (server/migrations.py: create_rill_oauth_*), Go reads
// and writes it; same split as rill_embed_handoff_tokens and rill_ai_prompt_log.
// Everything that must survive a Rill restart lives here: a merchant who starts
// connecting from Claude and then spends half an hour in onboarding must find
// their parked authorization still waiting when they finish.

// oauthClient is a registered OAuth client: either one that registered itself
// through Dynamic Client Registration (kind "dcr"), or a Client ID Metadata
// Document we fetched and cached (kind "cimd", where ClientID is the URL).
type oauthClient struct {
	ClientID     string
	ClientName   string
	RedirectURIs []string
	Kind         string
	FetchedAt    *time.Time
}

// pendingAuthorization is an authorize request parked while the user logs in,
// signs up, onboards and consents. UserID is stamped the first time a signed-in
// user opens the consent page; AuthCodeHash once they approve.
type pendingAuthorization struct {
	ResumeID          string     `db:"resume_id"`
	OAuthClientID     string     `db:"oauth_client_id"`
	RedirectURI       string     `db:"redirect_uri"`
	CodeChallenge     string     `db:"code_challenge"`
	State             string     `db:"state"`
	Scope             string     `db:"scope"`
	Resource          string     `db:"resource"`
	UserID            *int       `db:"user_id"`
	SelectedClientID  *string    `db:"selected_client_id"`
	AuthCodeExpiresAt *time.Time `db:"auth_code_expires_at"`
	ConsumedAt        *time.Time `db:"consumed_at"`
	DeniedAt          *time.Time `db:"denied_at"`
	GrantID           *string    `db:"grant_id"`
	CreatedAt         time.Time  `db:"created_at"`
	ExpiresAt         time.Time  `db:"expires_at"`
}

// oauthGrant is what a refresh token stands for: one user's consent for one
// OAuth client to reach one workspace. Every refresh token in a rotation chain
// shares the GrantID, so revoking the grant kills the whole family.
type oauthGrant struct {
	GrantID       string `db:"grant_id"`
	OAuthClientID string `db:"oauth_client_id"`
	UserID        int    `db:"user_id"`
	ClientID      string `db:"client_id"`
	ClickhouseDB  string `db:"clickhouse_db"`
	Role          string `db:"role"`
	Scope         string `db:"scope"`
}

// rotateOutcome says why a refresh token was or wasn't accepted.
type rotateOutcome int

const (
	rotateOK rotateOutcome = iota
	// rotateInvalid covers unknown, expired and revoked tokens.
	rotateInvalid
	// rotateReused means a token that was already exchanged came back after the
	// grace window: either a stolen copy or a confused client. The grant is
	// revoked as a precaution (OAuth 2.1 §4.3.1).
	rotateReused
)

// workspaceStatus is a workspace the signed-in user could bind a grant to,
// plus where it is in onboarding.
type workspaceStatus struct {
	ClientID     string  `db:"client_id"     json:"client_id"`
	CompanyName  string  `db:"company_name"  json:"company_name"`
	ClickhouseDB string  `db:"clickhouse_db" json:"-"`
	Step         *string `db:"step"          json:"step"`
}

// oauthStore is the persistence the OAuth service needs. An interface so the
// flow can be tested without Postgres.
type oauthStore interface {
	GetClient(ctx context.Context, clientID string) (*oauthClient, error)
	UpsertClient(ctx context.Context, c *oauthClient) error
	TouchClient(ctx context.Context, clientID string) error
	PruneUnusedClients(ctx context.Context, olderThan time.Duration) error

	CreatePending(ctx context.Context, p *pendingAuthorization) error
	GetPending(ctx context.Context, resumeID string) (*pendingAuthorization, error)
	// ClaimPending stamps userID on an unclaimed pending row. It reports false
	// when the row belongs to a different user.
	ClaimPending(ctx context.Context, resumeID string, userID int) (bool, error)
	SetAuthCode(ctx context.Context, resumeID string, userID int, selectedClientID, codeHash string, expiresAt time.Time) error
	DenyPending(ctx context.Context, resumeID string, userID int) error
	// ConsumeAuthCode atomically claims an unused, unexpired code. It returns
	// (nil, nil) when no such code exists.
	ConsumeAuthCode(ctx context.Context, codeHash string) (*pendingAuthorization, error)
	// GetPendingByCode finds a row by code hash regardless of state; used to
	// detect a code being replayed after it was exchanged.
	GetPendingByCode(ctx context.Context, codeHash string) (*pendingAuthorization, error)
	SetPendingGrant(ctx context.Context, resumeID, grantID string) error
	PruneStalePending(ctx context.Context) error

	InsertRefreshToken(ctx context.Context, tokenHash string, g *oauthGrant, expiresAt time.Time) error
	RotateRefreshToken(ctx context.Context, tokenHash string, grace time.Duration) (*oauthGrant, rotateOutcome, error)
	RevokeGrant(ctx context.Context, grantID string) error
	// GrantActive reports whether the grant has at least one unrevoked token
	// and its user still exists. Checked on every MCP request so a revoke in
	// Settings takes effect immediately rather than when the access token expires.
	GrantActive(ctx context.Context, grantID string) (bool, error)

	// WorkspacesForUser lists the workspaces a (non-super_admin) user may bind a
	// grant to: their own, plus siblings under a multi-store parent.
	WorkspacesForUser(ctx context.Context, user *User) ([]workspaceStatus, error)
	WorkspaceStatus(ctx context.Context, clientID string) (*workspaceStatus, error)
	OnboardingStep(ctx context.Context, clientID string) (string, error)
}

// pgOAuthStore implements oauthStore against the users database.
type pgOAuthStore struct {
	db *sqlx.DB
}

func newPGOAuthStore(db *sqlx.DB) *pgOAuthStore {
	return &pgOAuthStore{db: db}
}

func (s *pgOAuthStore) GetClient(ctx context.Context, clientID string) (*oauthClient, error) {
	var row struct {
		ClientID     string     `db:"client_id"`
		ClientName   string     `db:"client_name"`
		RedirectURIs []byte     `db:"redirect_uris"`
		Kind         string     `db:"kind"`
		FetchedAt    *time.Time `db:"metadata_fetched_at"`
	}
	err := s.db.GetContext(ctx, &row,
		`SELECT client_id, client_name, redirect_uris, kind, metadata_fetched_at
		   FROM rill_oauth_clients WHERE client_id = $1`,
		clientID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("bratrax oauth: client lookup failed: %w", err)
	}
	c := &oauthClient{ClientID: row.ClientID, ClientName: row.ClientName, Kind: row.Kind, FetchedAt: row.FetchedAt}
	if err := json.Unmarshal(row.RedirectURIs, &c.RedirectURIs); err != nil {
		return nil, fmt.Errorf("bratrax oauth: corrupt redirect_uris for %q: %w", clientID, err)
	}
	return c, nil
}

func (s *pgOAuthStore) UpsertClient(ctx context.Context, c *oauthClient) error {
	uris, err := json.Marshal(c.RedirectURIs)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO rill_oauth_clients (client_id, client_name, redirect_uris, kind, metadata_fetched_at)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (client_id) DO UPDATE
		    SET client_name = EXCLUDED.client_name,
		        redirect_uris = EXCLUDED.redirect_uris,
		        metadata_fetched_at = EXCLUDED.metadata_fetched_at`,
		c.ClientID, c.ClientName, string(uris), c.Kind, c.FetchedAt,
	)
	if err != nil {
		return fmt.Errorf("bratrax oauth: client upsert failed: %w", err)
	}
	return nil
}

func (s *pgOAuthStore) TouchClient(ctx context.Context, clientID string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE rill_oauth_clients SET last_used_at = NOW() WHERE client_id = $1`, clientID)
	return err
}

// PruneUnusedClients deletes DCR registrations that never completed a token
// exchange. Every Claude connection attempt that falls back to DCR registers a
// fresh client, so abandoned ones would otherwise accumulate forever.
func (s *pgOAuthStore) PruneUnusedClients(ctx context.Context, olderThan time.Duration) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM rill_oauth_clients
		  WHERE kind = 'dcr' AND last_used_at IS NULL
		    AND created_at < NOW() - make_interval(secs => $1)`,
		olderThan.Seconds(),
	)
	return err
}

const pendingColumns = `resume_id, oauth_client_id, redirect_uri, code_challenge,
	COALESCE(state, '') AS state, COALESCE(scope, '') AS scope, COALESCE(resource, '') AS resource,
	user_id, selected_client_id, auth_code_expires_at, consumed_at, denied_at, grant_id,
	created_at, expires_at`

func (s *pgOAuthStore) CreatePending(ctx context.Context, p *pendingAuthorization) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO rill_oauth_pending_authorizations
		   (resume_id, oauth_client_id, redirect_uri, code_challenge, state, scope, resource, user_id, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		p.ResumeID, p.OAuthClientID, p.RedirectURI, p.CodeChallenge, p.State, p.Scope, p.Resource, p.UserID, p.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("bratrax oauth: pending insert failed: %w", err)
	}
	return nil
}

func (s *pgOAuthStore) GetPending(ctx context.Context, resumeID string) (*pendingAuthorization, error) {
	var p pendingAuthorization
	err := s.db.GetContext(ctx, &p,
		`SELECT `+pendingColumns+` FROM rill_oauth_pending_authorizations WHERE resume_id = $1`, resumeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("bratrax oauth: pending lookup failed: %w", err)
	}
	return &p, nil
}

func (s *pgOAuthStore) ClaimPending(ctx context.Context, resumeID string, userID int) (bool, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE rill_oauth_pending_authorizations
		    SET user_id = $2
		  WHERE resume_id = $1 AND (user_id IS NULL OR user_id = $2)`,
		resumeID, userID,
	)
	if err != nil {
		return false, fmt.Errorf("bratrax oauth: pending claim failed: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n == 1, nil
}

func (s *pgOAuthStore) SetAuthCode(ctx context.Context, resumeID string, userID int, selectedClientID, codeHash string, expiresAt time.Time) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE rill_oauth_pending_authorizations
		    SET auth_code_hash = $4, auth_code_expires_at = $5, selected_client_id = $3
		  WHERE resume_id = $1 AND user_id = $2
		    AND consumed_at IS NULL AND denied_at IS NULL AND expires_at > NOW()`,
		resumeID, userID, selectedClientID, codeHash, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("bratrax oauth: set auth code failed: %w", err)
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return errPendingGone
	}
	return nil
}

func (s *pgOAuthStore) DenyPending(ctx context.Context, resumeID string, userID int) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE rill_oauth_pending_authorizations SET denied_at = NOW()
		  WHERE resume_id = $1 AND user_id = $2 AND consumed_at IS NULL`,
		resumeID, userID,
	)
	return err
}

func (s *pgOAuthStore) ConsumeAuthCode(ctx context.Context, codeHash string) (*pendingAuthorization, error) {
	var p pendingAuthorization
	err := s.db.GetContext(ctx, &p,
		`UPDATE rill_oauth_pending_authorizations
		    SET consumed_at = NOW()
		  WHERE auth_code_hash = $1
		    AND consumed_at IS NULL AND denied_at IS NULL
		    AND auth_code_expires_at > NOW()
		RETURNING `+pendingColumns,
		codeHash,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("bratrax oauth: consume code failed: %w", err)
	}
	return &p, nil
}

func (s *pgOAuthStore) GetPendingByCode(ctx context.Context, codeHash string) (*pendingAuthorization, error) {
	var p pendingAuthorization
	err := s.db.GetContext(ctx, &p,
		`SELECT `+pendingColumns+` FROM rill_oauth_pending_authorizations WHERE auth_code_hash = $1`, codeHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("bratrax oauth: code lookup failed: %w", err)
	}
	return &p, nil
}

func (s *pgOAuthStore) SetPendingGrant(ctx context.Context, resumeID, grantID string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE rill_oauth_pending_authorizations SET grant_id = $2 WHERE resume_id = $1`, resumeID, grantID)
	return err
}

// PruneStalePending drops parked authorizations a week past expiry. Kept that
// long, not deleted at expiry, so a replayed auth code can still be traced to
// its grant and the grant revoked.
func (s *pgOAuthStore) PruneStalePending(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM rill_oauth_pending_authorizations WHERE expires_at < NOW() - INTERVAL '7 days'`)
	return err
}

func (s *pgOAuthStore) InsertRefreshToken(ctx context.Context, tokenHash string, g *oauthGrant, expiresAt time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO rill_oauth_refresh_tokens
		   (token_hash, grant_id, oauth_client_id, user_id, client_id, clickhouse_db, role, scope, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		tokenHash, g.GrantID, g.OAuthClientID, g.UserID, g.ClientID, g.ClickhouseDB, g.Role, g.Scope, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("bratrax oauth: refresh token insert failed: %w", err)
	}
	return nil
}

// RotateRefreshToken marks a refresh token as exchanged and returns its grant.
//
// A token already exchanged within `grace` is accepted again. That covers a
// client whose refresh response was lost in transit (Claude gives the token
// endpoint 30s): without it, one dropped response would revoke the grant and
// log the merchant out. Past the grace window a second use is treated as theft.
func (s *pgOAuthStore) RotateRefreshToken(ctx context.Context, tokenHash string, grace time.Duration) (*oauthGrant, rotateOutcome, error) {
	var g oauthGrant
	err := s.db.GetContext(ctx, &g,
		`UPDATE rill_oauth_refresh_tokens
		    SET rotated_at = COALESCE(rotated_at, NOW()), last_used_at = NOW()
		  WHERE token_hash = $1
		    AND revoked_at IS NULL
		    AND expires_at > NOW()
		    AND (rotated_at IS NULL OR rotated_at > NOW() - make_interval(secs => $2))
		RETURNING grant_id, oauth_client_id, user_id, client_id, clickhouse_db, role, COALESCE(scope, '') AS scope`,
		tokenHash, grace.Seconds(),
	)
	if err == nil {
		return &g, rotateOK, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, rotateInvalid, fmt.Errorf("bratrax oauth: refresh rotate failed: %w", err)
	}

	// Rejected: find out whether it was a replay of an exchanged token.
	var row struct {
		GrantID   string     `db:"grant_id"`
		RotatedAt *time.Time `db:"rotated_at"`
		RevokedAt *time.Time `db:"revoked_at"`
	}
	err = s.db.GetContext(ctx, &row,
		`SELECT grant_id, rotated_at, revoked_at FROM rill_oauth_refresh_tokens WHERE token_hash = $1`, tokenHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, rotateInvalid, nil
		}
		return nil, rotateInvalid, fmt.Errorf("bratrax oauth: refresh lookup failed: %w", err)
	}
	if row.RotatedAt != nil && row.RevokedAt == nil {
		if err := s.RevokeGrant(ctx, row.GrantID); err != nil {
			return nil, rotateReused, err
		}
		return nil, rotateReused, nil
	}
	return nil, rotateInvalid, nil
}

func (s *pgOAuthStore) RevokeGrant(ctx context.Context, grantID string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE rill_oauth_refresh_tokens SET revoked_at = NOW()
		  WHERE grant_id = $1 AND revoked_at IS NULL`, grantID)
	if err != nil {
		return fmt.Errorf("bratrax oauth: revoke failed: %w", err)
	}
	return nil
}

func (s *pgOAuthStore) GrantActive(ctx context.Context, grantID string) (bool, error) {
	var ok bool
	err := s.db.GetContext(ctx, &ok,
		`SELECT EXISTS (
		   SELECT 1 FROM rill_oauth_refresh_tokens t
		     JOIN rill_users u ON u.id = t.user_id
		    WHERE t.grant_id = $1 AND t.revoked_at IS NULL AND t.expires_at > NOW())`,
		grantID,
	)
	if err != nil {
		return false, fmt.Errorf("bratrax oauth: grant check failed: %w", err)
	}
	return ok, nil
}

const workspaceSelect = `SELECT c.client_id, c.company_name, c.clickhouse_db, s.step
	  FROM rill_clients c
	  LEFT JOIN rill_onboarding_state s ON s.client_id = c.client_id`

func (s *pgOAuthStore) WorkspacesForUser(ctx context.Context, user *User) ([]workspaceStatus, error) {
	var out []workspaceStatus
	var err error
	if user.MultiClientID != nil && *user.MultiClientID != "" {
		err = s.db.SelectContext(ctx, &out,
			workspaceSelect+` WHERE c.multi_client_id = $1 ORDER BY c.company_name, c.created_at`,
			*user.MultiClientID)
	} else {
		err = s.db.SelectContext(ctx, &out,
			workspaceSelect+` JOIN rill_users u ON u.client_id = c.client_id WHERE u.id = $1`,
			user.ID)
	}
	if err != nil {
		return nil, fmt.Errorf("bratrax oauth: workspace list failed: %w", err)
	}
	return out, nil
}

func (s *pgOAuthStore) WorkspaceStatus(ctx context.Context, clientID string) (*workspaceStatus, error) {
	var w workspaceStatus
	err := s.db.GetContext(ctx, &w, workspaceSelect+` WHERE c.client_id = $1`, clientID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("bratrax oauth: workspace lookup failed: %w", err)
	}
	return &w, nil
}

// OnboardingStep returns the client's rill_onboarding_state.step, or "" when
// it has no onboarding row (clients that predate the Lite funnel).
func (s *pgOAuthStore) OnboardingStep(ctx context.Context, clientID string) (string, error) {
	var step sql.NullString
	err := s.db.GetContext(ctx, &step,
		`SELECT step FROM rill_onboarding_state WHERE client_id = $1`, clientID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("bratrax oauth: onboarding step lookup failed: %w", err)
	}
	return step.String, nil
}

var errPendingGone = errors.New("bratrax oauth: authorization request expired or already used")
