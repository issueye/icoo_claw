package dto

type RunRequest struct {
	SessionID     string         `json:"session_id" binding:"required"`
	RequestID     string         `json:"request_id"`
	Prompt        string         `json:"prompt" binding:"required"`
	Agent         map[string]any `json:"agent"`
	ToolWhitelist []string       `json:"tool_whitelist"`
	ForceSkills   []string       `json:"force_skills"`
	Metadata      map[string]any `json:"metadata"`
}

type RunResponse struct {
	SessionID  string `json:"session_id"`
	RequestID  string `json:"request_id,omitempty"`
	Output     string `json:"output"`
	StopReason string `json:"stop_reason"`
}

type StreamEvent struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
	RequestID string `json:"request_id,omitempty"`
	Output    string `json:"output,omitempty"`
}
