package bratrax

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
)

// newsletterHTTPClient is a short-timeout client for the fire-and-forget
// newsletter notify. Package-level so we reuse connections rather than
// allocating a client per signup.
var newsletterHTTPClient = &http.Client{Timeout: 5 * time.Second}

// notifyNewsletterSignup asks the Flask backend to auto-subscribe a freshly
// created user to the beehiiv newsletter. The admin-create path
// (POST /bratrax/auth/users) is self-contained in Go, so this is how that
// path reaches the single Python beehiiv implementation.
//
// Fire-and-forget: call it in a goroutine. It never blocks the caller, never
// panics, and only logs failures — a marketing side-effect must not affect
// user creation. The beehiiv API key and the on/off flag live ONLY in Flask,
// so this just forwards the email; Flask no-ops when the feature is disabled.
//
// Auth mirrors the existing internal convention (see internal_handlers.go):
// the shared BRATRAX_INTERNAL_SECRET sent in the X-Bratrax-Internal-Secret
// header. No-ops when the secret or the Flask base URL is unset.
func notifyNewsletterSignup(email string, logger *zap.Logger) {
	secret := os.Getenv("BRATRAX_INTERNAL_SECRET")
	if secret == "" || email == "" {
		return
	}
	base := os.Getenv("BRATRAX_API_URL")
	if base == "" {
		base = "http://localhost:8082"
	}
	endpoint := strings.TrimRight(base, "/") + "/internal/newsletter/subscribe"

	body, err := json.Marshal(map[string]string{
		"email":  email,
		"source": "bratrax_admin_create",
	})
	if err != nil {
		logger.Warn("newsletter notify: marshal failed", zap.Error(err))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		logger.Warn("newsletter notify: request build failed", zap.Error(err))
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Bratrax-Internal-Secret", secret)

	resp, err := newsletterHTTPClient.Do(req)
	if err != nil {
		logger.Warn("newsletter notify: request failed", zap.String("email", email), zap.Error(err))
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logger.Warn("newsletter notify: non-2xx response",
			zap.String("email", email), zap.Int("status", resp.StatusCode))
	}
}
