package agent_sdk

import (
	"context"
	"errors"
	"strings"

	"icoo_claw/server/claw/pkg/agent_sdk/sdk/api"
)

type SDKRunner struct {
	factory *RuntimeFactory
	history *HistoryAdapter
}

func NewSDKRunner(factory *RuntimeFactory, history *HistoryAdapter) *SDKRunner {
	return &SDKRunner{factory: factory, history: history}
}

func (r *SDKRunner) Run(ctx context.Context, req RunRequest) (*RunResponse, error) {
	if r == nil || r.factory == nil {
		return nil, errors.New("agent sdk runner is not configured")
	}

	rt, err := r.factory.New(ctx, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rt.Close() }()

	resp, err := rt.Run(ctx, api.Request{
		SessionID:     req.SessionID,
		RequestID:     req.RequestID,
		Prompt:        req.Prompt,
		ToolWhitelist: req.ToolWhitelist,
		ForceSkills:   req.ForceSkills,
		Metadata:      req.Metadata,
	})
	if err != nil {
		return nil, mapRuntimeError(err)
	}
	if err := r.saveSnapshot(ctx, rt, req.SessionID); err != nil {
		return nil, err
	}
	if resp == nil || resp.Result == nil {
		return nil, errors.New("agent runtime returned empty response")
	}

	return &RunResponse{
		SessionID:  req.SessionID,
		RequestID:  resp.RequestID,
		Output:     resp.Result.Output,
		StopReason: resp.Result.StopReason,
	}, nil
}

func (r *SDKRunner) RunStream(ctx context.Context, req RunRequest) (<-chan StreamEvent, error) {
	if r == nil || r.factory == nil {
		return nil, errors.New("agent sdk runner is not configured")
	}

	rt, err := r.factory.New(ctx, req)
	if err != nil {
		return nil, err
	}

	events, err := rt.RunStream(ctx, api.Request{
		SessionID:     req.SessionID,
		RequestID:     req.RequestID,
		Prompt:        req.Prompt,
		ToolWhitelist: req.ToolWhitelist,
		ForceSkills:   req.ForceSkills,
		Metadata:      req.Metadata,
	})
	if err != nil {
		_ = rt.Close()
		return nil, mapRuntimeError(err)
	}

	out := make(chan StreamEvent, 128)
	go func() {
		defer close(out)
		defer func() { _ = rt.Close() }()
		defer func() { _ = r.saveSnapshot(context.Background(), rt, req.SessionID) }()

		for event := range events {
			out <- mapRuntimeStreamEvent(event, req)
		}
	}()

	return out, nil
}

