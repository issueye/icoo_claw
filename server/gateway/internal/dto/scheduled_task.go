package dto

import "time"

type ScheduledTask struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Description   string         `json:"description,omitempty"`
	AgentID       string         `json:"agent_id,omitempty"`
	ScheduleType  string         `json:"schedule_type"`
	ScheduleValue string         `json:"schedule_value"`
	ActionType    string         `json:"action_type"`
	Payload       map[string]any `json:"payload,omitempty"`
	ForceSkills   []string       `json:"force_skills,omitempty"`
	Enabled       bool           `json:"enabled"`
	Status        string         `json:"status"`
	LastRunAt     *time.Time     `json:"last_run_at,omitempty"`
	NextRunAt     *time.Time     `json:"next_run_at,omitempty"`
	RunCount      int            `json:"run_count"`
	LastError     string         `json:"last_error,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

type CreateScheduledTaskRequest struct {
	ID            string         `json:"id"`
	Name          string         `json:"name" binding:"required"`
	Description   string         `json:"description"`
	AgentID       string         `json:"agent_id"`
	ScheduleType  string         `json:"schedule_type" binding:"required"`
	ScheduleValue string         `json:"schedule_value" binding:"required"`
	ActionType    string         `json:"action_type" binding:"required"`
	Payload       map[string]any `json:"payload"`
	ForceSkills   []string       `json:"force_skills"`
	Enabled       *bool          `json:"enabled"`
}

type UpdateScheduledTaskRequest struct {
	Name          *string         `json:"name"`
	Description   *string         `json:"description"`
	AgentID       *string         `json:"agent_id"`
	ScheduleType  *string         `json:"schedule_type"`
	ScheduleValue *string         `json:"schedule_value"`
	ActionType    *string         `json:"action_type"`
	Payload       *map[string]any `json:"payload"`
	ForceSkills   []string        `json:"force_skills"`
	Enabled       *bool           `json:"enabled"`
}
