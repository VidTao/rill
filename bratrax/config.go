package bratrax

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
)

// Config holds the Bratrax proxy configuration.
type Config struct {
	// TargetURL is the parsed URL of the Flask API backend.
	TargetURL *url.URL
	// UsersDSN is the PostgreSQL connection string for the rill_users table.
	UsersDSN string
	// IssuerURL is baked into issued JWTs and verified on incoming tokens.
	IssuerURL string
	// AudienceURL is the expected `aud` claim value.
	AudienceURL string
	// SecureCookie controls the `Secure` attribute on the auth cookie. Must be
	// true when Rill is served behind HTTPS (e.g. prod).
	SecureCookie bool
}

// ConfigFromEnv reads Bratrax configuration from environment variables.
func ConfigFromEnv() (*Config, error) {
	raw := os.Getenv("BRATRAX_API_URL")
	if raw == "" {
		raw = "http://localhost:8082"
	}

	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("bratrax: invalid BRATRAX_API_URL %q: %w", raw, err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("bratrax: BRATRAX_API_URL %q must use http or https scheme", raw)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("bratrax: BRATRAX_API_URL %q must include a host", raw)
	}

	usersDSN := os.Getenv("BRATRAX_USERS_DSN")
	if usersDSN == "" {
		usersDSN = "postgres://airflow:BratMaxPass@localhost:5434/airflow?sslmode=disable"
	}

	issuerURL := os.Getenv("BRATRAX_ISSUER_URL")
	if issuerURL == "" {
		issuerURL = "http://localhost:9009/bratrax"
	}

	audienceURL := os.Getenv("BRATRAX_AUDIENCE_URL")
	if audienceURL == "" {
		audienceURL = "http://localhost:9009"
	}

	secureCookie := false
	if rawSecure := os.Getenv("BRATRAX_SECURE_COOKIE"); rawSecure != "" {
		v, parseErr := strconv.ParseBool(rawSecure)
		if parseErr != nil {
			return nil, fmt.Errorf("bratrax: invalid BRATRAX_SECURE_COOKIE %q: %w", rawSecure, parseErr)
		}
		secureCookie = v
	}

	return &Config{
		TargetURL:    u,
		UsersDSN:     usersDSN,
		IssuerURL:    issuerURL,
		AudienceURL:  audienceURL,
		SecureCookie: secureCookie,
	}, nil
}
