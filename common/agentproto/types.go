package agentproto

import "encoding/json"

type RunRequest struct {
	SessionID     string               `json:"session_id"`
	RequestID     string               `json:"request_id"`
	Prompt        string               `json:"prompt"`
	Agent         *AgentRuntimeProfile `json:"agent,omitempty"`
	ToolWhitelist []string             `json:"tool_whitelist,omitempty"`
	Metadata      map[string]any       `json:"metadata,omitempty"`
}

type RunResponse struct {
	SessionID  string `json:"session_id"`
	RequestID  string `json:"request_id,omitempty"`
	Output     string `json:"output"`
	StopReason string `json:"stop_reason"`
}

type StreamEvent struct {
	Type               string              `json:"type"`
	SessionID          string              `json:"session_id"`
	RequestID          string              `json:"request_id,omitempty"`
	Update             *SessionUpdate      `json:"update,omitempty"`
	Permission         *PermissionRequest  `json:"permission,omitempty"`
	PermissionDecision chan PermissionVote `json:"-"`
	StopReason         string              `json:"stop_reason,omitempty"`
	Error              *StreamError        `json:"error,omitempty"`
}

const (
	StreamEventSessionUpdate     = "session/update"
	StreamEventSessionCompleted  = "session/completed"
	StreamEventSessionError      = "session/error"
	StreamEventPermissionRequest = "session/request_permission"
)

type SessionUpdate struct {
	SessionUpdate  string                `json:"sessionUpdate"`
	Content        *ContentBlock         `json:"content,omitempty"`
	MessageID      string                `json:"messageId,omitempty"`
	ToolCallID     string                `json:"toolCallId,omitempty"`
	Title          string                `json:"title,omitempty"`
	Kind           string                `json:"kind,omitempty"`
	Status         string                `json:"status,omitempty"`
	Locations      []ToolCallLocation    `json:"locations,omitempty"`
	RawInput       any                   `json:"rawInput,omitempty"`
	RawOutput      any                   `json:"rawOutput,omitempty"`
	Usage          *UsageUpdate          `json:"usage,omitempty"`
	PlanEntries    []PlanEntry           `json:"entries,omitempty"`
	ConfigOptions  []SessionConfigOption `json:"configOptions,omitempty"`
	CurrentModeID  string                `json:"currentModeId,omitempty"`
	AvailableModes []SessionMode         `json:"availableModes,omitempty"`
}

type ContentBlock struct {
	Type string          `json:"type,omitempty"`
	Text string          `json:"text,omitempty"`
	URI  string          `json:"uri,omitempty"`
	Mime string          `json:"mimeType,omitempty"`
	Data json.RawMessage `json:"data,omitempty"`
}

type ToolCallLocation struct {
	Path string `json:"path,omitempty"`
	Line *int   `json:"line,omitempty"`
}

type UsageUpdate struct {
	InputTokens  int `json:"inputTokens,omitempty"`
	OutputTokens int `json:"outputTokens,omitempty"`
	TotalTokens  int `json:"totalTokens,omitempty"`
}

type PlanEntry struct {
	Content  string         `json:"content"`
	Priority string         `json:"priority"`
	Status   string         `json:"status"`
	Metadata map[string]any `json:"_meta,omitempty"`
}

type SessionConfigOption struct {
	ID           string                      `json:"id"`
	Name         string                      `json:"name"`
	Description  string                      `json:"description,omitempty"`
	Category     string                      `json:"category,omitempty"`
	Type         string                      `json:"type"`
	CurrentValue any                         `json:"currentValue"`
	Options      []SessionConfigSelectOption `json:"options,omitempty"`
	Groups       []SessionConfigSelectGroup  `json:"groups,omitempty"`
	Metadata     map[string]any              `json:"_meta,omitempty"`
}

type SessionConfigSelectOption struct {
	Value       string         `json:"value"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Metadata    map[string]any `json:"_meta,omitempty"`
}

type SessionConfigSelectGroup struct {
	Group    string                      `json:"group"`
	Name     string                      `json:"name"`
	Options  []SessionConfigSelectOption `json:"options"`
	Metadata map[string]any              `json:"_meta,omitempty"`
}

type SessionMode struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type StreamError struct {
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}

type PermissionRequest struct {
	ID        string             `json:"id"`
	SessionID string             `json:"sessionId,omitempty"`
	ToolCall  PermissionToolCall `json:"toolCall"`
	Options   []PermissionOption `json:"options"`
	Metadata  map[string]any     `json:"metadata,omitempty"`
}

type PermissionToolCall struct {
	ToolCallID string             `json:"toolCallId,omitempty"`
	Title      string             `json:"title,omitempty"`
	Kind       string             `json:"kind,omitempty"`
	Status     string             `json:"status,omitempty"`
	Locations  []ToolCallLocation `json:"locations,omitempty"`
	RawInput   any                `json:"rawInput,omitempty"`
	RawOutput  any                `json:"rawOutput,omitempty"`
}

type PermissionOption struct {
	OptionID string         `json:"optionId"`
	Name     string         `json:"name"`
	Kind     string         `json:"kind"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type PermissionVote struct {
	ID       string
	Outcome  string
	OptionID string
}
