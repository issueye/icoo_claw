package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveExecutablePathUsesCurrentDirectoryBinary(t *testing.T) {
	dir := t.TempDir()
	name := "claw.exe"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(""), 0o755); err != nil {
		t.Fatalf("write binary: %v", err)
	}

	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	}()

	resolved, err := resolveExecutablePath(name, "")
	if err != nil {
		t.Fatalf("resolve executable: %v", err)
	}
	if resolved != filepath.Join(dir, name) {
		t.Fatalf("resolved = %q, want %q", resolved, filepath.Join(dir, name))
	}
}

func TestWriteClawConfigReturnsAbsolutePath(t *testing.T) {
	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	}()

	path, err := writeClawConfig(StartAgentInstanceSpec{
		InstanceID: "inst_test",
		AgentID:    "agent_test",
		Host:       "127.0.0.1",
		Port:       8101,
		ConfigDir:  "data/claw-configs",
	})
	if err != nil {
		t.Fatalf("write config: %v", err)
	}
	if !filepath.IsAbs(path) {
		t.Fatalf("path = %q, want absolute", path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stat config: %v", err)
	}
}
