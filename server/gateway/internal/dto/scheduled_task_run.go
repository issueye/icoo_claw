package dto

import "time"

type ScheduledTaskRun struct {
	ID         string    `json:"id"`
	TaskID     string    `json:"task_id"`
	AgentID    string    `json:"agent_id,omitempty"`
	Status     string    `json:"status"`
	Summary    string    `json:"summary,omitempty"`
	Error      string    `json:"error,omitempty"`
	ExecutedAt time.Time `json:"executed_at"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
