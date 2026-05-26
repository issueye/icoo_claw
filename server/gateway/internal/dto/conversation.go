package dto

import "time"

type Conversation struct {
	ID            string     `json:"id"`
	SessionID     string     `json:"session_id"`
	AgentID       string     `json:"agent_id"`
	UserID        string     `json:"user_id,omitempty"`
	Title         string     `json:"title"`
	Status        string     `json:"status"`
	LastMessageAt *time.Time `json:"last_message_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type CreateConversationRequest struct {
	AgentID string `json:"agent_id" binding:"required"`
	UserID  string `json:"user_id"`
	Title   string `json:"title"`
}

type SendMessageRequest struct {
	Prompt    string         `json:"prompt" binding:"required"`
	RequestID string         `json:"request_id"`
	Metadata  map[string]any `json:"metadata"`
}

type ChatResponse struct {
	ConversationID string `json:"conversation_id"`
	SessionID      string `json:"session_id"`
	RequestID      string `json:"request_id,omitempty"`
	Output         string `json:"output"`
	StopReason     string `json:"stop_reason"`
}

type ConversationMessagesResponse struct {
	Messages []SessionMessage `json:"messages"`
}

type SessionMessage struct {
	ID            string         `json:"id"`
	Role          string         `json:"role"`
	Content       string         `json:"content"`
	ContentBlocks []any          `json:"content_blocks,omitempty"`
	ToolCalls     []any          `json:"tool_calls,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
}
