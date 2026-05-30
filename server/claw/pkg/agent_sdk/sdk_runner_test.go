package agent_sdk

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"icoo_claw/server/claw/pkg/sessionstore"

	sdkmodel "icoo_claw/common/core/agent_sdk/model"
)

type memoryHistoryStore struct {
	messages []sessionstore.Message
}

func (m *memoryHistoryStore) ListMessages(context.Context, string) ([]sessionstore.Message, error) {
	return append([]sessionstore.Message(nil), m.messages...), nil
}

func (m *memoryHistoryStore) ReplaceMessages(_ context.Context, _ string, messages []sessionstore.Message) error {
	m.messages = append([]sessionstore.Message(nil), messages...)
	return nil
}

type staticModel struct{}

func (m staticModel) Complete(context.Context, sdkmodel.Request) (*sdkmodel.Response, error) {
	return nil, nil
}

type streamingDeltaModel struct{}

func (m streamingDeltaModel) Complete(context.Context, sdkmodel.Request) (*sdkmodel.Response, error) {
	return nil, nil
}

func (m streamingDeltaModel) CompleteStream(_ context.Context, _ sdkmodel.Request, cb sdkmodel.StreamHandler) error {
	for _, delta := range []string{"he", "llo"} {
		if err := cb(sdkmodel.StreamResult{Delta: delta}); err != nil {
			return err
		}
	}
	return cb(sdkmodel.StreamResult{
		Final: true,
		Response: &sdkmodel.Response{
			Message:    sdkmodel.Message{Role: "assistant", Content: "hello"},
			StopReason: "end_turn",
		},
	})
}

type toolCallingModel struct {
	t            *testing.T
	callCount    int
	toolResultOK bool
}

func (m *toolCallingModel) Complete(context.Context, sdkmodel.Request) (*sdkmodel.Response, error) {
	return nil, nil
}

func (m *toolCallingModel) CompleteStream(_ context.Context, req sdkmodel.Request, cb sdkmodel.StreamHandler) error {
	m.callCount++
	switch m.callCount {
	case 1:
		if !hasTool(req.Tools, "read") {
			m.t.Fatalf("model request tools = %+v, want read", req.Tools)
		}
		return cb(sdkmodel.StreamResult{
			Final: true,
			Response: &sdkmodel.Response{
				Message: sdkmodel.Message{
					Role: "assistant",
					ToolCalls: []sdkmodel.ToolCall{{
						ID:        "tool_call_1",
						Name:      "read",
						Arguments: map[string]any{"file_path": "fixture.txt"},
					}},
				},
				StopReason: "tool_calls",
			},
		})
	case 2:
		m.toolResultOK = requestHasToolResult(req, "read", "tool-call-secret")
		if !m.toolResultOK {
			m.t.Fatalf("second model request missing read result: %+v", req.Messages)
		}
		return cb(sdkmodel.StreamResult{
			Final: true,
			Response: &sdkmodel.Response{
				Message:    sdkmodel.Message{Role: "assistant", Content: "read tool-call-secret"},
				StopReason: "end_turn",
			},
		})
	default:
		m.t.Fatalf("unexpected model call count %d", m.callCount)
		return nil
	}
}

type writeFindModel struct {
	t         *testing.T
	callCount int
}

func (m *writeFindModel) Complete(context.Context, sdkmodel.Request) (*sdkmodel.Response, error) {
	return nil, nil
}

func (m *writeFindModel) CompleteStream(_ context.Context, req sdkmodel.Request, cb sdkmodel.StreamHandler) error {
	m.callCount++
	switch m.callCount {
	case 1:
		for _, name := range []string{"write", "find", "bash"} {
			if !hasTool(req.Tools, name) {
				m.t.Fatalf("model request tools = %+v, missing %s", req.Tools, name)
			}
		}
		return cb(sdkmodel.StreamResult{
			Final: true,
			Response: &sdkmodel.Response{
				Message: sdkmodel.Message{
					Role: "assistant",
					ToolCalls: []sdkmodel.ToolCall{{
						ID:   "tool_call_write",
						Name: "write",
						Arguments: map[string]any{
							"file_path": "generated.txt",
							"content":   "write-find-secret\n",
						},
					}},
				},
				StopReason: "tool_calls",
			},
		})
	case 2:
		if !requestHasToolResult(req, "write", "generated.txt") {
			m.t.Fatalf("second model request missing write result: %+v", req.Messages)
		}
		return cb(sdkmodel.StreamResult{
			Final: true,
			Response: &sdkmodel.Response{
				Message: sdkmodel.Message{
					Role: "assistant",
					ToolCalls: []sdkmodel.ToolCall{{
						ID:   "tool_call_find",
						Name: "find",
						Arguments: map[string]any{
							"pattern": "generated",
							"type":    "file",
						},
					}},
				},
				StopReason: "tool_calls",
			},
		})
	case 3:
		if !requestHasToolResult(req, "find", "generated.txt") {
			m.t.Fatalf("third model request missing find result: %+v", req.Messages)
		}
		return cb(sdkmodel.StreamResult{
			Final: true,
			Response: &sdkmodel.Response{
				Message:    sdkmodel.Message{Role: "assistant", Content: "write and find ok"},
				StopReason: "end_turn",
			},
		})
	default:
		m.t.Fatalf("unexpected model call count %d", m.callCount)
		return nil
	}
}

