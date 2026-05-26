package repository

import (
	"context"
	"time"

	"icoo_claw/server/gateway/internal/model"

	"gorm.io/gorm"
)

type AgentInstanceRepository interface {
	BaseRepository[model.AgentInstance]
	AdjustInflight(ctx context.Context, id string, delta int) error
}

type GormAgentInstanceRepository struct {
	*GormBaseRepository[model.AgentInstance]
}

func NewGormAgentInstanceRepository(db *gorm.DB) *GormAgentInstanceRepository {
	return &GormAgentInstanceRepository{
		GormBaseRepository: NewGormBaseRepository[model.AgentInstance](db, "created_at desc"),
	}
}

func (r *GormAgentInstanceRepository) AdjustInflight(ctx context.Context, id string, delta int) error {
	result := r.DB.WithContext(ctx).Model(&model.AgentInstance{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"inflight":   gorm.Expr("CASE WHEN inflight + ? < 0 THEN 0 ELSE inflight + ? END", delta, delta),
			"updated_at": time.Now().UTC(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
