package ai

import (
	"fmt"
	"strings"
	"testing"

	aiv1 "github.com/rilldata/rill/proto/gen/rill/ai/v1"
	"github.com/rilldata/rill/runtime/drivers"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestTruncateMessagesUnderLimit(t *testing.T) {
	// Many small messages: well past the old 20-message cutoff, but far under the size limit.
	messages := []*aiv1.CompletionMessage{
		NewTextCompletionMessage(RoleSystem, "system prompt"),
		NewTextCompletionMessage(RoleUser, "Which ad earned the most?"),
	}
	for i := range 30 {
		messages = append(messages, toolExchange(fmt.Sprintf("call%d", i), 100)...)
	}

	// Unchanged, so the prompt stays append-only across iterations.
	require.Equal(t, messages, maybeTruncateMessages(messages))
}

func TestTruncateMessagesOverLimit(t *testing.T) {
	messages := []*aiv1.CompletionMessage{
		NewTextCompletionMessage(RoleSystem, "system prompt"),
		toolExchange("seed", 50_000)[0],
		toolExchange("seed", 50_000)[1],
		NewTextCompletionMessage(RoleUser, "Which ad earned the most?"),
		NewTextCompletionMessage(RoleAssistant, "Let me check."),
	}
	for i := range 12 {
		messages = append(messages, toolExchange(fmt.Sprintf("call%d", i), 50_000)...)
	}
	messages = append(messages, NewTextCompletionMessage(RoleUser, "Tool call limit reached."))

	res := maybeTruncateMessages(messages)

	var size int
	for _, msg := range res {
		size += proto.Size(msg)
	}
	require.LessOrEqual(t, size, maxContextChars)

	// System prompt first, then the notice as a user message (not a system message, which the claude driver would hoist into the system prompt).
	require.Equal(t, string(RoleSystem), res[0].Role)
	require.Equal(t, string(RoleUser), res[1].Role)
	require.Equal(t, truncationNotice, res[1].Content[0].GetText())

	// Every user message survives, including the question.
	var userTexts []string
	for _, msg := range res {
		if msg.Role == string(RoleUser) {
			userTexts = append(userTexts, msg.Content[0].GetText())
		}
	}
	require.Equal(t, []string{truncationNotice, "Which ad earned the most?", "Tool call limit reached."}, userTexts)

	// The oldest tool exchanges went first, the newest is kept, and no call or result is orphaned.
	ids := make(map[string]int)
	for _, msg := range res {
		for _, block := range msg.Content {
			if call := block.GetToolCall(); call != nil {
				ids[call.Id]++
			} else if r := block.GetToolResult(); r != nil {
				ids[r.Id]++
			}
		}
	}
	for id, n := range ids {
		require.Equal(t, 2, n, "tool call %q is unbalanced", id)
	}
	require.NotContains(t, ids, "seed")
	require.NotContains(t, ids, "call0")
	require.Contains(t, ids, "call11")
}

func TestWeightedTokens(t *testing.T) {
	res := &drivers.CompleteResult{
		InputTokens:              1000,
		CacheCreationInputTokens: 4000,
		CacheReadInputTokens:     20000,
		OutputTokens:             500,
	}
	// 1000 + 4000*1.25 + 20000*0.1 + 500*5
	require.Equal(t, 1000+5000+2000+2500, weightedTokens(res))
}

func toolExchange(id string, resultSize int) []*aiv1.CompletionMessage {
	return []*aiv1.CompletionMessage{
		{Role: string(RoleAssistant), Content: []*aiv1.ContentBlock{{BlockType: &aiv1.ContentBlock_ToolCall{ToolCall: &aiv1.ToolCall{Id: id, Name: "query_metrics_view"}}}}},
		{Role: string(RoleTool), Content: []*aiv1.ContentBlock{{BlockType: &aiv1.ContentBlock_ToolResult{ToolResult: &aiv1.ToolResult{Id: id, Content: strings.Repeat("x", resultSize)}}}}},
	}
}
