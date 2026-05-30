package repository

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"

	sharedErrors "icoo_claw/common/errors"
	"icoo_claw/server/gateway/internal/sessionstore/model"

	"gorm.io/gorm"
)

type GormSessionRepository struct {
	db *gorm.DB
}

func NewGormSessionRepository(db *gorm.DB) *GormSessionRepository {
	return &GormSessionRepository{db: db}
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.SessionRecord{},
		&model.MessageRecord{},
		&model.RunRecord{},
		&model.RunEventRecord{},
	)
}

func (r *GormSessionRepository) Create(ctx context.Context, session model.Session) error {
	record, err := sessionToRecord(session)
	if err != nil {
		return err
	}
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		if isConstraintError(err) {
			return ErrConflict
		}
		return err
	}
	return nil
}

func (r *GormSessionRepository) Get(ctx context.Context, sessionID string) (*model.Session, error) {
	var record model.SessionRecord
	if err := r.db.WithContext(ctx).First(&record, "session_id = ?", sessionID).Error; err != nil {
		return nil, mapGormNotFound(err)
	}
	session, err := recordToSession(record)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *GormSessionRepository) List(ctx context.Context, filter model.SessionListFilter) (*model.SessionList, error) {
	offset, limit := listBounds(filter.Offset, filter.Limit)
	query := r.db.WithContext(ctx).Model(&model.SessionRecord{})
	if filter.UserID != "" {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if filter.AgentID != "" {
		query = query.Where("agent_id = ?", filter.AgentID)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	var records []model.SessionRecord
	if err := query.Order("updated_at DESC").Offset(offset).Limit(limit).Find(&records).Error; err != nil {
		return nil, err
	}
	sessions := make([]model.Session, 0, len(records))
	for _, record := range records {
		session, err := recordToSession(record)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}
	return &model.SessionList{Sessions: sessions}, nil
}

func (r *GormSessionRepository) Update(ctx context.Context, session model.Session) error {
	record, err := sessionToRecord(session)
	if err != nil {
		return err
	}
	result := r.db.WithContext(ctx).Model(&model.SessionRecord{}).
		Where("session_id = ?", session.SessionID).
		Updates(map[string]any{
			"user_id":    record.UserID,
			"agent_id":   record.AgentID,
			"title":      record.Title,
			"status":     record.Status,
			"metadata":   record.MetadataJSON,
			"revision":   record.Revision,
			"created_at": record.CreatedAt,
			"updated_at": record.UpdatedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *GormSessionRepository) Delete(ctx context.Context, sessionID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("session_id = ?", sessionID).Delete(&model.RunEventRecord{}).Error; err != nil {
			return err
		}
		if err := tx.Where("session_id = ?", sessionID).Delete(&model.RunRecord{}).Error; err != nil {
			return err
		}
		if err := tx.Where("session_id = ?", sessionID).Delete(&model.MessageRecord{}).Error; err != nil {
			return err
		}
		if err := tx.Where("session_id = ?", sessionID).Delete(&model.SessionRecord{}).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *GormSessionRepository) ListMessages(ctx context.Context, sessionID string, page model.MessagePage) (*model.MessageList, error) {
	var session model.SessionRecord
	if err := r.db.WithContext(ctx).First(&session, "session_id = ?", sessionID).Error; err != nil {
		return nil, mapGormNotFound(err)
	}

	query := r.db.WithContext(ctx).Where("session_id = ?", sessionID)
	order := "position ASC"
	limit := pageLimit(page.Limit)
	shouldReverse := false

	if page.Tail > 0 {
		query = query.Order("position DESC").Limit(page.Tail)
		shouldReverse = true
	} else if page.BeforeID != "" {
		position, err := r.messagePosition(ctx, sessionID, page.BeforeID)
		if err != nil {
			return nil, err
		}
		query = query.Where("position < ?", position).Order("position DESC")
		if limit > 0 {
			query = query.Limit(limit)
		}
		shouldReverse = true
	} else {
		if page.AfterID != "" {
			position, err := r.messagePosition(ctx, sessionID, page.AfterID)
			if err != nil {
				return nil, err
			}
			query = query.Where("position > ?", position)
		} else {
			query = query.Offset(pageOffset(page.Offset))
		}
		query = query.Order(order)
		if limit > 0 {
			query = query.Limit(limit)
		}
	}

	var records []model.MessageRecord
	if err := query.Find(&records).Error; err != nil {
		return nil, err
	}
	if shouldReverse {
		slices.Reverse(records)
	}
	messages, err := recordsToMessages(records)
	if err != nil {
		return nil, err
	}
	return &model.MessageList{Messages: messages, Revision: session.Revision}, nil
}

func (r *GormSessionRepository) AppendMessages(ctx context.Context, sessionID string, messages []model.Message) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		session, err := lockSession(tx, sessionID)
		if err != nil {
			return err
		}

		nextPosition, err := nextMessagePosition(tx, sessionID)
		if err != nil {
			return err
		}
		records := make([]model.MessageRecord, 0, len(messages))
		for _, message := range messages {
			record, err := messageToRecord(sessionID, nextPosition, message)
			if err != nil {
				return err
			}
			records = append(records, record)
			nextPosition++
		}
		if len(records) > 0 {
			if err := tx.Create(&records).Error; err != nil {
				if isConstraintError(err) {
					return ErrConflict
				}
				return err
			}
		}
		return touchSession(tx, session, time.Now().UTC(), true)
	})
}

func (r *GormSessionRepository) ReplaceMessages(ctx context.Context, sessionID string, messages []model.Message, expectedRevision *int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		session, err := lockSession(tx, sessionID)
		if err != nil {
			return err
		}
		if expectedRevision != nil && *expectedRevision != session.Revision {
			return ErrConflict
		}
		if err := tx.Where("session_id = ?", sessionID).Delete(&model.MessageRecord{}).Error; err != nil {
			return err
		}

		records := make([]model.MessageRecord, 0, len(messages))
		for i, message := range messages {
			record, err := messageToRecord(sessionID, int64(i+1), message)
			if err != nil {
				return err
			}
			records = append(records, record)
		}
		if len(records) > 0 {
			if err := tx.Create(&records).Error; err != nil {
				if isConstraintError(err) {
					return ErrConflict
				}
				return err
			}
		}
		return touchSession(tx, session, time.Now().UTC(), true)
	})
}

