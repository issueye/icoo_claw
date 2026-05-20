package service

import (
	"context"

	"icoo_claw/server/gateway/internal/dto"
	sessionmodel "icoo_claw/server/gateway/internal/sessionstore/model"
	sessionservice "icoo_claw/server/gateway/internal/sessionstore/service"
)

type LocalSessionBackend struct {
	sessions *sessionservice.SessionService
}

func NewLocalSessionBackend(sessions *sessionservice.SessionService) *LocalSessionBackend {
	return &LocalSessionBackend{sessions: sessions}
}

func (s *LocalSessionBackend) CreateSession(ctx context.Context, req SessionCreateRequest) error {
	_, err := s.sessions.Create(ctx, sessionmodel.Session{
		SessionID: req.SessionID,
		UserID:    req.UserID,
		AgentID:   req.AgentID,
		Title:     req.Title,
		Metadata:  req.Metadata,
	})
	return err
}

func (s *LocalSessionBackend) ListMessages(ctx context.Context, sessionID string) ([]dto.SessionMessage, error) {
	list, err := s.sessions.ListMessages(ctx, sessionID, sessionmodel.MessagePage{Limit: 0})
	if err != nil {
		return nil, err
	}
	out := make([]dto.SessionMessage, len(list.Messages))
	for i, msg := range list.Messages {
		out[i] = dto.SessionMessage{
			ID:        msg.ID,
			Role:      msg.Role,
			Content:   msg.Content,
			Metadata:  msg.Metadata,
			CreatedAt: msg.CreatedAt,
		}
	}
	return out, nil
}
