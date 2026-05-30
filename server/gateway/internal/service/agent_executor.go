package service

import (
	"context"
	"fmt"

	"icoo_claw/common/jsonutil"
	"icoo_claw/server/gateway/internal/client"
	"icoo_claw/server/gateway/internal/dto"
	"icoo_claw/server/gateway/internal/model"
	"icoo_claw/server/gateway/internal/repository"
)

type GatewayAgentExecutor struct {
	agents    repository.AgentRepository
	providers repository.ProviderRepository
	instances repository.AgentInstanceRepository
	starter   AgentInstanceStarter
	router    RouterPolicy
	runner    AgentRunner
	profile   *AgentRuntimeProfileBuilder
}

type GatewayAgentExecutorConfig struct {
	Agents    repository.AgentRepository
	Providers repository.ProviderRepository
	Instances repository.AgentInstanceRepository
	Starter   AgentInstanceStarter
	Router    RouterPolicy
	Runner    AgentRunner
}

type AgentExecutionRequest struct {
	AgentID      string
	Conversation *model.Conversation
	SessionID    string
	Prompt       string
	RequestID    string
	ForceSkills  []string
	Metadata     map[string]any
	InstanceName string
}

type AgentExecutionContext struct {
	Agent    *model.AgentProfile
	Provider *model.ProviderProfile
	Instance *model.AgentInstance
	Request  client.RunRequest
}

func NewGatewayAgentExecutor(cfg GatewayAgentExecutorConfig) *GatewayAgentExecutor {
	return &GatewayAgentExecutor{
		agents:    cfg.Agents,
		providers: cfg.Providers,
		instances: cfg.Instances,
		starter:   cfg.Starter,
		router:    cfg.Router,
		runner:    cfg.Runner,
		profile:   NewAgentRuntimeProfileBuilder(cfg.Providers),
	}
}

func (e *GatewayAgentExecutor) Prepare(ctx context.Context, req AgentExecutionRequest) (*AgentExecutionContext, error) {
	if e == nil || e.agents == nil {
		return nil, fmt.Errorf("agent executor is not configured")
	}
	agentID := req.AgentID
	if req.Conversation != nil && agentID == "" {
		agentID = req.Conversation.AgentID
	}
	agent, err := e.agents.Get(ctx, agentID)
	if err != nil {
		return nil, err
	}
	if err := EnsureAgentRunnable(agent); err != nil {
		return nil, err
	}
	provider, err := e.profile.ResolveProvider(ctx, agent)
	if err != nil {
		return nil, err
	}
	instance, err := e.selectInstance(ctx, req, agent)
	if err != nil {
		return nil, err
	}
	sessionID := req.SessionID
	if req.Conversation != nil && sessionID == "" {
		sessionID = req.Conversation.SessionID
	}
	payload := e.profile.BuildPayload(*agent, provider, req.Metadata)
	return &AgentExecutionContext{
		Agent:    agent,
		Provider: provider,
		Instance: instance,
		Request: client.RunRequest{
			SessionID:     sessionID,
			RequestID:     req.RequestID,
			Prompt:        req.Prompt,
			Agent:         payload,
			ToolWhitelist: jsonutil.UnmarshalStringSlice(agent.ToolWhitelistJSON),
			ForceSkills:   jsonutil.CleanStringSlice(req.ForceSkills),
			Metadata:      req.Metadata,
		},
	}, nil
}

func (e *GatewayAgentExecutor) Run(ctx context.Context, req AgentExecutionRequest) (*client.RunResponse, error) {
	execCtx, err := e.Prepare(ctx, req)
	if err != nil {
		return nil, err
	}
	if err := e.markInflight(ctx, execCtx.Instance.ID, 1); err != nil {
		return nil, err
	}
	defer func() { _ = e.markInflight(context.Background(), execCtx.Instance.ID, -1) }()
	return e.runner.Run(ctx, execCtx.Instance.BaseURL, execCtx.Request)
}

func (e *GatewayAgentExecutor) Stream(ctx context.Context, req AgentExecutionRequest) (*AgentExecutionContext, <-chan client.StreamEvent, func(), error) {
	execCtx, err := e.Prepare(ctx, req)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := e.markInflight(ctx, execCtx.Instance.ID, 1); err != nil {
		return nil, nil, nil, err
	}
	cleanup := func() { _ = e.markInflight(context.Background(), execCtx.Instance.ID, -1) }
	events, err := e.runner.Stream(ctx, execCtx.Instance.BaseURL, execCtx.Request)
	if err != nil {
		cleanup()
		return nil, nil, nil, err
	}
	return execCtx, events, cleanup, nil
}

func (e *GatewayAgentExecutor) selectInstance(ctx context.Context, req AgentExecutionRequest, agent *model.AgentProfile) (*model.AgentInstance, error) {
	if req.Conversation != nil && e.router != nil {
		return e.router.SelectInstance(ctx, req.Conversation)
	}
	if e.instances != nil {
		instances, err := e.instances.List(ctx)
		if err != nil {
			return nil, err
		}
		var selected *model.AgentInstance
		for i := range instances {
			instance := instances[i]
			if instance.AgentID != agent.ID || instance.Status != "ready" {
				continue
			}
			if selected == nil || instance.Inflight < selected.Inflight {
				copy := instance
				selected = &copy
			}
		}
		if selected != nil {
			return selected, nil
		}
	}
	if e.starter == nil {
		return nil, fmt.Errorf("no ready agent instance for agent %s", agent.ID)
	}
	started, err := e.starter.Start(ctx, dto.StartAgentInstanceRequest{AgentID: agent.ID, Name: req.InstanceName})
	if err != nil {
		return nil, err
	}
	return agentInstanceFromDTO(started), nil
}

func (e *GatewayAgentExecutor) markInflight(ctx context.Context, instanceID string, delta int) error {
	if e.router != nil {
		return e.router.MarkInflight(ctx, instanceID, delta)
	}
	if e.instances != nil {
		return e.instances.AdjustInflight(ctx, instanceID, delta)
	}
	return nil
}

func agentInstanceFromDTO(started *dto.AgentInstance) *model.AgentInstance {
	if started == nil {
		return nil
	}
	return &model.AgentInstance{
		ID:              started.ID,
		AgentID:         started.AgentID,
		Name:            started.Name,
		Status:          started.Status,
		PID:             started.PID,
		Host:            started.Host,
		Port:            started.Port,
		BaseURL:         started.BaseURL,
		Transport:       started.Transport,
		ProviderID:      started.ProviderID,
		ModelProvider:   started.ModelProvider,
		ModelName:       started.ModelName,
		ModelBaseURL:    started.ModelBaseURL,
		APIKeySet:       started.APIKeySet,
		LastHeartbeatAt: started.LastHeartbeatAt,
		LastError:       started.LastError,
		Inflight:        started.Inflight,
		CreatedAt:       started.CreatedAt,
		UpdatedAt:       started.UpdatedAt,
	}
}
