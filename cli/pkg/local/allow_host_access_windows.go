//go:build windows

package local

// allowHostAccess returns false on Windows MinGW dev builds.
// DuckDB's `aws` extension isn't published for windows_amd64_mingw, but Rill autoloads it
// when AllowHostAccess is true (to create S3/GCS/Azure fallback credential-chain secrets).
// Disabling host access skips those secrets, avoiding the autoload failure.
// Production deployments run on Linux, where this returns true (see allow_host_access_unix.go).
func allowHostAccess() bool {
	return false
}
