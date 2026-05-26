package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("resource not found")

type BaseRepository[T any] interface {
	Create(ctx context.Context, entity T) error
	Get(ctx context.Context, id string) (*T, error)
	List(ctx context.Context) ([]T, error)
	Update(ctx context.Context, entity T) error
	Delete(ctx context.Context, id string) error
}

type GormBaseRepository[T any] struct {
	DB      *gorm.DB
	OrderBy string
}

func NewGormBaseRepository[T any](db *gorm.DB, orderBy string) *GormBaseRepository[T] {
	return &GormBaseRepository[T]{DB: db, OrderBy: orderBy}
}

func (r *GormBaseRepository[T]) Create(ctx context.Context, entity T) error {
	return r.DB.WithContext(ctx).Create(&entity).Error
}

func (r *GormBaseRepository[T]) Get(ctx context.Context, id string) (*T, error) {
	var entity T
	err := r.DB.WithContext(ctx).First(&entity, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

func (r *GormBaseRepository[T]) List(ctx context.Context) ([]T, error) {
	var entities []T
	orderBy := r.OrderBy
	if orderBy == "" {
		orderBy = "created_at desc"
	}
	err := r.DB.WithContext(ctx).Order(orderBy).Find(&entities).Error
	return entities, err
}

func (r *GormBaseRepository[T]) Update(ctx context.Context, entity T) error {
	result := r.DB.WithContext(ctx).Save(&entity)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *GormBaseRepository[T]) Delete(ctx context.Context, id string) error {
	var entity T
	result := r.DB.WithContext(ctx).Delete(&entity, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