type fetchModel struct {
	t         *testing.T
	url       string
	callCount int
}

func (m *fetchModel) Complete(context.Context, sdkmodel.Request) (*sdkmodel.Response, error) {
	return nil, nil
}

func (m *fetchModel) CompleteStream(_ context.Context, req sdkmodel.Request, cb sdkmodel.StreamHandler) error {
	m.callCount++
	switch m.callCount {
	case 1:
		if !hasTool(req.Tools, "fetch") {
			m.t.Fatalf("model request tools = %+v, missing fetch", req.Tools)
		}
		return cb(sdkmodel.StreamResult{
			Final: true,
			Response: &sdkmodel.Response{
				Message: sdkmodel.Message{
					Role: "assistant",
					ToolCalls: []sdkmodel.ToolCall{{
						ID:        "tool_call_fetch",
						Name:      "fetch",
						Arguments: map[string]any{"url": m.url},
					}},
				},
				StopReason: "tool_calls",
			},
		})
	case 2:
		if !requestHasToolResult(req, "fetch", "network-secret") {
			m.t.Fatalf("second model request missing fetch result: %+v", req.Messages)
		}
		return cb(sdkmodel.StreamResult{
			Final: true,
			Response: &sdkmodel.Response{
				Message:    sdkmodel.Message{Role: "assistant", Content: "fetch ok"},
				StopReason: "end_turn",
			},
		})
	default:
		m.t.Fatalf("unexpected model call count %d", m.callCount)
		return nil
	}
}

type timeoutFetchModel struct {
	t         *testing.T
	url       string
	callCount int
}

func (m *timeoutFetchModel) Complete(context.Context, sdkmodel.Request) (*sdkmodel.Response, error) {
	return nil, nil
}

func (m *timeoutFetchModel) CompleteStream(_ context.Context, req sdkmodel.Request, cb sdkmodel.StreamHandler) error {
	m.callCount++
	switch m.callCount {
	case 1:
		if !hasTool(req.Tools, "fetch") {
			m.t.Fatalf("model request tools = %+v, missing fetch", req.Tools)
		}
		return cb(sdkmodel.StreamResult{
			Final: true,
			Response: &sdkmodel.Response{
				Message: sdkmodel.Message{
					Role: "assistant",
					ToolCalls: []sdkmodel.ToolCall{{
						ID:   "tool_call_fetch_timeout",
						Name: "fetch",
						Arguments: map[string]any{
							"url":     m.url,
							"timeout": 0.001,
						},
					}},
				},
				StopReason: "tool_calls",
			},
		})
	case 2:
		if !requestHasToolResult(req, "fetch", "context deadline exceeded") {
			m.t.Fatalf("second model request missing fetch timeout result: %+v", req.Messages)
		}
		return cb(sdkmodel.StreamResult{
			Final: true,
			Response: &sdkmodel.Response{
				Message:    sdkmodel.Message{Role: "assistant", Content: "fetch timeout handled"},
				StopReason: "end_turn",
			},
		})
	default:
		m.t.Fatalf("unexpected model call count %d", m.callCount)
		return nil
	}
}

func (m staticModel) CompleteStream(_ context.Context, req sdkmodel.Request, cb sdkmodel.StreamHandler) error {
	return cb(sdkmodel.StreamResult{
		Final: true,
		Response: &sdkmodel.Response{
			Message:    sdkmodel.Message{Role: "assistant", Content: "ok"},
			StopReason: "end_turn",
		},
	})
}

