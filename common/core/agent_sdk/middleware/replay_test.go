package middleware

import (
	"strings"
	"testing"
	"time"
)

func TestRedactorRedactsSensitiveKeys(t *testing.T) {
	redactor := Redactor{MaxStringLength: 128}
	input := map[string]any{
		"api_key":       "sk-live",
		"Authorization": "Bearer token",
		"safe":          "visible",
	}

	got := redactor.Redact(input).(map[string]any)

	if got["api_key"] != defaultRedactionReplacement {
		t.Fatalf("api_key was not redacted: %#v", got["api_key"])
	}
	if got["Authorization"] != defaultRedactionReplacement {
		t.Fatalf("Authorization was not redacted: %#v", got["Authorization"])
	}
	if got["safe"] != "visible" {
		t.Fatalf("safe value changed: %#v", got["safe"])
	}
}

func TestRedactorRedactsNestedMapAndSlice(t *testing.T) {
	redactor := DefaultRedactor()
	input := map[string]any{
		"outer": map[string]any{
			"password": "hunter2",
			"items": []any{
				map[string]any{"token": "abc", "name": "kept"},
				"plain",
			},
		},
	}

	got := redactor.Redact(input).(map[string]any)
	outer := got["outer"].(map[string]any)
	items := outer["items"].([]any)
	first := items[0].(map[string]any)

	if outer["password"] != defaultRedactionReplacement {
		t.Fatalf("nested password was not redacted: %#v", outer["password"])
	}
	if first["token"] != defaultRedactionReplacement {
		t.Fatalf("nested token was not redacted: %#v", first["token"])
	}
	if first["name"] != "kept" || items[1] != "plain" {
		t.Fatalf("non-sensitive nested values changed: %#v", got)
	}
}

func TestRedactorTruncatesLongStrings(t *testing.T) {
	redactor := Redactor{MaxStringLength: 20}
	got := redactor.Redact(strings.Repeat("a", 64)).(string)

	if len(got) != 20 {
		t.Fatalf("truncated length = %d, want 20", len(got))
	}
	if !strings.HasSuffix(got, truncatedSuffix) {
		t.Fatalf("truncated string missing suffix: %q", got)
	}
}

func TestBuildReplayArtifactDoesNotMutateInputs(t *testing.T) {
	createdAt := time.Date(2026, 5, 29, 10, 11, 12, 0, time.UTC)
	settings := map[string]any{
		"api_key": "original-secret",
		"nested":  map[string]any{"token": "nested-secret"},
	}
	metadata := map[string]any{
		"headers": map[string]any{"Authorization": "Bearer secret"},
	}
	events := []TraceEvent{
		{
			Timestamp:    createdAt,
			Stage:        "before_tool",
			SessionID:    "session-1",
			ModelRequest: map[string]any{"api_key": "event-secret", "prompt": "hello"},
			ToolCall:     map[string]any{"input": map[string]any{"password": "tool-secret"}},
		},
	}

	artifact := BuildReplayArtifact(ReplayArtifactOptions{
		RunID:     "run-1",
		SessionID: "session-1",
		RequestID: "request-1",
		CreatedAt: createdAt,
		Events:    events,
		Settings:  settings,
		Metadata:  metadata,
	})

	if artifact.RunID != "run-1" || artifact.SessionID != "session-1" || artifact.RequestID != "request-1" {
		t.Fatalf("artifact ids not preserved: %#v", artifact)
	}
	if !artifact.CreatedAt.Equal(createdAt) {
		t.Fatalf("created_at = %s, want %s", artifact.CreatedAt, createdAt)
	}
	if artifact.Settings["api_key"] != defaultRedactionReplacement {
		t.Fatalf("artifact settings api_key was not redacted: %#v", artifact.Settings)
	}
	if artifact.Events[0].ModelRequest["api_key"] != defaultRedactionReplacement {
		t.Fatalf("artifact event model request was not redacted: %#v", artifact.Events[0].ModelRequest)
	}
	input := artifact.Events[0].ToolCall["input"].(map[string]any)
	if input["password"] != defaultRedactionReplacement {
		t.Fatalf("artifact event tool input was not redacted: %#v", artifact.Events[0].ToolCall)
	}

	if settings["api_key"] != "original-secret" {
		t.Fatalf("settings mutated: %#v", settings)
	}
	if settings["nested"].(map[string]any)["token"] != "nested-secret" {
		t.Fatalf("nested settings mutated: %#v", settings)
	}
	if metadata["headers"].(map[string]any)["Authorization"] != "Bearer secret" {
		t.Fatalf("metadata mutated: %#v", metadata)
	}
	if events[0].ModelRequest["api_key"] != "event-secret" {
		t.Fatalf("events mutated: %#v", events[0].ModelRequest)
	}
	toolInput := events[0].ToolCall["input"].(map[string]any)
	if toolInput["password"] != "tool-secret" {
		t.Fatalf("nested event tool input mutated: %#v", toolInput)
	}
}
