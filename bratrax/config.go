package bratrax

import (
	"fmt"
	"net/url"
	"os"
)

// Config holds the Bratrax proxy configuration.
type Config struct {
	// TargetURL is the parsed URL of the Flask API backend.
	TargetURL *url.URL
	// UsersDSN is the PostgreSQL connection string for the bratrax_users table.
	UsersDSN string
}

// ConfigFromEnv reads Bratrax configuration from environment variables.
// BRATRAX_API_URL defaults to "http://localhost:8082" if not set.
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

	return &Config{
		TargetURL: u,
		UsersDSN:  usersDSN,
	}, nil
}
