package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"icoo_claw/server/session_store/internal/model"
	"icoo_claw/server/session_store/internal/repository"
)

type SessionService struct {
	repo repository.SessionRepository
}

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

func (s *SessionService) ListMessages(ctx context.Context, sessionID string, page model.MessagePage) ([]model.Message, error) {
	return s.repo.ListMessages(ctx, sessionID, page)
}

func (s *SessionService) AppendMessages(ctx context.Context, sessionID string, messages []model.Message) error {
	return s.repo.AppendMessages(ctx, sessionID, normalizeMessages(messages))
}

func (s *SessionService) ReplaceMessages(ctx context.Context, sessionID string, messages []model.Message) error {
	return s.repo.ReplaceMessages(ctx, sessionID, normalizeMessages(messages))
}

func normalizeMessages(messages []model.Message) []model.Message {
	now := time.Now().UTC()
	out := make([]model.Message, len(messages))
	for i, msg := range messages {
		if strings.TrimSpace(msg.ID) == "" {
			msg.ID = "msg_" + randomID()
		}
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

func randomID() string {
	var buf [12]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format("20060102150405.000000000")))
	}
	return hex.EncodeToString(buf[:])
}
