package service

import (
	"context"
	"fmt"
	"time"

	"icoo_claw/server/gateway/internal/dto"
	"icoo_claw/server/gateway/internal/model"
	"icoo_claw/server/gateway/internal/repository"
)

type RouterPolicy interface {
	SelectInstance(ctx context.Context, conversation *model.Conversation) (*model.AgentInstance, error)
	MarkInflight(ctx context.Context, instanceID string, delta int) error
}

type AgentInstanceStarter interface {
	Start(ctx context.Context, req dto.StartAgentInstanceRequest) (*dto.AgentInstance, error)
}

type AgentInstanceHealthRefresher interface {
	ProbeInstances(ctx context.Context) error
}

type DefaultRouterPolicy struct {
	conversations repository.ConversationRepository
	instances     repository.AgentInstanceRepository
	starter       AgentInstanceStarter
}

func NewDefaultRouterPolicy(conversations repository.ConversationRepository, instances repository.AgentInstanceRepository, starter AgentInstanceStarter) *DefaultRouterPolicy {
	return &DefaultRouterPolicy{
		conversations: conversations,
		instances:     instances,
		starter:       starter,
	}
}

func (p *DefaultRouterPolicy) SelectInstance(ctx context.Context, conversation *model.Conversation) (*model.AgentInstance, error) {
	if conversation == nil {
		return nil, fmt.Errorf("conversation is required")
	}
	if refresher, ok := p.starter.(AgentInstanceHealthRefresher); ok {
		if err := refresher.ProbeInstances(ctx); err != nil {
			return nil, err
		}
	}

	instances, err := p.instances.List(ctx)
	if err != nil {
		return nil, err
	}

	var selected *model.AgentInstance
	for i := range instances {
		instance := instances[i]
		if instance.AgentID != conversation.AgentID || instance.Status != "ready" {
			continue
		}
		if conversation.StickyInstanceID != "" && instance.ID == conversation.StickyInstanceID {
			return &instance, nil
		}
		if selected == nil || instance.Inflight < selected.Inflight {
			copy := instance
			selected = &copy
		}
	}

	if selected == nil {
		started, err := p.startInstance(ctx, conversation.AgentID)
		if err != nil {
			return nil, err
		}
		selected = started
	}

	if conversation.StickyInstanceID != selected.ID {
		conversation.StickyInstanceID = selected.ID
		conversation.UpdatedAt = time.Now().UTC()
		if err := p.conversations.Update(ctx, *conversation); err != nil {
			return nil, err
		}
	}
	return selected, nil
}

func (p *DefaultRouterPolicy) MarkInflight(ctx context.Context, instanceID string, delta int) error {
	return p.instances.AdjustInflight(ctx, instanceID, delta)
}

func (p *DefaultRouterPolicy) startInstance(ctx context.Context, agentID string) (*model.AgentInstance, error) {
	if p.starter == nil {
		return nil, fmt.Errorf("no ready agent instance for agent %s", agentID)
	}
	started, err := p.starter.Start(ctx, dto.StartAgentInstanceRequest{AgentID: agentID})
	if err != nil {
		return nil, err
	}
	return &model.AgentInstance{
		ID:              started.ID,
		AgentID:         started.AgentID,
		Status:          started.Status,
		PID:             started.PID,
		Host:            started.Host,
		Port:            started.Port,
		BaseURL:         started.BaseURL,
		LastHeartbeatAt: started.LastHeartbeatAt,
		LastError:       started.LastError,
		Inflight:        started.Inflight,
		CreatedAt:       started.CreatedAt,
		UpdatedAt:       started.UpdatedAt,
	}, nil
}
