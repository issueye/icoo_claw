package model

import "time"

type ScheduledTaskRun struct {
	ID         string    `gorm:"primaryKey;size:64" json:"id"`
	TaskID     string    `gorm:"size:64;index:idx_scheduled_task_runs_task_executed,priority:1" json:"task_id"`
	AgentID    string    `gorm:"size:64;index" json:"agent_id,omitempty"`
	Status     string    `gorm:"size:32;not null;index" json:"status"`
	Summary    string    `gorm:"type:text" json:"summary,omitempty"`
	Error      string    `gorm:"type:text" json:"error,omitempty"`
	ExecutedAt time.Time `gorm:"index:idx_scheduled_task_runs_task_executed,priority:2" json:"executed_at"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
