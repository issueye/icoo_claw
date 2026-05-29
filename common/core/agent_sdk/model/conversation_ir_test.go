package model

import "testing"

func TestBuildConversationUserAssistant(t *testing.T) {
	conversation := BuildConversation([]Message{
		{Role: "USER", Content: "hello"},
		{Role: "assistant", Content: "hi"},
	})

	if len(conversation.Messages) != 2 {
		t.Fatalf("messages len = %d, want 2", len(conversation.Messages))
	}
	if conversation.Messages[0].Role != "user" || conversation.Messages[0].Text != "hello" {
		t.Fatalf("user message = %#v, want normalized user text", conversation.Messages[0])
	}
	if conversation.Messages[1].Role != "assistant" || conversation.Messages[1].Text != "hi" {
		t.Fatalf("assistant message = %#v, want assistant text", conversation.Messages[1])
	}
}

func TestBuildConversationAssistantToolCalls(t *testing.T) {
	conversation := BuildConversation([]Message{{
		Role:    "assistant",
		Content: "checking",
		ToolCalls: []ToolCall{{
			ID:   "call_1",
			Name: "read",
			Arguments: map[string]any{
				"path": "README.md",
				"opts": map[string]any{"limit": 10},
			},
		}},
	}})

	msg := conversation.Messages[0]
	if len(msg.ToolCalls) != 1 {
		t.Fatalf("tool calls len = %d, want 1", len(msg.ToolCalls))
	}
	call := msg.ToolCalls[0]
	if call.ID != "call_1" || call.Name != "read" {
		t.Fatalf("tool call = %#v, want id/name preserved", call)
	}
	opts, ok := call.Arguments["opts"].(map[string]any)
	if !ok || opts["limit"] != 10 {
		t.Fatalf("nested args = %#v, want cloned nested map", call.Arguments)
	}
	if len(msg.ToolResults) != 0 {
		t.Fatalf("tool results len = %d, want 0 for assistant", len(msg.ToolResults))
	}
}

func TestBuildConversationToolResult(t *testing.T) {
	conversation := BuildConversation([]Message{{
		Role:    "tool",
		Content: "fallback result",
		ToolCalls: []ToolCall{{
			ID:     "call_1",
			Name:   "read",
			Result: "file contents",
		}},
	}})

	msg := conversation.Messages[0]
	if msg.Role != "tool" {
		t.Fatalf("role = %q, want tool", msg.Role)
	}
	if len(msg.ToolResults) != 1 {
		t.Fatalf("tool results len = %d, want 1", len(msg.ToolResults))
	}
	result := msg.ToolResults[0]
	if result.ID != "call_1" || result.Name != "read" || result.Content != "file contents" {
		t.Fatalf("tool result = %#v, want structured result", result)
	}
	if len(msg.ToolCalls) != 0 {
		t.Fatalf("tool calls len = %d, want 0 for tool role", len(msg.ToolCalls))
	}
}

func TestBuildConversationToolResultFallsBackToMessageContent(t *testing.T) {
	conversation := BuildConversation([]Message{{
		Role:    "tool",
		Content: "fallback result",
		ToolCalls: []ToolCall{{
			ID:   "call_1",
			Name: "read",
		}},
	}})

	result := conversation.Messages[0].ToolResults[0]
	if result.Content != "fallback result" {
		t.Fatalf("result content = %q, want fallback message content", result.Content)
	}
}

