// Package persist provides pluggable session persistence for the agent SDK.
//
// Usage:
//
//	store := persist.NewFileStore("/var/lib/agent/sessions")
//	rt, err := api.New(ctx, api.Options{
//	    SessionStore: store,
//	    // HistoryLoader / HistorySaver 由 SessionStore 自动注入，无需手动设置
//	})
package persist

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"icoo_claw/common/core/agent_sdk/api"
	"icoo_claw/common/core/agent_sdk/message"
)

// ─── 核心接口 ──────────────────────────────────────────────────────────────────

// SessionStore 定义会话持久化的完整契约。
type SessionStore = api.SessionStore

// SessionMeta 描述会话的轻量级元数据，用于列表展示、排序、过期清理等场景。
type SessionMeta = api.SessionMeta

// DeferredStateStore 持久化工具延迟激活状态（哪个 session 已激活哪些工具）。
type DeferredStateStore = api.DeferredStateStore

// Store 组合了会话历史和工具状态的完整持久化接口。
type Store = api.Store

// ─── 文件系统实现 ──────────────────────────────────────────────────────────────

const (
	historyFileName  = "history.jsonl"
	metaFileName     = "meta.json"
	deferredFileName = "deferred.json"

	defaultDirPerm  = 0o700
	defaultFilePerm = 0o600
)

// FileStore 将会话历史以 JSONL 格式落盘至 baseDir/<sessionID>/ 目录。
// 所有公共方法均为并发安全。
type FileStore struct {
	baseDir string
	mu      sync.RWMutex // 保护元数据缓存
	metaCache map[string]SessionMeta
}

// FileStoreOption 配置 FileStore。
type FileStoreOption func(*FileStore)

// NewFileStore 创建文件系统持久化后端。
// baseDir 目录不存在时会自动创建。
func NewFileStore(baseDir string, opts ...FileStoreOption) (*FileStore, error) {
	baseDir = filepath.Clean(strings.TrimSpace(baseDir))
	if baseDir == "" {
		return nil, errors.New("persist: baseDir cannot be empty")
	}
	if err := os.MkdirAll(baseDir, defaultDirPerm); err != nil {
		return nil, fmt.Errorf("persist: create base dir %q: %w", baseDir, err)
	}
	s := &FileStore{
		baseDir:   baseDir,
		metaCache: make(map[string]SessionMeta),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s, nil
}

// ── SessionStore 实现 ────────────────────────────────────────────────────────

func (s *FileStore) LoadHistory(ctx context.Context, sessionID string) ([]message.Message, error) {
	if err := validateSessionID(sessionID); err != nil {
		return nil, err
	}
	path := s.historyPath(sessionID)
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil // 会话不存在，返回空
		}
		return nil, fmt.Errorf("persist: open history %q: %w", path, err)
	}
	defer f.Close()

	var msgs []message.Message
	dec := json.NewDecoder(f)
	for dec.More() {
		var msg message.Message
		if err := dec.Decode(&msg); err != nil {
			return nil, fmt.Errorf("persist: decode history %q: %w", path, err)
		}
		msgs = append(msgs, msg)
	}
	return msgs, nil
}

func (s *FileStore) SaveHistory(ctx context.Context, sessionID string, msgs []message.Message) error {
	if err := validateSessionID(sessionID); err != nil {
		return err
	}
	if err := s.ensureSessionDir(sessionID); err != nil {
		return err
	}
	path := s.historyPath(sessionID)
	// 原子写：先写临时文件再重命名
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, defaultFilePerm)
	if err != nil {
		return fmt.Errorf("persist: create temp history %q: %w", tmp, err)
	}
	enc := json.NewEncoder(f)
	for _, msg := range msgs {
		if err := enc.Encode(msg); err != nil {
			f.Close()
			os.Remove(tmp)
			return fmt.Errorf("persist: encode history: %w", err)
		}
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("persist: flush history %q: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("persist: commit history %q: %w", path, err)
	}
	return s.updateMeta(ctx, sessionID, len(msgs))
}

