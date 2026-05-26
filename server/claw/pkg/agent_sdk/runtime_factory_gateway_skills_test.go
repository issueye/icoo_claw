package agent_sdk

import "testing"

func TestGatewaySkillsFromJSON(t *testing.T) {
	got := GatewaySkillsFromJSON(`{"gateway_skills":{"path":"E:/skills/active","skills":[{"name":"doc-writer","Description":"Write docs"}]}}`)
	if got.Path != "E:/skills/active" {
		t.Fatalf("path = %q", got.Path)
	}
	if len(got.Skills) != 1 || got.Skills[0].Name != "doc-writer" || got.Skills[0].Description != "Write docs" {
		t.Fatalf("skills = %+v", got.Skills)
	}
}

func TestParseAgentProfileUsesGatewaySkillsPathAsProjectRoot(t *testing.T) {
	profile := parseAgentProfile(map[string]any{
		"gateway_skills": map[string]any{
			"path": "E:/skills/active",
			"skills": []any{
				map[string]any{"name": "doc-writer", "description": "Write docs"},
			},
		},
	})
	if profile.ProjectRoot != "E:/skills/active" {
		t.Fatalf("project root = %q", profile.ProjectRoot)
	}
	if len(profile.GatewaySkills.Skills) != 1 || profile.GatewaySkills.Skills[0].Name != "doc-writer" {
		t.Fatalf("gateway skills = %+v", profile.GatewaySkills)
	}
}
