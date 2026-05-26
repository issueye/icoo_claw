package repository

import (
	"context"
	"errors"
	"strings"

	"icoo_claw/server/gateway/internal/model"

	"gorm.io/gorm"
)

type ProviderRepository interface {
	BaseRepository[model.ProviderProfile]
	GetEnabledByType(ctx context.Context, providerType string) (*model.ProviderProfile, error)
}

type GormProviderRepository struct {
	*GormBaseRepository[model.ProviderProfile]
}

func NewGormProviderRepository(db *gorm.DB) *GormProviderRepository {
	return &GormProviderRepository{
		GormBaseRepository: NewGormBaseRepository[model.ProviderProfile](db, "created_at desc"),
	}
}

func (r *GormProviderRepository) GetEnabledByType(ctx context.Context, providerType string) (*model.ProviderProfile, error) {
	var provider model.ProviderProfile
	err := r.DB.WithContext(ctx).
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
