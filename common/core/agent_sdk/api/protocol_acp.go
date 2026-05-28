package api

// ACPProtocol encodes AgentEvents into structures compatible with the Agent
// Communication Protocol (ACP). The encoded output maps to the session-level
// event format that downstream consumers (e.g. claw's ACP agent, gateway's
// ACP runner) can directly translate into acp.SessionUpdate values.
//
// This encoder does NOT depend on the external acp-go-sdk package, keeping the
// core SDK free of transport-specific dependencies. The returned ACPEvent can
// be further marshalled by consumers that hold the ACP SDK dependency.
type ACPProtocol struct{}

// NewACPProtocol returns the ACP protocol encoder.
func NewACPProtocol() *ACPProtocol {
	return &ACPProtocol{}
}

// ACPEvent is the wire-format representation for ACP-compatible consumers.
// It mirrors the session-level protocol used between claw and gateway.
type ACPEvent struct {
	Type       string        `json:"type"`
	SessionID  string        `json:"session_id"`
	RequestID  string        `json:"request_id,omitempty"`
	StopReason string        `json:"stop_reason,omitempty"`
	Error      *ACPError     `json:"error,omitempty"`
	Update     *ACPUpdate    `json:"update,omitempty"`
}

// ACPError represents an error in the ACP session protocol.
type ACPError struct {
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}

// ACPUpdate carries the session update payload for ACP consumers.
type ACPUpdate struct {
	SessionUpdate string        `json:"sessionUpdate"`
	Content       *ACPContent   `json:"content,omitempty"`
	ToolCallID    string        `json:"toolCallId,omitempty"`
	Title         string        `json:"title,omitempty"`
	Kind          string        `json:"kind,omitempty"`
	Status        string        `json:"status,omitempty"`
	RawInput      interface{}   `json:"rawInput,omitempty"`
	RawOutput     interface{}   `json:"rawOutput,omitempty"`
	Usage         *ACPUsage     `json:"usage,omitempty"`
}

// ACPContent represents a content block in the ACP format.
type ACPContent struct {
	Type string `json:"type,omitempty"`
	Text string `json:"text,omitempty"`
}

// ACPUsage represents token usage in the ACP format.
type ACPUsage struct {
	InputTokens  int `json:"inputTokens,omitempty"`
	OutputTokens int `json:"outputTokens,omitempty"`
	TotalTokens  int `json:"totalTokens,omitempty"`
}

// Encode converts a protocol-agnostic AgentEvent into an ACP-compatible
// ACPEvent. Events that have no ACP mapping are returned as nil (the
// caller should skip them).
func (p *ACPProtocol) Encode(event AgentEvent) interface{} {
	switch event.Type {
	case AETextDelta:
		return ACPEvent{
			Type:      "session/update",
			SessionID: event.SessionID,
			RequestID: event.RequestID,
			Update: &ACPUpdate{
				SessionUpdate: "agent_message_chunk",
				Content:       &ACPContent{Type: "text", Text: event.Text},
			},
		}

	case AEToolCallStart:
		return ACPEvent{
			Type:      "session/update",
			SessionID: event.SessionID,
			RequestID: event.RequestID,
			Update: &ACPUpdate{
				SessionUpdate: "tool_call",
				ToolCallID:    event.ToolUseID,
				Title:         toolDisplayName(event.ToolName),
				Kind:          classifyToolKind(event.ToolName),
				Status:        "pending",
			},
		}

	case AEToolExecutionStart:
		return ACPEvent{
			Type:      "session/update",
			SessionID: event.SessionID,
			RequestID: event.RequestID,
			Update: &ACPUpdate{
				SessionUpdate: "tool_call",
				ToolCallID:    event.ToolUseID,
				Title:         toolDisplayName(event.ToolName),
				Kind:          classifyToolKind(event.ToolName),
				Status:        "in_progress",
				RawInput:      event.ToolInput,
			},
		}

	case AEToolExecutionOutput:
		return ACPEvent{
			Type:      "session/update",
			SessionID: event.SessionID,
			RequestID: event.RequestID,
			Update: &ACPUpdate{
				SessionUpdate: "tool_call_update",
				ToolCallID:    event.ToolUseID,
				Status:        "in_progress",
				RawOutput:     event.ToolOutput,
			},
		}

	case AEToolExecutionResult:
		status := "completed"
		if event.IsError {
			status = "failed"
		}
		return ACPEvent{
			Type:      "session/update",
			SessionID: event.SessionID,
			RequestID: event.RequestID,
			Update: &ACPUpdate{
				SessionUpdate: "tool_call_update",
				ToolCallID:    event.ToolUseID,
				Status:        status,
				RawOutput:     event.ToolOutput,
			},
		}

	case AEToolCallInputDelta:
		return ACPEvent{
			Type:      "session/update",
			SessionID: event.SessionID,
			RequestID: event.RequestID,
			Update: &ACPUpdate{
				SessionUpdate: "tool_call_update",
				ToolCallID:    event.ToolUseID,
				RawInput:      event.ToolOutputRaw,
			},
		}

	case AEUsageUpdate:
		return ACPEvent{
			Type:      "session/update",
			SessionID: event.SessionID,
			RequestID: event.RequestID,
			Update: &ACPUpdate{
				SessionUpdate: "usage_update",
				Usage: &ACPUsage{
					InputTokens:  event.InputTokens,
					OutputTokens: event.OutputTokens,
					TotalTokens:  event.InputTokens + event.OutputTokens,
				},
			},
		}

	case AEAgentStop:
		return ACPEvent{
			Type:       "session/completed",
			SessionID:  event.SessionID,
			RequestID:  event.RequestID,
			StopReason: "end_turn",
		}

	case AEFatalError:
		return ACPEvent{
			Type:      "session/error",
			SessionID: event.SessionID,
			RequestID: event.RequestID,
			Error: &ACPError{
				Message: event.ErrorMessage,
			},
		}

	case AEStopReason, AEMessageEnvelopeStart, AEMessageEnvelopeStop,
		AEAgentStart, AEIterationStart, AEIterationStop,
		AETextStart, AETextStop, AEToolCallInputStop:
		return nil

	default:
		return nil
	}
}

// classifyToolKind maps a tool name to an ACP tool kind string.
func classifyToolKind(name string) string {
	switch name {
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
		return "other"
	}
}

func toolDisplayName(name string) string {
	if name == "" {
		return "Tool call"
	}
	return name
}
