package repository

import (
	"context"
	"errors"

	"icoo_claw/server/gateway/internal/model"

	"gorm.io/gorm"
)

type AgentInstanceRepository interface {
	Create(ctx context.Context, instance model.AgentInstance) error
	Get(ctx context.Context, id string) (*model.AgentInstance, error)
	List(ctx context.Context) ([]model.AgentInstance, error)
	Update(ctx context.Context, instance model.AgentInstance) error
}

type GormAgentInstanceRepository struct {
	db *gorm.DB
}

func NewGormAgentInstanceRepository(db *gorm.DB) *GormAgentInstanceRepository {
	return &GormAgentInstanceRepository{db: db}
}

func (r *GormAgentInstanceRepository) Create(ctx context.Context, instance model.AgentInstance) error {
	return r.db.WithContext(ctx).Create(&instance).Error
}

func (r *GormAgentInstanceRepository) Get(ctx context.Context, id string) (*model.AgentInstance, error) {
	var instance model.AgentInstance
	err := r.db.WithContext(ctx).First(&instance, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

func (r *GormAgentInstanceRepository) List(ctx context.Context) ([]model.AgentInstance, error) {
	var instances []model.AgentInstance
	err := r.db.WithContext(ctx).Order("created_at desc").Find(&instances).Error
	return instances, err
}

func (r *GormAgentInstanceRepository) Update(ctx context.Context, instance model.AgentInstance) error {
	result := r.db.WithContext(ctx).Save(&instance)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