func TestBuildConversationMultimodalBlocks(t *testing.T) {
	conversation := BuildConversation([]Message{{
		Role:    "user",
		Content: "look",
		ContentBlocks: []ContentBlock{
			{Type: ContentBlockText, Text: "caption"},
			{Type: ContentBlockImage, MediaType: "image/png", Data: "abc"},
			{Type: ContentBlockDocument, MediaType: "application/pdf", URL: "https://example.test/doc.pdf"},
		},
	}})

	msg := conversation.Messages[0]
	if msg.Text != "look" {
		t.Fatalf("text = %q, want original content", msg.Text)
	}
	if len(msg.ContentBlocks) != 3 {
		t.Fatalf("content blocks len = %d, want 3", len(msg.ContentBlocks))
	}
	if msg.ContentBlocks[1].Type != ContentBlockImage || msg.ContentBlocks[1].Data != "abc" {
		t.Fatalf("image block = %#v, want image data preserved", msg.ContentBlocks[1])
	}
}

func TestBuildConversationReasoningContent(t *testing.T) {
	conversation := BuildConversation([]Message{{
		Role:             "assistant",
		Content:          "answer",
		ReasoningContent: "private reasoning",
	}})

	msg := conversation.Messages[0]
	if msg.ReasoningContent != "private reasoning" {
		t.Fatalf("reasoning = %q, want preserved", msg.ReasoningContent)
	}
}

func TestBuildConversationInputMutationDoesNotAffectIR(t *testing.T) {
	args := map[string]any{
		"path": "README.md",
		"nested": map[string]any{
			"limit": 10,
		},
		"list": []any{"a", map[string]any{"b": "c"}},
	}
	messages := []Message{{
		Role: "assistant",
		ContentBlocks: []ContentBlock{{
			Type: ContentBlockText,
			Text: "before",
		}},
		ToolCalls: []ToolCall{{
			ID:        "call_1",
			Name:      "read",
			Arguments: args,
		}},
	}}

	conversation := BuildConversation(messages)

	messages[0].ContentBlocks[0].Text = "after"
	messages[0].ToolCalls[0].ID = "changed"
	args["path"] = "changed.md"
	args["nested"].(map[string]any)["limit"] = 20
	args["list"].([]any)[1].(map[string]any)["b"] = "changed"

	msg := conversation.Messages[0]
	if msg.ContentBlocks[0].Text != "before" {
		t.Fatalf("content block text = %q, want original", msg.ContentBlocks[0].Text)
	}
	call := msg.ToolCalls[0]
	if call.ID != "call_1" || call.Arguments["path"] != "README.md" {
		t.Fatalf("tool call = %#v, want original id/path", call)
	}
	nested := call.Arguments["nested"].(map[string]any)
	if nested["limit"] != 10 {
		t.Fatalf("nested limit = %v, want 10", nested["limit"])
	}
	listMap := call.Arguments["list"].([]any)[1].(map[string]any)
	if listMap["b"] != "c" {
		t.Fatalf("list map value = %v, want c", listMap["b"])
	}
}

func TestConversationCloneAndNormalize(t *testing.T) {
	conversation := Conversation{Messages: []ConversationMessage{{
		Role:      " ASSISTANT ",
		CacheHint: "unknown",
		ContentBlocks: []ContentBlock{{
			Type: ContentBlockText,
			Text: "hello",
		}},
		ToolCalls: []ConversationToolCall{{
			ID:        "call_1",
			Name:      "read",
			Arguments: map[string]any{"path": "README.md"},
		}},
	}}}

	normalized := conversation.Normalize()
	conversation.Messages[0].ContentBlocks[0].Text = "changed"
	conversation.Messages[0].ToolCalls[0].Arguments["path"] = "changed.md"

	msg := normalized.Messages[0]
	if msg.Role != "assistant" {
		t.Fatalf("role = %q, want assistant", msg.Role)
	}
	if msg.CacheHint != CacheHintNone {
		t.Fatalf("cache hint = %q, want none", msg.CacheHint)
	}
	if msg.ContentBlocks[0].Text != "hello" {
		t.Fatalf("content block text = %q, want cloned value", msg.ContentBlocks[0].Text)
	}
	if msg.ToolCalls[0].Arguments["path"] != "README.md" {
		t.Fatalf("path = %v, want cloned value", msg.ToolCalls[0].Arguments["path"])
	}
}
