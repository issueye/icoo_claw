package model

import "time"

type ScheduledTask struct {
	ID            string     `gorm:"primaryKey;size:64"`
	Name          string     `gorm:"size:128;not null"`
	Description   string     `gorm:"type:text"`
	AgentID       string     `gorm:"size:64;index"`
	ScheduleType  string     `gorm:"size:32;not null;index:idx_scheduled_tasks_due,priority:1"`
	ScheduleValue string     `gorm:"size:128;not null"`
	ActionType    string     `gorm:"size:64;not null"`
	PayloadJSON   string     `gorm:"type:text"`
	Enabled       bool       `gorm:"index:idx_scheduled_tasks_due,priority:2"`
	Status        string     `gorm:"size:32;not null;index"`
	LastRunAt     *time.Time `gorm:"index"`
	NextRunAt     *time.Time `gorm:"index:idx_scheduled_tasks_due,priority:3"`
	RunCount      int
	LastError     string `gorm:"type:text"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
