package agentproto

import (
	"errors"
	"strings"
)

type CollectedStream struct {
	Output     string
	StopReason string
	SessionID  string
	RequestID  string
}

func CollectTextStream(events <-chan StreamEvent, fallbackSessionID string, fallbackRequestID string) (*CollectedStream, error) {
	var output strings.Builder
	result := &CollectedStream{
		StopReason: "stream_closed",
		SessionID:  fallbackSessionID,
		RequestID:  fallbackRequestID,
	}
	completed := false

	handler := StreamEventHandlerFunc{
		OnUpdate: func(event StreamEvent) error {
			if event.Update != nil && event.Update.SessionUpdate == "agent_message_chunk" && event.Update.Content != nil {
				output.WriteString(event.Update.Content.Text)
			}
			return nil
		},
		OnCompleted: func(event StreamEvent) error {
			result.StopReason = defaultString(event.StopReason, "end_turn")
			completed = true
			return nil
		},
		OnError: func(event StreamEvent) error {
			message := ""
			if event.Error != nil {
				message = event.Error.Message
			}
			return errors.New(defaultString(message, "stream error"))
		},
	}

	for event := range events {
		if event.SessionID != "" {
			result.SessionID = event.SessionID
		}
		if event.RequestID != "" {
			result.RequestID = event.RequestID
		}
		if err := DispatchStreamEvent(event, handler); err != nil {
			result.Output = output.String()
			return result, err
		}
	}

	result.Output = output.String()
	if !completed {
		return result, errors.New("agent stream closed before completion")
	}
	return result, nil
}

func defaultString(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
