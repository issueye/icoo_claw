package model

import "time"

type Session struct {
	SessionID string
	UserID    string
	AgentID   string
	Title     string
	Status    string
	Metadata  map[string]any
	Revision  int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Message struct {
	ID            string
	Role          string
	Content       string
	ContentBlocks []any
	ToolCalls     []any
	Metadata      map[string]any
	CreatedAt     time.Time
}

type MessagePage struct {
	Offset   int
	Limit    int
	Tail     int
	BeforeID string
	AfterID  string
}

type MessageList struct {
	Messages []Message
	Revision int64
}

type Run struct {
	ID          string
	RequestID   string
	Status      string
	StopReason  string
	Error       string
	Usage       map[string]any
	Metadata    map[string]any
	StartedAt   time.Time
	CompletedAt *time.Time
}

type RunPage struct {
	Offset int
	Limit  int
}

type RunEvent struct {
	ID        string
	RunID     string
	Type      string
	Sequence  int64
	Payload   map[string]any
	Metadata  map[string]any
	CreatedAt time.Time
}

type RunEventPage struct {
	Offset int
	Limit  int
}

type SessionListFilter struct {
	UserID  string
	AgentID string
	Status  string
	Offset  int
	Limit   int
}

type SessionList struct {
	Sessions []Session
}
