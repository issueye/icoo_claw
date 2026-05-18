package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"icoo_claw/server/session_store/internal/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestGormSessionRepositoryMessages(t *testing.T) {
	repo := newTestGormRepository(t)
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
		{ID: "msg_3", Role: "user", Content: "again", CreatedAt: now},
	}); err != nil {
		t.Fatalf("append messages: %v", err)
	}

	list, err := repo.ListMessages(ctx, "sess_1", model.MessagePage{Limit: 0})
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	messages := list.Messages
	if len(messages) != 3 || messages[0].Content != "hello" || messages[1].Content != "hi" {
		t.Fatalf("messages = %+v", messages)
	}
	if list.Revision != 1 {
		t.Fatalf("revision = %d, want 1", list.Revision)
	}
	tail, err := repo.ListMessages(ctx, "sess_1", model.MessagePage{Tail: 2})
	if err != nil {
		t.Fatalf("tail messages: %v", err)
	}
	if len(tail.Messages) != 2 || tail.Messages[0].ID != "msg_2" || tail.Messages[1].ID != "msg_3" {
		t.Fatalf("tail = %+v", tail.Messages)
	}
	before, err := repo.ListMessages(ctx, "sess_1", model.MessagePage{BeforeID: "msg_3", Limit: 1})
	if err != nil {
		t.Fatalf("before messages: %v", err)
	}
	if len(before.Messages) != 1 || before.Messages[0].ID != "msg_2" {
		t.Fatalf("before = %+v", before.Messages)
	}
	after, err := repo.ListMessages(ctx, "sess_1", model.MessagePage{AfterID: "msg_1", Limit: 1})
	if err != nil {
		t.Fatalf("after messages: %v", err)
	}
	if len(after.Messages) != 1 || after.Messages[0].ID != "msg_2" {
		t.Fatalf("after = %+v", after.Messages)
	}

	expectedRevision := list.Revision
	if err := repo.ReplaceMessages(ctx, "sess_1", []model.Message{
		{ID: "msg_4", Role: "assistant", Content: "snapshot", CreatedAt: now},
	}, &expectedRevision); err != nil {
		t.Fatalf("replace messages: %v", err)
	}

	list, err = repo.ListMessages(ctx, "sess_1", model.MessagePage{Limit: 0})
	if err != nil {
		t.Fatalf("list snapshot: %v", err)
	}
	messages = list.Messages
	if len(messages) != 1 || messages[0].Content != "snapshot" {
		t.Fatalf("snapshot messages = %+v", messages)
	}
	if list.Revision != 2 {
		t.Fatalf("snapshot revision = %d, want 2", list.Revision)
	}
	if err := repo.ReplaceMessages(ctx, "sess_1", []model.Message{
		{ID: "msg_5", Role: "user", Content: "stale", CreatedAt: now},
	}, &expectedRevision); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale replace err = %v, want ErrConflict", err)
	}
}

func TestGormSessionRepositoryListSessions(t *testing.T) {
	repo := newTestGormRepository(t)
	ctx := context.Background()
	now := time.Now().UTC()
	for _, session := range []model.Session{
		{SessionID: "sess_old", UserID: "user_1", AgentID: "agent_1", Status: "active", Metadata: map[string]any{}, CreatedAt: now, UpdatedAt: now},
		{SessionID: "sess_new", UserID: "user_1", AgentID: "agent_1", Status: "archived", Metadata: map[string]any{}, CreatedAt: now, UpdatedAt: now.Add(time.Second)},
		{SessionID: "sess_other", UserID: "user_2", AgentID: "agent_1", Status: "active", Metadata: map[string]any{}, CreatedAt: now, UpdatedAt: now.Add(2 * time.Second)},
	} {
		if err := repo.Create(ctx, session); err != nil {
			t.Fatalf("create %s: %v", session.SessionID, err)
		}
	}

	list, err := repo.List(ctx, model.SessionListFilter{UserID: "user_1", Limit: 10})
	if err != nil {
		t.Fatalf("list by user: %v", err)
	}
	if len(list.Sessions) != 2 || list.Sessions[0].SessionID != "sess_new" || list.Sessions[1].SessionID != "sess_old" {
		t.Fatalf("user sessions = %+v", list.Sessions)
	}
	list, err = repo.List(ctx, model.SessionListFilter{Status: "active", Limit: 10})
	if err != nil {
		t.Fatalf("list active: %v", err)
	}
	if len(list.Sessions) != 2 {
		t.Fatalf("active sessions = %+v", list.Sessions)
	}
}

