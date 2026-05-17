package model

import "time"

type AgentInstance struct {
	ID              string `gorm:"primaryKey;size:64"`
	AgentID         string `gorm:"index;size:64;not null"`
	Name            string `gorm:"size:128"`
	Status          string `gorm:"index;size:32;not null"`
	PID             int
	Host            string `gorm:"size:128"`
	Port            int
	BaseURL         string `gorm:"size:512"`
	LastHeartbeatAt *time.Time
	LastError       string `gorm:"type:text"`
	Inflight        int
	MetadataJSON    string `gorm:"column:metadata;type:text"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
