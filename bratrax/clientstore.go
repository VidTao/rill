package bratrax

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// Client represents a row in the rill_clients table.
type Client struct {
	ClientID      string    `db:"client_id"        json:"client_id"`
	CompanyName   string    `db:"company_name"     json:"company_name"`
	ClickhouseDB  string    `db:"clickhouse_db"    json:"clickhouse_db"`
	RillProjectID *string   `db:"rill_project_id"  json:"rill_project_id"`
	// MultiClientID is the parent rill_multi_clients row this sub-store
	// belongs to, or NULL for legacy single-store clients.
	MultiClientID *string   `db:"multi_client_id"  json:"multi_client_id,omitempty"`
	CreatedAt     time.Time `db:"created_at"       json:"created_at"`
}

// ClientWithAdmin pairs a Client with the earliest-created admin user's
// email for that client. AdminEmail is nil when no admin user exists yet
// (e.g. a client row created mid-onboarding before the user record
// finalises). Used by the super_admin client-switcher dropdown to show
// who owns each client at a glance.
type ClientWithAdmin struct {
	Client
	AdminEmail *string `db:"admin_email" json:"admin_email,omitempty"`
}

// ClientStoreInterface abstracts client persistence for testing.
type ClientStoreInterface interface {
	GetByRillProjectID(ctx context.Context, projectID string) (*Client, error)
	GetDefault(ctx context.Context) (*Client, error)
	GetByUserID(ctx context.Context, userID int) (*Client, error)
	GetAnthropicKey(ctx context.Context, clientDB string) (string, error)
	GetByMCPToken(ctx context.Context, token string) (*Client, error)
	GetByClientID(ctx context.Context, clientID string) (*Client, error)
	ListAll(ctx context.Context) ([]Client, error)
	ListAllWithAdminEmail(ctx context.Context) ([]ClientWithAdmin, error)
	ListByMultiClientID(ctx context.Context, multiClientID string) ([]Client, error)
}

// ClientStore provides read operations on rill_clients.
type ClientStore struct {
	db *sqlx.DB
}

// NewClientStore creates a ClientStore sharing the given database connection.
func NewClientStore(db *sqlx.DB) *ClientStore {
	return &ClientStore{db: db}
}

// GetByRillProjectID returns the client associated with a Rill project ID (viewer path).
func (s *ClientStore) GetByRillProjectID(ctx context.Context, projectID string) (*Client, error) {
	var c Client
	err := s.db.GetContext(ctx, &c,
		`SELECT client_id, company_name, clickhouse_db, rill_project_id, multi_client_id, created_at
		 FROM rill_clients WHERE rill_project_id = $1`,
		projectID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("bratrax clientstore: query failed: %w", err)
	}
	return &c, nil
}

// GetDefault returns the first client by creation order (admin path — single-tenant fallback).
func (s *ClientStore) GetDefault(ctx context.Context) (*Client, error) {
	var c Client
	err := s.db.GetContext(ctx, &c,
		`SELECT client_id, company_name, clickhouse_db, rill_project_id, multi_client_id, created_at
		 FROM rill_clients ORDER BY created_at, client_id LIMIT 1`,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("bratrax clientstore: query failed: %w", err)
	}
	return &c, nil
}

// GetAnthropicKey returns the per-client Anthropic API key for BYOK chat.
// Returns ("", nil) if the client has no key configured (chat will be disabled).
// Returns ("", err) only on actual database errors. The clientDB argument is
// the rill_clients.clickhouse_db value (the per-client slug like "vyne") —
// matches what InstanceRouterMiddleware passes to ensure(). NOTE: this is
// distinct from rill_clients.client_id (which is a UUID).
func (s *ClientStore) GetAnthropicKey(ctx context.Context, clientDB string) (string, error) {
	var key sql.NullString
	err := s.db.GetContext(ctx, &key,
		`SELECT anthropic_api_key FROM rill_clients WHERE clickhouse_db = $1`,
		clientDB,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("bratrax clientstore: query failed: %w", err)
	}
	if !key.Valid {
		return "", nil
	}
	return key.String, nil
}

// GetByMCPToken returns the client whose mcp_token matches the given opaque
// token (used by the /bratrax/mcp endpoint to authenticate Claude Desktop and
// resolve which per-client Rill instance to forward to). Returns (nil, nil) if
// no client has that token. The token is exact-match — partial / prefix matches
// are not allowed.
func (s *ClientStore) GetByMCPToken(ctx context.Context, token string) (*Client, error) {
	if token == "" {
		return nil, nil
	}
	var c Client
	err := s.db.GetContext(ctx, &c,
		`SELECT client_id, company_name, clickhouse_db, rill_project_id, multi_client_id, created_at
		 FROM rill_clients WHERE mcp_token = $1`,
		token,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("bratrax clientstore: query failed: %w", err)
	}
	return &c, nil
}

