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
	return s.runner.Run(ctx, req)
}

func (s *AgentService) RunStream(ctx context.Context, req dto.RunRequest) (<-chan dto.StreamEvent, error) {
	events, err := s.runner.RunStream(ctx, req)
	if err != nil {
		return nil, err
	}
	out := make(chan dto.StreamEvent)
	go func() {
		defer close(out)
		for event := range events {
			out <- dto.StreamEvent(event)
		}
	}()
	return out, nil
}
