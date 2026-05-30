package dto

import "icoo_claw/common/agentproto"

type ChatWSRequest struct {
	Type           string         `json:"type"`
	ConversationID string         `json:"conversation_id,omitempty"`
	Prompt         string         `json:"prompt,omitempty"`
	RequestID      string         `json:"request_id,omitempty"`
	ForceSkills    []string       `json:"force_skills,omitempty"`
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

type SessionUpdate = agentproto.SessionUpdate
type ContentBlock = agentproto.ContentBlock
type ToolCallLocation = agentproto.ToolCallLocation
type UsageUpdate = agentproto.UsageUpdate
