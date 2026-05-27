package dto

import "icoo_claw/server/claw/pkg/agent_sdk"

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

type SessionUpdate = agent_sdk.SessionUpdate
type ContentBlock = agent_sdk.ContentBlock
type ToolCallLocation = agent_sdk.ToolCallLocation
type UsageUpdate = agent_sdk.UsageUpdate
