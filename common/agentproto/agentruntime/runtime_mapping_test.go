package agentruntime

import (
	"encoding/json"
	"testing"

	"icoo_claw/common/agentproto"
	"icoo_claw/common/core/agent_sdk/api"
)

func TestMapStreamEventTextDelta(t *testing.T) {
	got := MapStreamEvent(api.StreamEvent{
		Type:  api.EventContentBlockDelta,
		Delta: &api.Delta{Type: "text_delta", Text: "hello"},
	}, "session_1", "request_1")

	if got.Type != agentproto.StreamEventSessionUpdate || got.SessionID != "session_1" || got.RequestID != "request_1" {
		t.Fatalf("event metadata = %#v", got)
	}
	if got.Update == nil || got.Update.SessionUpdate != "agent_message_chunk" {
		t.Fatalf("update = %#v, want agent message chunk", got.Update)
	}
	if got.Update.Content == nil || got.Update.Content.Text != "hello" {
		t.Fatalf("content = %#v, want hello", got.Update.Content)
	}
}

func TestMapStreamEventToolLifecycle(t *testing.T) {
	rawInput := json.RawMessage(`{"path":"README.md"}`)
	start := MapStreamEvent(api.StreamEvent{
		Type:      api.EventContentBlockStart,
		ToolUseID: "fallback_id",
		ContentBlock: &api.ContentBlock{
			Type:  "tool_use",
			ID:    "call_1",
			Name:  "read",
			Input: rawInput,
		},
	}, "session_1", "request_1")

	if start.Update == nil || start.Update.SessionUpdate != "tool_call" {
		t.Fatalf("start update = %#v, want tool_call", start.Update)
	}
	if start.Update.ToolCallID != "call_1" || start.Update.Kind != "read" || start.Update.Status != "pending" {
		t.Fatalf("start update = %#v, want pending read call_1", start.Update)
	}
	if start.Update.RawInput != string(rawInput) {
		t.Fatalf("raw input = %#v, want raw JSON string", start.Update.RawInput)
	}

	completed := MapStreamEvent(api.StreamEvent{
		Type:      api.EventToolExecutionResult,
		ToolUseID: "call_1",
		Output:    map[string]any{"output": "ok"},
		IsError:   boolPtr(false),
	}, "session_1", "request_1")

	if completed.Update == nil || completed.Update.Status != "completed" || completed.Update.RawOutput == nil {
		t.Fatalf("completed update = %#v, want completed with output", completed.Update)
	}

	failed := MapStreamEvent(api.StreamEvent{
		Type:      api.EventToolExecutionResult,
		ToolUseID: "call_1",
		Output:    map[string]any{"error": "boom"},
		IsError:   boolPtr(true),
	}, "session_1", "request_1")

	if failed.Update == nil || failed.Update.Status != "failed" {
		t.Fatalf("failed update = %#v, want failed", failed.Update)
	}
}

func TestMapStreamEventUsageAndCompletion(t *testing.T) {
	usage := MapStreamEvent(api.StreamEvent{
		Type:  api.EventMessageDelta,
		Usage: &api.Usage{InputTokens: 3, OutputTokens: 5},
	}, "session_1", "request_1")

	if usage.Update == nil || usage.Update.Usage == nil {
		t.Fatalf("usage update = %#v, want usage", usage.Update)
	}
	if usage.Update.Usage.TotalTokens != 8 {
		t.Fatalf("total tokens = %d, want 8", usage.Update.Usage.TotalTokens)
	}

	completed := MapStreamEvent(api.StreamEvent{Type: api.EventAgentStop}, "session_1", "request_1")
	if completed.Type != agentproto.StreamEventSessionCompleted || completed.StopReason != "end_turn" {
		t.Fatalf("completed = %#v, want end_turn completion", completed)
	}
}

func boolPtr(v bool) *bool {
	return &v
}
