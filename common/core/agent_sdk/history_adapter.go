package agent_sdk

import (
	"context"
	"encoding/json"
	"fmt"
	sdkmessage "icoo_claw/common/core/agent_sdk/sdk/message"
	"sync"
	"time"
)

type HistoryMessage struct {
	ID            string
	Role          string
	Content       string
	ContentBlocks []any
	ToolCalls     []any
	Metadata      map[string]any
	CreatedAt     time.Time
}

type HistoryStore interface {
	ListMessages(ctx context.Context, sessionID string) ([]HistoryMessage, error)
	ReplaceMessages(ctx context.Context, sessionID string, messages []HistoryMessage) error
}

type RevisionedHistoryStore interface {
	ListMessagesWithRevision(ctx context.Context, sessionID string) ([]HistoryMessage, int64, error)
	ReplaceMessagesWithRevision(ctx context.Context, sessionID string, messages []HistoryMessage, expectedRevision *int64) error
}

type HistoryAdapter struct {
	store     HistoryStore
	mu        sync.Mutex
	revisions map[string]int64
}

func NewHistoryAdapter(store HistoryStore) *HistoryAdapter {
	return &HistoryAdapter{store: store, revisions: map[string]int64{}}
}

func (h *HistoryAdapter) Load(ctx context.Context, sessionID string) ([]sdkmessage.Message, error) {
	if h == nil || h.store == nil {
		return nil, nil
	}
	var messages []HistoryMessage
	if store, ok := h.store.(RevisionedHistoryStore); ok {
		loaded, revision, err := store.ListMessagesWithRevision(ctx, sessionID)
		if err != nil {
			return nil, err
		}
		messages = loaded
		h.mu.Lock()
		h.revisions[sessionID] = revision
		h.mu.Unlock()
	} else {
		loaded, err := h.store.ListMessages(ctx, sessionID)
		if err != nil {
			return nil, err
		}
		messages = loaded
	}
	return toSDKMessages(messages)
}

func (h *HistoryAdapter) SaveSnapshot(ctx context.Context, sessionID string, messages []sdkmessage.Message) error {
	if h == nil || h.store == nil {
		return nil
	}
	if store, ok := h.store.(RevisionedHistoryStore); ok {
		expected := h.expectedRevision(sessionID)
		return store.ReplaceMessagesWithRevision(ctx, sessionID, fromSDKMessages(messages), expected)
	}
	return h.store.ReplaceMessages(ctx, sessionID, fromSDKMessages(messages))
}

func (h *HistoryAdapter) expectedRevision(sessionID string) *int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	revision, ok := h.revisions[sessionID]
	if !ok {
		return nil
	}
	return &revision
}

func toSDKMessages(messages []HistoryMessage) ([]sdkmessage.Message, error) {
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

func fromSDKMessages(messages []sdkmessage.Message) []HistoryMessage {
	out := make([]HistoryMessage, len(messages))
	for i, msg := range messages {
		out[i] = HistoryMessage{
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