func (r *GormSessionRepository) ListRuns(ctx context.Context, sessionID string, page model.RunPage) ([]model.Run, error) {
	if err := ensureSessionExists(r.db.WithContext(ctx), sessionID); err != nil {
		return nil, err
	}
	query := r.db.WithContext(ctx).Where("session_id = ?", sessionID).Order("position ASC").Offset(pageOffset(page.Offset))
	if limit := pageLimit(page.Limit); limit > 0 {
		query = query.Limit(limit)
	}
	var records []model.RunRecord
	if err := query.Find(&records).Error; err != nil {
		return nil, err
	}
	return recordsToRuns(records)
}

func (r *GormSessionRepository) AppendRuns(ctx context.Context, sessionID string, runs []model.Run) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		session, err := lockSession(tx, sessionID)
		if err != nil {
			return err
		}
		nextPosition, err := nextRunPosition(tx, sessionID)
		if err != nil {
			return err
		}
		records := make([]model.RunRecord, 0, len(runs))
		for _, run := range runs {
			record, err := runToRecord(sessionID, nextPosition, run)
			if err != nil {
				return err
			}
			records = append(records, record)
			nextPosition++
		}
		if len(records) > 0 {
			if err := tx.Create(&records).Error; err != nil {
				if isConstraintError(err) {
					return ErrConflict
				}
				return err
			}
		}
		return touchSession(tx, session, time.Now().UTC(), false)
	})
}

func (r *GormSessionRepository) ListRunEvents(ctx context.Context, sessionID string, runID string, page model.RunEventPage) ([]model.RunEvent, error) {
	if err := ensureSessionExists(r.db.WithContext(ctx), sessionID); err != nil {
		return nil, err
	}
	query := r.db.WithContext(ctx).
		Where("session_id = ? AND run_id = ?", sessionID, runID).
		Order("sequence ASC").
		Order("created_at ASC").
		Offset(pageOffset(page.Offset))
	if limit := pageLimit(page.Limit); limit > 0 {
		query = query.Limit(limit)
	}
	var records []model.RunEventRecord
	if err := query.Find(&records).Error; err != nil {
		return nil, err
	}
	return recordsToRunEvents(records)
}

func (r *GormSessionRepository) AppendRunEvents(ctx context.Context, sessionID string, runID string, events []model.RunEvent) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		session, err := lockSession(tx, sessionID)
		if err != nil {
			return err
		}
		records := make([]model.RunEventRecord, 0, len(events))
		for _, event := range events {
			record, err := runEventToRecord(sessionID, runID, event)
			if err != nil {
				return err
			}
			records = append(records, record)
		}
		if len(records) > 0 {
			if err := tx.Create(&records).Error; err != nil {
				if isConstraintError(err) {
					return ErrConflict
				}
				return err
			}
		}
		return touchSession(tx, session, time.Now().UTC(), false)
	})
}

func (r *GormSessionRepository) messagePosition(ctx context.Context, sessionID string, messageID string) (int64, error) {
	var record model.MessageRecord
	if err := r.db.WithContext(ctx).
		Select("position").
		First(&record, "session_id = ? AND id = ?", sessionID, messageID).Error; err != nil {
		return 0, mapGormNotFound(err)
	}
	return record.Position, nil
}

