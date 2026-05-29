package api

import "encoding/json"

// AnthropicSSEProtocol encodes AgentEvents into Anthropic-compatible SSE
// StreamEvents. This is the default protocol and preserves full backward
// compatibility with existing consumers.
type AnthropicSSEProtocol struct{}

// NewAnthropicSSEProtocol returns the default Anthropic SSE encoder.
func NewAnthropicSSEProtocol() *AnthropicSSEProtocol {
	return &AnthropicSSEProtocol{}
}

// Encode converts a protocol-agnostic AgentEvent into an Anthropic-compatible
// StreamEvent.
func (p *AnthropicSSEProtocol) Encode(event AgentEvent) interface{} {
	switch event.Type {
	case AEAgentStart:
		return StreamEvent{Type: EventAgentStart}

	case AEAgentStop:
		return StreamEvent{Type: EventAgentStop}

	case AEIterationStart:
		iter := event.Iteration
		return StreamEvent{Type: EventIterationStart, Iteration: &iter}

	case AEIterationStop:
		iter := event.Iteration
		return StreamEvent{Type: EventIterationStop, Iteration: &iter}

	case AETextStart:
		idx := 0
		return StreamEvent{Type: EventContentBlockStart, Index: &idx, ContentBlock: &ContentBlock{Type: "text"}}

	case AETextDelta:
		idx := 0
		return StreamEvent{Type: EventContentBlockDelta, Index: &idx, Delta: &Delta{Type: "text_delta", Text: event.Text}}

	case AETextStop:
		idx := 0
		return StreamEvent{Type: EventContentBlockStop, Index: &idx}

	case AEToolCallStart:
		return p.encodeToolCallStart(event)

	case AEToolCallInputDelta:
		return p.encodeToolCallInputDelta(event)

	case AEToolCallInputStop:
		return StreamEvent{
			Type:  EventContentBlockStop,
			Index: intPtr(0),
		}

	case AEToolExecutionStart:
		iter := event.Iteration
		return StreamEvent{
			Type:      EventToolExecutionStart,
			ToolUseID: event.ToolUseID,
			Name:      event.ToolName,
			Output:    event.ToolInput,
			Iteration: &iter,
		}

	case AEToolExecutionOutput:
		evt := StreamEvent{
			Type:      EventToolExecutionOutput,
			ToolUseID: event.ToolUseID,
			Name:      event.ToolName,
			Output:    event.ToolOutput,
		}
		evt.IsStderr = &event.IsStderr
		return evt

	case AEToolExecutionResult:
		evt := StreamEvent{
			Type:      EventToolExecutionResult,
			ToolUseID: event.ToolUseID,
			Name:      event.ToolName,
			Output:    event.ToolOutput,
		}
		evt.IsError = &event.IsError
		return evt

	case AEUsageUpdate:
		return StreamEvent{
			Type: EventMessageDelta,
			Usage: &Usage{
				InputTokens:         event.InputTokens,
				OutputTokens:        event.OutputTokens,
				TotalTokens:         event.TotalTokens,
				CacheReadTokens:     event.CacheReadTokens,
				CacheCreationTokens: event.CacheCreationTokens,
			},
		}

	case AEStopReason:
		return StreamEvent{
			Type:  EventMessageDelta,
			Delta: &Delta{StopReason: event.StopReason},
			Usage: &Usage{},
		}

	case AEMessageEnvelopeStart:
		return StreamEvent{
			Type:    EventMessageStart,
			Message: &Message{Role: "assistant"},
		}

	case AEMessageEnvelopeStop:
		return StreamEvent{Type: EventMessageStop}

	case AEFatalError:
		isErr := true
		return StreamEvent{Type: EventError, Output: event.ErrorMessage, IsError: &isErr}

	default:
		return StreamEvent{Type: EventError, Output: "unknown agent event type"}
	}
}

func (p *AnthropicSSEProtocol) encodeToolCallStart(event AgentEvent) StreamEvent {
	idx := 0
	return StreamEvent{
		Type:      EventContentBlockStart,
		Index:     &idx,
		ToolUseID: event.ToolUseID,
		ContentBlock: &ContentBlock{
			Type: "tool_use",
			ID:   event.ToolUseID,
			Name: event.ToolName,
		},
	}
}

func (p *AnthropicSSEProtocol) encodeToolCallInputDelta(event AgentEvent) StreamEvent {
	idx := 0
	encoded, err := json.Marshal(event.ToolOutputRaw)
	if err != nil {
		encoded = []byte(`""`)
	}
	return StreamEvent{
		Type:      EventContentBlockDelta,
		Index:     &idx,
		ToolUseID: event.ToolUseID,
		Delta: &Delta{
			Type:        "input_json_delta",
			PartialJSON: json.RawMessage(encoded),
		},
	}
}

func intPtr(v int) *int { return &v }
