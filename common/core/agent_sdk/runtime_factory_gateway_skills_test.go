package agent_sdk

import (
	"strings"
	"testing"
)

func TestParseAgentProfileIgnoresGatewaySkillsRequestPayload(t *testing.T) {
	profile := parseAgentProfile(map[string]any{
		"gateway_skills": map[string]any{
			"path": "E:/skills/active",
			"skills": []any{
				map[string]any{"name": "doc-writer", "description": "Write docs"},
			},
		},
	})
	if profile.ProjectRoot != "" {
		t.Fatalf("project root = %q, want empty request profile root", profile.ProjectRoot)
	}
}

func TestRuntimeFactoryDefaultProjectRootKeepsRequestProjectRoot(t *testing.T) {
	factory := NewRuntimeFactory(nil, nil)
	factory.SetDefaultProjectRoot("E:/skills/agent")

	profile := parseAgentProfile(map[string]any{"project_root": "E:/user/project"})
	if profile.ProjectRoot != "E:/user/project" {
		t.Fatalf("parsed project root = %q", profile.ProjectRoot)
	}
	if strings.TrimSpace(profile.ProjectRoot) == "" && factory.defaultProjectRoot != "" {
		profile.ProjectRoot = factory.defaultProjectRoot
	}
	if profile.ProjectRoot != "E:/user/project" {
		t.Fatalf("runtime project root = %q, want request project root", profile.ProjectRoot)
	}
}

func TestRuntimeFactoryDefaultProjectRootFillsEmptyProjectRoot(t *testing.T) {
	factory := NewRuntimeFactory(nil, nil)
	factory.SetDefaultProjectRoot("E:/skills/agent")

	profile := parseAgentProfile(nil)
	if strings.TrimSpace(profile.ProjectRoot) == "" && factory.defaultProjectRoot != "" {
		profile.ProjectRoot = factory.defaultProjectRoot
	}
	if profile.ProjectRoot != "E:/skills/agent" {
		t.Fatalf("runtime project root = %q, want gateway skills root", profile.ProjectRoot)
	}
}
