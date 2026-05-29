package persist

import (
	"context"
	"sync"

	"icoo_claw/common/core/agent_sdk/message"
)

// ─── 内存实现（用于测试） ────────────────────────────────────────────────────

// MemoryStore 是 Store 的内存实现，适用于单元测试和非持久化场景。
// 所有方法均为并发安全。
type MemoryStore struct {
	mu        sync.RWMutex
	history   map[string][]message.Message
	deferred  map[string][]string
	summaries map[string]string
}

// NewMemoryStore 创建一个空的内存持久化后端。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		history:   make(map[string][]message.Message),
		deferred:  make(map[string][]string),
		summaries: make(map[string]string),
	}
}

func (m *MemoryStore) LoadHistory(_ context.Context, sessionID string) ([]message.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	msgs := m.history[sessionID]
	if len(msgs) == 0 {
		return nil, nil
	}
	return message.CloneMessages(msgs), nil
}

func (m *MemoryStore) SaveHistory(_ context.Context, sessionID string, msgs []message.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(msgs) == 0 {
		delete(m.history, sessionID)
		return nil
	}
	m.history[sessionID] = message.CloneMessages(msgs)
	return nil
}

func (m *MemoryStore) AppendMessages(_ context.Context, sessionID string, msgs []message.Message) error {
	if len(msgs) == 0 {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	existing := m.history[sessionID]
	m.history[sessionID] = append(existing, message.CloneMessages(msgs)...)
	return nil
}

func (m *MemoryStore) DeleteSession(_ context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.history, sessionID)
	delete(m.deferred, sessionID)
	delete(m.summaries, sessionID)
	return nil
}

func (m *MemoryStore) ListSessions(_ context.Context) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ids := make([]string, 0, len(m.history))
	for id := range m.history {
		ids = append(ids, id)
	}
	return ids, nil
}

func (m *MemoryStore) SessionMeta(_ context.Context, sessionID string) (SessionMeta, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	msgs := m.history[sessionID]
	summary := m.summaries[sessionID]
	if len(msgs) == 0 && summary == "" {
		return SessionMeta{}, nil
	}
	return SessionMeta{
		SessionID:    sessionID,
		MessageCount: len(msgs),
		Summary:      summary,
	}, nil
}

func (m *MemoryStore) UpdateSessionSummary(_ context.Context, sessionID string, summary string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.summaries[sessionID] = summary
	return nil
}

func (m *MemoryStore) LoadDeferredState(_ context.Context, sessionID string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tools := m.deferred[sessionID]
	if len(tools) == 0 {
		return nil, nil
	}
	out := make([]string, len(tools))
	copy(out, tools)
	return out, nil
}

func (m *MemoryStore) SaveDeferredState(_ context.Context, sessionID string, activeTools []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(activeTools) == 0 {
		delete(m.deferred, sessionID)
		return nil
	}
	cp := make([]string, len(activeTools))
	copy(cp, activeTools)
	m.deferred[sessionID] = cp
	return nil
}

func (m *MemoryStore) DeleteDeferredState(_ context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.deferred, sessionID)
	return nil
}

// Reset 清空所有数据，便于测试复用。
func (m *MemoryStore) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.history = make(map[string][]message.Message)
	m.deferred = make(map[string][]string)
	m.summaries = make(map[string]string)
}
