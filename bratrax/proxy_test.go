package bratrax

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"go.uber.org/zap"
)

func TestStripBratraxPrefix(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"full path", "/bratrax/connectors/foo", "/connectors/foo"},
		{"root no slash", "/bratrax", "/"},
		{"root with slash", "/bratrax/", "/"},
		{"nested path", "/bratrax/api/v1/health", "/api/v1/health"},
		{"not our prefix", "/other/path", "/other/path"},
		{"partial match", "/bratraxfoo/bar", "/bratraxfoo/bar"},
		{"empty", "", ""},
		{"just slash", "/", "/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripBratraxPrefix(tt.path)
			if got != tt.want {
				t.Errorf("stripBratraxPrefix(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestConfigFromEnv(t *testing.T) {
	tests := []struct {
		name    string
		envVal  string
		wantURL string
		wantErr bool
	}{
		{"default", "", "http://localhost:8081", false},
		{"custom", "https://api.bratrax.com", "https://api.bratrax.com", false},
		{"with port", "http://flask:5000", "http://flask:5000", false},
		{"no scheme", "localhost:8081", "", true},
		{"ftp scheme rejected", "ftp://internal:21", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use t.Setenv for both set and unset to avoid test pollution.
			t.Setenv("BRATRAX_API_URL", tt.envVal)

			cfg, err := ConfigFromEnv()
			if tt.wantErr {
				if err == nil {
					t.Errorf("ConfigFromEnv() expected error for %q, got nil", tt.envVal)
				}
				return
			}
			if err != nil {
				t.Fatalf("ConfigFromEnv() unexpected error: %v", err)
			}
			if cfg.TargetURL.String() != tt.wantURL {
				t.Errorf("ConfigFromEnv().TargetURL = %q, want %q", cfg.TargetURL.String(), tt.wantURL)
			}
		})
	}
}

func TestNewProxy_PathRewrite(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Received-Path", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	target, _ := url.Parse(backend.URL)
	proxy := NewProxy(target, zap.NewNop())

	tests := []struct {
		name     string
		reqPath  string
		wantPath string
	}{
		{"strips prefix", "/bratrax/connectors/foo", "/connectors/foo"},
		{"strips prefix root", "/bratrax/", "/"},
		{"no prefix passthrough", "/other/path", "/other/path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.reqPath, nil)
			rec := httptest.NewRecorder()
			proxy.ServeHTTP(rec, req)

			got := rec.Header().Get("X-Received-Path")
			if got != tt.wantPath {
				t.Errorf("backend received path %q, want %q", got, tt.wantPath)
			}
		})
	}
}

func TestNewProxy_BackendDown(t *testing.T) {
	target, _ := url.Parse("http://127.0.0.1:1") // nothing listening
	proxy := NewProxy(target, zap.NewNop())

	req := httptest.NewRequest("GET", "/bratrax/health", nil)
	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want 502", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode 502 body: %v", err)
	}
	if body["error"] != "bad_gateway" {
		t.Errorf("error field = %q, want %q", body["error"], "bad_gateway")
	}
}
