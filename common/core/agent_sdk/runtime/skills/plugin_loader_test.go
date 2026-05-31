package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromFSIncludesExplicitSkillDirs(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "plugin", "skills", "review")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	content := `---
name: review
description: Review code changes
---
Review the current change set.`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}

	regs, errs := LoadFromFS(LoaderOptions{
		ProjectRoot: root,
		SkillDirs:   []string{filepath.Join(root, "plugin", "skills")},
	})
	if len(errs) != 0 {
		t.Fatalf("errs = %v, want none", errs)
	}
	if len(regs) != 1 {
		t.Fatalf("registrations len = %d, want 1", len(regs))
	}
	if regs[0].Definition.Name != "review" {
		t.Fatalf("skill name = %q, want review", regs[0].Definition.Name)
	}
}

func TestLoadFromFSIncludesVersionedSkillRoot(t *testing.T) {
	root := t.TempDir()
	for _, version := range []string{"20260530200000", "20260530210600"} {
		skillDir := filepath.Join(root, "weather", version)
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatalf("mkdir skill dir: %v", err)
		}
		content := `---
name: weather
description: Query weather
---
Query the weather.`
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
			t.Fatalf("write skill: %v", err)
		}
	}

	regs, errs := LoadFromFS(LoaderOptions{ProjectRoot: root})
	if len(errs) != 0 {
		t.Fatalf("errs = %v, want none", errs)
	}
	if len(regs) != 1 {
		t.Fatalf("registrations len = %d, want 1", len(regs))
	}
	if regs[0].Definition.Name != "weather" {
		t.Fatalf("skill name = %q, want weather", regs[0].Definition.Name)
	}
	if filepath.Base(filepath.Dir(regs[0].Definition.Metadata["source"])) != "20260530210600" {
		t.Fatalf("source = %v, want latest version", regs[0].Definition.Metadata["source"])
	}
}
