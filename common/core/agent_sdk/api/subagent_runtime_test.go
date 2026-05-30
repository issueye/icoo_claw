package api

import (
	"context"
	"testing"

	"icoo_claw/common/core/agent_sdk/model"
	"icoo_claw/common/core/agent_sdk/runtime/subagents"
	"icoo_claw/common/core/agent_sdk/tool"
	toolbuiltin "icoo_claw/common/core/agent_sdk/tool/builtin"
)

func TestSubagentToolAllowExcludesSkillForSkillExecutor(t *testing.T) {
	registry := tool.NewRegistry()
	for _, impl := range []tool.Tool{
		toolbuiltin.NewBashToolWithRoot(t.TempDir()),
		toolbuiltin.NewSkillTool(nil, nil),
		toolbuiltin.NewSkillExecuteTool(nil, nil),
	} {
		if err := registry.Register(impl); err != nil {
			t.Fatalf("register %s: %v", impl.Name(), err)
		}
	}

	rt := &Runtime{registry: registry}
	allow := subagentToolAllow(rt, nil, subagents.TypeSkillExecutor)
	if _, ok := allow["skill"]; ok {
		t.Fatalf("skill-executor should not expose skill tool")
	}
	if _, ok := allow["skill_execute"]; ok {
		t.Fatalf("skill-executor should not expose skill_execute tool")
	}
	if _, ok := allow["bash"]; !ok {
		t.Fatalf("skill-executor should keep non-skill tools")
	}
}

func TestNormalizedSubagentModelClearsBuiltinAliases(t *testing.T) {
	for _, value := range []string{subagents.ModelSonnet, subagents.ModelHaiku, " " + subagents.ModelSonnet + " "} {
		if got := normalizedSubagentModel(value); got != "" {
			t.Fatalf("normalizedSubagentModel(%q) = %q, want empty", value, got)
		}
	}
	if got := normalizedSubagentModel("gpt-test-model"); got != "gpt-test-model" {
		t.Fatalf("normalizedSubagentModel custom = %q", got)
	}
}

func TestRuntimeSubagentRunsModelLoop(t *testing.T) {
	rt := &Runtime{
		opts: Options{
			Model:         subagentLoopModel{t: t},
			MaxIterations: 3,
		},
		registry: tool.NewRegistry(),
	}

	res, err := rt.runSubagent(context.Background(), subagents.Context{}, subagents.Request{
		Target:      subagents.TypeSkillExecutor,
		Instruction: "今天成都的天气",
	})
	if err != nil {
		t.Fatalf("run subagent: %v", err)
	}
	if res.Output != "成都今天晴。" {
		t.Fatalf("output = %q", res.Output)
	}
}

func TestRuntimeSubagentUsesModelPoolForBuiltinAlias(t *testing.T) {
	rt := &Runtime{
		opts: Options{
			ModelPool: map[ModelTier]model.Model{
				ModelTierLow: subagentStaticModel{output: "low-tier"},
			},
			MaxIterations: 3,
		},
		registry: tool.NewRegistry(),
	}

	res, err := rt.runSubagent(context.Background(), subagents.Context{Model: subagents.ModelHaiku}, subagents.Request{
		Target:      subagents.TypeExplore,
		Instruction: "route this",
	})
	if err != nil {
		t.Fatalf("run subagent: %v", err)
	}
	if res.Output != "low-tier" {
		t.Fatalf("output = %q, want low-tier", res.Output)
	}
}

func TestRuntimeSubagentModelMappingOverridesBuiltinAlias(t *testing.T) {
	rt := &Runtime{
		opts: Options{
			ModelPool: map[ModelTier]model.Model{
				ModelTierLow:  subagentStaticModel{output: "low-tier"},
				ModelTierHigh: subagentStaticModel{output: "high-tier"},
			},
			SubagentModelMapping: map[string]ModelTier{
				subagents.TypeExplore: ModelTierHigh,
			},
			MaxIterations: 3,
		},
		registry: tool.NewRegistry(),
	}

	res, err := rt.runSubagent(context.Background(), subagents.Context{Model: subagents.ModelHaiku}, subagents.Request{
		Target:      subagents.TypeExplore,
		Instruction: "route this",
	})
	if err != nil {
		t.Fatalf("run subagent: %v", err)
	}
	if res.Output != "high-tier" {
		t.Fatalf("output = %q, want high-tier", res.Output)
	}
}

type subagentLoopModel struct {
	t *testing.T
}

func (m subagentLoopModel) Complete(context.Context, model.Request) (*model.Response, error) {
	return nil, nil
}

func (m subagentLoopModel) CompleteStream(_ context.Context, req model.Request, cb model.StreamHandler) error {
	if len(req.Messages) != 1 || req.Messages[0].Content != "今天成都的天气" {
		m.t.Fatalf("messages = %#v", req.Messages)
	}
	return cb(model.StreamResult{
		Final: true,
		Response: &model.Response{
			Message:    model.Message{Role: "assistant", Content: "成都今天晴。"},
			StopReason: "end_turn",
		},
	})
}

type subagentStaticModel struct {
	output string
}

func (m subagentStaticModel) Complete(context.Context, model.Request) (*model.Response, error) {
	return nil, nil
}

func (m subagentStaticModel) CompleteStream(_ context.Context, _ model.Request, cb model.StreamHandler) error {
	return cb(model.StreamResult{
		Final: true,
		Response: &model.Response{
			Message:    model.Message{Role: "assistant", Content: m.output},
			StopReason: "end_turn",
		},
	})
}
