package repository

import (
	"context"
	"time"

	"icoo_claw/server/gateway/internal/model"

	"gorm.io/gorm"
)

type ScheduledTaskRepository interface {
	BaseRepository[model.ScheduledTask]
	ListDue(ctx context.Context, now time.Time) ([]model.ScheduledTask, error)
}

type GormScheduledTaskRepository struct {
	*GormBaseRepository[model.ScheduledTask]
}

func NewGormScheduledTaskRepository(db *gorm.DB) *GormScheduledTaskRepository {
	return &GormScheduledTaskRepository{
		GormBaseRepository: NewGormBaseRepository[model.ScheduledTask](db, "created_at desc"),
	}
}

func (r *GormScheduledTaskRepository) ListDue(ctx context.Context, now time.Time) ([]model.ScheduledTask, error) {
	var tasks []model.ScheduledTask
	err := r.DB.WithContext(ctx).
		Where("enabled = ? AND next_run_at IS NOT NULL AND next_run_at <= ?", true, now.UTC()).
		Order("next_run_at asc").
		Find(&tasks).
		Error
	return tasks, err
}