func TestGormSessionRepositoryRuns(t *testing.T) {
	repo := newTestGormRepository(t)
	ctx := context.Background()
	now := time.Now().UTC()
	completedAt := now.Add(time.Second)
	if err := repo.Create(ctx, model.Session{
		SessionID: "sess:runs",
		UserID:    "user:1",
		AgentID:   "agent:1",
		Status:    "active",
		Metadata:  map[string]any{},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := repo.AppendRuns(ctx, "sess:runs", []model.Run{
		{
			ID:          "run_1",
			RequestID:   "req_1",
			Status:      "completed",
			StopReason:  "end_turn",
			Usage:       map[string]any{"tokens": float64(12)},
			StartedAt:   now,
			CompletedAt: &completedAt,
		},
		{ID: "run_2", RequestID: "req_2", Status: "failed", Error: "boom", StartedAt: now},
	}); err != nil {
		t.Fatalf("append runs: %v", err)
	}
	if err := repo.AppendRunEvents(ctx, "sess:runs", "run_1", []model.RunEvent{
		{ID: "evt_1", RunID: "run_1", Type: "delta", Sequence: 1, Payload: map[string]any{"text": "hel"}, CreatedAt: now},
		{ID: "evt_2", RunID: "run_1", Type: "delta", Sequence: 2, Payload: map[string]any{"text": "lo"}, CreatedAt: now},
	}); err != nil {
		t.Fatalf("append run events: %v", err)
	}

	runs, err := repo.ListRuns(ctx, "sess:runs", model.RunPage{Limit: 0})
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if len(runs) != 2 || runs[0].RequestID != "req_1" || runs[1].Error != "boom" {
		t.Fatalf("runs = %+v", runs)
	}
	events, err := repo.ListRunEvents(ctx, "sess:runs", "run_1", model.RunEventPage{Limit: 0})
	if err != nil {
		t.Fatalf("list run events: %v", err)
	}
	if len(events) != 2 || events[0].RunID != "run_1" || events[1].Sequence != 2 {
		t.Fatalf("events = %+v", events)
	}

	session, err := repo.Get(ctx, "sess:runs")
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if !session.UpdatedAt.After(now) {
		t.Fatalf("updated_at was not touched: %s <= %s", session.UpdatedAt, now)
	}
	if err := repo.Delete(ctx, "sess:runs"); err != nil {
		t.Fatalf("delete session: %v", err)
	}
	if events, err := repo.ListRunEvents(ctx, "sess:runs", "run_1", model.RunEventPage{Limit: 0}); !errors.Is(err, ErrNotFound) || len(events) != 0 {
		t.Fatalf("events after delete = %+v, err = %v", events, err)
	}
}

func TestGormSessionRepositoryDeleteMissingIsIdempotent(t *testing.T) {
	repo := newTestGormRepository(t)
	if err := repo.Delete(context.Background(), "missing"); err != nil {
		t.Fatalf("delete missing: %v", err)
	}
}

func newTestGormRepository(t *testing.T) *GormSessionRepository {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("unwrap sqlite: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := AutoMigrate(db); err != nil {
		t.Fatalf("migrate sqlite: %v", err)
	}
	return NewGormSessionRepository(db)
}
