package agent_sdk

import (
	"context"
	"testing"

	"icoo_claw/server/claw/pkg/sessionstore"

	sdkmodel "icoo_claw/server/claw/pkg/agent_sdk/sdk/model"
)

type memoryHistoryStore struct {
	messages []sessionstore.Message
}

func (m *memoryHistoryStore) ListMessages(context.Context, string) ([]sessionstore.Message, error) {
	return append([]sessionstore.Message(nil), m.messages...), nil
}

func (m *memoryHistoryStore) ReplaceMessages(_ context.Context, _ string, messages []sessionstore.Message) error {
	m.messages = append([]sessionstore.Message(nil), messages...)
	return nil
}

type staticModel struct{}

func (m staticModel) Complete(context.Context, sdkmodel.Request) (*sdkmodel.Response, error) {
	return nil, nil
}

func (m staticModel) CompleteStream(_ context.Context, req sdkmodel.Request, cb sdkmodel.StreamHandler) error {
	return cb(sdkmodel.StreamResult{
		Final: true,
		Response: &sdkmodel.Response{
			Message:    sdkmodel.Message{Role: "assistant", Content: "ok"},
			StopReason: "end_turn",
		},
	})
}

func TestSDKRunnerRunLoadsAndSavesHistory(t *testing.T) {
	store := &memoryHistoryStore{
		messages: []sessionstore.Message{{Role: "user", Content: "previous"}},
	}
	history := NewHistoryAdapter(store)
	factory := NewRuntimeFactory(history, staticModel{})
	runner := NewSDKRunner(factory, history)

	resp, err := runner.Run(context.Background(), RunRequest{
		SessionID: "sess_1",
		Prompt:    "next",
		Agent: map[string]any{
			"enabled_builtin_tools": []any{},
		},
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if resp.Output != "ok" {
		t.Fatalf("output = %q, want ok", resp.Output)
	}
	if len(store.messages) != 3 {
		t.Fatalf("saved messages = %+v", store.messages)
	}
	if store.messages[0].Content != "previous" || store.messages[1].Content != "next" || store.messages[2].Content != "ok" {
		t.Fatalf("unexpected saved messages = %+v", store.messages)
	}
}
