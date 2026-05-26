package agent_sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"icoo_claw/server/claw/pkg/agent_sdk/sdk/api"
	sdkmessage "icoo_claw/server/claw/pkg/agent_sdk/sdk/message"
	sdkmodel "icoo_claw/server/claw/pkg/agent_sdk/sdk/model"
)

type RuntimeFactory struct {
	history       *HistoryAdapter
	model         sdkmodel.Model
	gatewaySkills GatewaySkills
}

func NewRuntimeFactory(history *HistoryAdapter, model sdkmodel.Model) *RuntimeFactory {
	return &RuntimeFactory{history: history, model: model}
}

func (f *RuntimeFactory) SetGatewaySkills(skills GatewaySkills) {
	if f == nil {
		return
	}
	f.gatewaySkills = skills
}

func (f *RuntimeFactory) New(ctx context.Context, req RunRequest) (*api.Runtime, error) {
	profile := parseAgentProfile(req.Agent)
	if strings.TrimSpace(profile.GatewaySkills.Path) == "" {
		profile.GatewaySkills = f.gatewaySkills
	}
	if strings.TrimSpace(profile.ProjectRoot) == "" && strings.TrimSpace(profile.GatewaySkills.Path) != "" {
		profile.ProjectRoot = profile.GatewaySkills.Path
	}
	options := api.Options{
		EntryPoint:          api.EntryPointPlatform,
		ProjectRoot:         profile.ProjectRoot,
		SystemPrompt:        profile.SystemPrompt,
		MaxIterations:       profile.MaxIterations,
		EnabledBuiltinTools: profile.EnabledBuiltinTools,
		MCPServers:          profile.MCPServers,
		Sandbox: api.SandboxOptions{
			NetworkAllow: profile.NetworkAllow,
		},
		HistoryLoader: func(sessionID string) ([]sdkmessage.Message, error) {
			if f.history == nil {
				return nil, nil
			}
			return f.history.Load(ctx, sessionID)
		},
	}

	disabledRules := false
	options.RulesEnabled = &disabledRules

	if f.model != nil {
		options.Model = f.model
	} else {
		provider, err := modelProvider(profile)
		if err != nil {
			return nil, err
		}
		options.ModelFactory = provider
	}

	rt, err := api.New(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("create agent runtime: %w", err)
	}
	return rt, nil
}

type AgentProfile struct {
	ModelProvider       string
	ModelName           string
	APIKey              string
	BaseURL             string
	ProjectRoot         string
	SystemPrompt        string
	MaxIterations       int
	EnabledBuiltinTools []string
	MCPServers          []string
	NetworkAllow        []string
	GatewaySkills       GatewaySkills
}

type GatewaySkills struct {
	Path   string
	Skills []GatewaySkill
}

type GatewaySkill struct {
	Name        string
	Description string
}

func parseAgentProfile(input map[string]any) AgentProfile {
	profile := AgentProfile{
		ModelProvider:       firstStringValue(input, "model_provider", "ICOO_AGENT_MODEL_PROVIDER"),
		ModelName:           firstStringValue(input, "model_name", "ICOO_AGENT_MODEL_NAME"),
		APIKey:              firstStringValue(input, "api_key", "ICOO_AGENT_API_KEY"),
		BaseURL:             firstStringValue(input, "base_url", "ICOO_AGENT_BASE_URL"),
		ProjectRoot:         stringValue(input, "project_root"),
		SystemPrompt:        stringValue(input, "system_prompt"),
		MaxIterations:       intValue(input, "max_iterations"),
		EnabledBuiltinTools: stringSlice(input, "enabled_builtin_tools"),
		MCPServers:          stringSlice(input, "mcp_servers"),
		NetworkAllow:        firstStringSlice(input, "network_allow", "allowed_domains"),
	}
	profile.GatewaySkills = gatewaySkillsValue(input, "gateway_skills")
	if strings.TrimSpace(profile.ProjectRoot) == "" && strings.TrimSpace(profile.GatewaySkills.Path) != "" {
		profile.ProjectRoot = profile.GatewaySkills.Path
	}
	return profile
}

func modelProvider(profile AgentProfile) (api.ModelFactory, error) {
	switch strings.ToLower(strings.TrimSpace(profile.ModelProvider)) {
	case "", "openai":
		return &sdkmodel.OpenAIProvider{
			APIKey:    profile.APIKey,
			BaseURL:   profile.BaseURL,
			ModelName: profile.ModelName,
		}, nil
	case "anthropic":
		return &sdkmodel.AnthropicProvider{
			APIKey:    profile.APIKey,
			BaseURL:   profile.BaseURL,
			ModelName: profile.ModelName,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported model provider %q", profile.ModelProvider)
	}
}

func stringValue(input map[string]any, key string) string {
	if input == nil {
		return ""
	}
	value, _ := input[key].(string)
	return strings.TrimSpace(value)
}

func firstStringValue(input map[string]any, key, envKey string) string {
	if value := stringValue(input, key); value != "" {
		return value
	}
	return strings.TrimSpace(os.Getenv(envKey))
}

func intValue(input map[string]any, key string) int {
	if input == nil {
		return 0
	}
	switch value := input[key].(type) {
	case int:
		return value
	case float64:
		return int(value)
	default:
		return 0
	}
}

func stringSlice(input map[string]any, key string) []string {
	if input == nil {
		return nil
	}
	raw, ok := input[key].([]any)
	if !ok {
		if values, ok := input[key].([]string); ok {
			return values
		}
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if value, ok := item.(string); ok && strings.TrimSpace(value) != "" {
			out = append(out, strings.TrimSpace(value))
		}
	}
	return out
}

func firstStringSlice(input map[string]any, keys ...string) []string {
	for _, key := range keys {
		if values := stringSlice(input, key); len(values) > 0 {
			return values
		}
	}
	return nil
}

func gatewaySkillsValue(input map[string]any, key string) GatewaySkills {
	if input == nil {
		return GatewaySkills{}
	}
	raw, ok := input[key]
	if !ok {
		return GatewaySkills{}
	}
	payload, ok := raw.(map[string]any)
	if !ok {
		return GatewaySkills{}
	}
	out := GatewaySkills{Path: stringValue(payload, "path")}
	for _, item := range anySlice(payload, "skills") {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name := stringValue(entry, "name")
		if name == "" {
			continue
		}
		out.Skills = append(out.Skills, GatewaySkill{
			Name:        name,
			Description: firstMapStringValue(entry, "description", "Description"),
		})
	}
	return out
}

func GatewaySkillsFromJSON(raw string) GatewaySkills {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return GatewaySkills{}
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return GatewaySkills{}
	}
	return gatewaySkillsValue(payload, "gateway_skills")
}

func anySlice(input map[string]any, key string) []any {
	if input == nil {
		return nil
	}
	switch raw := input[key].(type) {
	case []any:
		return raw
	default:
		return nil
	}
}

func firstMapStringValue(input map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := stringValue(input, key); value != "" {
			return value
		}
	}
	return ""
}
