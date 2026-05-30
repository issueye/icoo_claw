package subagents

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromFSIncludesExplicitSubagentDirs(t *testing.T) {
	root := t.TempDir()
	agentDir := filepath.Join(root, "plugin", "subagents")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("mkdir agent dir: %v", err)
	}
	content := `---
name: reviewer
description: Review code changes
tools: read,grep
model: haiku
---
Inspect the change set.`
	if err := os.WriteFile(filepath.Join(agentDir, "reviewer.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write subagent: %v", err)
	}

	regs, errs := LoadFromFS(LoaderOptions{
		ProjectRoot:  root,
		SubagentDirs: []string{agentDir},
	})
	if len(errs) != 0 {
		t.Fatalf("errs = %v, want none", errs)
	}
	if len(regs) != 1 {
		t.Fatalf("registrations len = %d, want 1", len(regs))
	}
	if regs[0].Definition.Name != "reviewer" {
		t.Fatalf("subagent name = %q, want reviewer", regs[0].Definition.Name)
	}
	if regs[0].Definition.DefaultModel != ModelHaiku {
		t.Fatalf("model = %q, want haiku", regs[0].Definition.DefaultModel)
	}
}
