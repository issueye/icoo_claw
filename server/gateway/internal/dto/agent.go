package dto

import "time"

type NetworkProxy struct {
	HTTPProxy  string `json:"http_proxy,omitempty"`
	HTTPSProxy string `json:"https_proxy,omitempty"`
	NoProxy    string `json:"no_proxy,omitempty"`
}

type AgentProfile struct {
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	ProviderID    string       `json:"provider_id,omitempty"`
	ModelProvider string       `json:"model_provider"`
	ModelName     string       `json:"model_name"`
	BaseURL       string       `json:"base_url,omitempty"`
	Transport     string       `json:"transport"`
	CommandArgs   []string     `json:"command_args"`
	SystemPrompt  string       `json:"system_prompt"`
	MaxIterations int          `json:"max_iterations"`
	ToolWhitelist []string     `json:"tool_whitelist"`
	NetworkAllow  []string     `json:"network_allow"`
	NetworkProxy  NetworkProxy `json:"network_proxy"`
	MCPServerIDs  []string     `json:"mcp_server_ids"`
	SkillNames    []string     `json:"skill_names"`
	Enabled       bool         `json:"enabled"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
}

type CreateAgentRequest struct {
	ID            string       `json:"id"`
	Name          string       `json:"name" binding:"required"`
	ProviderID    string       `json:"provider_id"`
	ModelProvider string       `json:"model_provider"`
	ModelName     string       `json:"model_name"`
	BaseURL       string       `json:"base_url"`
	Transport     string       `json:"transport"`
	CommandArgs   []string     `json:"command_args"`
	SystemPrompt  string       `json:"system_prompt"`
	MaxIterations int          `json:"max_iterations"`
	ToolWhitelist []string     `json:"tool_whitelist"`
	NetworkAllow  []string     `json:"network_allow"`
	NetworkProxy  NetworkProxy `json:"network_proxy"`
	MCPServerIDs  []string     `json:"mcp_server_ids"`
	SkillNames    []string     `json:"skill_names"`
	Enabled       *bool        `json:"enabled"`
}

type UpdateAgentRequest struct {
	Name          *string       `json:"name"`
	ProviderID    *string       `json:"provider_id"`
	ModelProvider *string       `json:"model_provider"`
	ModelName     *string       `json:"model_name"`
	BaseURL       *string       `json:"base_url"`
	Transport     *string       `json:"transport"`
	CommandArgs   []string      `json:"command_args"`
	SystemPrompt  *string       `json:"system_prompt"`
	MaxIterations *int          `json:"max_iterations"`
	ToolWhitelist []string      `json:"tool_whitelist"`
	NetworkAllow  []string      `json:"network_allow"`
	NetworkProxy  *NetworkProxy `json:"network_proxy"`
	MCPServerIDs  []string      `json:"mcp_server_ids"`
	SkillNames    []string      `json:"skill_names"`
	Enabled       *bool         `json:"enabled"`
}
