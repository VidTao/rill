package bratrax

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// The "Open in Bratrax" bridge.
//
// A merchant working inside the Shopify admin iframe is authenticated by an App
// Bridge session token, not by the bratrax_auth cookie — that cookie is
// third-party in the iframe and blocked by Safari's ITP. So they have no cookie
// on bratrax.com, and clicking a link to the full app in a new tab would land
// on /login.
//
// Flask mints a single-use token (POST /bratrax/auth/embed-handoff, which
// reaches Flask with the embedded identity already resolved by AuthMapper).
// This handler consumes it and sets the normal cookie, so the new tab is a
// completely ordinary logged-in session from that point on.
//
// Registered at GET /auth/handoff — deliberately OUTSIDE the /bratrax/ prefix,
// so it must be declared explicitly on the mux. A path under /bratrax/ would be
// swallowed by the auth catch-all and 401 before reaching here, and a path the
// proxy doesn't know about falls through to the SPA, whose layout guard bounces
// it to /login. That is the same trap /email/pause hit.

// handoffRedirectPath is where a successful hand-off lands. The embedded
// dashboard shows a single canvas, so the natural full-app destination is the
// same canvas with the normal chrome around it.
const handoffRedirectPath = "/canvas/campaign_deep_dive"

// EmbedHandoffService consumes single-use hand-off tokens and issues a cookie.
type EmbedHandoffService struct {
	db           *sqlx.DB
	auth         *AuthService
	logger       *zap.Logger
	secureCookie bool
}

// NewEmbedHandoffService wires the hand-off handler to the same user database
// Flask writes tokens into and the auth service that issues session JWTs.
func NewEmbedHandoffService(db *sqlx.DB, auth *AuthService, logger *zap.Logger, secureCookie bool) *EmbedHandoffService {
	return &EmbedHandoffService{db: db, auth: auth, logger: logger, secureCookie: secureCookie}
}

// HandleHandoff handles GET /auth/handoff?t=<token>.
//
// Always redirects rather than returning JSON — a human is following this link
// from a new browser tab. A bad or expired token sends them to /login, which is
// the useful destination anyway: they can sign in normally from there.
func (s *EmbedHandoffService) HandleHandoff(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("t")
	if token == "" {
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return
	}

	userID, err := s.consumeToken(r, token)
	if err != nil {
		// Expired, already used, or forged. Deliberately not distinguished in
		// the response — the merchant's next move is the same either way.
		s.logger.Info("embed hand-off rejected", zap.Error(err))
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return
	}

	user, err := s.auth.store.GetByID(r.Context(), userID)
	if err != nil || user == nil {
		s.logger.Error("embed hand-off user lookup failed", zap.Int("user_id", userID), zap.Error(err))
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return
	}

	jwtToken, err := s.auth.issueToken(user)
	if err != nil {
		s.logger.Error("embed hand-off token issuance failed", zap.Error(err))
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return
	}

	// Identical cookie to HandleLogin — this is an ordinary session from here.
	http.SetCookie(w, &http.Cookie{
		Name:     bratraxCookieName,
		Value:    jwtToken,
		Path:     "/",
		MaxAge:   int(bratraxTokenTTL.Seconds()),
		HttpOnly: true,
		Secure:   s.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})

	s.logger.Info("embed hand-off completed", zap.Int("user_id", user.ID))
	http.Redirect(w, r, handoffRedirectPath, http.StatusTemporaryRedirect)
}

// consumeToken atomically claims an unused, unexpired token and returns its
// user_id. The UPDATE ... WHERE used_at IS NULL RETURNING does the check and
// the claim in one statement, so two concurrent requests with the same token
// cannot both succeed — the second matches zero rows.
func (s *EmbedHandoffService) consumeToken(r *http.Request, token string) (int, error) {
	var userID int
	err := s.db.GetContext(r.Context(), &userID,
		`UPDATE rill_embed_handoff_tokens
		    SET used_at = NOW()
		  WHERE token = $1
		    AND used_at IS NULL
		    AND expires_at > NOW()
		RETURNING user_id`,
		token,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, errors.New("bratrax: hand-off token unknown, expired, or already used")
		}
		return 0, fmt.Errorf("bratrax: hand-off token lookup failed: %w", err)
	}
	return userID, nil
}
