package api

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"icoo_claw/common/core/agent_sdk/message"
	"icoo_claw/common/core/agent_sdk/middleware"
	"icoo_claw/common/core/agent_sdk/model"
	"icoo_claw/common/core/agent_sdk/tool"
)

func TestBeforeToolMiddlewareErrorBlocksExecution(t *testing.T) {
	registry := tool.NewRegistry()
	impl := &countingTool{name: "blocked_tool"}
	if err := registry.Register(impl); err != nil {
		t.Fatalf("register tool: %v", err)
	}
	history := message.NewHistory()
	executor := &runtimeToolExecutor{
		executor:  tool.NewExecutor(registry, nil),
		history:   history,
		sessionID: "session-test",
	}
	blockErr := errors.New("blocked by policy")
	chain := middleware.NewChain([]middleware.Middleware{
		middleware.Funcs{
			Identifier: "blocker",
			OnBeforeTool: func(context.Context, *middleware.State) error {
				return blockErr
			},
		},
	})

	err := (&Runtime{registry: registry}).executeToolCalls(context.Background(), []model.ToolCall{{
		ID:        "call-1",
		Name:      "blocked_tool",
		Arguments: map[string]any{"value": "x"},
	}}, executor, chain, &middleware.State{Values: map[string]any{}}, nil, nil, Request{SessionID: "session-test"})
	if !errors.Is(err, blockErr) {
		t.Fatalf("executeToolCalls err = %v, want %v", err, blockErr)
	}
	if got := impl.calls.Load(); got != 0 {
		t.Fatalf("tool executed %d times, want 0", got)
	}
	messages := history.All()
	if len(messages) != 1 || messages[0].Role != "tool" {
		t.Fatalf("history = %#v, want one tool result", messages)
	}
	if len(messages[0].ToolCalls) != 1 || !strings.Contains(messages[0].ToolCalls[0].Result, "blocked by policy") {
		t.Fatalf("tool result = %#v, want blocked error", messages[0].ToolCalls)
	}
}

type countingTool struct {
	name  string
	calls atomic.Int64
}

func (t *countingTool) Name() string {
	return t.name
}

func (t *countingTool) Description() string {
	return "count executions"
}

func (t *countingTool) Schema() *tool.JSONSchema {
	return &tool.JSONSchema{Type: "object"}
}

func (t *countingTool) Execute(context.Context, map[string]interface{}) (*tool.ToolResult, error) {
	t.calls.Add(1)
	return &tool.ToolResult{Success: true, Output: "executed"}, nil
}
