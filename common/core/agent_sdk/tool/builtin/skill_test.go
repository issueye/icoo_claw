package toolbuiltin

import (
	"context"
	"strings"
	"testing"

	"icoo_claw/common/core/agent_sdk/runtime/skills"
	"icoo_claw/common/core/agent_sdk/runtime/subagents"
)

type fakeSkillDispatcher struct {
	req subagents.Request
}

func (f *fakeSkillDispatcher) Dispatch(_ context.Context, req subagents.Request) (subagents.Result, error) {
	f.req = req
	return subagents.Result{
		Subagent: subagents.TypeSkillExecutor,
		Output:   "成都今天多云，气温约 18-25 摄氏度。",
		Metadata: map[string]any{"ok": true},
	}, nil
}

func TestSkillToolExecutesViaSubagentAndReturnsSummary(t *testing.T) {
	reg := skills.NewRegistry()
	err := reg.Register(skills.Definition{Name: "weather-zh", Description: "天气查询"}, skills.HandlerFunc(func(context.Context, skills.ActivationContext) (skills.Result, error) {
		return skills.Result{
			Skill: "weather-zh",
			Output: map[string]any{
				"body": "运行 scripts/weather-cn.sh 查询天气。",
				"support_files": map[string][]string{
					"scripts": {"weather-cn.sh"},
				},
			},
			Metadata: map[string]any{
				"source":        "C:/gateway/.skills/active/.agents/skills/weather-zh/SKILL.md",
				"allowed-tools": []string{"bash", "read", "skill"},
			},
		}, nil
	}))
	if err != nil {
		t.Fatalf("register skill: %v", err)
	}

	dispatcher := &fakeSkillDispatcher{}
	tool := NewSkillToolWithSubagent(reg, func(context.Context) skills.ActivationContext {
		return skills.ActivationContext{Prompt: "今天成都的天气"}
	}, dispatcher)

	res, err := tool.Execute(context.Background(), map[string]any{"command": "weather-zh"})
	if err != nil {
		t.Fatalf("execute skill: %v", err)
	}
	if res.Output != "成都今天多云，气温约 18-25 摄氏度。" {
		t.Fatalf("output = %q", res.Output)
	}
	if dispatcher.req.Target != subagents.TypeSkillExecutor {
		t.Fatalf("target = %q, want %q", dispatcher.req.Target, subagents.TypeSkillExecutor)
	}
	if !strings.Contains(dispatcher.req.Instruction, "今天成都的天气") {
		t.Fatalf("instruction missing user request: %s", dispatcher.req.Instruction)
	}
	if !strings.Contains(dispatcher.req.Instruction, "scripts/weather-cn.sh") {
		t.Fatalf("instruction missing support file: %s", dispatcher.req.Instruction)
	}
	if strings.Contains(strings.Join(dispatcher.req.ToolWhitelist, ","), "skill") {
		t.Fatalf("tool whitelist should exclude skill to avoid recursion: %#v", dispatcher.req.ToolWhitelist)
	}
}
