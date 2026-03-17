package ai

import (
	"context"

	"github.com/rilldata/rill/runtime"
)

// checkBratraxAccess verifies the session has permission for Bratrax workshop tools.
// For now, requires UseAI permission. Can be tightened later.
func checkBratraxAccess(ctx context.Context) (bool, error) {
	s := GetSession(ctx)
	return s.Claims().Can(runtime.UseAI), nil
}
