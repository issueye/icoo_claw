package dto

import "time"

type SyncEvent struct {
	ID             string         `json:"id"`
	Time           time.Time      `json:"time"`
	Source         string         `json:"source"`
	Protocol       string         `json:"protocol"`
	Direction      string         `json:"direction"`
	Type           string         `json:"type"`
	ConversationID string         `json:"conversation_id,omitempty"`
	SessionID      string         `json:"session_id,omitempty"`
	RequestID      string         `json:"request_id,omitempty"`
	Payload        any            `json:"payload,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}
