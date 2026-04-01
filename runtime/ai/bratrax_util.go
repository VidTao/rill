package ai

import (
	"context"

	"github.com/rilldata/rill/runtime"
)

// checkBratraxAccess verifies the session has UseAI permission.
// This is the base check for all Bratrax AI tools (read-only data tools + Claude chat).
func checkBratraxAccess(ctx context.Context) (bool, error) {
	s := GetSession(ctx)
	return s.Claims().Can(runtime.UseAI), nil
}

// checkBratraxWriteAccess verifies the session has admin role.
// Workshop tools (create, write, compile, deploy) require this.
// Lite viewer users get read-only MCP (query tools + Claude chat, no workshop).
func checkBratraxWriteAccess(ctx context.Context) (bool, error) {
	s := GetSession(ctx)
	if !s.Claims().Can(runtime.UseAI) {
		return false, nil
	}
	return getBratraxUserRole(ctx) == "admin", nil
}

// getBratraxClientID extracts the client_id from the session's security claims.
// Returns the client_id if available, empty string otherwise.
// The client_id is set in UserAttributes by the auth pipeline when the user
// has an associated bratrax_clients record (via project_id → clientstore lookup).
func getBratraxClientID(ctx context.Context) string {
	s := GetSession(ctx)
	claims := s.Claims()
	if claims == nil || claims.UserAttributes == nil {
		return ""
	}
	// Check for explicit client_id attribute
	if cid, ok := claims.UserAttributes["client_id"].(string); ok && cid != "" {
		return cid
	}
	// Fall back to project_id (used as rill_project_id in bratrax_clients)
	if pid, ok := claims.UserAttributes["project_id"].(string); ok && pid != "" {
		return pid
	}
	return ""
}

// getBratraxUserRole extracts the user role from session claims.
// Returns "admin" or "viewer".
func getBratraxUserRole(ctx context.Context) string {
	s := GetSession(ctx)
	claims := s.Claims()
	if claims == nil || claims.UserAttributes == nil {
		return ""
	}
	if role, ok := claims.UserAttributes["role"].(string); ok {
		return role
	}
	return ""
}
