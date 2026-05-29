package api

import (
	"context"
	"testing"

	"icoo_claw/common/core/agent_sdk/model"
	"icoo_claw/common/core/agent_sdk/tool"
)

func TestRuntimeBuilderOptions(t *testing.T) {
	mdl := builderModel{}
	prompter := PermissionPrompterFunc(func(context.Context, PermissionRequest) (bool, error) { return true, nil })

	opts := NewRuntimeBuilder(t.TempDir()).
		WithModel(mdl).
		WithSystemPrompt("system").
		WithBuiltinTools("read", "grep").
		WithPermissionPrompter(prompter).
		WithMaxIterations(3).
		Options()

	if opts.Model == nil {
		t.Fatalf("model not set")
	}
	if opts.SystemPrompt != "system" {
		t.Fatalf("system prompt = %q", opts.SystemPrompt)
	}
	if len(opts.EnabledBuiltinTools) != 2 || !containsString(opts.EnabledBuiltinTools, "grep") || !containsString(opts.EnabledBuiltinTools, "read") {
		t.Fatalf("builtin tools = %#v", opts.EnabledBuiltinTools)
	}
	if opts.PermissionPrompter == nil {
		t.Fatalf("permission prompter not set")
	}
	if opts.MaxIterations != 3 {
		t.Fatalf("max iterations = %d", opts.MaxIterations)
	}
}

func TestRuntimeBuilderBuild(t *testing.T) {
	rt, err := NewRuntimeBuilder(t.TempDir()).
		WithModel(builderModel{}).
		WithBuiltinTools("read").
		Build(context.Background())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	defer rt.Close()

	if rt.registry == nil {
		t.Fatalf("registry not initialised")
	}
	if _, err := rt.registry.Get("read"); err != nil {
		t.Fatalf("read tool not registered: %v", err)
	}
}

func TestRuntimeBuilderPresets(t *testing.T) {
	coding := NewCodingAgent("project").Options()
	if !containsString(coding.EnabledBuiltinTools, "bash") || !containsString(coding.EnabledBuiltinTools, "edit") {
		t.Fatalf("coding tools = %#v", coding.EnabledBuiltinTools)
	}

	research := NewResearchAgent("project").Options()
	if containsString(research.EnabledBuiltinTools, "write") || !containsString(research.EnabledBuiltinTools, "web_search") {
		t.Fatalf("research tools = %#v", research.EnabledBuiltinTools)
	}

	safe := NewSafeLocalAgent("project").Options()
	if containsString(safe.EnabledBuiltinTools, "bash") || containsString(safe.EnabledBuiltinTools, "write") {
		t.Fatalf("safe local tools = %#v", safe.EnabledBuiltinTools)
	}
}

func TestRuntimeBuilderToolOnlyAgent(t *testing.T) {
	custom := builderTool{name: "custom"}
	opts := NewToolOnlyAgent("project", custom).Options()
	if len(opts.Tools) != 1 || opts.Tools[0].Name() != "custom" {
		t.Fatalf("tools = %#v", opts.Tools)
	}
	if len(opts.EnabledBuiltinTools) != 0 {
		t.Fatalf("enabled builtins = %#v, want none", opts.EnabledBuiltinTools)
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

type builderModel struct{}

func (builderModel) Complete(context.Context, model.Request) (*model.Response, error) {
	return &model.Response{Message: model.Message{Role: "assistant", Content: "ok"}}, nil
}

func (builderModel) CompleteStream(_ context.Context, _ model.Request, cb model.StreamHandler) error {
	return cb(model.StreamResult{Final: true, Response: &model.Response{Message: model.Message{Role: "assistant", Content: "ok"}}})
}

type builderTool struct {
	name string
}

func (t builderTool) Name() string { return t.name }

func (t builderTool) Description() string { return t.name }

func (t builderTool) Schema() *tool.JSONSchema { return &tool.JSONSchema{Type: "object"} }

func (t builderTool) Execute(context.Context, map[string]interface{}) (*tool.ToolResult, error) {
	return &tool.ToolResult{Success: true}, nil
}