func ensureSessionExists(db *gorm.DB, sessionID string) error {
	var count int64
	if err := db.Model(&model.SessionRecord{}).Where("session_id = ?", sessionID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrNotFound
	}
	return nil
}

func lockSession(tx *gorm.DB, sessionID string) (model.SessionRecord, error) {
	var session model.SessionRecord
	if err := tx.First(&session, "session_id = ?", sessionID).Error; err != nil {
		return model.SessionRecord{}, mapGormNotFound(err)
	}
	return session, nil
}

func touchSession(tx *gorm.DB, session model.SessionRecord, now time.Time, incrementRevision bool) error {
	updates := map[string]any{"updated_at": now}
	if incrementRevision {
		updates["revision"] = session.Revision + 1
	}
	return tx.Model(&model.SessionRecord{}).Where("session_id = ?", session.SessionID).Updates(updates).Error
}

func nextMessagePosition(tx *gorm.DB, sessionID string) (int64, error) {
	var maxPosition *int64
	if err := tx.Model(&model.MessageRecord{}).
		Select("MAX(position)").
		Where("session_id = ?", sessionID).
		Scan(&maxPosition).Error; err != nil {
		return 0, err
	}
	if maxPosition == nil {
		return 1, nil
	}
	return *maxPosition + 1, nil
}

func nextRunPosition(tx *gorm.DB, sessionID string) (int64, error) {
	var maxPosition *int64
	if err := tx.Model(&model.RunRecord{}).
		Select("MAX(position)").
		Where("session_id = ?", sessionID).
		Scan(&maxPosition).Error; err != nil {
		return 0, err
	}
	if maxPosition == nil {
		return 1, nil
	}
	return *maxPosition + 1, nil
}

func sessionToRecord(session model.Session) (model.SessionRecord, error) {
	metadata, err := marshalJSON(session.Metadata, map[string]any{})
	if err != nil {
		return model.SessionRecord{}, sharedErrors.Wrap("encode session metadata", err)
	}
	return model.SessionRecord{
		SessionID:    session.SessionID,
		UserID:       session.UserID,
		AgentID:      session.AgentID,
		Title:        session.Title,
		Status:       session.Status,
		MetadataJSON: metadata,
		Revision:     session.Revision,
		CreatedAt:    session.CreatedAt,
		UpdatedAt:    session.UpdatedAt,
	}, nil
}

func recordToSession(record model.SessionRecord) (model.Session, error) {
	metadata := map[string]any{}
	if err := unmarshalJSON(record.MetadataJSON, &metadata); err != nil {
		return model.Session{}, sharedErrors.Wrap("decode session metadata", err)
	}
	return model.Session{
		SessionID: record.SessionID,
		UserID:    record.UserID,
		AgentID:   record.AgentID,
		Title:     record.Title,
		Status:    record.Status,
		Metadata:  metadata,
		Revision:  record.Revision,
		CreatedAt: record.CreatedAt,
		UpdatedAt: record.UpdatedAt,
	}, nil
}

func messageToRecord(sessionID string, position int64, message model.Message) (model.MessageRecord, error) {
	contentBlocks, err := marshalJSON(message.ContentBlocks, []any{})
	if err != nil {
		return model.MessageRecord{}, sharedErrors.Wrap("encode message content blocks", err)
	}
	toolCalls, err := marshalJSON(message.ToolCalls, []any{})
	if err != nil {
		return model.MessageRecord{}, sharedErrors.Wrap("encode message tool calls", err)
	}
	metadata, err := marshalJSON(message.Metadata, map[string]any{})
	if err != nil {
		return model.MessageRecord{}, sharedErrors.Wrap("encode message metadata", err)
	}
	return model.MessageRecord{
		ID:                message.ID,
		SessionID:         sessionID,
		Position:          position,
		Role:              message.Role,
		Content:           message.Content,
		ContentBlocksJSON: contentBlocks,
		ToolCallsJSON:     toolCalls,
		MetadataJSON:      metadata,
		CreatedAt:         message.CreatedAt,
	}, nil
}

func recordToMessage(record model.MessageRecord) (model.Message, error) {
	contentBlocks := []any{}
	if err := unmarshalJSON(record.ContentBlocksJSON, &contentBlocks); err != nil {
		return model.Message{}, sharedErrors.Wrap("decode message content blocks", err)
	}
	toolCalls := []any{}
	if err := unmarshalJSON(record.ToolCallsJSON, &toolCalls); err != nil {
		return model.Message{}, sharedErrors.Wrap("decode message tool calls", err)
	}
	metadata := map[string]any{}
	if err := unmarshalJSON(record.MetadataJSON, &metadata); err != nil {
		return model.Message{}, sharedErrors.Wrap("decode message metadata", err)
	}
	return model.Message{
		ID:            record.ID,
		Role:          record.Role,
		Content:       record.Content,
		ContentBlocks: contentBlocks,
		ToolCalls:     toolCalls,
		Metadata:      metadata,
		CreatedAt:     record.CreatedAt,
	}, nil
}

