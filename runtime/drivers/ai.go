package drivers

import (
	"context"

	"github.com/google/jsonschema-go/jsonschema"
	aiv1 "github.com/rilldata/rill/proto/gen/rill/ai/v1"
)

type AIService interface {
	Complete(ctx context.Context, opts *CompleteOptions) (*CompleteResult, error)
}

type CompleteOptions struct {
	Messages     []*aiv1.CompletionMessage
	Tools        []*aiv1.Tool
	OutputSchema *jsonschema.Schema
	// NoToolCalls forbids the model from calling any of Tools on this request.
	// The tool definitions are still sent where the provider allows it: a history
	// that contains tool calls must be accompanied by the tools it references, and
	// dropping them would also invalidate any cached prompt prefix.
	NoToolCalls bool
}

type CompleteResult struct {
	Message      *aiv1.CompletionMessage
	InputTokens  int
	OutputTokens int
	// CacheCreationInputTokens and CacheReadInputTokens are input tokens written to
	// and read from the provider's prompt cache. They are not included in
	// InputTokens. Zero for providers that don't report them.
	CacheCreationInputTokens int
	CacheReadInputTokens     int
}
