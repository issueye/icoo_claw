package acp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	acp "github.com/coder/acp-go-sdk"
	"icoo_claw/server/claw/pkg/agent_sdk"
)

type errorRunner struct{}

func (r errorRunner) Run(context.Context, agent_sdk.RunRequest) (*agent_sdk.RunResponse, error) {
	return nil, nil
}

func (r errorRunner) RunStream(context.Context, agent_sdk.RunRequest) (<-chan agent_sdk.StreamEvent, error) {
	events := make(chan agent_sdk.StreamEvent, 1)
	events <- agent_sdk.StreamEvent{Type: agent_sdk.StreamEventSessionError, Error: &agent_sdk.StreamError{Message: "boom"}}
	close(events)
	return events, nil
}

func TestPromptReturnsErrorForRuntimeSessionError(t *testing.T) {
	agent := NewAgent(errorRunner{})
	_, err := agent.Prompt(context.Background(), acp.PromptRequest{
		SessionId: acp.SessionId("sess_1"),
		Prompt:    []acp.ContentBlock{acp.TextBlock("hello")},
	})
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("error = %v, want runtime session error", err)
	}
}

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