// GetByUserID returns the client linked to the given user via rill_users.client_id.
// This is a proper FK lookup replacing the legacy 1:1 user.id == organization_id mapping.
// One client can have many users (future multi-user tenancy); a user has at most one client.
// Returns (nil, nil) if the user has no client or the user does not exist.
//
// Note: super_admin users have client_id=NULL; this returns nil for them. Use
// GetByClientID after resolving the super_admin's active client (cookie /
// last_client_id / first-client) instead.
func (s *ClientStore) GetByUserID(ctx context.Context, userID int) (*Client, error) {
	var c Client
	err := s.db.GetContext(ctx, &c,
		`SELECT c.client_id, c.company_name, c.clickhouse_db, c.rill_project_id, c.created_at
		 FROM rill_clients c
		 INNER JOIN rill_users u ON u.client_id = c.client_id
		 WHERE u.id = $1`,
		userID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("bratrax clientstore: query failed: %w", err)
	}
	return &c, nil
}

// GetByClientID returns the client with the given client_id (UUID). Returns
// (nil, nil) if no such client exists. Used for super_admin client resolution
// (active-client cookie validation, switch-client endpoint, last_client_id
// fallback).
func (s *ClientStore) GetByClientID(ctx context.Context, clientID string) (*Client, error) {
	if clientID == "" {
		return nil, nil
	}
	var c Client
	err := s.db.GetContext(ctx, &c,
		`SELECT client_id, company_name, clickhouse_db, rill_project_id, multi_client_id, created_at
		 FROM rill_clients WHERE client_id = $1`,
		clientID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("bratrax clientstore: query failed: %w", err)
	}
	return &c, nil
}

// ListAll returns every client in the system, ordered by company_name (the
// dropdown order shown to super_admins). Used by the /bratrax/auth/clients
// endpoint that powers the super_admin client switcher.
func (s *ClientStore) ListAll(ctx context.Context) ([]Client, error) {
	var out []Client
	err := s.db.SelectContext(ctx, &out,
		`SELECT client_id, company_name, clickhouse_db, rill_project_id, multi_client_id, created_at
		 FROM rill_clients
		 ORDER BY company_name ASC, created_at ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("bratrax clientstore: list all failed: %w", err)
	}
	return out, nil
}

// ListByMultiClientID returns every sub-store under a given multi-client,
// ordered the same way ListAll is so the switcher dropdown ordering matches
// the super_admin experience. Returns an empty slice (no error) if there are
// no rows. Used by the /bratrax/auth/list-clients endpoint to scope the
// switcher for multi-store users to their parent's siblings.
func (s *ClientStore) ListByMultiClientID(ctx context.Context, multiClientID string) ([]Client, error) {
	if multiClientID == "" {
		return nil, nil
	}
	var out []Client
	err := s.db.SelectContext(ctx, &out,
		`SELECT client_id, company_name, clickhouse_db, rill_project_id, multi_client_id, created_at
		 FROM rill_clients
		 WHERE multi_client_id = $1
		 ORDER BY company_name ASC, created_at ASC`,
		multiClientID,
	)
	if err != nil {
		return nil, fmt.Errorf("bratrax clientstore: list by multi_client_id failed: %w", err)
	}
	return out, nil
}

// ListAllWithAdminEmail returns every client with the earliest-created admin
// user's email attached. NULL when no admin exists for a client (e.g. a
// half-onboarded row). The subquery filters `role = 'admin'` so super_admins
// — who have client_id IS NULL per Track K — are correctly excluded.
//
// Ordering: active clients first (alphabetical), then the rest
// (alphabetical). `rill_clients.active` is the SoT flag the reconcile +
// event-ingest path gates on; `IS TRUE` lumps NULL + FALSE together as
// "not active" so the dropdown surfaces in-use workspaces at the top.
func (s *ClientStore) ListAllWithAdminEmail(ctx context.Context) ([]ClientWithAdmin, error) {
	var out []ClientWithAdmin
	err := s.db.SelectContext(ctx, &out,
		`SELECT
		   c.client_id, c.company_name, c.clickhouse_db, c.rill_project_id, c.created_at,
		   (SELECT u.email
		      FROM rill_users u
		     WHERE u.client_id = c.client_id AND u.role = 'admin'
		     ORDER BY u.created_at ASC
		     LIMIT 1) AS admin_email
		 FROM rill_clients c
		 ORDER BY (c.active IS TRUE) DESC, c.company_name ASC, c.created_at ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("bratrax clientstore: list all with admin email failed: %w", err)
	}
	return out, nil
}