func (s *FileStore) AppendMessages(ctx context.Context, sessionID string, msgs []message.Message) error {
	if err := validateSessionID(sessionID); err != nil {
		return err
	}
	if len(msgs) == 0 {
		return nil
	}
	if err := s.ensureSessionDir(sessionID); err != nil {
		return err
	}
	path := s.historyPath(sessionID)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, defaultFilePerm)
	if err != nil {
		return fmt.Errorf("persist: open history for append %q: %w", path, err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	for _, msg := range msgs {
		if err := enc.Encode(msg); err != nil {
			return fmt.Errorf("persist: append message: %w", err)
		}
	}
	// 更新元数据（消息数增量计算）
	meta, _ := s.SessionMeta(ctx, sessionID)
	return s.updateMeta(ctx, sessionID, meta.MessageCount+len(msgs))
}

func (s *FileStore) DeleteSession(ctx context.Context, sessionID string) error {
	if err := validateSessionID(sessionID); err != nil {
		return err
	}
	dir := s.sessionDir(sessionID)
	if err := os.RemoveAll(dir); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("persist: delete session %q: %w", sessionID, err)
	}
	s.mu.Lock()
	delete(s.metaCache, sessionID)
	s.mu.Unlock()
	return nil
}

func (s *FileStore) ListSessions(ctx context.Context) ([]string, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("persist: list sessions: %w", err)
	}
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			ids = append(ids, entry.Name())
		}
	}
	sort.Strings(ids)
	return ids, nil
}

func (s *FileStore) SessionMeta(ctx context.Context, sessionID string) (SessionMeta, error) {
	if err := validateSessionID(sessionID); err != nil {
		return SessionMeta{}, err
	}
	// 先查缓存
	s.mu.RLock()
	if meta, ok := s.metaCache[sessionID]; ok {
		s.mu.RUnlock()
		return meta, nil
	}
	s.mu.RUnlock()

	// 从磁盘读取
	path := filepath.Join(s.sessionDir(sessionID), metaFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return SessionMeta{}, nil
		}
		return SessionMeta{}, fmt.Errorf("persist: read meta %q: %w", path, err)
	}
	var meta SessionMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return SessionMeta{}, fmt.Errorf("persist: decode meta %q: %w", path, err)
	}
	// 写回缓存
	s.mu.Lock()
	s.metaCache[sessionID] = meta
	s.mu.Unlock()
	return meta, nil
}

// ── DeferredStateStore 实现 ──────────────────────────────────────────────────

func (s *FileStore) LoadDeferredState(ctx context.Context, sessionID string) ([]string, error) {
	if err := validateSessionID(sessionID); err != nil {
		return nil, err
	}
	path := filepath.Join(s.sessionDir(sessionID), deferredFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("persist: read deferred state %q: %w", path, err)
	}
	var tools []string
	if err := json.Unmarshal(data, &tools); err != nil {
		return nil, fmt.Errorf("persist: decode deferred state %q: %w", path, err)
	}
	return tools, nil
}

func (s *FileStore) SaveDeferredState(ctx context.Context, sessionID string, activeTools []string) error {
	if err := validateSessionID(sessionID); err != nil {
		return err
	}
	if err := s.ensureSessionDir(sessionID); err != nil {
		return err
	}
	path := filepath.Join(s.sessionDir(sessionID), deferredFileName)
	data, err := json.Marshal(activeTools)
	if err != nil {
		return fmt.Errorf("persist: encode deferred state: %w", err)
	}
	if err := os.WriteFile(path, data, defaultFilePerm); err != nil {
		return fmt.Errorf("persist: write deferred state %q: %w", path, err)
	}
	return nil
}

func (s *FileStore) DeleteDeferredState(ctx context.Context, sessionID string) error {
	if err := validateSessionID(sessionID); err != nil {
		return err
	}
	path := filepath.Join(s.sessionDir(sessionID), deferredFileName)
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("persist: delete deferred state %q: %w", path, err)
	}
	return nil
}

// ── 清理工具 ────────────────────────────────────────────────────────────────

