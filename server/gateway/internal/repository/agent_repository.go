package repository

import (
	"context"
	"errors"

	"icoo_claw/server/gateway/internal/model"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("resource not found")

type AgentRepository interface {
	Create(ctx context.Context, agent model.AgentProfile) error
	Get(ctx context.Context, id string) (*model.AgentProfile, error)
	List(ctx context.Context) ([]model.AgentProfile, error)
	Update(ctx context.Context, agent model.AgentProfile) error
	Delete(ctx context.Context, id string) error
}

type GormAgentRepository struct {
	db *gorm.DB
}

func NewGormAgentRepository(db *gorm.DB) *GormAgentRepository {
	return &GormAgentRepository{db: db}
}

func (r *GormAgentRepository) Create(ctx context.Context, agent model.AgentProfile) error {
	return r.db.WithContext(ctx).Create(&agent).Error
}

func (r *GormAgentRepository) Get(ctx context.Context, id string) (*model.AgentProfile, error) {
	var agent model.AgentProfile
	err := r.db.WithContext(ctx).First(&agent, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &agent, nil
}

func (r *GormAgentRepository) List(ctx context.Context) ([]model.AgentProfile, error) {
	var agents []model.AgentProfile
	err := r.db.WithContext(ctx).Order("created_at desc").Find(&agents).Error
	return agents, err
}

func (r *GormAgentRepository) Update(ctx context.Context, agent model.AgentProfile) error {
	result := r.db.WithContext(ctx).Save(&agent)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *GormAgentRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&model.AgentProfile{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
