package api

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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

func TestRuntimeLoadsPluginManifestsWithoutEnablingCapabilities(t *testing.T) {
	root := t.TempDir()
	pluginRoot := filepath.Join(root, "external-plugin")
	manifestDir := filepath.Join(pluginRoot, ".codex-plugin")
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatalf("mkdir manifest dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(manifestDir, "plugin.json"), []byte(`{
  "name": "external-plugin",
  "version": "0.1.0",
  "capabilities": {"skills": ["skills/review"]}
}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	dirs := []string{pluginRoot}
	rt, err := NewRuntimeBuilder(root).
		WithModel(builderModel{}).
		WithBuiltinTools("read").
		WithPluginDirs(dirs...).
		Build(context.Background())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	defer rt.Close()

	dirs[0] = "changed"
	if len(rt.opts.PluginDirs) != 1 || rt.opts.PluginDirs[0] != pluginRoot {
		t.Fatalf("plugin dirs = %#v, want frozen original", rt.opts.PluginDirs)
	}
	if len(rt.opts.plugins) != 1 || rt.opts.plugins[0].Manifest.Name != "external-plugin" {
		t.Fatalf("plugins = %#v, want loaded manifest", rt.opts.plugins)
	}
	if _, ok := rt.opts.skReg.Get("review"); ok {
		t.Fatalf("plugin skill should not be registered without a SKILL.md")
	}
}

func TestRuntimeLoadsPluginSkillCapability(t *testing.T) {
	root := t.TempDir()
	pluginRoot := filepath.Join(root, ".agents", "plugins", "reviewer")
	manifestDir := filepath.Join(pluginRoot, ".codex-plugin")
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatalf("mkdir manifest dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(manifestDir, "plugin.json"), []byte(`{
  "name": "reviewer",
  "version": "0.1.0",
  "capabilities": {"skills": ["skills"]}
}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	skillDir := filepath.Join(pluginRoot, "skills", "review")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(`---
name: review
description: Review current changes
---
Review the current change set.`), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}

	rt, err := NewRuntimeBuilder(root).
		WithModel(builderModel{}).
		WithBuiltinTools("read").
		Build(context.Background())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	defer rt.Close()

	skill, ok := rt.opts.skReg.Get("review")
	if !ok {
		t.Fatalf("plugin skill was not registered")
	}
	source := skill.Definition().Metadata["source"]
	if source == "" || filepath.Base(filepath.Dir(source)) != "review" {
		t.Fatalf("source = %q, want plugin review skill", source)
	}
}

func TestRuntimeScansSkillDirectoryAtSessionStart(t *testing.T) {
	root := t.TempDir()
	model := &capturingToolsModel{}
	rt, err := NewRuntimeBuilder(root).
		WithModel(model).
		WithBuiltinTools("skill_execute").
		Build(context.Background())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	defer rt.Close()

	if _, err := rt.Run(context.Background(), Request{SessionID: "sess_before", Prompt: "hello"}); err != nil {
		t.Fatalf("run before skill: %v", err)
	}
	if model.lastSkillDescriptionContains("weather") {
		t.Fatalf("skill was available before it existed")
	}

	skillDir := filepath.Join(root, "weather", "20260530210600")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(`---
name: weather
description: Query weather
---
Query the weather.`), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}

	if _, err := rt.Run(context.Background(), Request{SessionID: "sess_after", Prompt: "hello again"}); err != nil {
		t.Fatalf("run after skill: %v", err)
	}
	if !model.lastSkillDescriptionContains("weather") {
		t.Fatalf("skill_execute description did not include newly scanned skill: %q", model.lastSkillDescription())
	}
}

