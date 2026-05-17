package repository

import (
	"context"
	"testing"
	"time"

	"icoo_claw/server/session_store/internal/model"

	"github.com/nalgeon/redka"
	_ "modernc.org/sqlite"
)

func TestRedkaSessionRepositoryMessages(t *testing.T) {
	db, err := redka.Open("file:/redka.db?vfs=memdb", &redka.Options{DriverName: "sqlite"})
	if err != nil {
		t.Fatalf("open redka: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	repo := NewRedkaSessionRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()
	if err := repo.Create(ctx, model.Session{
		SessionID: "sess_1",
		UserID:    "user_1",
		AgentID:   "agent_1",
		Status:    "active",
		Metadata:  map[string]any{"source": "test"},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := repo.AppendMessages(ctx, "sess_1", []model.Message{
		{ID: "msg_1", Role: "user", Content: "hello", CreatedAt: now},
		{ID: "msg_2", Role: "assistant", Content: "hi", CreatedAt: now},
	}); err != nil {
		t.Fatalf("append messages: %v", err)
	}

	messages, err := repo.ListMessages(ctx, "sess_1", model.MessagePage{Limit: 0})
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(messages) != 2 || messages[0].Content != "hello" || messages[1].Content != "hi" {
		t.Fatalf("messages = %+v", messages)
	}

	if err := repo.ReplaceMessages(ctx, "sess_1", []model.Message{
		{ID: "msg_3", Role: "assistant", Content: "snapshot", CreatedAt: now},
	}); err != nil {
		t.Fatalf("replace messages: %v", err)
	}

	messages, err = repo.ListMessages(ctx, "sess_1", model.MessagePage{Limit: 0})
	if err != nil {
		t.Fatalf("list snapshot: %v", err)
	}
	if len(messages) != 1 || messages[0].Content != "snapshot" {
		t.Fatalf("snapshot messages = %+v", messages)
	}
}
