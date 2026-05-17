package repository

import (
	"context"
	"errors"

	"icoo_claw/server/gateway/internal/model"

	"gorm.io/gorm"
)

type ConversationRepository interface {
	Create(ctx context.Context, conversation model.Conversation) error
	Get(ctx context.Context, id string) (*model.Conversation, error)
	List(ctx context.Context) ([]model.Conversation, error)
	Update(ctx context.Context, conversation model.Conversation) error
	Delete(ctx context.Context, id string) error
}

type GormConversationRepository struct {
	db *gorm.DB
}

func NewGormConversationRepository(db *gorm.DB) *GormConversationRepository {
	return &GormConversationRepository{db: db}
}

func (r *GormConversationRepository) Create(ctx context.Context, conversation model.Conversation) error {
	return r.db.WithContext(ctx).Create(&conversation).Error
}

func (r *GormConversationRepository) Get(ctx context.Context, id string) (*model.Conversation, error) {
	var conversation model.Conversation
	err := r.db.WithContext(ctx).First(&conversation, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &conversation, nil
}

func (r *GormConversationRepository) List(ctx context.Context) ([]model.Conversation, error) {
	var conversations []model.Conversation
	err := r.db.WithContext(ctx).Order("updated_at desc").Find(&conversations).Error
	return conversations, err
}

func (r *GormConversationRepository) Update(ctx context.Context, conversation model.Conversation) error {
	result := r.db.WithContext(ctx).Save(&conversation)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *GormConversationRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&model.Conversation{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
