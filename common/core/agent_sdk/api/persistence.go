package api

import (
	"context"
	"time"

	"icoo_claw/common/core/agent_sdk/message"
)

// SessionMeta 描述会话的轻量级元数据，用于列表展示、排序、过期清理以及展示会话摘要等场景。
type SessionMeta struct {
	SessionID    string    `json:"session_id"`
	MessageCount int       `json:"message_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	SizeBytes    int64     `json:"size_bytes"`
	Summary      string    `json:"summary,omitempty"` // 会话的长文本摘要
}

// SessionStore 定义会话持久化的完整契约。
// 实现者可以选择文件系统、数据库、Redis 等任意后端。
type SessionStore interface {
	// LoadHistory 加载指定会话的消息历史。
	// 若会话不存在应返回 (nil, nil)，而非错误。
	LoadHistory(ctx context.Context, sessionID string) ([]message.Message, error)

	// SaveHistory 持久化会话的完整消息历史（全量覆写）。
	SaveHistory(ctx context.Context, sessionID string, msgs []message.Message) error

	// AppendMessages 将新消息追加到已有持久化历史（增量写入）。
	AppendMessages(ctx context.Context, sessionID string, msgs []message.Message) error

	// DeleteSession 删除会话的所有持久化数据。
	DeleteSession(ctx context.Context, sessionID string) error

	// ListSessions 返回所有已持久化的会话 ID。
	ListSessions(ctx context.Context) ([]string, error)

	// SessionMeta 返回会话的元数据（无需加载完整消息）。
	SessionMeta(ctx context.Context, sessionID string) (SessionMeta, error)

	// UpdateSessionSummary 更新指定会话的摘要信息。
	UpdateSessionSummary(ctx context.Context, sessionID string, summary string) error
}

// DeferredStateStore 持久化工具延迟激活状态（哪个 session 已激活哪些工具）。
type DeferredStateStore interface {
	// LoadDeferredState 加载指定会话已激活的工具集合。
	LoadDeferredState(ctx context.Context, sessionID string) ([]string, error)

	// SaveDeferredState 保存指定会话已激活的工具集合。
	SaveDeferredState(ctx context.Context, sessionID string, activeTools []string) error

	// DeleteDeferredState 删除会话的工具激活状态（随会话删除同步调用）。
	DeleteDeferredState(ctx context.Context, sessionID string) error
}

// Store 组合了会话历史和工具状态的完整持久化接口。
type Store interface {
	SessionStore
	DeferredStateStore
}
