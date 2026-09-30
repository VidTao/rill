package ai_test

import (
	"context"
	"testing"

	aiv1 "github.com/rilldata/rill/proto/gen/rill/ai/v1"
	"github.com/rilldata/rill/runtime/ai"
	"github.com/rilldata/rill/runtime/drivers"
	"github.com/rilldata/rill/runtime/testruntime"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestCompleteTokenBudget(t *testing.T) {
	// Each call costs 100 input-token equivalents, so a budget of 250 is used up after 3 calls and the 4th is the final one.
	calls := runToolLoop(t, &ai.CompleteOptions{MaxIterations: 10, TokenBudget: 250})

	require.Len(t, calls, 4)
	for _, c := range calls[:3] {
		require.False(t, c.NoToolCalls)
	}
	last := calls[3]
	require.True(t, last.NoToolCalls)
	require.NotEmpty(t, last.Tools, "the final call must still carry the tools referenced by earlier tool calls")
	require.Contains(t, last.Messages[len(last.Messages)-1].Content[0].GetText(), "Tool call limit reached")
}

func TestCompleteIterationCapKeepsTools(t *testing.T) {
	calls := runToolLoop(t, &ai.CompleteOptions{MaxIterations: 3})

	require.Len(t, calls, 3)
	require.False(t, calls[1].NoToolCalls)
	require.True(t, calls[2].NoToolCalls)
	require.NotEmpty(t, calls[2].Tools)
}

// runToolLoop runs Session.Complete against a fake LLM that keeps calling list_metrics_views until it's told not to.
// It returns the options of every LLM call.
func runToolLoop(t *testing.T, opts *ai.CompleteOptions) []*drivers.CompleteOptions {
	rt, instanceID := testruntime.NewInstanceWithOptions(t, testruntime.InstanceOptions{
		Files: map[string]string{"rill.yaml": ""},
	})
	s := newSession(t, rt, instanceID)

	llm := &toolLoopLLM{}
	s.SetLLM(func(ctx context.Context) (drivers.AIService, func(), error) {
		return llm, func() {}, nil
	})

	opts.Messages = []*aiv1.CompletionMessage{ai.NewTextCompletionMessage(ai.RoleUser, "List the metrics views.")}
	opts.Tools = []string{ai.ListMetricsViewsName}
	var out string
	err := s.Complete(ai.WithSession(t.Context(), s), "test loop", &out, opts)
	require.NoError(t, err)
	require.Equal(t, "done", out)

	return llm.calls
}

type toolLoopLLM struct {
	calls []*drivers.CompleteOptions
}

func (l *toolLoopLLM) Complete(ctx context.Context, opts *drivers.CompleteOptions) (*drivers.CompleteResult, error) {
	l.calls = append(l.calls, opts)

	block := &aiv1.ContentBlock{BlockType: &aiv1.ContentBlock_ToolCall{ToolCall: &aiv1.ToolCall{
		Id:    "call",
		Name:  ai.ListMetricsViewsName,
		Input: &structpb.Struct{},
	}}}
	if opts.NoToolCalls {
		block = &aiv1.ContentBlock{BlockType: &aiv1.ContentBlock_Text{Text: "done"}}
	}

	return &drivers.CompleteResult{
		Message:     &aiv1.CompletionMessage{Role: "assistant", Content: []*aiv1.ContentBlock{block}},
		InputTokens: 100,
	}, nil
}
