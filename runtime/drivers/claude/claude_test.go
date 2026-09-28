package claude

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	aiv1 "github.com/rilldata/rill/proto/gen/rill/ai/v1"
	"github.com/rilldata/rill/runtime/drivers"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
)

// cannedResponse is a minimal Messages API response with prompt cache usage.
const cannedResponse = `{
	"id": "msg_test",
	"type": "message",
	"role": "assistant",
	"model": "claude-sonnet-5",
	"content": [{"type": "text", "text": "done"}],
	"stop_reason": "end_turn",
	"stop_sequence": null,
	"usage": {"input_tokens": 10, "output_tokens": 5, "cache_creation_input_tokens": 100, "cache_read_input_tokens": 200}
}`

func TestCompleteRequest(t *testing.T) {
	h, captured := newTestHandle(t, map[string]any{})

	res, err := h.Complete(t.Context(), &drivers.CompleteOptions{
		Messages: agentLoopMessages(),
		Tools:    testTools(),
	})
	require.NoError(t, err)

	body := captured()
	require.Equal(t, "claude-sonnet-5", body["model"])
	require.EqualValues(t, 16000, body["max_tokens"])
	require.NotContains(t, body, "temperature")
	require.Equal(t, "auto", body["tool_choice"].(map[string]any)["type"])

	// Breakpoints: last system block (caches tools + system) and last message block (caches the conversation).
	system := body["system"].([]any)
	require.Equal(t, "ephemeral", cacheControlType(system[len(system)-1]))
	msgs := body["messages"].([]any)
	lastContent := msgs[len(msgs)-1].(map[string]any)["content"].([]any)
	require.Equal(t, "ephemeral", cacheControlType(lastContent[len(lastContent)-1]))
	tools := body["tools"].([]any)
	require.Empty(t, cacheControlType(tools[len(tools)-1]))

	require.Equal(t, 10, res.InputTokens)
	require.Equal(t, 5, res.OutputTokens)
	require.Equal(t, 100, res.CacheCreationInputTokens)
	require.Equal(t, 200, res.CacheReadInputTokens)
}

func TestCompleteNoToolCallsKeepsTools(t *testing.T) {
	h, captured := newTestHandle(t, map[string]any{})

	_, err := h.Complete(t.Context(), &drivers.CompleteOptions{
		Messages:    agentLoopMessages(),
		Tools:       testTools(),
		NoToolCalls: true,
	})
	require.NoError(t, err)

	body := captured()
	require.Len(t, body["tools"], 1)
	require.Equal(t, "none", body["tool_choice"].(map[string]any)["type"])
}

func TestCompleteCachesToolsWithoutSystem(t *testing.T) {
	h, captured := newTestHandle(t, map[string]any{"model": "claude-haiku-4-5"})

	_, err := h.Complete(t.Context(), &drivers.CompleteOptions{
		Messages: agentLoopMessages()[1:],
		Tools:    testTools(),
	})
	require.NoError(t, err)

	body := captured()
	require.Equal(t, "claude-haiku-4-5", body["model"])
	require.NotContains(t, body, "system")
	tools := body["tools"].([]any)
	require.Equal(t, "ephemeral", cacheControlType(tools[len(tools)-1]))
}

// newTestHandle opens the driver against a fake API server. The returned func gives the last request body.
func newTestHandle(t *testing.T, config map[string]any) (*handle, func() map[string]any) {
	var body []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		body, err = io.ReadAll(r.Body)
		require.NoError(t, err)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(cannedResponse))
	}))
	t.Cleanup(srv.Close)

	config["api_key"] = "sk-ant-test"
	config["base_url"] = srv.URL
	conn, err := driver{}.Open("test", config, nil, nil, zap.NewNop())
	require.NoError(t, err)

	return conn.(*handle), func() map[string]any {
		var m map[string]any
		require.NoError(t, json.Unmarshal(body, &m))
		return m
	}
}

// agentLoopMessages is a conversation shaped like an agent loop iteration: system prompt, question, one tool exchange.
func agentLoopMessages() []*aiv1.CompletionMessage {
	return []*aiv1.CompletionMessage{
		{Role: "system", Content: []*aiv1.ContentBlock{{BlockType: &aiv1.ContentBlock_Text{Text: "You are an analyst."}}}},
		{Role: "user", Content: []*aiv1.ContentBlock{{BlockType: &aiv1.ContentBlock_Text{Text: "Which ad earned the most?"}}}},
		{Role: "assistant", Content: []*aiv1.ContentBlock{{BlockType: &aiv1.ContentBlock_ToolCall{ToolCall: &aiv1.ToolCall{Id: "call1", Name: "query", Input: &structpb.Struct{}}}}}},
		{Role: "tool", Content: []*aiv1.ContentBlock{{BlockType: &aiv1.ContentBlock_ToolResult{ToolResult: &aiv1.ToolResult{Id: "call1", Content: `{"rows":[]}`}}}}},
	}
}

func testTools() []*aiv1.Tool {
	return []*aiv1.Tool{{Name: "query", Description: "Run a query.", InputSchema: `{"type":"object","properties":{"sql":{"type":"string"}}}`}}
}

func cacheControlType(block any) string {
	cc, ok := block.(map[string]any)["cache_control"].(map[string]any)
	if !ok {
		return ""
	}
	return cc["type"].(string)
}