func TestSDKRunnerRunLoadsAndSavesHistory(t *testing.T) {
	store := &memoryHistoryStore{
		messages: []sessionstore.Message{{Role: "user", Content: "previous"}},
	}
	history := NewHistoryAdapter(store)
	factory := NewRuntimeFactory(history, staticModel{})
	runner := NewSDKRunner(factory, history)

	resp, err := runner.Run(context.Background(), RunRequest{
		SessionID: "sess_1",
		Prompt:    "next",
		Agent: map[string]any{
			"enabled_builtin_tools": []any{},
		},
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if resp.Output != "ok" {
		t.Fatalf("output = %q, want ok", resp.Output)
	}
	if len(store.messages) != 3 {
		t.Fatalf("saved messages = %+v", store.messages)
	}
	if store.messages[0].Content != "previous" || store.messages[1].Content != "next" || store.messages[2].Content != "ok" {
		t.Fatalf("unexpected saved messages = %+v", store.messages)
	}
}

func TestSDKRunnerStreamEmitsTextWithoutNilPlaceholders(t *testing.T) {
	history := NewHistoryAdapter(&memoryHistoryStore{})
	runner := NewSDKRunner(NewRuntimeFactory(history, staticModel{}), history)

	events, err := runner.RunStream(context.Background(), RunRequest{
		SessionID: "sess_stream",
		Prompt:    "hello",
		Agent: map[string]any{
			"enabled_builtin_tools": []any{},
		},
	})
	if err != nil {
		t.Fatalf("run stream: %v", err)
	}

	output := ""
	for event := range events {
		text := streamEventText(event)
		if text == "<nil>" {
			t.Fatalf("unexpected nil placeholder event: %+v", event)
		}
		output += text
	}
	if output != "ok" {
		t.Fatalf("stream output = %q, want ok", output)
	}
}

func TestSDKRunnerStreamsModelDeltasWithoutFinalDuplicate(t *testing.T) {
	history := NewHistoryAdapter(&memoryHistoryStore{})
	runner := NewSDKRunner(NewRuntimeFactory(history, streamingDeltaModel{}), history)

	events, err := runner.RunStream(context.Background(), RunRequest{
		SessionID: "sess_delta",
		Prompt:    "hello",
		Agent: map[string]any{
			"enabled_builtin_tools": []any{},
		},
	})
	if err != nil {
		t.Fatalf("run stream: %v", err)
	}

	output := ""
	for event := range events {
		output += streamEventText(event)
	}
	if output != "hello" {
		t.Fatalf("stream output = %q, want only realtime deltas", output)
	}
}

func TestSDKRunnerStreamCompletesOnlyAfterToolLoopFinishes(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(root+"/fixture.txt", []byte("tool-call-secret\n"), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	store := &memoryHistoryStore{}
	history := NewHistoryAdapter(store)
	model := &toolCallingModel{t: t}
	runner := NewSDKRunner(NewRuntimeFactory(history, model), history)

	events, err := runner.RunStream(context.Background(), RunRequest{
		SessionID:     "sess_tool_stream",
		Prompt:        "read the fixture",
		ToolWhitelist: []string{"read"},
		Agent: map[string]any{
			"project_root":          root,
			"enabled_builtin_tools": []any{"read"},
			"max_iterations":        float64(3),
		},
	})
	if err != nil {
		t.Fatalf("run stream: %v", err)
	}

	output := ""
	toolResultSeen := false
	toolInputWithIDSeen := false
	completedCount := 0
	completedBeforeToolResult := false
	for event := range events {
		output += streamEventText(event)
		if event.Type == StreamEventSessionUpdate && event.Update != nil &&
			event.Update.SessionUpdate == "tool_call_update" && event.Update.ToolCallID == "tool_call_1" && event.Update.RawInput != nil {
			toolInputWithIDSeen = true
		}
		if event.Type == StreamEventSessionUpdate && event.Update != nil &&
			event.Update.SessionUpdate == "tool_call_update" && event.Update.Status == "completed" {
			toolResultSeen = true
		}
		if event.Type == StreamEventSessionCompleted {
			completedCount++
			if !toolResultSeen {
				completedBeforeToolResult = true
			}
		}
	}

	if output != "read tool-call-secret" {
		t.Fatalf("stream output = %q, want final tool-informed response", output)
	}
	if !toolResultSeen {
		t.Fatalf("stream did not surface completed tool result")
	}
	if !toolInputWithIDSeen {
		t.Fatalf("stream did not surface tool input with matching tool id")
	}
	if completedBeforeToolResult {
		t.Fatalf("stream completed before tool result was emitted")
	}
	if completedCount != 1 {
		t.Fatalf("completed events = %d, want 1", completedCount)
	}
	if model.callCount != 2 || !model.toolResultOK {
		t.Fatalf("model callCount=%d toolResultOK=%v", model.callCount, model.toolResultOK)
	}
}

func streamEventText(event StreamEvent) string {
	if event.Type != StreamEventSessionUpdate || event.Update == nil {
		return ""
	}
	if event.Update.SessionUpdate != "agent_message_chunk" || event.Update.Content == nil {
		return ""
	}
	return event.Update.Content.Text
}

func TestSDKRunnerExecutesBuiltinReadToolAndSavesResult(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(root+"/fixture.txt", []byte("tool-call-secret\n"), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	store := &memoryHistoryStore{}
	history := NewHistoryAdapter(store)
	model := &toolCallingModel{t: t}
	factory := NewRuntimeFactory(history, model)
	runner := NewSDKRunner(factory, history)

	resp, err := runner.Run(context.Background(), RunRequest{
		SessionID:     "sess_tool",
		Prompt:        "read the fixture",
		ToolWhitelist: []string{"read"},
		Agent: map[string]any{
			"project_root":          root,
			"enabled_builtin_tools": []any{"read"},
			"max_iterations":        float64(3),
		},
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if resp.Output != "read tool-call-secret" {
		t.Fatalf("output = %q, want final tool-informed response", resp.Output)
	}
	if model.callCount != 2 || !model.toolResultOK {
		t.Fatalf("model callCount=%d toolResultOK=%v", model.callCount, model.toolResultOK)
	}
	if len(store.messages) != 4 {
		t.Fatalf("saved messages = %+v, want user + assistant tool call + tool result + assistant", store.messages)
	}
	assistantToolCall := mustJSON(t, store.messages[1].ToolCalls)
	if store.messages[1].Role != "assistant" || !strings.Contains(assistantToolCall, `"Name":"read"`) {
		t.Fatalf("assistant tool call message = %+v", store.messages[1])
	}
	toolResult := mustJSON(t, store.messages[2].ToolCalls)
	if store.messages[2].Role != "tool" || !strings.Contains(toolResult, "tool-call-secret") {
		t.Fatalf("tool result message = %+v", store.messages[2])
	}
	if store.messages[3].Role != "assistant" || store.messages[3].Content != "read tool-call-secret" {
		t.Fatalf("final assistant message = %+v", store.messages[3])
	}
}

func TestSDKRunnerExecutesBuiltinWriteAndFindTools(t *testing.T) {
	root := t.TempDir()

	store := &memoryHistoryStore{}
	history := NewHistoryAdapter(store)
	model := &writeFindModel{t: t}
	factory := NewRuntimeFactory(history, model)
	runner := NewSDKRunner(factory, history)

	resp, err := runner.Run(context.Background(), RunRequest{
		SessionID:     "sess_write_find",
		Prompt:        "write and find a file",
		ToolWhitelist: []string{"write", "find", "bash"},
		Agent: map[string]any{
			"project_root": root,
			"enabled_builtin_tools": []any{
				"write",
				"find",
				"bash",
			},
			"max_iterations": float64(4),
		},
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if resp.Output != "write and find ok" {
		t.Fatalf("output = %q, want write and find ok", resp.Output)
	}
	content, err := os.ReadFile(root + "/generated.txt")
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	if string(content) != "write-find-secret\n" {
		t.Fatalf("generated file = %q", content)
	}
	if model.callCount != 3 {
		t.Fatalf("model callCount=%d, want 3", model.callCount)
	}
	if len(store.messages) != 6 {
		t.Fatalf("saved messages = %+v, want user + two tool rounds + final assistant", store.messages)
	}
}

func TestSDKRunnerExecutesFetchTool(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("network-secret"))
	}))
	defer server.Close()

	root := t.TempDir()
	store := &memoryHistoryStore{}
	history := NewHistoryAdapter(store)
	model := &fetchModel{t: t, url: server.URL}
	factory := NewRuntimeFactory(history, model)
	runner := NewSDKRunner(factory, history)

	resp, err := runner.Run(context.Background(), RunRequest{
		SessionID:     "sess_fetch",
		Prompt:        "fetch the url",
		ToolWhitelist: []string{"fetch"},
		Agent: map[string]any{
			"project_root":          root,
			"enabled_builtin_tools": []any{"fetch"},
			"network_allow":         []any{"127.0.0.1"},
			"max_iterations":        float64(3),
		},
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if resp.Output != "fetch ok" {
		t.Fatalf("output = %q, want fetch ok", resp.Output)
	}
	if model.callCount != 2 {
		t.Fatalf("model callCount=%d, want 2", model.callCount)
	}
}

func TestSDKRunnerStreamContinuesAfterFetchToolTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		_, _ = w.Write([]byte("late response"))
	}))
	defer server.Close()

	root := t.TempDir()
	store := &memoryHistoryStore{}
	history := NewHistoryAdapter(store)
	model := &timeoutFetchModel{t: t, url: server.URL}
	factory := NewRuntimeFactory(history, model)
	runner := NewSDKRunner(factory, history)

	events, err := runner.RunStream(context.Background(), RunRequest{
		SessionID:     "sess_fetch_timeout",
		Prompt:        "fetch the slow url",
		ToolWhitelist: []string{"fetch"},
		Agent: map[string]any{
			"project_root":          root,
			"enabled_builtin_tools": []any{"fetch"},
			"network_allow":         []any{"127.0.0.1"},
			"max_iterations":        float64(3),
		},
	})
	if err != nil {
		t.Fatalf("run stream: %v", err)
	}

	output := ""
	toolFailedSeen := false
	completedCount := 0
	for event := range events {
		if event.Type == StreamEventSessionError {
			t.Fatalf("stream should continue after tool timeout, got error: %+v", event.Error)
		}
		output += streamEventText(event)
		if event.Type == StreamEventSessionUpdate && event.Update != nil &&
			event.Update.SessionUpdate == "tool_call_update" &&
			event.Update.ToolCallID == "tool_call_fetch_timeout" &&
			event.Update.Status == "failed" {
			toolFailedSeen = true
		}
		if event.Type == StreamEventSessionCompleted {
			completedCount++
		}
	}

	if output != "fetch timeout handled" {
		t.Fatalf("stream output = %q, want timeout recovery response", output)
	}
	if !toolFailedSeen {
		t.Fatalf("stream did not surface failed tool status")
	}
	if completedCount != 1 {
		t.Fatalf("completed events = %d, want 1", completedCount)
	}
	if model.callCount != 2 {
		t.Fatalf("model callCount=%d, want 2", model.callCount)
	}
	if len(store.messages) != 4 {
		t.Fatalf("saved messages = %+v, want user + assistant tool call + failed tool result + assistant", store.messages)
	}
	toolResult := mustJSON(t, store.messages[2].ToolCalls)
	if store.messages[2].Role != "tool" || !strings.Contains(toolResult, "context deadline exceeded") {
		t.Fatalf("failed tool result message = %+v", store.messages[2])
	}
}

func TestRuntimeFactoryExposesCoreBuiltinTools(t *testing.T) {
	root := t.TempDir()
	history := NewHistoryAdapter(&memoryHistoryStore{})
	factory := NewRuntimeFactory(history, staticModel{})
	rt, err := factory.New(context.Background(), RunRequest{
		SessionID: "sess_tools",
		Agent: map[string]any{
			"project_root": root,
			"enabled_builtin_tools": []any{
				"read",
				"write",
				"bash",
				"find",
				"fetch",
				"web_search",
			},
		},
	})
	if err != nil {
		t.Fatalf("create runtime: %v", err)
	}
	defer func() { _ = rt.Close() }()

	defs := rt.AvailableToolsForWhitelist([]string{"read", "write", "bash", "find", "fetch", "web_search"})
	names := make(map[string]bool, len(defs))
	for _, def := range defs {
		names[def.Name] = true
	}
	for _, name := range []string{"read", "write", "bash", "find", "fetch", "web_search"} {
		if !names[name] {
			t.Fatalf("available tools = %+v, missing %s", defs, name)
		}
	}
	if names["grep"] || names["glob"] || names["edit"] {
		t.Fatalf("available tools = %+v, expected only requested core tools", defs)
	}
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal value: %v", err)
	}
	return string(payload)
}

func hasTool(tools []sdkmodel.ToolDefinition, name string) bool {
	for _, tool := range tools {
		if tool.Name == name {
			return true
		}
	}
	return false
}

func requestHasToolResult(req sdkmodel.Request, name string, content string) bool {
	for _, msg := range req.Messages {
		for _, call := range msg.ToolCalls {
			if call.Name == name && strings.Contains(call.Result, content) {
				return true
			}
		}
	}
	return false
}
