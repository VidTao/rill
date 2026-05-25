package bratrax

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"

	_ "github.com/jackc/pgx/v4/stdlib" // pgx driver for database/sql
)

// User represents a row in the rill_users table.
type User struct {
	ID           int       `db:"id"            json:"id"`
	Email        string    `db:"email"         json:"email"`
	PasswordHash string    `db:"password_hash" json:"-"`
	Name         string    `db:"name"          json:"name"`
	Role         string    `db:"role"          json:"role"`
	ProjectID    *string   `db:"project_id"    json:"project_id"`
	ClientID     *string   `db:"client_id"     json:"client_id,omitempty"`
	// LastClientID is the most recent client a super_admin was active on. Used
	// as a fallback when the bratrax_active_client cookie is missing/invalid
	// (e.g. fresh login). NULL for new super_admins until their first switch.
	LastClientID *string   `db:"last_client_id" json:"last_client_id,omitempty"`
	// MultiClientID ties a non-super_admin user to a rill_multi_clients parent.
	// NULL for legacy single-store users (no behavior change). When set, the
	// user can switch between sibling sub-stores via the same client-switcher
	// UI super_admins use, and can add new sub-stores via the "Add store"
	// header button (POST /bratrax/multi-client/add-store).
	MultiClientID *string   `db:"multi_client_id" json:"multi_client_id,omitempty"`
	CreatedAt     time.Time `db:"created_at"    json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"    json:"updated_at"`
}

// UserStoreInterface abstracts user persistence for testing.
type UserStoreInterface interface {
	Authenticate(ctx context.Context, email, password string) (*User, error)
	GetByID(ctx context.Context, id int) (*User, error)
	CreateUser(ctx context.Context, email, password, name, role string, projectID *string) (*User, error)
	ListUsers(ctx context.Context) ([]User, error)
	LinkUserToClient(ctx context.Context, userID int, clientID string) error
	SetLastClientID(ctx context.Context, userID int, clientID string) error
}

// UserStore provides CRUD operations on rill_users.
type UserStore struct {
	db *sqlx.DB
}

// NewUserStore connects to the users database and returns a UserStore.
func NewUserStore(dsn string) (*UserStore, error) {
	db, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("bratrax userstore: failed to connect: %w", err)
	}
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(30 * time.Minute)

	return &UserStore{db: db}, nil
}

// DB returns the underlying database connection pool (used to share with ClientStore).
func (s *UserStore) DB() *sqlx.DB {
	return s.db
}

// Close closes the underlying database connection pool.
func (s *UserStore) Close() error {
	return s.db.Close()
}

// Authenticate verifies email+password and returns the user on success.
// Returns (nil, nil) if credentials are invalid (prevents user enumeration).
func (s *UserStore) Authenticate(ctx context.Context, email, password string) (*User, error) {
	var u User
	err := s.db.GetContext(ctx, &u,
		"SELECT id, email, password_hash, name, role, project_id, client_id, last_client_id, multi_client_id, created_at, updated_at FROM rill_users WHERE email = $1",
		email,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("bratrax userstore: query failed: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, nil
	}

	return &u, nil
}

// GetByID retrieves a user by primary key.
func (s *UserStore) GetByID(ctx context.Context, id int) (*User, error) {
	var u User
	err := s.db.GetContext(ctx, &u,
		"SELECT id, email, '' AS password_hash, name, role, project_id, client_id, last_client_id, multi_client_id, created_at, updated_at FROM rill_users WHERE id = $1",
		id,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("bratrax userstore: query failed: %w", err)
	}
	return &u, nil
}

// CreateUser inserts a new user with a bcrypt-hashed password.
// Passwords longer than 72 bytes are rejected (bcrypt truncation limit).
// The client_id column is left NULL at creation; the onboarding flow links
// the user to a client later via LinkUserToClient.
func (s *UserStore) CreateUser(ctx context.Context, email, password, name, role string, projectID *string) (*User, error) {
	if len(password) > 72 {
		return nil, fmt.Errorf("bratrax userstore: password exceeds 72 bytes (bcrypt limit)")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("bratrax userstore: hash failed: %w", err)
	}

	var u User
	err = s.db.QueryRowxContext(ctx,
		`INSERT INTO rill_users (email, password_hash, name, role, project_id)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, email, '' AS password_hash, name, role, project_id, client_id, last_client_id, multi_client_id, created_at, updated_at`,
		email, string(hash), name, role, projectID,
	).StructScan(&u)
	if err != nil {
		return nil, fmt.Errorf("bratrax userstore: insert failed: %w", err)
	}

	return &u, nil
}

// ListUsers returns all users, ordered by ID.
func (s *UserStore) ListUsers(ctx context.Context) ([]User, error) {
	var users []User
	err := s.db.SelectContext(ctx, &users,
		"SELECT id, email, '' AS password_hash, name, role, project_id, client_id, last_client_id, multi_client_id, created_at, updated_at FROM rill_users ORDER BY id",
	)
	if err != nil {
		return nil, fmt.Errorf("bratrax userstore: query failed: %w", err)
	}
	return users, nil
}

// LinkUserToClient sets the client_id FK on a user row. Used by onboarding
// when a new client is created and needs to be associated with its user.
func (s *UserStore) LinkUserToClient(ctx context.Context, userID int, clientID string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE rill_users SET client_id = $1, updated_at = NOW() WHERE id = $2",
		clientID, userID,
	)
	if err != nil {
		return fmt.Errorf("bratrax userstore: link user to client failed: %w", err)
	}
	return nil
}

// SetLastClientID records the last client a super_admin was active on so the
// next login lands them back on it. Also written when a super_admin switches
// clients via /bratrax/auth/switch-client.
func (s *UserStore) SetLastClientID(ctx context.Context, userID int, clientID string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE rill_users SET last_client_id = $1, updated_at = NOW() WHERE id = $2",
		clientID, userID,
	)
	if err != nil {
		return fmt.Errorf("bratrax userstore: set last client id failed: %w", err)
	}
	return nil
}
