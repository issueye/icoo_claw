package toolbuiltin

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"icoo_claw/common/core/agent_sdk/runtime/skills"
)

func TestSkillCreateToolWritesVersionedSkillAndRefreshesRegistry(t *testing.T) {
	root := t.TempDir()
	reg := skills.NewRegistry()
	create := NewSkillCreateTool(root, reg)

	res, err := create.Execute(context.Background(), map[string]any{
		"name":          "weather",
		"description":   "Query weather",
		"content":       "Use weather data to answer.",
		"version":       "v1",
		"allowed_tools": []any{"fetch", "read"},
		"files": []any{
			map[string]any{"path": "references/cities.md", "content": "成都"},
		},
	})
	if err != nil {
		t.Fatalf("create skill: %v", err)
	}
	if !res.Success {
		t.Fatalf("success = false")
	}
	path := filepath.Join(root, "weather", "v1", "SKILL.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read skill: %v", err)
	}
	if !strings.Contains(string(data), `name: "weather"`) || !strings.Contains(string(data), `- "fetch"`) {
		t.Fatalf("skill content = %s", data)
	}
	if _, err := os.Stat(filepath.Join(root, "weather", "v1", "references", "cities.md")); err != nil {
		t.Fatalf("support file missing: %v", err)
	}
	if _, ok := reg.Get("weather"); !ok {
		t.Fatalf("registry was not refreshed with created skill")
	}
}

func TestSkillCreateToolGeneratesTimestampVersion(t *testing.T) {
	root := t.TempDir()
	create := NewSkillCreateTool(root, nil)

	res, err := create.Execute(context.Background(), map[string]any{
		"name":        "weather",
		"description": "Query weather",
		"content":     "Use weather data to answer.",
	})
	if err != nil {
		t.Fatalf("create skill: %v", err)
	}
	data, _ := res.Data.(map[string]any)
	version, _ := data["version"].(string)
	if !regexp.MustCompile(`^\d{14}$`).MatchString(version) {
		t.Fatalf("version = %q, want timestamp", version)
	}
	if _, err := os.Stat(filepath.Join(root, "weather", version, "SKILL.md")); err != nil {
		t.Fatalf("timestamp skill missing: %v", err)
	}
}

func TestSkillCreateToolRejectsEscapingSupportFile(t *testing.T) {
	create := NewSkillCreateTool(t.TempDir(), nil)

	_, err := create.Execute(context.Background(), map[string]any{
		"name":        "weather",
		"description": "Query weather",
		"content":     "Use weather data to answer.",
		"files": []any{
			map[string]any{"path": "../outside.txt", "content": "nope"},
		},
	})
	if err == nil {
		t.Fatal("expected invalid support file path error")
	}
}
