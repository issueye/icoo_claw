package model

import "time"

type ProviderProfile struct {
	ID           string `gorm:"primaryKey;size:64"`
	Name         string `gorm:"size:128;not null"`
	Type         string `gorm:"size:32;not null;index:idx_provider_type_enabled"`
	BaseURL      string `gorm:"size:512"`
	APIKey       string `gorm:"type:text"`
	DefaultModel string `gorm:"size:128"`
	Enabled      bool   `gorm:"index:idx_provider_type_enabled"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
