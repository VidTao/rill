//go:build !windows

package local

// allowHostAccess returns whether DuckDB can access cloud-credential providers from the host.
// On Linux/macOS this is true — the platform supports the `aws` DuckDB extension natively,
// so S3/GCS/Azure fallback secrets work as designed.
func allowHostAccess() bool {
	return true
}
