package service

import (
	"context"
	"errors"
	"strings"

	"icoo_claw/server/gateway/internal/model"
	"icoo_claw/server/gateway/internal/repository"
)

var ErrAgentDisabled = errors.New("agent is disabled")

type AgentRuntimeProfileBuilder struct {
	providers repository.ProviderRepository
}

func NewAgentRuntimeProfileBuilder(providers repository.ProviderRepository) *AgentRuntimeProfileBuilder {
	return &AgentRuntimeProfileBuilder{providers: providers}
}

func EnsureAgentRunnable(agent *model.AgentProfile) error {
	if agent == nil {
		return nil
	}
	if !agent.Enabled {
		return ErrAgentDisabled
	}
	return nil
}

func (b *AgentRuntimeProfileBuilder) ResolveProvider(ctx context.Context, agent *model.AgentProfile) (*model.ProviderProfile, error) {
	if b == nil || b.providers == nil || agent == nil {
		return nil, nil
	}
	if agent.ProviderID != "" {
		return b.providers.Get(ctx, agent.ProviderID)
	}
	if agent.ModelProvider == "" {
		return nil, nil
	}
	provider, err := b.providers.GetEnabledByType(ctx, agent.ModelProvider)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return provider, nil
}

func (b *AgentRuntimeProfileBuilder) BuildPayload(agent model.AgentProfile, provider *model.ProviderProfile, metadata map[string]any) map[string]any {
	return agentProfileMap(agent, provider, metadata)
}

func (b *AgentRuntimeProfileBuilder) BuildLaunchConfig(agent model.AgentProfile, provider *model.ProviderProfile) AgentLaunchConfig {
	launch := AgentLaunchConfig{
		ProviderID:    agent.ProviderID,
		ModelProvider: agent.ModelProvider,
		ModelName:     agent.ModelName,
		BaseURL:       agent.BaseURL,
	}
	if provider == nil {
		return launch
	}
	launch.ProviderID = provider.ID
	if provider.Type != "" {
		launch.ModelProvider = provider.Type
	}
	if launch.ModelName == "" {
		launch.ModelName = provider.DefaultModel
	}
	if launch.BaseURL == "" {
		launch.BaseURL = provider.BaseURL
	}
	launch.APIKey = provider.APIKey
	return launch
}

func agentProfileMap(agent model.AgentProfile, provider *model.ProviderProfile, metadata map[string]any) map[string]any {
	modelProvider := agent.ModelProvider
	modelName := agent.ModelName
	baseURL := agent.BaseURL
	apiKey := ""
	if provider != nil {
		if provider.Type != "" {
			modelProvider = provider.Type
		}
		if modelName == "" {
			modelName = provider.DefaultModel
		}
		if baseURL == "" {
			baseURL = provider.BaseURL
		}
		apiKey = provider.APIKey
	}
	profile := map[string]any{
		"model_provider": modelProvider,
		"model_name":     modelName,
		"base_url":       baseURL,
		"api_key":        apiKey,
		"system_prompt":  agent.SystemPrompt,
		"max_iterations": maxAgentIterations(agent.MaxIterations),
		"network_allow":  parseStringSlice(agent.NetworkAllowJSON),
		"mcp_servers":    parseStringSlice(agent.MCPServerIDsJSON),
	}
	if tools := parseStringSlice(agent.ToolWhitelistJSON); len(tools) > 0 {
		profile["enabled_builtin_tools"] = tools
	}
	if projectRoot := metadataString(metadata, "project_root"); projectRoot != "" {
		profile["project_root"] = projectRoot
	}
	return profile
}

func maxAgentIterations(value int) int {
	if value < 4 {
		return 4
	}
	return value
}

func metadataString(metadata map[string]any, key string) string {
	if len(metadata) == 0 {
		return ""
	}
	value, ok := metadata[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}
