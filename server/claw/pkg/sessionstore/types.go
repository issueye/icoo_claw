package sessionstore

import "time"

type Session struct {
	SessionID string         `json:"session_id"`
	UserID    string         `json:"user_id,omitempty"`
	AgentID   string         `json:"agent_id,omitempty"`
	Title     string         `json:"title,omitempty"`
	Status    string         `json:"status"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type CreateSessionRequest struct {
	SessionID string         `json:"session_id"`
	UserID    string         `json:"user_id,omitempty"`
	AgentID   string         `json:"agent_id,omitempty"`
	Title     string         `json:"title,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type Message struct {
	ID            string         `json:"id"`
	Role          string         `json:"role"`
	Content       string         `json:"content"`
	ContentBlocks []any          `json:"content_blocks,omitempty"`
	ToolCalls     []any          `json:"tool_calls,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
}

type MessagesRequest struct {
	Messages []Message `json:"messages"`
}

type MessagesResponse struct {
	Messages []Message `json:"messages"`
}
