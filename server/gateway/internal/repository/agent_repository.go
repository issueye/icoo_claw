package repository

import (
	"icoo_claw/server/gateway/internal/model"

	"gorm.io/gorm"
)

type AgentRepository interface {
	BaseRepository[model.AgentProfile]
}

type GormAgentRepository struct {
	*GormBaseRepository[model.AgentProfile]
}

func NewGormAgentRepository(db *gorm.DB) *GormAgentRepository {
	return &GormAgentRepository{
		GormBaseRepository: NewGormBaseRepository[model.AgentProfile](db, "created_at desc"),
	}
}
