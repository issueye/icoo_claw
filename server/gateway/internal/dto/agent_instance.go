package dto

import "time"

type AgentInstance struct {
	ID              string     `json:"id"`
	AgentID         string     `json:"agent_id"`
	Status          string     `json:"status"`
	PID             int        `json:"pid"`
	Host            string     `json:"host"`
	Port            int        `json:"port"`
	BaseURL         string     `json:"base_url"`
	LastHeartbeatAt *time.Time `json:"last_heartbeat_at,omitempty"`
	LastError       string     `json:"last_error,omitempty"`
	Inflight        int        `json:"inflight"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type StartAgentInstanceRequest struct {
	AgentID string `json:"agent_id" binding:"required"`
	Name    string `json:"name"`
}
