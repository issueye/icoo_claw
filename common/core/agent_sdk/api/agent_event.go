package api

import "encoding/json"

// AgentEventType enumerates the semantic events produced by the agent runtime.
// These are protocol-agnostic: they describe what happened, not how it is
// serialised on the wire.
type AgentEventType int

const (
	// Agent lifecycle.
	AEAgentStart AgentEventType = iota
	AEAgentStop

	// Iteration lifecycle within an agent run.
	AEIterationStart
	AEIterationStop

	// Text content streaming from the model.
	AETextStart
	AETextDelta
	AETextStop

	// Tool call lifecycle.
	AEToolCallStart
	AEToolCallInputDelta
	AEToolCallInputStop
	AEToolExecutionStart
	AEToolExecutionOutput
	AEToolExecutionResult

	// Message-level metadata.
	AEUsageUpdate
	AEStopReason

	// Message envelope markers (primarily for Anthropic SSE compat).
	AEMessageEnvelopeStart
	AEMessageEnvelopeStop

	// Terminal error.
	AEFatalError
)

// AgentEvent is the protocol-agnostic representation of a single streaming
// event. Consumers translate these into wire-format specific structures
// (Anthropic SSE events, ACP SessionUpdate, etc.) via StreamProtocol.
type AgentEvent struct {
	Type AgentEventType

	// Session identifiers.
	SessionID string
	RequestID string

	// Iteration tracking (AEIterationStart/Stop).
	Iteration int

	// Text payload (AETextDelta).
	Text string

	// Tool call fields (AEToolCallStart/AEToolExecution*/AEToolCallResult).
	ToolUseID string
	ToolName  string
	ToolKind  string

	// Tool status: "pending", "in_progress", "completed", "failed".
	ToolStatus string

	// Tool input/output payloads.
	ToolInput     map[string]any
	ToolInputJSON json.RawMessage
	ToolOutput    interface{}
	ToolOutputRaw string
	IsStderr      bool
	IsError       bool

	// Usage metrics (AEUsageUpdate).
	InputTokens  int
	OutputTokens int

	// Stop reason (AEStopReason).
	StopReason string

	// Fatal error message (AEFatalError).
	ErrorMessage string
}

// StreamProtocol converts protocol-agnostic AgentEvents into wire-format
// structures suitable for a specific transport.
type StreamProtocol interface {
	// Encode translates an AgentEvent into the protocol's wire representation.
	// The returned interface{} should be the concrete event type used by the
	// protocol consumer (e.g. StreamEvent for Anthropic SSE).
	Encode(event AgentEvent) interface{}
}
