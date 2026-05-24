package agent_sdk

import (
	"context"
	"encoding/json"
)

type Runner interface {
	Run(ctx context.Context, req RunRequest) (*RunResponse, error)
	RunStream(ctx context.Context, req RunRequest) (<-chan StreamEvent, error)
}

type RunRequest struct {
	SessionID     string
	RequestID     string
	Prompt        string
	Agent         map[string]any
	ToolWhitelist []string
	ForceSkills   []string
	Metadata      map[string]any
}

type RunResponse struct {
	SessionID  string
	RequestID  string
	Output     string
	StopReason string
}

type StreamEvent struct {
	Type       string         `json:"type"`
	SessionID  string         `json:"session_id"`
	RequestID  string         `json:"request_id,omitempty"`
	Update     *SessionUpdate `json:"update,omitempty"`
	StopReason string         `json:"stop_reason,omitempty"`
	Error      *StreamError   `json:"error,omitempty"`
}

const (
	StreamEventSessionUpdate    = "session/update"
	StreamEventSessionCompleted = "session/completed"
	StreamEventSessionError     = "session/error"
)

type SessionUpdate struct {
	SessionUpdate string             `json:"sessionUpdate"`
	Content       *ContentBlock      `json:"content,omitempty"`
	MessageID     string             `json:"messageId,omitempty"`
	ToolCallID    string             `json:"toolCallId,omitempty"`
	Title         string             `json:"title,omitempty"`
	Kind          string             `json:"kind,omitempty"`
	Status        string             `json:"status,omitempty"`
	Locations     []ToolCallLocation `json:"locations,omitempty"`
	RawInput      any                `json:"rawInput,omitempty"`
	RawOutput     any                `json:"rawOutput,omitempty"`
	Usage         *UsageUpdate       `json:"usage,omitempty"`
}

type ContentBlock struct {
	Type  string          `json:"type,omitempty"`
	Text  string          `json:"text,omitempty"`
	URI   string          `json:"uri,omitempty"`
	Mime  string          `json:"mimeType,omitempty"`
	Data  json.RawMessage `json:"data,omitempty"`
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

type StreamError struct {
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}
