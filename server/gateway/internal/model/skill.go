package model

import "time"

type SkillProfile struct {
	ID           string `gorm:"primaryKey;size:64"`
	Name         string `gorm:"size:128;not null;uniqueIndex"`
	Description  string `gorm:"type:text;not null"`
	Path         string `gorm:"size:512;not null"`
	Content      string `gorm:"type:text"`
	Version      string `gorm:"size:64;not null;index"`
	Status       string `gorm:"size:32;not null;index"`
	Source           string `gorm:"size:64"`
	AllowedToolsJSON string `gorm:"column:allowed_tools;type:text"`
	MetadataJSON     string `gorm:"column:metadata;type:text"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
