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
				Type:       event.Type,
				SessionID:  event.SessionID,
				RequestID:  event.RequestID,
				Update:     toDTOUpdate(event.Update),
				StopReason: event.StopReason,
				Error:      toDTOStreamError(event.Error),
			}
		}
	}()
	return out, nil
}

func toDTOUpdate(update *agent_sdk.SessionUpdate) *dto.SessionUpdate {
	if update == nil {
		return nil
	}
	return &dto.SessionUpdate{
		SessionUpdate: update.SessionUpdate,
		Content:       toDTOContentBlock(update.Content),
		MessageID:     update.MessageID,
		ToolCallID:    update.ToolCallID,
		Title:         update.Title,
		Kind:          update.Kind,
		Status:        update.Status,
		Locations:     toDTOToolCallLocations(update.Locations),
		RawInput:      update.RawInput,
		RawOutput:     update.RawOutput,
		Usage:         toDTOUsage(update.Usage),
	}
}

func toDTOContentBlock(content *agent_sdk.ContentBlock) *dto.ContentBlock {
	if content == nil {
		return nil
	}
	return &dto.ContentBlock{
		Type: content.Type,
		Text: content.Text,
		URI:  content.URI,
		Mime: content.Mime,
		Data: content.Data,
	}
}

func toDTOToolCallLocations(locations []agent_sdk.ToolCallLocation) []dto.ToolCallLocation {
	if len(locations) == 0 {
		return nil
	}
	out := make([]dto.ToolCallLocation, len(locations))
	for i, location := range locations {
		out[i] = dto.ToolCallLocation{Path: location.Path, Line: location.Line}
	}
	return out
}

func toDTOUsage(usage *agent_sdk.UsageUpdate) *dto.UsageUpdate {
	if usage == nil {
		return nil
	}
	return &dto.UsageUpdate{
		InputTokens:  usage.InputTokens,
		OutputTokens: usage.OutputTokens,
		TotalTokens:  usage.TotalTokens,
	}
}

func toDTOStreamError(streamError *agent_sdk.StreamError) *dto.StreamError {
	if streamError == nil {
		return nil
	}
	return &dto.StreamError{Message: streamError.Message, Code: streamError.Code}
}
