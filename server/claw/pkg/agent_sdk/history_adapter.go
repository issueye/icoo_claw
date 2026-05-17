package agent_sdk

import (
	"context"
	"encoding/json"
	"fmt"

	"icoo_claw/server/claw/pkg/sessionstore"

	sdkmessage "github.com/stellarlinkco/agentsdk-go/pkg/message"
)

type HistoryStore interface {
	ListMessages(ctx context.Context, sessionID string) ([]sessionstore.Message, error)
	ReplaceMessages(ctx context.Context, sessionID string, messages []sessionstore.Message) error
}

type HistoryAdapter struct {
	store HistoryStore
}

func NewHistoryAdapter(store HistoryStore) *HistoryAdapter {
	return &HistoryAdapter{store: store}
}

func (h *HistoryAdapter) Load(ctx context.Context, sessionID string) ([]sdkmessage.Message, error) {
	if h == nil || h.store == nil {
		return nil, nil
	}
	messages, err := h.store.ListMessages(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return toSDKMessages(messages)
}

func (h *HistoryAdapter) SaveSnapshot(ctx context.Context, sessionID string, messages []sdkmessage.Message) error {
	if h == nil || h.store == nil {
		return nil
	}
	return h.store.ReplaceMessages(ctx, sessionID, fromSDKMessages(messages))
}

func toSDKMessages(messages []sessionstore.Message) ([]sdkmessage.Message, error) {
	out := make([]sdkmessage.Message, len(messages))
	for i, msg := range messages {
		toolCalls, err := convertSlice[sdkmessage.ToolCall](msg.ToolCalls)
		if err != nil {
			return nil, fmt.Errorf("convert tool calls: %w", err)
		}
		contentBlocks, err := convertSlice[sdkmessage.ContentBlock](msg.ContentBlocks)
		if err != nil {
			return nil, fmt.Errorf("convert content blocks: %w", err)
		}
		out[i] = sdkmessage.Message{
			Role:          msg.Role,
			Content:       msg.Content,
			ContentBlocks: contentBlocks,
			ToolCalls:     toolCalls,
			Metadata:      msg.Metadata,
		}
	}
	return out, nil
}

func fromSDKMessages(messages []sdkmessage.Message) []sessionstore.Message {
	out := make([]sessionstore.Message, len(messages))
	for i, msg := range messages {
		out[i] = sessionstore.Message{
			Role:          msg.Role,
			Content:       msg.Content,
			ContentBlocks: toAnySlice(msg.ContentBlocks),
			ToolCalls:     toAnySlice(msg.ToolCalls),
			Metadata:      msg.Metadata,
		}
	}
	return out
}

func convertSlice[T any](input []any) ([]T, error) {
	if len(input) == 0 {
		return nil, nil
	}
	payload, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	var out []T
	if err := json.Unmarshal(payload, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func toAnySlice[T any](input []T) []any {
	if len(input) == 0 {
		return nil
	}
	out := make([]any, len(input))
	for i, item := range input {
		out[i] = item
	}
	return out
}
