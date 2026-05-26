package repository

import (
	"context"
	"errors"
	"strings"

	"icoo_claw/server/gateway/internal/model"

	"gorm.io/gorm"
)

type SkillRepository interface {
	BaseRepository[model.SkillProfile]
	GetByName(ctx context.Context, name string) (*model.SkillProfile, error)
	ListActive(ctx context.Context) ([]model.SkillProfile, error)
}

type GormSkillRepository struct {
	*GormBaseRepository[model.SkillProfile]
}

func NewGormSkillRepository(db *gorm.DB) *GormSkillRepository {
	return &GormSkillRepository{
		GormBaseRepository: NewGormBaseRepository[model.SkillProfile](db, "created_at desc"),
	}
}

func (r *GormSkillRepository) GetByName(ctx context.Context, name string) (*model.SkillProfile, error) {
	var skill model.SkillProfile
	err := r.DB.WithContext(ctx).
		Where("name = ?", strings.TrimSpace(name)).
		First(&skill).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &skill, nil
}

func (r *GormSkillRepository) ListActive(ctx context.Context) ([]model.SkillProfile, error) {
	var skills []model.SkillProfile
	err := r.DB.WithContext(ctx).
		Where("status = ?", "active").
		Order("updated_at desc").
		Find(&skills).
		Error
	return skills, err
}
