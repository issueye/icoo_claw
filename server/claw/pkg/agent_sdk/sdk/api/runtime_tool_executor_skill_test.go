package api

import (
	"context"
	"strings"
	"testing"

	"icoo_claw/server/claw/pkg/agent_sdk/sdk/message"
	"icoo_claw/server/claw/pkg/agent_sdk/sdk/model"
	"icoo_claw/server/claw/pkg/agent_sdk/sdk/runtime/skills"
	"icoo_claw/server/claw/pkg/agent_sdk/sdk/runtime/subagents"
	"icoo_claw/server/claw/pkg/agent_sdk/sdk/tool"
	toolbuiltin "icoo_claw/server/claw/pkg/agent_sdk/sdk/tool/builtin"
)

func TestRuntimeToolExecutorPassesUserPromptToSkill(t *testing.T) {
	reg := skills.NewRegistry()
	if err := reg.Register(skills.Definition{Name: "weather-zh", Description: "天气"}, skills.HandlerFunc(func(context.Context, skills.ActivationContext) (skills.Result, error) {
		return skills.Result{
			Skill:  "weather-zh",
			Output: map[string]any{"body": "查询天气"},
		}, nil
	})); err != nil {
		t.Fatalf("register skill: %v", err)
	}

	dispatcher := &capturePromptDispatcher{}
	toolRegistry := tool.NewRegistry()
	if err := toolRegistry.Register(toolbuiltin.NewSkillToolWithSubagent(reg, nil, dispatcher)); err != nil {
		t.Fatalf("register tool: %v", err)
	}

	history := message.NewHistory()
	history.Append(message.Message{Role: "user", Content: "今天成都的天气"})
	executor := &runtimeToolExecutor{
		executor: tool.NewExecutor(toolRegistry, nil),
		history:  history,
	}

	_, err := executor.Execute(context.Background(), model.ToolCall{
		ID:        "call_1",
		Name:      "skill",
		Arguments: map[string]any{"command": "weather-zh"},
	})
	if err != nil {
		t.Fatalf("execute skill: %v", err)
	}
	if !strings.Contains(dispatcher.req.Instruction, "今天成都的天气") {
		t.Fatalf("skill subagent instruction missing user prompt: %s", dispatcher.req.Instruction)
	}
}

type capturePromptDispatcher struct {
	req subagents.Request
}

func (f *capturePromptDispatcher) Dispatch(_ context.Context, req subagents.Request) (subagents.Result, error) {
	f.req = req
	return subagents.Result{Output: "ok"}, nil
}
