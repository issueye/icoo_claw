package acp

import (
	"encoding/json"
	"testing"

	"icoo_claw/server/claw/pkg/agent_sdk"
)

func TestToACPUpdateSkipsUnsupportedUpdates(t *testing.T) {
	update, ok := toACPUpdate(&agent_sdk.SessionUpdate{SessionUpdate: "content_block_start"})
	if ok {
		t.Fatalf("ok = true for unsupported update: %+v", update)
	}
}

func TestToACPUpdateReturnsMarshalableAgentMessage(t *testing.T) {
	update, ok := toACPUpdate(&agent_sdk.SessionUpdate{
		SessionUpdate: "agent_message_chunk",
		Content:       &agent_sdk.ContentBlock{Type: "text", Text: "hello"},
	})
	if !ok {
		t.Fatal("ok = false, want supported update")
	}
	if _, err := json.Marshal(update); err != nil {
		t.Fatalf("marshal update: %v", err)
	}
}