func mapRuntimeStreamEvent(event api.StreamEvent, req RunRequest) StreamEvent {
	sessionID := event.SessionID
	if sessionID == "" {
		sessionID = req.SessionID
	}
	base := StreamEvent{
		Type:      StreamEventSessionUpdate,
		SessionID: sessionID,
		RequestID: req.RequestID,
	}

	switch event.Type {
	case api.EventContentBlockDelta:
		if event.Delta != nil && event.Delta.Type == "text_delta" && event.Delta.Text != "" {
			base.Update = &SessionUpdate{
				SessionUpdate: "agent_message_chunk",
				Content:       &ContentBlock{Type: "text", Text: event.Delta.Text},
			}
			return base
		}
		if event.Delta != nil && event.Delta.Type == "input_json_delta" && len(event.Delta.PartialJSON) > 0 {
			base.Update = &SessionUpdate{
				SessionUpdate: "tool_call_update",
				ToolCallID:    event.ToolUseID,
				RawInput:      jsonRawMessageString(event.Delta.PartialJSON),
			}
			return base
		}
	case api.EventContentBlockStart:
		if event.ContentBlock != nil && event.ContentBlock.Type == "tool_use" {
			toolID := firstNonEmpty(event.ContentBlock.ID, event.ToolUseID)
			base.Update = &SessionUpdate{
				SessionUpdate: "tool_call",
				ToolCallID:    toolID,
				Title:         defaultString(event.ContentBlock.Name, "Tool call"),
				Kind:          toolKind(event.ContentBlock.Name),
				Status:        "pending",
				RawInput:      jsonRawMessageString(event.ContentBlock.Input),
			}
			return base
		}
	case api.EventToolExecutionStart:
		base.Update = &SessionUpdate{
			SessionUpdate: "tool_call",
			ToolCallID:    event.ToolUseID,
			Title:         defaultString(event.Name, "Tool call"),
			Kind:          toolKind(event.Name),
			Status:        "in_progress",
		}
		return base
	case api.EventToolExecutionOutput:
		base.Update = &SessionUpdate{
			SessionUpdate: "tool_call_update",
			ToolCallID:    event.ToolUseID,
			Status:        "in_progress",
			RawOutput:     event.Output,
		}
		return base
	case api.EventToolExecutionResult:
		status := "completed"
		if event.IsError != nil && *event.IsError {
			status = "failed"
		}
		base.Update = &SessionUpdate{
			SessionUpdate: "tool_call_update",
			ToolCallID:    event.ToolUseID,
			Status:        status,
			RawOutput:     event.Output,
		}
		return base
	case api.EventMessageDelta:
		if event.Usage != nil {
			base.Update = &SessionUpdate{
				SessionUpdate: "usage_update",
				Usage: &UsageUpdate{
					InputTokens:  event.Usage.InputTokens,
					OutputTokens: event.Usage.OutputTokens,
					TotalTokens:  event.Usage.InputTokens + event.Usage.OutputTokens,
				},
			}
			return base
		}
		if event.Delta != nil && event.Delta.StopReason != "" {
			base.Update = &SessionUpdate{SessionUpdate: event.Type}
			return base
		}
	case api.EventMessageStop:
		base.Update = &SessionUpdate{SessionUpdate: event.Type}
		return base
	case api.EventAgentStop:
		return StreamEvent{
			Type:       StreamEventSessionCompleted,
			SessionID:  sessionID,
			RequestID:  req.RequestID,
			StopReason: "end_turn",
		}
	case api.EventError:
		message := streamEventOutput(event)
		return StreamEvent{
			Type:      StreamEventSessionError,
			SessionID: sessionID,
			RequestID: req.RequestID,
			Error:     &StreamError{Message: defaultString(message, "stream error")},
		}
	}

	base.Update = &SessionUpdate{SessionUpdate: event.Type}
	return base
}

func (r *SDKRunner) saveSnapshot(ctx context.Context, rt *api.Runtime, sessionID string) error {
	if r.history == nil {
		return nil
	}
	snapshot, ok := rt.SessionHistory(sessionID)
	if !ok {
		return nil
	}
	return r.history.SaveSnapshot(ctx, sessionID, snapshot)
}

func streamEventOutput(event api.StreamEvent) string {
	switch event.Type {
	case api.EventContentBlockDelta:
		if event.Delta != nil && event.Delta.Type == "text_delta" {
			return event.Delta.Text
		}
	case api.EventToolExecutionOutput:
		if output, ok := event.Output.(string); ok {
			return output
		}
	case api.EventError:
		if output, ok := event.Output.(string); ok {
			return output
		}
	}
	return ""
}

func normalizeStopReason(value string) string {
	switch value {
	case "", "stop_sequence":
		return "end_turn"
	default:
		return value
	}
}

func jsonRawMessageString(value []byte) any {
	if len(value) == 0 {
		return nil
	}
	return string(value)
}

func toolKind(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "read", "list", "view":
		return "read"
	case "edit", "write", "patch":
		return "edit"
	case "delete", "remove":
		return "delete"
	case "move", "rename":
		return "move"
	case "search", "grep", "find":
		return "search"
	case "bash", "shell", "command", "terminal", "exec":
		return "execute"
	case "fetch", "http", "web_fetch", "web_search":
		return "fetch"
	default:
		if strings.Contains(strings.ToLower(name), "search") {
			return "search"
		}
		return "other"
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func defaultString(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func mapRuntimeError(err error) error {
	if errors.Is(err, api.ErrConcurrentExecution) {
		return ErrSessionBusy
	}
	return err
}

var ErrSessionBusy = errors.New("session busy")
