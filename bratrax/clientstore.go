package bratrax

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// Client represents a row in the bratrax_clients table.
type Client struct {
	ClientID       string    `db:"client_id"        json:"client_id"`
	OrganizationID int       `db:"organization_id"  json:"organization_id"`
	CompanyName    string    `db:"company_name"     json:"company_name"`
	ClickhouseDB   string    `db:"clickhouse_db"    json:"clickhouse_db"`
	RillProjectID  *string   `db:"rill_project_id"  json:"rill_project_id"`
	CreatedAt      time.Time `db:"created_at"       json:"created_at"`
}

// ClientStoreInterface abstracts client persistence for testing.
type ClientStoreInterface interface {
	GetByRillProjectID(ctx context.Context, projectID string) (*Client, error)
	GetDefault(ctx context.Context) (*Client, error)
	GetByOrganizationID(ctx context.Context, orgID int) (*Client, error)
}

// ClientStore provides read operations on bratrax_clients.
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
		`SELECT client_id, organization_id, company_name, clickhouse_db, rill_project_id, created_at
		 FROM bratrax_clients WHERE rill_project_id = $1`,
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
		`SELECT client_id, organization_id, company_name, clickhouse_db, rill_project_id, created_at
		 FROM bratrax_clients ORDER BY created_at, client_id LIMIT 1`,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("bratrax clientstore: query failed: %w", err)
	}
	return &c, nil
}

// GetByOrganizationID returns the client whose organization_id matches the given ID.
// In Bratrax's data model each user (bratrax_users.id) maps 1:1 to a client row via
// bratrax_clients.organization_id. Returns (nil, nil) if no row matches.
func (s *ClientStore) GetByOrganizationID(ctx context.Context, orgID int) (*Client, error) {
	var c Client
	err := s.db.GetContext(ctx, &c,
		`SELECT client_id, organization_id, company_name, clickhouse_db, rill_project_id, created_at
		 FROM bratrax_clients WHERE organization_id = $1`,
		orgID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("bratrax clientstore: query failed: %w", err)
	}
	return &c, nil
}
