package model

import (
	"encoding/json"
	"testing"
)

func TestOpenAIChatEncodingPreservesToolHistoryFromConversationIR(t *testing.T) {
	messages := convertMessagesToOpenAI(toolHistoryFixture(), "default system")
	data, err := json.Marshal(messages)
	if err != nil {
		t.Fatalf("marshal openai messages: %v", err)
	}
	var got []map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal openai messages %s: %v", data, err)
	}

	if len(got) != 5 {
		t.Fatalf("messages len = %d, want 5: %#v", len(got), got)
	}
	if got[0]["role"] != "system" || got[0]["content"] != "default system" {
		t.Fatalf("system message = %#v", got[0])
	}
	if got[1]["role"] != "user" || got[1]["content"] != "read README" {
		t.Fatalf("user message = %#v", got[1])
	}
	if got[2]["role"] != "assistant" {
		t.Fatalf("assistant message role = %#v", got[2])
	}
	toolCalls, ok := got[2]["tool_calls"].([]any)
	if !ok || len(toolCalls) != 1 {
		t.Fatalf("tool calls = %#v, want one", got[2]["tool_calls"])
	}
	call := toolCalls[0].(map[string]any)
	if call["id"] != "call_1" {
		t.Fatalf("tool call = %#v, want call_1", call)
	}
	function := call["function"].(map[string]any)
	if function["name"] != "read" {
		t.Fatalf("tool function = %#v, want read", function)
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(function["arguments"].(string)), &args); err != nil {
		t.Fatalf("unmarshal arguments: %v", err)
	}
	if args["path"] != "README.md" {
		t.Fatalf("arguments path = %v, want README.md", args["path"])
	}
	if got[3]["role"] != "tool" || got[3]["tool_call_id"] != "call_1" || got[3]["content"] != "file contents" {
		t.Fatalf("tool result = %#v", got[3])
	}
	if got[4]["role"] != "assistant" || got[4]["content"] != "done" {
		t.Fatalf("final assistant = %#v", got[4])
	}
}

func TestAnthropicEncodingPreservesToolHistoryFromConversationIR(t *testing.T) {
	system, messages := convertMessages(toolHistoryFixture(), false, "default system")
	if len(system) != 1 || system[0].Text != "default system" {
		t.Fatalf("system = %#v, want default system", system)
	}

	data, err := json.Marshal(messages)
	if err != nil {
		t.Fatalf("marshal anthropic messages: %v", err)
	}
	var got []map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal anthropic messages %s: %v", data, err)
	}

	if len(got) != 4 {
		t.Fatalf("messages len = %d, want 4: %#v", len(got), got)
	}
	if got[0]["role"] != "user" {
		t.Fatalf("user role = %#v", got[0])
	}
	if got[1]["role"] != "assistant" {
		t.Fatalf("assistant role = %#v", got[1])
	}
	assistantContent := got[1]["content"].([]any)
	if len(assistantContent) != 2 {
		t.Fatalf("assistant content len = %d, want text + tool_use: %#v", len(assistantContent), assistantContent)
	}
	toolUse := assistantContent[1].(map[string]any)
	if toolUse["type"] != "tool_use" || toolUse["id"] != "call_1" || toolUse["name"] != "read" {
		t.Fatalf("tool use = %#v, want call_1/read", toolUse)
	}
	input := toolUse["input"].(map[string]any)
	if input["path"] != "README.md" {
		t.Fatalf("tool input path = %v, want README.md", input["path"])
	}
	if got[2]["role"] != "user" {
		t.Fatalf("tool result wrapper role = %#v", got[2])
	}
	toolResult := got[2]["content"].([]any)[0].(map[string]any)
	if toolResult["type"] != "tool_result" || toolResult["tool_use_id"] != "call_1" {
		t.Fatalf("tool result = %#v, want tool_result call_1", toolResult)
	}
	if got[3]["role"] != "assistant" {
		t.Fatalf("final assistant role = %#v", got[3])
	}
}

func toolHistoryFixture() []Message {
	return []Message{
		{Role: "user", Content: "read README"},
		{
			Role:    "assistant",
			Content: "I'll inspect it.",
			ToolCalls: []ToolCall{{
				ID:        "call_1",
				Name:      "read",
				Arguments: map[string]any{"path": "README.md"},
			}},
		},
		{
			Role: "tool",
			ToolCalls: []ToolCall{{
				ID:     "call_1",
				Name:   "read",
				Result: "file contents",
			}},
		},
		{Role: "assistant", Content: "done"},
	}
}
