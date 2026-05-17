package model

import "time"

type Session struct {
	SessionID string
	UserID    string
	AgentID   string
	Title     string
	Status    string
	Metadata  map[string]any
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
	Offset int
	Limit  int
}
