package dto

type ChatWSRequest struct {
	Type           string         `json:"type"`
	ConversationID string         `json:"conversation_id,omitempty"`
	Prompt         string         `json:"prompt,omitempty"`
	RequestID      string         `json:"request_id,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

type ChatWSResponse struct {
	Type           string         `json:"type"`
	ConversationID string         `json:"conversation_id,omitempty"`
	SessionID      string         `json:"session_id,omitempty"`
	RequestID      string         `json:"request_id,omitempty"`
	Update         *SessionUpdate `json:"update,omitempty"`
	StopReason     string         `json:"stop_reason,omitempty"`
	Code           string         `json:"code,omitempty"`
	Error          string         `json:"error,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

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
	Type string `json:"type,omitempty"`
	Text string `json:"text,omitempty"`
	URI  string `json:"uri,omitempty"`
	Mime string `json:"mimeType,omitempty"`
	Data any    `json:"data,omitempty"`
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
