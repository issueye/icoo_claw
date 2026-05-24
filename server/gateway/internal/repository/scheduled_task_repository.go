package repository

import (
	"context"
	"errors"
	"time"

	"icoo_claw/server/gateway/internal/model"

	"gorm.io/gorm"
)

type ScheduledTaskRepository interface {
	Create(ctx context.Context, task model.ScheduledTask) error
	Get(ctx context.Context, id string) (*model.ScheduledTask, error)
	List(ctx context.Context) ([]model.ScheduledTask, error)
	ListDue(ctx context.Context, now time.Time) ([]model.ScheduledTask, error)
	Update(ctx context.Context, task model.ScheduledTask) error
	Delete(ctx context.Context, id string) error
}

type GormScheduledTaskRepository struct {
	db *gorm.DB
}

func NewGormScheduledTaskRepository(db *gorm.DB) *GormScheduledTaskRepository {
	return &GormScheduledTaskRepository{db: db}
}

func (r *GormScheduledTaskRepository) Create(ctx context.Context, task model.ScheduledTask) error {
	return r.db.WithContext(ctx).Create(&task).Error
}

func (r *GormScheduledTaskRepository) Get(ctx context.Context, id string) (*model.ScheduledTask, error) {
	var task model.ScheduledTask
	err := r.db.WithContext(ctx).First(&task, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *GormScheduledTaskRepository) List(ctx context.Context) ([]model.ScheduledTask, error) {
	var tasks []model.ScheduledTask
	err := r.db.WithContext(ctx).Order("created_at desc").Find(&tasks).Error
	return tasks, err
}

func (r *GormScheduledTaskRepository) ListDue(ctx context.Context, now time.Time) ([]model.ScheduledTask, error) {
	var tasks []model.ScheduledTask
	err := r.db.WithContext(ctx).
		Where("enabled = ? AND next_run_at IS NOT NULL AND next_run_at <= ?", true, now.UTC()).
		Order("next_run_at asc").
		Find(&tasks).
		Error
	return tasks, err
}

func (r *GormScheduledTaskRepository) Update(ctx context.Context, task model.ScheduledTask) error {
	result := r.db.WithContext(ctx).Save(&task)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *GormScheduledTaskRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&model.ScheduledTask{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
