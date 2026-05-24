package repository

import (
	"context"
	"errors"
	"strings"

	"icoo_claw/server/gateway/internal/model"

	"gorm.io/gorm"
)

type ProviderRepository interface {
	Create(ctx context.Context, provider model.ProviderProfile) error
	Get(ctx context.Context, id string) (*model.ProviderProfile, error)
	GetEnabledByType(ctx context.Context, providerType string) (*model.ProviderProfile, error)
	List(ctx context.Context) ([]model.ProviderProfile, error)
	Update(ctx context.Context, provider model.ProviderProfile) error
	Delete(ctx context.Context, id string) error
}

type GormProviderRepository struct {
	db *gorm.DB
}

func NewGormProviderRepository(db *gorm.DB) *GormProviderRepository {
	return &GormProviderRepository{db: db}
}

func (r *GormProviderRepository) Create(ctx context.Context, provider model.ProviderProfile) error {
	return r.db.WithContext(ctx).Create(&provider).Error
}

func (r *GormProviderRepository) Get(ctx context.Context, id string) (*model.ProviderProfile, error) {
	var provider model.ProviderProfile
	err := r.db.WithContext(ctx).First(&provider, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &provider, nil
}

func (r *GormProviderRepository) GetEnabledByType(ctx context.Context, providerType string) (*model.ProviderProfile, error) {
	var provider model.ProviderProfile
	err := r.db.WithContext(ctx).
		Where("type = ? AND enabled = ?", strings.TrimSpace(providerType), true).
		Order("updated_at desc").
		First(&provider).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &provider, nil
}

func (r *GormProviderRepository) List(ctx context.Context) ([]model.ProviderProfile, error) {
	var providers []model.ProviderProfile
	err := r.db.WithContext(ctx).Order("created_at desc").Find(&providers).Error
	return providers, err
}

func (r *GormProviderRepository) Update(ctx context.Context, provider model.ProviderProfile) error {
	result := r.db.WithContext(ctx).Save(&provider)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *GormProviderRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&model.ProviderProfile{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
