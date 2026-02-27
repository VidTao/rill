package bratrax

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"go.uber.org/zap"
)

// NewProxy creates a reverse proxy that forwards requests to the Flask API.
// It strips the /bratrax prefix from the path before forwarding.
func NewProxy(target *url.URL, logger *zap.Logger) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(target)

	// Override Director to strip /bratrax prefix before forwarding.
	defaultDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		defaultDirector(req)
		req.URL.Path = stripBratraxPrefix(req.URL.Path)
		if req.URL.RawPath != "" {
			req.URL.RawPath = stripBratraxPrefix(req.URL.RawPath)
		}
		req.Host = target.Host
	}

	// Clone default transport to preserve connection pooling, timeouts, and HTTP/2.
	// Only skip TLS verification for non-HTTPS targets (dev) or explicit opt-in.
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if target.Scheme != "https" {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // HTTP target, no TLS to verify
	}
	proxy.Transport = transport

	// Return a JSON 502 on proxy errors instead of a bare text response.
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		logger.Error("bratrax proxy error",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Error(err),
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		if encErr := json.NewEncoder(w).Encode(map[string]string{
			"error":   "bad_gateway",
			"message": "Bratrax API is unreachable",
		}); encErr != nil {
			logger.Debug("failed to write 502 response", zap.Error(encErr))
		}
	}

	return proxy
}

// stripBratraxPrefix removes the leading /bratrax segment from a URL path.
// Examples:
//
//	"/bratrax/connectors/foo" -> "/connectors/foo"
//	"/bratrax"                -> "/"
//	"/bratrax/"               -> "/"
//	"/other/path"             -> "/other/path"
func stripBratraxPrefix(path string) string {
	const prefix = "/bratrax"
	if !strings.HasPrefix(path, prefix) {
		return path
	}
	rest := path[len(prefix):]
	if rest == "" || rest == "/" {
		return "/"
	}
	if rest[0] != '/' {
		// Path was e.g. "/bratraxfoo" — not our prefix.
		return path
	}
	return rest
}
