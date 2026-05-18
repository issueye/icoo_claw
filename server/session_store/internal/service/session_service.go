package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"icoo_claw/server/session_store/internal/model"
	"icoo_claw/server/session_store/internal/repository"
)

type SessionService struct {
	repo repository.SessionRepository
}

var ErrInvalidInput = errors.New("invalid session input")

func NewSessionService(repo repository.SessionRepository) *SessionService {
	return &SessionService{repo: repo}
}

func (s *SessionService) Create(ctx context.Context, session model.Session) (*model.Session, error) {
	now := time.Now().UTC()
	session.SessionID = strings.TrimSpace(session.SessionID)
	if session.SessionID == "" {
		session.SessionID = "sess_" + randomID()
	}
	if session.Status == "" {
		session.Status = "active"
	}
	session.CreatedAt = now
	session.UpdatedAt = now
	if session.Metadata == nil {
		session.Metadata = map[string]any{}
	}

	if err := s.repo.Create(ctx, session); err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *SessionService) Get(ctx context.Context, sessionID string) (*model.Session, error) {
	return s.repo.Get(ctx, sessionID)
}

func (s *SessionService) List(ctx context.Context, filter model.SessionListFilter) (*model.SessionList, error) {
	return s.repo.List(ctx, filter)
}

func (s *SessionService) Update(ctx context.Context, session model.Session) (*model.Session, error) {
	current, err := s.repo.Get(ctx, session.SessionID)
	if err != nil {
		return nil, err
	}
	if session.Title != "" {
		current.Title = session.Title
	}
	if session.Status != "" {
		current.Status = session.Status
	}
	if session.Metadata != nil {
		current.Metadata = session.Metadata
	}
	current.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, *current); err != nil {
		return nil, err
	}
	return current, nil
}

func (s *SessionService) Delete(ctx context.Context, sessionID string) error {
	return s.repo.Delete(ctx, sessionID)
}

func (s *SessionService) ListMessages(ctx context.Context, sessionID string, page model.MessagePage) (*model.MessageList, error) {
	return s.repo.ListMessages(ctx, sessionID, page)
}

func (s *SessionService) AppendMessages(ctx context.Context, sessionID string, messages []model.Message) error {
	if err := validateMessages(messages); err != nil {
		return err
	}
	return s.repo.AppendMessages(ctx, sessionID, normalizeMessages(messages))
}

func (s *SessionService) ReplaceMessages(ctx context.Context, sessionID string, messages []model.Message, expectedRevision *int64) error {
	if err := validateMessages(messages); err != nil {
		return err
	}
	return s.repo.ReplaceMessages(ctx, sessionID, normalizeMessages(messages), expectedRevision)
}

func (s *SessionService) ListRuns(ctx context.Context, sessionID string, page model.RunPage) ([]model.Run, error) {
	return s.repo.ListRuns(ctx, sessionID, page)
}

func (s *SessionService) AppendRuns(ctx context.Context, sessionID string, runs []model.Run) error {
	if len(runs) == 0 {
		return ErrInvalidInput
	}
	return s.repo.AppendRuns(ctx, sessionID, normalizeRuns(runs))
}

func (s *SessionService) ListRunEvents(ctx context.Context, sessionID string, runID string, page model.RunEventPage) ([]model.RunEvent, error) {
	return s.repo.ListRunEvents(ctx, sessionID, runID, page)
}

func (s *SessionService) AppendRunEvents(ctx context.Context, sessionID string, runID string, events []model.RunEvent) error {
	if strings.TrimSpace(runID) == "" || len(events) == 0 {
		return ErrInvalidInput
	}
	for _, event := range events {
		if strings.TrimSpace(event.Type) == "" {
			return ErrInvalidInput
		}
		if event.RunID != "" && event.RunID != runID {
			return ErrInvalidInput
		}
	}
	return s.repo.AppendRunEvents(ctx, sessionID, runID, normalizeRunEvents(runID, events))
}

func validateMessages(messages []model.Message) error {
	if len(messages) == 0 {
		return ErrInvalidInput
	}
	for _, msg := range messages {
		switch strings.TrimSpace(msg.Role) {
		case "system", "user", "assistant", "tool":
		default:
			return ErrInvalidInput
		}
		if strings.TrimSpace(msg.Content) == "" && len(msg.ContentBlocks) == 0 && len(msg.ToolCalls) == 0 {
			return ErrInvalidInput
		}
	}
	return nil
}

func normalizeMessages(messages []model.Message) []model.Message {
	now := time.Now().UTC()
	out := make([]model.Message, len(messages))
	for i, msg := range messages {
		if strings.TrimSpace(msg.ID) == "" {
			msg.ID = "msg_" + randomID()
		}
		msg.Role = strings.TrimSpace(msg.Role)
		if msg.CreatedAt.IsZero() {
			msg.CreatedAt = now
		}
		if msg.Metadata == nil {
			msg.Metadata = map[string]any{}
		}
		out[i] = msg
	}
	return out
}

func normalizeRuns(runs []model.Run) []model.Run {
	now := time.Now().UTC()
	out := make([]model.Run, len(runs))
	for i, run := range runs {
		if strings.TrimSpace(run.ID) == "" {
			run.ID = "run_" + randomID()
		}
		if run.Status == "" {
			run.Status = "completed"
		}
		if run.StartedAt.IsZero() {
			run.StartedAt = now
		}
		if run.Metadata == nil {
			run.Metadata = map[string]any{}
		}
		if run.Usage == nil {
			run.Usage = map[string]any{}
		}
		out[i] = run
	}
	return out
}

func normalizeRunEvents(runID string, events []model.RunEvent) []model.RunEvent {
	now := time.Now().UTC()
	out := make([]model.RunEvent, len(events))
	for i, event := range events {
		if strings.TrimSpace(event.ID) == "" {
			event.ID = "evt_" + randomID()
		}
		event.RunID = strings.TrimSpace(event.RunID)
		if event.RunID == "" {
			event.RunID = runID
		}
		event.Type = strings.TrimSpace(event.Type)
		if event.CreatedAt.IsZero() {
			event.CreatedAt = now
		}
		if event.Payload == nil {
			event.Payload = map[string]any{}
		}
		if event.Metadata == nil {
			event.Metadata = map[string]any{}
		}
		out[i] = event
	}
	return out
}

func randomID() string {
	var buf [12]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format("20060102150405.000000000")))
	}
	return hex.EncodeToString(buf[:])
}
