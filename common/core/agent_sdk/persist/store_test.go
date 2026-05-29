package persist_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"icoo_claw/common/core/agent_sdk/message"
	"icoo_claw/common/core/agent_sdk/persist"
)

// ─── 通用测试套件（对 Store 接口的任意实现均适用）────────────────────────────

func runStoreTests(t *testing.T, store persist.Store) {
	t.Helper()
	ctx := context.Background()

	t.Run("加载不存在的会话返回空", func(t *testing.T) {
		msgs, err := store.LoadHistory(ctx, "ghost-session")
		if err != nil {
			t.Fatalf("LoadHistory 返回错误: %v", err)
		}
		if len(msgs) != 0 {
			t.Fatalf("期望空 slice，得到 %d 条消息", len(msgs))
		}
	})

	t.Run("保存和加载消息历史", func(t *testing.T) {
		sid := "test-session-1"
		in := []message.Message{
			{Role: "user", Content: "你好"},
			{Role: "assistant", Content: "你好！有什么可以帮你的？"},
		}
		if err := store.SaveHistory(ctx, sid, in); err != nil {
			t.Fatalf("SaveHistory 错误: %v", err)
		}
		out, err := store.LoadHistory(ctx, sid)
		if err != nil {
			t.Fatalf("LoadHistory 错误: %v", err)
		}
		if len(out) != len(in) {
			t.Fatalf("期望 %d 条，得到 %d 条", len(in), len(out))
		}
		for i, msg := range out {
			if msg.Role != in[i].Role || msg.Content != in[i].Content {
				t.Errorf("第 %d 条消息不匹配: %+v vs %+v", i, msg, in[i])
			}
		}
	})

	t.Run("增量追加消息", func(t *testing.T) {
		sid := "test-session-append"
		first := []message.Message{{Role: "user", Content: "第一条"}}
		second := []message.Message{{Role: "assistant", Content: "第二条"}}

		if err := store.SaveHistory(ctx, sid, first); err != nil {
			t.Fatalf("SaveHistory 错误: %v", err)
		}
		if err := store.AppendMessages(ctx, sid, second); err != nil {
			t.Fatalf("AppendMessages 错误: %v", err)
		}
		out, err := store.LoadHistory(ctx, sid)
		if err != nil {
			t.Fatalf("LoadHistory 错误: %v", err)
		}
		if len(out) != 2 {
			t.Fatalf("期望 2 条，得到 %d 条", len(out))
		}
	})

	t.Run("全量覆写清除旧消息", func(t *testing.T) {
		sid := "test-session-overwrite"
		old := []message.Message{{Role: "user", Content: "旧消息"}}
		new_ := []message.Message{{Role: "assistant", Content: "新消息"}}

		if err := store.SaveHistory(ctx, sid, old); err != nil {
			t.Fatalf("SaveHistory 错误: %v", err)
		}
		if err := store.SaveHistory(ctx, sid, new_); err != nil {
			t.Fatalf("SaveHistory 覆写错误: %v", err)
		}
		out, err := store.LoadHistory(ctx, sid)
		if err != nil {
			t.Fatalf("LoadHistory 错误: %v", err)
		}
		if len(out) != 1 || out[0].Content != "新消息" {
			t.Fatalf("覆写未生效: %+v", out)
		}
	})

	t.Run("删除会话", func(t *testing.T) {
		sid := "test-session-delete"
		msgs := []message.Message{{Role: "user", Content: "待删除"}}
		if err := store.SaveHistory(ctx, sid, msgs); err != nil {
			t.Fatalf("SaveHistory 错误: %v", err)
		}
		if err := store.DeleteSession(ctx, sid); err != nil {
			t.Fatalf("DeleteSession 错误: %v", err)
		}
		out, err := store.LoadHistory(ctx, sid)
		if err != nil {
			t.Fatalf("LoadHistory after delete 错误: %v", err)
		}
		if len(out) != 0 {
			t.Fatalf("删除后仍有 %d 条消息", len(out))
		}
	})

	t.Run("列出所有会话", func(t *testing.T) {
		// 清理确保当前只有我们创建的会话（FileStore 可能有残留，MemoryStore 没有）
		// 只验证我们创建的会话在列表中出现
		sid := "test-session-list-" + time.Now().Format("20060102150405")
		if err := store.SaveHistory(ctx, sid, []message.Message{{Role: "user", Content: "x"}}); err != nil {
			t.Fatalf("SaveHistory 错误: %v", err)
		}
		ids, err := store.ListSessions(ctx)
		if err != nil {
			t.Fatalf("ListSessions 错误: %v", err)
		}
		found := false
		for _, id := range ids {
			if id == sid {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("会话 %q 未出现在列表 %v 中", sid, ids)
		}
	})

	t.Run("会话元数据", func(t *testing.T) {
		sid := "test-session-meta"
		msgs := []message.Message{
			{Role: "user", Content: "msg1"},
			{Role: "assistant", Content: "msg2"},
		}
		if err := store.SaveHistory(ctx, sid, msgs); err != nil {
			t.Fatalf("SaveHistory 错误: %v", err)
		}
		meta, err := store.SessionMeta(ctx, sid)
		if err != nil {
			t.Fatalf("SessionMeta 错误: %v", err)
		}
		if meta.MessageCount != 2 {
			t.Errorf("期望 MessageCount=2，得到 %d", meta.MessageCount)
		}
	})

	t.Run("会话摘要更新与加载", func(t *testing.T) {
		sid := "test-session-summary"
		if err := store.UpdateSessionSummary(ctx, sid, "这是一个会话摘要"); err != nil {
			t.Fatalf("UpdateSessionSummary 错误: %v", err)
		}
		meta, err := store.SessionMeta(ctx, sid)
		if err != nil {
			t.Fatalf("SessionMeta 错误: %v", err)
		}
		if meta.Summary != "这是一个会话摘要" {
			t.Errorf("期望 Summary='这是一个会话摘要'，得到 %q", meta.Summary)
		}
	})

	t.Run("延迟工具状态保存与加载", func(t *testing.T) {
		sid := "test-session-deferred"
		tools := []string{"bash", "web_search"}
		if err := store.SaveDeferredState(ctx, sid, tools); err != nil {
			t.Fatalf("SaveDeferredState 错误: %v", err)
		}
		loaded, err := store.LoadDeferredState(ctx, sid)
		if err != nil {
			t.Fatalf("LoadDeferredState 错误: %v", err)
		}
		if len(loaded) != len(tools) {
			t.Fatalf("期望 %d 个工具，得到 %d 个", len(tools), len(loaded))
		}
	})

	t.Run("延迟工具状态删除", func(t *testing.T) {
		sid := "test-session-deferred-del"
		if err := store.SaveDeferredState(ctx, sid, []string{"bash"}); err != nil {
			t.Fatalf("SaveDeferredState 错误: %v", err)
		}
		if err := store.DeleteDeferredState(ctx, sid); err != nil {
			t.Fatalf("DeleteDeferredState 错误: %v", err)
		}
		loaded, err := store.LoadDeferredState(ctx, sid)
		if err != nil {
			t.Fatalf("LoadDeferredState after delete 错误: %v", err)
		}
		if len(loaded) != 0 {
			t.Fatalf("删除后仍有 %d 个工具", len(loaded))
		}
	})

	t.Run("ToolCall 字段持久化", func(t *testing.T) {
		sid := "test-session-toolcall"
		msgs := []message.Message{
			{
				Role:    "assistant",
				Content: "执行工具",
				ToolCalls: []message.ToolCall{
					{ID: "tc1", Name: "bash", Arguments: map[string]any{"command": "ls"}, Result: "file.txt"},
				},
			},
		}
		if err := store.SaveHistory(ctx, sid, msgs); err != nil {
			t.Fatalf("SaveHistory 错误: %v", err)
		}
		out, err := store.LoadHistory(ctx, sid)
		if err != nil {
			t.Fatalf("LoadHistory 错误: %v", err)
		}
		if len(out) != 1 || len(out[0].ToolCalls) != 1 {
			t.Fatalf("ToolCalls 未正确持久化: %+v", out)
		}
		tc := out[0].ToolCalls[0]
		if tc.ID != "tc1" || tc.Name != "bash" || tc.Result != "file.txt" {
			t.Errorf("ToolCall 字段错误: %+v", tc)
		}
	})
}

// ─── MemoryStore 测试 ────────────────────────────────────────────────────────

func TestMemoryStore(t *testing.T) {
	store := persist.NewMemoryStore()
	runStoreTests(t, store)
}

func TestMemoryStore_Reset(t *testing.T) {
	ctx := context.Background()
	store := persist.NewMemoryStore()
	_ = store.SaveHistory(ctx, "s1", []message.Message{{Role: "user", Content: "x"}})
	store.Reset()
	msgs, _ := store.LoadHistory(ctx, "s1")
	if len(msgs) != 0 {
		t.Fatal("Reset 后仍有数据")
	}
}

// ─── FileStore 测试 ──────────────────────────────────────────────────────────

func TestFileStore(t *testing.T) {
	dir := t.TempDir()
	store, err := persist.NewFileStore(dir)
	if err != nil {
		t.Fatalf("NewFileStore 错误: %v", err)
	}
	runStoreTests(t, store)
}

func TestFileStore_AtomicWrite(t *testing.T) {
	dir := t.TempDir()
	store, err := persist.NewFileStore(dir)
	if err != nil {
		t.Fatalf("NewFileStore 错误: %v", err)
	}
	ctx := context.Background()
	sid := "atomic-test"
	msgs := make([]message.Message, 100)
	for i := range msgs {
		msgs[i] = message.Message{Role: "user", Content: "消息内容，用于测试原子写入的完整性验证"}
	}
	if err := store.SaveHistory(ctx, sid, msgs); err != nil {
		t.Fatalf("SaveHistory 错误: %v", err)
	}
	// 确认没有 .tmp 残留
	tmp := filepath.Join(dir, sid, "history.jsonl.tmp")
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Error("原子写入后 .tmp 文件未被清理")
	}
}

func TestFileStore_PurgeOlderThan(t *testing.T) {
	dir := t.TempDir()
	store, err := persist.NewFileStore(dir)
	if err != nil {
		t.Fatalf("NewFileStore 错误: %v", err)
	}
	ctx := context.Background()

	// 保存两个会话
	_ = store.SaveHistory(ctx, "old-session", []message.Message{{Role: "user", Content: "x"}})
	_ = store.SaveHistory(ctx, "new-session", []message.Message{{Role: "user", Content: "y"}})

	// 清理 1 毫秒后以前的数据（old-session 已是"旧"的）
	time.Sleep(2 * time.Millisecond)
	cutoff := time.Now()

	// 保存 new-session 让它的 UpdatedAt 在 cutoff 之后
	_ = store.SaveHistory(ctx, "new-session", []message.Message{{Role: "user", Content: "y2"}})

	deleted, err := store.PurgeOlderThan(ctx, cutoff)
	if err != nil {
		t.Fatalf("PurgeOlderThan 错误: %v", err)
	}
	if deleted < 1 {
		t.Fatalf("期望至少删除 1 个会话，实际删除 %d 个", deleted)
	}
}
