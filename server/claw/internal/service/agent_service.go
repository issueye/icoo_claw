package service

import (
	"context"

	"icoo_claw/server/claw/internal/dto"
	"icoo_claw/server/claw/pkg/agent_sdk"
)

type AgentService struct {
	runner agent_sdk.Runner
}

func NewAgentService(runner agent_sdk.Runner) *AgentService {
	return &AgentService{runner: runner}
}

func (s *AgentService) Run(ctx context.Context, req dto.RunRequest) (*dto.RunResponse, error) {
	resp, err := s.runner.Run(ctx, agent_sdk.RunRequest{
		SessionID:     req.SessionID,
		RequestID:     req.RequestID,
		Prompt:        req.Prompt,
		Agent:         req.Agent,
		ToolWhitelist: req.ToolWhitelist,
		ForceSkills:   req.ForceSkills,
		Metadata:      req.Metadata,
	})
	if err != nil {
		return nil, err
	}
	return &dto.RunResponse{
		SessionID:  resp.SessionID,
		RequestID:  resp.RequestID,
		Output:     resp.Output,
		StopReason: resp.StopReason,
	}, nil
}

func (s *AgentService) RunStream(ctx context.Context, req dto.RunRequest) (<-chan dto.StreamEvent, error) {
	events, err := s.runner.RunStream(ctx, agent_sdk.RunRequest{
		SessionID:     req.SessionID,
		RequestID:     req.RequestID,
		Prompt:        req.Prompt,
		Agent:         req.Agent,
		ToolWhitelist: req.ToolWhitelist,
		ForceSkills:   req.ForceSkills,
		Metadata:      req.Metadata,
	})
	if err != nil {
		return nil, err
	}

	out := make(chan dto.StreamEvent)
	go func() {
		defer close(out)
		for event := range events {
			out <- dto.StreamEvent{
				Type:      event.Type,
				SessionID: event.SessionID,
				RequestID: event.RequestID,
				Output:    event.Output,
			}
		}
	}()
	return out, nil
}
