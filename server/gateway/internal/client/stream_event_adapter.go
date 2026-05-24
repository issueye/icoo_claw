package client

import (
	"encoding/json"
	"strings"

	agent_sdk "icoo_claw/server/claw/pkg/agent_sdk"
)

func (e StreamEvent) ToAgentSDK() agent_sdk.StreamEvent {
	return agent_sdk.StreamEvent{
		Type:       e.Type,
		SessionID:  e.SessionID,
		RequestID:  e.RequestID,
		Update:     e.Update.ToAgentSDK(),
		StopReason: e.StopReason,
		Error:      e.Error.ToAgentSDK(),
	}
}

func (u *SessionUpdate) ToAgentSDK() *agent_sdk.SessionUpdate {
	if u == nil {
		return nil
	}
	return &agent_sdk.SessionUpdate{
		SessionUpdate: u.SessionUpdate,
		Content:       u.Content.ToAgentSDK(),
		MessageID:     u.MessageID,
		ToolCallID:    u.ToolCallID,
		Title:         u.Title,
		Kind:          u.Kind,
		Status:        u.Status,
		Locations:     toAgentSDKLocations(u.Locations),
		RawInput:      u.RawInput,
		RawOutput:     u.RawOutput,
		Usage:         u.Usage.ToAgentSDK(),
	}
}

func (c *ContentBlock) ToAgentSDK() *agent_sdk.ContentBlock {
	if c == nil {
		return nil
	}
	return &agent_sdk.ContentBlock{
		Type: c.Type,
		Text: c.Text,
		URI:  c.URI,
		Mime: c.Mime,
		Data: toAgentSDKRawMessage(c.Data),
	}
}

func (l *ToolCallLocation) ToAgentSDK() agent_sdk.ToolCallLocation {
	if l == nil {
		return agent_sdk.ToolCallLocation{}
	}
	return agent_sdk.ToolCallLocation{Path: l.Path, Line: l.Line}
}

func (u *UsageUpdate) ToAgentSDK() *agent_sdk.UsageUpdate {
	if u == nil {
		return nil
	}
	return &agent_sdk.UsageUpdate{
		InputTokens:  u.InputTokens,
		OutputTokens: u.OutputTokens,
		TotalTokens:  u.TotalTokens,
	}
}

func (e *StreamError) ToAgentSDK() *agent_sdk.StreamError {
	if e == nil {
		return nil
	}
	return &agent_sdk.StreamError{
		Message: e.Message,
		Code:    e.Code,
	}
}

func toAgentSDKLocations(locations []ToolCallLocation) []agent_sdk.ToolCallLocation {
	if len(locations) == 0 {
		return nil
	}
	out := make([]agent_sdk.ToolCallLocation, len(locations))
	for i, location := range locations {
		out[i] = location.ToAgentSDK()
	}
	return out
}

func toAgentSDKRawMessage(value any) json.RawMessage {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case json.RawMessage:
		return append(json.RawMessage(nil), v...)
	case []byte:
		return append(json.RawMessage(nil), v...)
	case string:
		if strings.TrimSpace(v) == "" {
			return nil
		}
		return json.RawMessage([]byte(v))
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return nil
		}
		return json.RawMessage(data)
	}
}