func recordsToMessages(records []model.MessageRecord) ([]model.Message, error) {
	messages := make([]model.Message, 0, len(records))
	for _, record := range records {
		message, err := recordToMessage(record)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	return messages, nil
}

func runToRecord(sessionID string, position int64, run model.Run) (model.RunRecord, error) {
	usage, err := marshalJSON(run.Usage, map[string]any{})
	if err != nil {
		return model.RunRecord{}, sharedErrors.Wrap("encode run usage", err)
	}
	metadata, err := marshalJSON(run.Metadata, map[string]any{})
	if err != nil {
		return model.RunRecord{}, sharedErrors.Wrap("encode run metadata", err)
	}
	return model.RunRecord{
		ID:           run.ID,
		SessionID:    sessionID,
		Position:     position,
		RequestID:    run.RequestID,
		Status:       run.Status,
		StopReason:   run.StopReason,
		Error:        run.Error,
		UsageJSON:    usage,
		MetadataJSON: metadata,
		StartedAt:    run.StartedAt,
		CompletedAt:  run.CompletedAt,
	}, nil
}

func recordToRun(record model.RunRecord) (model.Run, error) {
	usage := map[string]any{}
	if err := unmarshalJSON(record.UsageJSON, &usage); err != nil {
		return model.Run{}, sharedErrors.Wrap("decode run usage", err)
	}
	metadata := map[string]any{}
	if err := unmarshalJSON(record.MetadataJSON, &metadata); err != nil {
		return model.Run{}, sharedErrors.Wrap("decode run metadata", err)
	}
	return model.Run{
		ID:          record.ID,
		RequestID:   record.RequestID,
		Status:      record.Status,
		StopReason:  record.StopReason,
		Error:       record.Error,
		Usage:       usage,
		Metadata:    metadata,
		StartedAt:   record.StartedAt,
		CompletedAt: record.CompletedAt,
	}, nil
}

func recordsToRuns(records []model.RunRecord) ([]model.Run, error) {
	runs := make([]model.Run, 0, len(records))
	for _, record := range records {
		run, err := recordToRun(record)
		if err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, nil
}

func runEventToRecord(sessionID string, runID string, event model.RunEvent) (model.RunEventRecord, error) {
	payload, err := marshalJSON(event.Payload, map[string]any{})
	if err != nil {
		return model.RunEventRecord{}, sharedErrors.Wrap("encode run event payload", err)
	}
	metadata, err := marshalJSON(event.Metadata, map[string]any{})
	if err != nil {
		return model.RunEventRecord{}, sharedErrors.Wrap("encode run event metadata", err)
	}
	if event.RunID != "" {
		runID = event.RunID
	}
	return model.RunEventRecord{
		ID:           event.ID,
		SessionID:    sessionID,
		RunID:        runID,
		Type:         event.Type,
		Sequence:     event.Sequence,
		PayloadJSON:  payload,
		MetadataJSON: metadata,
		CreatedAt:    event.CreatedAt,
	}, nil
}

func recordToRunEvent(record model.RunEventRecord) (model.RunEvent, error) {
	payload := map[string]any{}
	if err := unmarshalJSON(record.PayloadJSON, &payload); err != nil {
		return model.RunEvent{}, sharedErrors.Wrap("decode run event payload", err)
	}
	metadata := map[string]any{}
	if err := unmarshalJSON(record.MetadataJSON, &metadata); err != nil {
		return model.RunEvent{}, sharedErrors.Wrap("decode run event metadata", err)
	}
	return model.RunEvent{
		ID:        record.ID,
		RunID:     record.RunID,
		Type:      record.Type,
		Sequence:  record.Sequence,
		Payload:   payload,
		Metadata:  metadata,
		CreatedAt: record.CreatedAt,
	}, nil
}

func recordsToRunEvents(records []model.RunEventRecord) ([]model.RunEvent, error) {
	events := make([]model.RunEvent, 0, len(records))
	for _, record := range records {
		event, err := recordToRunEvent(record)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}

func marshalJSON[T any](value T, fallback T) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		payload, err = json.Marshal(fallback)
	}
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func unmarshalJSON(raw string, target any) error {
	if raw == "" {
		return nil
	}
	return json.Unmarshal([]byte(raw), target)
}

func mapGormNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}

func isConstraintError(err error) bool {
	if err == nil {
		return false
	}
	message := err.Error()
	return errors.Is(err, gorm.ErrDuplicatedKey) ||
		strings.Contains(message, "constraint failed") ||
		strings.Contains(message, "UNIQUE constraint failed") ||
		strings.Contains(message, "duplicate")
}
