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
