package bratrax

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// AIPromptStore meters AI-sidebar prompts per user against rill_ai_prompt_log.
//
// Demo visitors get a lifetime budget of AI prompts on the platform's Anthropic
// key (see Config.DemoUserMaxPrompts). The count of rows IS the quota — the
// same append-only shape Flask's emails/cap.py and support_bot/cap.py use,
// rather than a counter column that can drift.
//
// The table is created by Flask's migrations; this side only reads and appends.
type AIPromptStore struct {
	db *sqlx.DB
}

// NewAIPromptStore creates an AIPromptStore sharing the given database connection.
func NewAIPromptStore(db *sqlx.DB) *AIPromptStore {
	return &AIPromptStore{db: db}
}

// CountForUser returns how many prompts the user has ever sent. Lifetime, not
// windowed: the demo budget is a one-time trial, not a refilling allowance.
func (s *AIPromptStore) CountForUser(ctx context.Context, userID int) (int, error) {
	var n int
	err := s.db.GetContext(ctx, &n,
		`SELECT count(*) FROM rill_ai_prompt_log WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return 0, fmt.Errorf("bratrax promptstore: count failed: %w", err)
	}
	return n, nil
}

// Record appends one prompt. clientID may be empty and model may be empty; both
// are stored for cost attribution only and neither affects the count.
func (s *AIPromptStore) Record(ctx context.Context, userID int, clientID, model string) error {
	var clientArg, modelArg any
	if clientID != "" {
		clientArg = clientID
	}
	if model != "" {
		modelArg = model
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO rill_ai_prompt_log (user_id, client_id, model) VALUES ($1, $2, $3)`,
		userID, clientArg, modelArg,
	)
	if err != nil {
		return fmt.Errorf("bratrax promptstore: insert failed: %w", err)
	}
	return nil
}

// DemoAIQuota decides whether a demo user may send another AI-sidebar prompt.
//
// Scoped to viewers on the shared demo workspace: they are the self-serve demo
// signups, and unlike a real client they can never supply their own Anthropic
// key (the key is bound to the instance, and one instance serves every demo
// visitor). Admins and super_admins on that workspace are internal accounts and
// stay uncapped so debugging isn't rationed.
type DemoAIQuota struct {
	store      *AIPromptStore
	clientSlug string
	model      string
	maxPrompts int
}

// NewDemoAIQuota returns nil when the feature is inert — no platform key
// configured, or no store. A nil *DemoAIQuota meters nothing; every method is
// nil-safe.
func NewDemoAIQuota(store *AIPromptStore, cfg *Config) *DemoAIQuota {
	if store == nil || cfg == nil || cfg.AnthropicAPIKey == "" {
		return nil
	}
	return &DemoAIQuota{
		store:      store,
		clientSlug: cfg.DemoClientSlug,
		model:      cfg.DemoUsersModel,
		maxPrompts: cfg.DemoUserMaxPrompts,
	}
}

// applies reports whether this request is a demo user spending one prompt.
// pathTail is the portion after /v1/instances/{id}, e.g. "/ai/complete/stream".
func (q *DemoAIQuota) applies(r *http.Request, user *User, client *Client, pathTail string) bool {
	if q == nil || user == nil || client == nil {
		return false
	}
	if r.Method != http.MethodPost {
		return false
	}
	// Exactly the two AI entrypoints. Everything else under /v1/instances is
	// untouched, and one request here is one user-visible prompt: metering any
	// deeper would count the agent tool loop's many LLM round-trips instead.
	if pathTail != "/ai/complete" && pathTail != "/ai/complete/stream" {
		return false
	}
	return client.ClickhouseDB == q.clientSlug && user.Role == "viewer"
}

// admit consumes one prompt from the user's lifetime budget, reporting whether
// the request may proceed. Records on admission rather than on success: a
// prompt that later errors still counts, which is the cost of not wrapping the
// ResponseWriter — a wrapper that failed to forward http.Flusher would silently
// break the SSE stream this gate sits in front of.
//
// Fails open on database errors, matching the Flask-side caps. The Anthropic
// console spend limit is the real backstop.
func (q *DemoAIQuota) admit(ctx context.Context, user *User, client *Client, logger *zap.Logger) bool {
	if q == nil {
		return true
	}
	used, err := q.store.CountForUser(ctx, user.ID)
	if err != nil {
		logger.Warn("demo AI quota: count failed; permitting prompt",
			zap.Int("user_id", user.ID), zap.Error(err))
		return true
	}
	if used >= q.maxPrompts {
		logger.Info("demo AI quota exhausted",
			zap.Int("user_id", user.ID), zap.Int("used", used), zap.Int("limit", q.maxPrompts))
		return false
	}
	if err := q.store.Record(ctx, user.ID, client.ClientID, q.model); err != nil {
		logger.Warn("demo AI quota: record failed; permitting prompt",
			zap.Int("user_id", user.ID), zap.Error(err))
	}
	return true
}
