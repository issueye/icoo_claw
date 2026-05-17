package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"icoo_claw/server/session_store/internal/model"

	"github.com/nalgeon/redka"
)

var ErrNotFound = errors.New("session not found")

type SessionRepository interface {
	Create(ctx context.Context, session model.Session) error
	Get(ctx context.Context, sessionID string) (*model.Session, error)
	Update(ctx context.Context, session model.Session) error
	Delete(ctx context.Context, sessionID string) error
	ListMessages(ctx context.Context, sessionID string, page model.MessagePage) ([]model.Message, error)
	AppendMessages(ctx context.Context, sessionID string, messages []model.Message) error
	ReplaceMessages(ctx context.Context, sessionID string, messages []model.Message) error
}

type RedkaSessionRepository struct {
	db *redka.DB
}

func NewRedkaSessionRepository(db *redka.DB) *RedkaSessionRepository {
	return &RedkaSessionRepository{db: db}
}

func (r *RedkaSessionRepository) Create(ctx context.Context, session model.Session) error {
	return r.db.UpdateContext(ctx, func(tx *redka.Tx) error {
		return r.saveSession(tx, session)
	})
}

func (r *RedkaSessionRepository) Get(ctx context.Context, sessionID string) (*model.Session, error) {
	var session *model.Session
	err := r.db.ViewContext(ctx, func(tx *redka.Tx) error {
		exists, err := tx.Key().Exists(sessionMetaKey(sessionID))
		if err != nil {
			return err
		}
		if !exists {
			return ErrNotFound
		}

		items, err := tx.Hash().Items(sessionMetaKey(sessionID))
		if err != nil {
			return err
		}
		parsed, err := parseSession(items)
		if err != nil {
			return err
		}
		session = parsed
		return nil
	})
	if err != nil {
		return nil, err
	}
	return session, nil
}

func (r *RedkaSessionRepository) Update(ctx context.Context, session model.Session) error {
	return r.db.UpdateContext(ctx, func(tx *redka.Tx) error {
		exists, err := tx.Key().Exists(sessionMetaKey(session.SessionID))
		if err != nil {
			return err
		}
		if !exists {
			return ErrNotFound
		}
		return r.saveSession(tx, session)
	})
}

func (r *RedkaSessionRepository) Delete(ctx context.Context, sessionID string) error {
	return r.db.UpdateContext(ctx, func(tx *redka.Tx) error {
		_, err := tx.Key().Delete(
			sessionMetaKey(sessionID),
			sessionMessagesKey(sessionID),
			sessionRunsKey(sessionID),
		)
		return err
	})
}

func (r *RedkaSessionRepository) ListMessages(ctx context.Context, sessionID string, page model.MessagePage) ([]model.Message, error) {
	var messages []model.Message
	err := r.db.ViewContext(ctx, func(tx *redka.Tx) error {
		exists, err := tx.Key().Exists(sessionMetaKey(sessionID))
		if err != nil {
			return err
		}
		if !exists {
			return ErrNotFound
		}

		start, stop := pageBounds(page)
		values, err := tx.List().Range(sessionMessagesKey(sessionID), start, stop)
		if err != nil {
			return err
		}

		messages = make([]model.Message, 0, len(values))
		for _, value := range values {
			var msg model.Message
			if err := json.Unmarshal(value.Bytes(), &msg); err != nil {
				return fmt.Errorf("decode message: %w", err)
			}
			messages = append(messages, msg)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return messages, nil
}

func (r *RedkaSessionRepository) AppendMessages(ctx context.Context, sessionID string, messages []model.Message) error {
	return r.db.UpdateContext(ctx, func(tx *redka.Tx) error {
		exists, err := tx.Key().Exists(sessionMetaKey(sessionID))
		if err != nil {
			return err
		}
		if !exists {
			return ErrNotFound
		}
		for _, msg := range messages {
			payload, err := json.Marshal(msg)
			if err != nil {
				return fmt.Errorf("encode message: %w", err)
			}
			if _, err := tx.List().PushBack(sessionMessagesKey(sessionID), payload); err != nil {
				return err
			}
		}
		return touchSession(tx, sessionID, time.Now().UTC())
	})
}

func (r *RedkaSessionRepository) ReplaceMessages(ctx context.Context, sessionID string, messages []model.Message) error {
	return r.db.UpdateContext(ctx, func(tx *redka.Tx) error {
		exists, err := tx.Key().Exists(sessionMetaKey(sessionID))
		if err != nil {
			return err
		}
		if !exists {
			return ErrNotFound
		}
		if _, err := tx.Key().Delete(sessionMessagesKey(sessionID)); err != nil {
			return err
		}
		for _, msg := range messages {
			payload, err := json.Marshal(msg)
			if err != nil {
				return fmt.Errorf("encode message: %w", err)
			}
			if _, err := tx.List().PushBack(sessionMessagesKey(sessionID), payload); err != nil {
				return err
			}
		}
		return touchSession(tx, sessionID, time.Now().UTC())
	})
}

func (r *RedkaSessionRepository) saveSession(tx *redka.Tx, session model.Session) error {
	metadata, err := json.Marshal(session.Metadata)
	if err != nil {
		return fmt.Errorf("encode session metadata: %w", err)
	}

	_, err = tx.Hash().SetMany(sessionMetaKey(session.SessionID), map[string]any{
		"session_id": session.SessionID,
		"user_id":    session.UserID,
		"agent_id":   session.AgentID,
		"title":      session.Title,
		"status":     session.Status,
		"metadata":   string(metadata),
		"created_at": session.CreatedAt.Format(time.RFC3339Nano),
		"updated_at": session.UpdatedAt.Format(time.RFC3339Nano),
	})
	if err != nil {
		return err
	}

	score := float64(session.UpdatedAt.UnixMilli())
	if session.UserID != "" {
		if _, err := tx.ZSet().Add(userSessionsKey(session.UserID), session.SessionID, score); err != nil {
			return err
		}
	}
	if session.AgentID != "" {
		if _, err := tx.ZSet().Add(agentSessionsKey(session.AgentID), session.SessionID, score); err != nil {
			return err
		}
	}
	return nil
}

func touchSession(tx *redka.Tx, sessionID string, now time.Time) error {
	_, err := tx.Hash().Set(sessionMetaKey(sessionID), "updated_at", now.Format(time.RFC3339Nano))
	return err
}

func parseSession(items map[string]redka.Value) (*model.Session, error) {
	createdAt, err := parseTime(items["created_at"].String())
	if err != nil {
		return nil, err
	}
	updatedAt, err := parseTime(items["updated_at"].String())
	if err != nil {
		return nil, err
	}

	metadata := map[string]any{}
	if raw := items["metadata"].String(); raw != "" {
		if err := json.Unmarshal([]byte(raw), &metadata); err != nil {
			return nil, fmt.Errorf("decode session metadata: %w", err)
		}
	}

	return &model.Session{
		SessionID: items["session_id"].String(),
		UserID:    items["user_id"].String(),
		AgentID:   items["agent_id"].String(),
		Title:     items["title"].String(),
		Status:    items["status"].String(),
		Metadata:  metadata,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

func parseTime(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse time %q: %w", value, err)
	}
	return parsed, nil
}

func pageBounds(page model.MessagePage) (int, int) {
	if page.Offset < 0 {
		page.Offset = 0
	}
	if page.Limit <= 0 {
		return page.Offset, -1
	}
	return page.Offset, page.Offset + page.Limit - 1
}
