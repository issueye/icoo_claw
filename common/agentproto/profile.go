package agentproto

type AgentRuntimeProfile struct {
	ModelProvider       string   `json:"model_provider,omitempty"`
	ModelName           string   `json:"model_name,omitempty"`
	APIKey              string   `json:"api_key,omitempty"`
	BaseURL             string   `json:"base_url,omitempty"`
	ProjectRoot         string   `json:"project_root,omitempty"`
	SystemPrompt        string   `json:"system_prompt,omitempty"`
	MaxIterations       int      `json:"max_iterations,omitempty"`
	EnabledBuiltinTools []string `json:"enabled_builtin_tools,omitempty"`
	MCPServers          []string `json:"mcp_servers,omitempty"`
	NetworkAllow        []string `json:"network_allow,omitempty"`
}

type AgentLaunchConfig struct {
	ProviderID    string `json:"provider_id,omitempty"`
	ModelProvider string `json:"model_provider,omitempty"`
	ModelName     string `json:"model_name,omitempty"`
	APIKey        string `json:"api_key,omitempty"`
	BaseURL       string `json:"base_url,omitempty"`
}
