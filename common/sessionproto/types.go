package sessionproto

import "time"

type Session struct {
	SessionID string         `json:"session_id"`
	UserID    string         `json:"user_id,omitempty"`
	AgentID   string         `json:"agent_id,omitempty"`
	Title     string         `json:"title,omitempty"`
	Status    string         `json:"status"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	Revision  int64          `json:"revision"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type CreateSessionRequest struct {
	SessionID string         `json:"session_id"`
	UserID    string         `json:"user_id"`
	AgentID   string         `json:"agent_id"`
	Title     string         `json:"title"`
	Metadata  map[string]any `json:"metadata"`
}

type UpdateSessionRequest struct {
	Title    *string        `json:"title"`
	Status   *string        `json:"status"`
	Metadata map[string]any `json:"metadata"`
}

type ListSessionsOptions struct {
	UserID  string
	AgentID string
	Status  string
	Offset  int
	Limit   int
}

type SessionsResponse struct {
	Sessions []Session `json:"sessions"`
}

type Message struct {
	ID            string         `json:"id"`
	Role          string         `json:"role" binding:"required"`
	Content       string         `json:"content"`
	ContentBlocks []any          `json:"content_blocks,omitempty"`
	ToolCalls     []any          `json:"tool_calls,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
}

type MessagesRequest struct {
	Messages         []Message `json:"messages" binding:"required"`
	ExpectedRevision *int64    `json:"expected_revision,omitempty"`
}

type MessagesResponse struct {
	Messages []Message `json:"messages"`
	Revision int64     `json:"revision"`
}

type Run struct {
	ID          string         `json:"id"`
	RequestID   string         `json:"request_id,omitempty"`
	Status      string         `json:"status"`
	StopReason  string         `json:"stop_reason,omitempty"`
	Error       string         `json:"error,omitempty"`
	Usage       map[string]any `json:"usage,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	StartedAt   time.Time      `json:"started_at"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
}

type RunsRequest struct {
	Runs []Run `json:"runs" binding:"required"`
}

type RunsResponse struct {
	Runs []Run `json:"runs"`
}

type RunEvent struct {
	ID        string         `json:"id"`
	RunID     string         `json:"run_id,omitempty"`
	Type      string         `json:"type" binding:"required"`
	Sequence  int64          `json:"sequence"`
	Payload   map[string]any `json:"payload,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

type RunEventsRequest struct {
	Events []RunEvent `json:"events" binding:"required"`
}

type RunEventsResponse struct {
	Events []RunEvent `json:"events"`
}
