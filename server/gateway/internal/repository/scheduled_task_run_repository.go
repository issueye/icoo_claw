package repository

import (
	"context"
	"errors"

	"icoo_claw/server/gateway/internal/model"

	"gorm.io/gorm"
)

type ScheduledTaskRunRepository interface {
	Create(ctx context.Context, run model.ScheduledTaskRun) error
	ListByTaskID(ctx context.Context, taskID string, limit int) ([]model.ScheduledTaskRun, error)
}

type GormScheduledTaskRunRepository struct {
	db *gorm.DB
}

func NewGormScheduledTaskRunRepository(db *gorm.DB) *GormScheduledTaskRunRepository {
	return &GormScheduledTaskRunRepository{db: db}
}

func (r *GormScheduledTaskRunRepository) Create(ctx context.Context, run model.ScheduledTaskRun) error {
	return r.db.WithContext(ctx).Create(&run).Error
}

func (r *GormScheduledTaskRunRepository) ListByTaskID(ctx context.Context, taskID string, limit int) ([]model.ScheduledTaskRun, error) {
	if limit <= 0 {
		limit = 20
	}
	var runs []model.ScheduledTaskRun
	err := r.db.WithContext(ctx).
		Where("task_id = ?", taskID).
		Order("executed_at desc").
		Limit(limit).
		Find(&runs).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return []model.ScheduledTaskRun{}, nil
	}
	return runs, err
}
