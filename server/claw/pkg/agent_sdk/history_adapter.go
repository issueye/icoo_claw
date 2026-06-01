package agent_sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	sdkmessage "icoo_claw/common/core/agent_sdk/message"
	"icoo_claw/server/claw/pkg/sessionstore"
)

type HistoryStore interface {
	ListMessages(ctx context.Context, sessionID string) ([]sessionstore.Message, error)
	ReplaceMessages(ctx context.Context, sessionID string, messages []sessionstore.Message) error
}

type RevisionedHistoryStore interface {
	ListMessagesWithRevision(ctx context.Context, sessionID string) ([]sessionstore.Message, int64, error)
	ReplaceMessagesWithRevision(ctx context.Context, sessionID string, messages []sessionstore.Message, expectedRevision *int64) error
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
	var messages []sessionstore.Message
	if store, ok := h.store.(RevisionedHistoryStore); ok {
		loaded, revision, err := store.ListMessagesWithRevision(ctx, sessionID)
		if err != nil {
			if isHistoryNotFound(err) {
				h.mu.Lock()
				h.revisions[sessionID] = 0
				h.mu.Unlock()
				return nil, nil
			}
			return nil, err
		}
		messages = loaded
		h.mu.Lock()
		h.revisions[sessionID] = revision
		h.mu.Unlock()
	} else {
		loaded, err := h.store.ListMessages(ctx, sessionID)
		if err != nil {
			if isHistoryNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		messages = loaded
	}
	return toSDKMessages(messages)
}

func isHistoryNotFound(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "status 404") ||
		strings.Contains(message, "not_found") ||
		strings.Contains(message, "not found")
}

func (h *HistoryAdapter) SaveSnapshot(ctx context.Context, sessionID string, messages []sdkmessage.Message) error {
	if h == nil || h.store == nil {
		return nil
	}
	if store, ok := h.store.(RevisionedHistoryStore); ok {
		expected := h.expectedRevision(sessionID)
		if err := store.ReplaceMessagesWithRevision(ctx, sessionID, fromSDKMessages(messages), expected); err != nil {
			if isHistoryNotFound(err) {
				return nil
			}
			return err
		}
		return nil
	}
	if err := h.store.ReplaceMessages(ctx, sessionID, fromSDKMessages(messages)); err != nil {
		if isHistoryNotFound(err) {
			return nil
		}
		return err
	}
	return nil
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