func TestRuntimeFiltersLoadedSkillsByAllowedNames(t *testing.T) {
	root := t.TempDir()
	writeBuilderSkill(t, root, "weather", "v1", "Query weather")
	writeBuilderSkill(t, root, "doc-writer", "v1", "Write docs")

	model := &capturingToolsModel{}
	rt, err := New(context.Background(), Options{
		ProjectRoot:          root,
		Model:                model,
		EnabledBuiltinTools:  []string{"skill_execute"},
		AllowedSkills:        []string{"weather"},
		DefaultEnableCache:   false,
		DisableSafetyHook:    true,
		PermissionPrompter:   PermissionPrompterFunc(func(context.Context, PermissionRequest) (bool, error) { return true, nil }),
		MaxIterations:        4,
		StopReinjectionLimit: 1,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer rt.Close()

	if _, err := rt.Run(context.Background(), Request{SessionID: "sess_filtered", Prompt: "hello"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	desc := model.lastSkillDescription()
	if !strings.Contains(desc, "weather") {
		t.Fatalf("skill description = %q, want weather", desc)
	}
	if strings.Contains(desc, "doc-writer") {
		t.Fatalf("skill description = %q, want doc-writer filtered out", desc)
	}
}

func TestRuntimeLoadsPluginSubagentCapability(t *testing.T) {
	root := t.TempDir()
	pluginRoot := filepath.Join(root, ".agents", "plugins", "reviewer")
	manifestDir := filepath.Join(pluginRoot, ".codex-plugin")
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatalf("mkdir manifest dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(manifestDir, "plugin.json"), []byte(`{
  "name": "reviewer",
  "version": "0.1.0",
  "capabilities": {"subagents": ["subagents"]}
}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	subagentDir := filepath.Join(pluginRoot, "subagents")
	if err := os.MkdirAll(subagentDir, 0o755); err != nil {
		t.Fatalf("mkdir subagent dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subagentDir, "reviewer.md"), []byte(`---
name: reviewer
description: Review current changes
tools: read,grep
model: haiku
---
Review the current change set.`), 0o644); err != nil {
		t.Fatalf("write subagent: %v", err)
	}

	rt, err := NewRuntimeBuilder(root).
		WithModel(builderModel{}).
		WithBuiltinTools("read").
		Build(context.Background())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	defer rt.Close()

	defs := rt.opts.subMgr.List()
	for _, def := range defs {
		if def.Name == "reviewer" {
			if def.DefaultModel != "haiku" {
				t.Fatalf("reviewer model = %q, want haiku", def.DefaultModel)
			}
			return
		}
	}
	t.Fatalf("plugin subagent was not registered: %#v", defs)
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func writeBuilderSkill(t *testing.T, root, name, version, description string) {
	t.Helper()
	dir := filepath.Join(root, name, version)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	content := strings.Join([]string{
		"---",
		"name: " + name,
		"description: " + description,
		"metadata:",
		"  version: " + version,
		"---",
		description,
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
}

type builderModel struct{}

func (builderModel) Complete(context.Context, model.Request) (*model.Response, error) {
	return &model.Response{Message: model.Message{Role: "assistant", Content: "ok"}}, nil
}

func (builderModel) CompleteStream(_ context.Context, _ model.Request, cb model.StreamHandler) error {
	return cb(model.StreamResult{Final: true, Response: &model.Response{Message: model.Message{Role: "assistant", Content: "ok"}}})
}

type capturingToolsModel struct {
	tools []model.ToolDefinition
}

func (m *capturingToolsModel) Complete(_ context.Context, req model.Request) (*model.Response, error) {
	m.tools = append([]model.ToolDefinition(nil), req.Tools...)
	return &model.Response{Message: model.Message{Role: "assistant", Content: "ok"}}, nil
}

func (m *capturingToolsModel) CompleteStream(_ context.Context, req model.Request, cb model.StreamHandler) error {
	m.tools = append([]model.ToolDefinition(nil), req.Tools...)
	return cb(model.StreamResult{Final: true, Response: &model.Response{Message: model.Message{Role: "assistant", Content: "ok"}}})
}

func (m *capturingToolsModel) lastSkillDescription() string {
	for _, tool := range m.tools {
		if tool.Name == "skill_execute" {
			return tool.Description
		}
	}
	return ""
}

func (m *capturingToolsModel) lastSkillDescriptionContains(value string) bool {
	return strings.Contains(m.lastSkillDescription(), value)
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
