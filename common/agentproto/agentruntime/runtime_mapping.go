package agentruntime

import (
	"encoding/json"
	"strings"

	"icoo_claw/common/agentproto"
	"icoo_claw/common/core/agent_sdk/api"
)

// MapStreamEvent converts the core SDK's Anthropic-compatible runtime stream
// event into the shared Claw/Gateway session stream protocol.
func MapStreamEvent(event api.StreamEvent, sessionID string, requestID string) agentproto.StreamEvent {
	if event.SessionID != "" {
		sessionID = event.SessionID
	}
	base := agentproto.StreamEvent{
		Type:      agentproto.StreamEventSessionUpdate,
		SessionID: sessionID,
		RequestID: requestID,
	}

	switch event.Type {
	case api.EventContentBlockDelta:
		if event.Delta != nil && event.Delta.Type == "text_delta" && event.Delta.Text != "" {
			base.Update = &agentproto.SessionUpdate{
				SessionUpdate: "agent_message_chunk",
				Content:       &agentproto.ContentBlock{Type: "text", Text: event.Delta.Text},
			}
			return base
		}
		if event.Delta != nil && event.Delta.Type == "input_json_delta" && len(event.Delta.PartialJSON) > 0 {
			base.Update = &agentproto.SessionUpdate{
				SessionUpdate: "tool_call_update",
				ToolCallID:    event.ToolUseID,
				RawInput:      jsonRawMessageString(event.Delta.PartialJSON),
			}
			return base
		}
	case api.EventContentBlockStart:
		if event.ContentBlock != nil && event.ContentBlock.Type == "tool_use" {
			toolID := firstNonEmpty(event.ContentBlock.ID, event.ToolUseID)
			base.Update = &agentproto.SessionUpdate{
				SessionUpdate: "tool_call",
				ToolCallID:    toolID,
				Title:         defaultString(event.ContentBlock.Name, "Tool call"),
				Kind:          ToolKind(event.ContentBlock.Name),
				Status:        "pending",
				RawInput:      jsonRawMessageString(event.ContentBlock.Input),
			}
			return base
		}
	case api.EventToolExecutionStart:
		base.Update = &agentproto.SessionUpdate{
			SessionUpdate: "tool_call",
			ToolCallID:    event.ToolUseID,
			Title:         defaultString(event.Name, "Tool call"),
			Kind:          ToolKind(event.Name),
			Status:        "in_progress",
			RawInput:      event.Output,
		}
		return base
	case api.EventToolExecutionOutput:
		base.Update = &agentproto.SessionUpdate{
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
		base.Update = &agentproto.SessionUpdate{
			SessionUpdate: "tool_call_update",
			ToolCallID:    event.ToolUseID,
			Status:        status,
			RawOutput:     event.Output,
		}
		return base
	case api.EventMessageDelta:
		if event.Usage != nil {
			base.Update = &agentproto.SessionUpdate{
				SessionUpdate: "usage_update",
				Usage: &agentproto.UsageUpdate{
					InputTokens:  event.Usage.InputTokens,
					OutputTokens: event.Usage.OutputTokens,
					TotalTokens:  event.Usage.InputTokens + event.Usage.OutputTokens,
				},
			}
			return base
		}
		if event.Delta != nil && event.Delta.StopReason != "" {
			base.Update = &agentproto.SessionUpdate{SessionUpdate: event.Type}
			return base
		}
	case api.EventMessageStop:
		base.Update = &agentproto.SessionUpdate{SessionUpdate: event.Type}
		return base
	case api.EventAgentStop:
		return agentproto.StreamEvent{
			Type:       agentproto.StreamEventSessionCompleted,
			SessionID:  sessionID,
			RequestID:  requestID,
			StopReason: "end_turn",
		}
	case api.EventError:
		message := StreamEventOutput(event)
		return agentproto.StreamEvent{
			Type:      agentproto.StreamEventSessionError,
			SessionID: sessionID,
			RequestID: requestID,
			Error:     &agentproto.StreamError{Message: defaultString(message, "stream error")},
		}
	}

	base.Update = &agentproto.SessionUpdate{SessionUpdate: event.Type}
	return base
}

// StreamEventOutput extracts a displayable output string from runtime events
// that carry text or error payloads.
func StreamEventOutput(event api.StreamEvent) string {
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

// ToolKind maps a tool name to the shared stream protocol kind.
func ToolKind(name string) string {
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

func jsonRawMessageString(value []byte) any {
	if len(value) == 0 {
		return nil
	}
	var text string
	if err := json.Unmarshal(value, &text); err == nil {
		return text
	}
	return string(value)
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
