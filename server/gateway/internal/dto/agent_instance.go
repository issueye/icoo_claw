package dto

import "time"

type AgentInstance struct {
	ID              string     `json:"id"`
	AgentID         string     `json:"agent_id"`
	Name            string     `json:"name,omitempty"`
	Status          string     `json:"status"`
	PID             int        `json:"pid"`
	Host            string     `json:"host"`
	Port            int        `json:"port"`
	BaseURL         string     `json:"base_url"`
	ProviderID      string     `json:"provider_id,omitempty"`
	ModelProvider   string     `json:"model_provider,omitempty"`
	ModelName       string     `json:"model_name,omitempty"`
	ModelBaseURL    string     `json:"model_base_url,omitempty"`
	APIKeySet       bool       `json:"api_key_set"`
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
