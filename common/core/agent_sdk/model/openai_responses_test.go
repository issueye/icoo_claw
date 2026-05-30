package model

import (
	"encoding/json"
	"testing"
)

func TestBuildResponsesInputPreservesStructuredHistory(t *testing.T) {
	input := buildResponsesInput([]Message{
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
		{Role: "assistant", Content: "README contains file contents."},
	})

	items := marshalResponsesInputItems(t, input)
	if len(items) != 5 {
		t.Fatalf("items len = %d, want 5: %#v", len(items), items)
	}

	assertResponsesMessage(t, items[0], "user", "read README")
	assertResponsesMessage(t, items[1], "assistant", "I'll inspect it.")

	if items[2]["type"] != "function_call" {
		t.Fatalf("item[2] type = %v, want function_call: %#v", items[2]["type"], items[2])
	}
	if items[2]["call_id"] != "call_1" || items[2]["name"] != "read" {
		t.Fatalf("function call = %#v, want call_1/read", items[2])
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(items[2]["arguments"].(string)), &args); err != nil {
		t.Fatalf("unmarshal arguments: %v", err)
	}
	if args["path"] != "README.md" {
		t.Fatalf("arguments path = %v, want README.md", args["path"])
	}

	if items[3]["type"] != "function_call_output" {
		t.Fatalf("item[3] type = %v, want function_call_output: %#v", items[3]["type"], items[3])
	}
	if items[3]["call_id"] != "call_1" || items[3]["output"] != "file contents" {
		t.Fatalf("function call output = %#v, want call_1/file contents", items[3])
	}

	assertResponsesMessage(t, items[4], "assistant", "README contains file contents.")
}

func TestBuildResponsesInputPreservesMultimodalMessage(t *testing.T) {
	input := buildResponsesInput([]Message{{
		Role: "user",
		ContentBlocks: []ContentBlock{
			{Type: ContentBlockText, Text: "describe"},
			{Type: ContentBlockImage, URL: "https://example.test/image.png"},
		},
	}})

	items := marshalResponsesInputItems(t, input)
	if len(items) != 1 {
		t.Fatalf("items len = %d, want 1", len(items))
	}
	content, ok := items[0]["content"].([]any)
	if !ok {
		t.Fatalf("content = %#v, want list", items[0]["content"])
	}
	if len(content) != 2 {
		t.Fatalf("content len = %d, want 2: %#v", len(content), content)
	}
	if content[0].(map[string]any)["type"] != "input_text" || content[0].(map[string]any)["text"] != "describe" {
		t.Fatalf("text part = %#v, want input_text describe", content[0])
	}
	if content[1].(map[string]any)["type"] != "input_image" || content[1].(map[string]any)["image_url"] != "https://example.test/image.png" {
		t.Fatalf("image part = %#v, want input_image URL", content[1])
	}
}

func TestBuildResponsesInputEmptyConversationUsesPlaceholderMessage(t *testing.T) {
	input := buildResponsesInput(nil)
	items := marshalResponsesInputItems(t, input)
	if len(items) != 1 {
		t.Fatalf("items len = %d, want 1", len(items))
	}
	assertResponsesMessage(t, items[0], "user", ".")
}

func marshalResponsesInputItems(t *testing.T, input any) []map[string]any {
	t.Helper()
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}
	var items []map[string]any
	if err := json.Unmarshal(data, &items); err != nil {
		t.Fatalf("unmarshal input %s: %v", data, err)
	}
	return items
}

func assertResponsesMessage(t *testing.T, item map[string]any, role string, text string) {
	t.Helper()
	if itemType, ok := item["type"]; ok && itemType != "message" {
		t.Fatalf("message type = %v, want message or omitted", itemType)
	}
	if item["role"] != role {
		t.Fatalf("message metadata = %#v, want role %s", item, role)
	}
	if contentText, ok := item["content"].(string); ok {
		if contentText != text {
			t.Fatalf("content = %q, want %q", contentText, text)
		}
		return
	}
	content, ok := item["content"].([]any)
	if !ok || len(content) != 1 {
		t.Fatalf("content = %#v, want single content part", item["content"])
	}
	part, ok := content[0].(map[string]any)
	if !ok {
		t.Fatalf("content part = %#v, want object", content[0])
	}
	if part["type"] != "input_text" || part["text"] != text {
		t.Fatalf("content part = %#v, want input_text %q", part, text)
	}
}
