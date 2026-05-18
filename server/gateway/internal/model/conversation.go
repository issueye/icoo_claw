package model

import "time"

type Conversation struct {
	ID               string `gorm:"primaryKey;size:64"`
	SessionID        string `gorm:"uniqueIndex;size:64;not null"`
	AgentID          string `gorm:"index;size:64;not null"`
	StickyInstanceID string `gorm:"index;size:64"`
	UserID           string `gorm:"index;size:64"`
	Title            string `gorm:"size:256"`
	Status           string `gorm:"index;size:32;not null"`
	LastMessageAt    *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
