package api

import "testing"

func TestACPProtocolToolKindNormalization(t *testing.T) {
	protocol := NewACPProtocol()
	encoded := protocol.Encode(AgentEvent{
		Type:      AEToolCallStart,
		SessionID: "session_1",
		RequestID: "request_1",
		ToolUseID: "call_1",
		ToolName:  "  Code_Search  ",
	})

	event, ok := encoded.(ACPEvent)
	if !ok {
		t.Fatalf("encoded = %#v, want ACPEvent", encoded)
	}
	if event.Update == nil {
		t.Fatalf("update is nil")
	}
	if event.Update.Kind != "search" {
		t.Fatalf("kind = %q, want search", event.Update.Kind)
	}
	if event.Update.Title != "Code_Search" {
		t.Fatalf("title = %q, want trimmed display name", event.Update.Title)
	}
}
