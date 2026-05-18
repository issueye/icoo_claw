package repository

import (
	"context"
	"errors"

	"icoo_claw/server/session_store/internal/model"
)

var ErrNotFound = errors.New("session not found")
var ErrConflict = errors.New("session revision conflict")

type SessionRepository interface {
	Create(ctx context.Context, session model.Session) error
	Get(ctx context.Context, sessionID string) (*model.Session, error)
	List(ctx context.Context, filter model.SessionListFilter) (*model.SessionList, error)
	Update(ctx context.Context, session model.Session) error
	Delete(ctx context.Context, sessionID string) error
	ListMessages(ctx context.Context, sessionID string, page model.MessagePage) (*model.MessageList, error)
	AppendMessages(ctx context.Context, sessionID string, messages []model.Message) error
	ReplaceMessages(ctx context.Context, sessionID string, messages []model.Message, expectedRevision *int64) error
	ListRuns(ctx context.Context, sessionID string, page model.RunPage) ([]model.Run, error)
	AppendRuns(ctx context.Context, sessionID string, runs []model.Run) error
	ListRunEvents(ctx context.Context, sessionID string, runID string, page model.RunEventPage) ([]model.RunEvent, error)
	AppendRunEvents(ctx context.Context, sessionID string, runID string, events []model.RunEvent) error
}

func listBounds(offset, limit int) (int, int) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return offset, limit
}

func pageLimit(limit int) int {
	if limit < 0 {
		return 0
	}
	return limit
}

func pageOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}
