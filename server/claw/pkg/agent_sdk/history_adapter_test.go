package agent_sdk

import (
	"context"
	"errors"
	"testing"

	"icoo_claw/server/claw/pkg/sessionstore"
)

type missingHistoryStore struct{}

func (s missingHistoryStore) ListMessages(context.Context, string) ([]sessionstore.Message, error) {
	return nil, errors.New(`session api GET /v1/sessions/sess_missing/messages?limit=0: status 404: {"code":"not_found","error":"session not found"}`)
}

func (s missingHistoryStore) ReplaceMessages(context.Context, string, []sessionstore.Message) error {
	return errors.New(`session api PUT /v1/sessions/sess_missing/messages/snapshot: status 404: {"code":"not_found","error":"session not found"}`)
}

func TestHistoryAdapterTreatsMissingSessionAsEmptyHistory(t *testing.T) {
	adapter := NewHistoryAdapter(missingHistoryStore{})

	messages, err := adapter.Load(context.Background(), "sess_missing")
	if err != nil {
		t.Fatalf("Load() error = %v, want nil for missing session", err)
	}
	if len(messages) != 0 {
		t.Fatalf("messages len = %d, want empty history", len(messages))
	}
}

func TestHistoryAdapterIgnoresMissingSessionWhenSavingSnapshot(t *testing.T) {
	adapter := NewHistoryAdapter(missingHistoryStore{})

	if err := adapter.SaveSnapshot(context.Background(), "sess_missing", nil); err != nil {
		t.Fatalf("SaveSnapshot() error = %v, want nil for missing session", err)
	}
}
