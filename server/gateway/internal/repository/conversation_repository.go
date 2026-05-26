package repository

import (
	"icoo_claw/server/gateway/internal/model"

	"gorm.io/gorm"
)

type ConversationRepository interface {
	BaseRepository[model.Conversation]
}

type GormConversationRepository struct {
	*GormBaseRepository[model.Conversation]
}

func NewGormConversationRepository(db *gorm.DB) *GormConversationRepository {
	return &GormConversationRepository{
		GormBaseRepository: NewGormBaseRepository[model.Conversation](db, "updated_at desc"),
	}
}
