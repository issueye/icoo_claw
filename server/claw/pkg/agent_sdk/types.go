package agent_sdk

import "context"

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
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
	RequestID string `json:"request_id,omitempty"`
	Output    string `json:"output,omitempty"`
}
