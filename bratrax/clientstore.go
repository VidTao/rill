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
	CreatedAt     time.Time `db:"created_at"       json:"created_at"`
}

// ClientStoreInterface abstracts client persistence for testing.
type ClientStoreInterface interface {
	GetByRillProjectID(ctx context.Context, projectID string) (*Client, error)
	GetDefault(ctx context.Context) (*Client, error)
	GetByUserID(ctx context.Context, userID int) (*Client, error)
	GetAnthropicKey(ctx context.Context, clientDB string) (string, error)
	GetByMCPToken(ctx context.Context, token string) (*Client, error)
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
		`SELECT client_id, company_name, clickhouse_db, rill_project_id, created_at
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
		`SELECT client_id, company_name, clickhouse_db, rill_project_id, created_at
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
		`SELECT client_id, company_name, clickhouse_db, rill_project_id, created_at
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