// PurgeOlderThan 删除所有最后更新时间早于 cutoff 的会话。
// 用于定期清理过期数据。返回删除的会话数和首个错误。
func (s *FileStore) PurgeOlderThan(ctx context.Context, cutoff time.Time) (int, error) {
	ids, err := s.ListSessions(ctx)
	if err != nil {
		return 0, err
	}
	var firstErr error
	deleted := 0
	for _, id := range ids {
		meta, err := s.SessionMeta(ctx, id)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if meta.UpdatedAt.IsZero() || meta.UpdatedAt.Before(cutoff) {
			if err := s.DeleteSession(ctx, id); err != nil {
				if firstErr == nil {
					firstErr = err
				}
			} else {
				deleted++
			}
		}
	}
	return deleted, firstErr
}

// ── 内部辅助 ─────────────────────────────────────────────────────────────────

func (s *FileStore) sessionDir(sessionID string) string {
	return filepath.Join(s.baseDir, sanitizeID(sessionID))
}

func (s *FileStore) historyPath(sessionID string) string {
	return filepath.Join(s.sessionDir(sessionID), historyFileName)
}

func (s *FileStore) ensureSessionDir(sessionID string) error {
	dir := s.sessionDir(sessionID)
	if err := os.MkdirAll(dir, defaultDirPerm); err != nil {
		return fmt.Errorf("persist: create session dir %q: %w", dir, err)
	}
	return nil
}

func (s *FileStore) updateMeta(ctx context.Context, sessionID string, msgCount int) error {
	dir := s.sessionDir(sessionID)
	path := filepath.Join(dir, metaFileName)

	var existing SessionMeta
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &existing)
	}

	now := time.Now().UTC()
	meta := SessionMeta{
		SessionID:    sessionID,
		MessageCount: msgCount,
		UpdatedAt:    now,
		Summary:      existing.Summary,
	}
	if existing.CreatedAt.IsZero() {
		meta.CreatedAt = now
	} else {
		meta.CreatedAt = existing.CreatedAt
	}
	// 计算 history 文件大小
	if info, err := os.Stat(filepath.Join(dir, historyFileName)); err == nil {
		meta.SizeBytes = info.Size()
	}

	data, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("persist: encode meta: %w", err)
	}
	if err := os.WriteFile(path, data, defaultFilePerm); err != nil {
		return fmt.Errorf("persist: write meta %q: %w", path, err)
	}
	s.mu.Lock()
	s.metaCache[sessionID] = meta
	s.mu.Unlock()
	return nil
}

// UpdateSessionSummary 更新指定会话的长文本摘要。
func (s *FileStore) UpdateSessionSummary(ctx context.Context, sessionID string, summary string) error {
	if err := validateSessionID(sessionID); err != nil {
		return err
	}
	if err := s.ensureSessionDir(sessionID); err != nil {
		return err
	}
	dir := s.sessionDir(sessionID)
	path := filepath.Join(dir, metaFileName)

	var existing SessionMeta
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &existing)
	}

	now := time.Now().UTC()
	meta := SessionMeta{
		SessionID:    sessionID,
		MessageCount: existing.MessageCount,
		UpdatedAt:    now,
		Summary:      summary,
	}
	if existing.CreatedAt.IsZero() {
		meta.CreatedAt = now
	} else {
		meta.CreatedAt = existing.CreatedAt
	}
	if info, err := os.Stat(filepath.Join(dir, historyFileName)); err == nil {
		meta.SizeBytes = info.Size()
	}

	data, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("persist: encode meta: %w", err)
	}
	if err := os.WriteFile(path, data, defaultFilePerm); err != nil {
		return fmt.Errorf("persist: write meta %q: %w", path, err)
	}

	s.mu.Lock()
	s.metaCache[sessionID] = meta
	s.mu.Unlock()
	return nil
}

func validateSessionID(id string) error {
	if strings.TrimSpace(id) == "" {
		return errors.New("persist: sessionID cannot be empty")
	}
	return nil
}

// sanitizeID 将 sessionID 转换为安全的文件系统路径组件。
// 与 api 包中的 sanitizePathComponent 逻辑对齐。
func sanitizeID(id string) string {
	const fallback = "default"
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return fallback
	}
	var b strings.Builder
	b.Grow(len(trimmed))
	for _, r := range trimmed {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	sanitized := strings.Trim(b.String(), "-")
	if sanitized == "" {
		return fallback
	}
	return sanitized
}
