package model

import "time"

type ScheduledTaskRun struct {
	ID         string    `gorm:"primaryKey;size:64"`
	TaskID     string    `gorm:"size:64;index:idx_scheduled_task_runs_task_executed,priority:1"`
	AgentID    string    `gorm:"size:64;index"`
	Status     string    `gorm:"size:32;not null;index"`
	Summary    string    `gorm:"type:text"`
	Error      string    `gorm:"type:text"`
	ExecutedAt time.Time `gorm:"index:idx_scheduled_task_runs_task_executed,priority:2"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
