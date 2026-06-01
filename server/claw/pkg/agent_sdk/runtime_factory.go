package agent_sdk

import (
	"context"
	"fmt"
	"os"
	"strings"

	"icoo_claw/common/agentproto"
	"icoo_claw/common/core/agent_sdk/api"
	sdkmessage "icoo_claw/common/core/agent_sdk/message"
	sdkmodel "icoo_claw/common/core/agent_sdk/model"
)

type RuntimeFactory struct {
	history            *HistoryAdapter
	model              sdkmodel.Model
	defaultProjectRoot string
	permissionPrompter api.PermissionPrompter
}

func NewRuntimeFactory(history *HistoryAdapter, model sdkmodel.Model) *RuntimeFactory {
	return &RuntimeFactory{history: history, model: model}
}

func (f *RuntimeFactory) SetDefaultProjectRoot(projectRoot string) {
	if f == nil {
		return
	}
	f.defaultProjectRoot = strings.TrimSpace(projectRoot)
}

func (f *RuntimeFactory) SetPermissionPrompter(prompter api.PermissionPrompter) {
	if f == nil {
		return
	}
	f.permissionPrompter = prompter
}

func (f *RuntimeFactory) New(ctx context.Context, req RunRequest) (*api.Runtime, error) {
	profile := parseAgentProfile(req.Agent)
	if strings.TrimSpace(profile.ProjectRoot) == "" && f.defaultProjectRoot != "" {
		profile.ProjectRoot = f.defaultProjectRoot
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
			NetworkProxy: api.NetworkProxyOptions{
				HTTPProxy:  profile.NetworkProxy.HTTPProxy,
				HTTPSProxy: profile.NetworkProxy.HTTPSProxy,
				NoProxy:    profile.NetworkProxy.NoProxy,
			},
		},
		PermissionPrompter: f.permissionPrompter,
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

	if options.PermissionPrompter == nil {
		// Claw runs as the platform service layer, not an interactive CLI. Keep
		// existing HTTP/Gateway behavior by approving SDK permission prompts here
		// while ACP stdio mode can inject a real client-backed prompter.
		options.PermissionPrompter = api.PermissionPrompterFunc(func(context.Context, api.PermissionRequest) (bool, error) {
			return true, nil
		})
	}

	rt, err := api.New(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("create agent runtime: %w", err)
	}
	return rt, nil
}

type AgentProfile = agentproto.AgentRuntimeProfile

func parseAgentProfile(input *agentproto.AgentRuntimeProfile) AgentProfile {
	profile := AgentProfile{}
	if input != nil {
		profile = *input
	}
	if strings.TrimSpace(profile.ModelProvider) == "" {
		profile.ModelProvider = strings.TrimSpace(os.Getenv("ICOO_AGENT_MODEL_PROVIDER"))
	}
	if strings.TrimSpace(profile.ModelName) == "" {
		profile.ModelName = strings.TrimSpace(os.Getenv("ICOO_AGENT_MODEL_NAME"))
	}
	if strings.TrimSpace(profile.APIKey) == "" {
		profile.APIKey = strings.TrimSpace(os.Getenv("ICOO_AGENT_API_KEY"))
	}
	if strings.TrimSpace(profile.BaseURL) == "" {
		profile.BaseURL = strings.TrimSpace(os.Getenv("ICOO_AGENT_BASE_URL"))
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
