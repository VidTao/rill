package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

// bratraxAPIBaseURL returns the Flask API base URL from the environment.
func bratraxAPIBaseURL() string {
	u := os.Getenv("BRATRAX_API_URL")
	if u == "" {
		u = "http://localhost:8081"
	}
	return u + "/api/v1"
}

// bratraxGet performs a GET request to the Flask API and decodes the JSON response.
func bratraxGet(ctx context.Context, path string, query url.Values, out any) error {
	u := bratraxAPIBaseURL() + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return fmt.Errorf("bratrax API: failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	return bratraxDo(req, out)
}

// bratraxPost performs a POST request to the Flask API with a JSON body and decodes the response.
func bratraxPost(ctx context.Context, path string, body any, out any, timeout time.Duration) error {
	u := bratraxAPIBaseURL() + path

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("bratrax API: failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bodyReader)
	if err != nil {
		return fmt.Errorf("bratrax API: failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	return bratraxDoWithTimeout(req, out, timeout)
}

// bratraxPut performs a PUT request to the Flask API with a JSON body and decodes the response.
func bratraxPut(ctx context.Context, path string, body any, out any) error {
	u := bratraxAPIBaseURL() + path

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("bratrax API: failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u, bodyReader)
	if err != nil {
		return fmt.Errorf("bratrax API: failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	return bratraxDo(req, out)
}

// bratraxDo executes the HTTP request with a default 30s timeout and decodes JSON into out.
func bratraxDo(req *http.Request, out any) error {
	return bratraxDoWithTimeout(req, out, 30*time.Second)
}

// bratraxDoWithTimeout executes the HTTP request with a custom timeout and decodes JSON into out.
func bratraxDoWithTimeout(req *http.Request, out any, timeout time.Duration) error {
	client := &http.Client{Timeout: timeout}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("bratrax API: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("bratrax API: failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("bratrax API: %s %s returned %d: %s", req.Method, req.URL.Path, resp.StatusCode, string(respBody))
	}

	if out != nil {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("bratrax API: failed to decode response: %w", err)
		}
	}

	return nil
}

// bratraxEnvelope is the standard response wrapper from Flask/Lightdash APIs.
type bratraxEnvelope struct {
	Status  string          `json:"status"`
	Results json.RawMessage `json:"results"`
}

// bratraxGetResults performs a GET and unwraps the {status, results} envelope.
func bratraxGetResults(ctx context.Context, path string, query url.Values, out any) error {
	var env bratraxEnvelope
	if err := bratraxGet(ctx, path, query, &env); err != nil {
		return err
	}
	if env.Status != "ok" {
		return fmt.Errorf("bratrax API: status=%s, results=%s", env.Status, string(env.Results))
	}
	if out != nil {
		if err := json.Unmarshal(env.Results, out); err != nil {
			return fmt.Errorf("bratrax API: failed to decode results: %w", err)
		}
	}
	return nil
}

// bratraxPostResults performs a POST and unwraps the {status, results} envelope.
func bratraxPostResults(ctx context.Context, path string, body any, out any, timeout time.Duration) error {
	var env bratraxEnvelope
	if err := bratraxPost(ctx, path, body, &env, timeout); err != nil {
		return err
	}
	if env.Status != "ok" {
		return fmt.Errorf("bratrax API: status=%s, results=%s", env.Status, string(env.Results))
	}
	if out != nil {
		if err := json.Unmarshal(env.Results, out); err != nil {
			return fmt.Errorf("bratrax API: failed to decode results: %w", err)
		}
	}
	return nil
}

// bratraxPutResults performs a PUT and unwraps the {status, results} envelope.
func bratraxPutResults(ctx context.Context, path string, body any, out any) error {
	var env bratraxEnvelope
	if err := bratraxPut(ctx, path, body, &env); err != nil {
		return err
	}
	if env.Status != "ok" {
		return fmt.Errorf("bratrax API: status=%s, results=%s", env.Status, string(env.Results))
	}
	if out != nil {
		if err := json.Unmarshal(env.Results, out); err != nil {
			return fmt.Errorf("bratrax API: failed to decode results: %w", err)
		}
	}
	return nil
}
