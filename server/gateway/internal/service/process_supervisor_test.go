package service

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"icoo_claw/server/gateway/internal/config"
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

func TestResolveExecutablePathUsesExtraExecutableDirectory(t *testing.T) {
	dir := t.TempDir()
	name := "claw.exe"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(""), 0o755); err != nil {
		t.Fatalf("write binary: %v", err)
	}

	commandName := name
	if runtime.GOOS == "windows" {
		commandName = "claw"
	}
	resolved, err := resolveExecutablePath(commandName, "", dir)
	if err != nil {
		t.Fatalf("resolve executable: %v", err)
	}
	if filepath.Dir(resolved) != dir {
		t.Fatalf("resolved = %q, want directory %q", resolved, dir)
	}
}

func TestAgentProcessEnvPrependsBinDirToPath(t *testing.T) {
	got := agentProcessEnv([]string{"Path=C:\\Windows"}, "C:\\pkg\\bin")
	if len(got) != 1 {
		t.Fatalf("env = %+v", got)
	}
	if got[0] != "Path=C:\\pkg\\bin;C:\\Windows" {
		t.Fatalf("Path env = %q", got[0])
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
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if strings.Contains(string(payload), "json =") {
		t.Fatalf("gateway skills JSON should not be written: %s", payload)
	}
}

func TestACPCommandArgvSplitsSingleLineCommand(t *testing.T) {
	got, err := acpCommandArgv([]string{`npx @zed-industries/codex-acp --flag "two words"`})
	if err != nil {
		t.Fatalf("parse acp command: %v", err)
	}
	want := []string{"npx", "@zed-industries/codex-acp", "--flag", "two words"}
	if !stringSlicesEqual(got, want) {
		t.Fatalf("argv = %+v, want %+v", got, want)
	}
}

func TestACPCommandArgvKeepsWindowsPathBackslashes(t *testing.T) {
	got, err := acpCommandArgv([]string{`"C:\Program Files\Claw\claw.exe" --acp`})
	if err != nil {
		t.Fatalf("parse acp command: %v", err)
	}
	want := []string{`C:\Program Files\Claw\claw.exe`, "--acp"}
	if !stringSlicesEqual(got, want) {
		t.Fatalf("argv = %+v, want %+v", got, want)
	}
}

func TestACPCommandArgvKeepsMultiLineCommandParts(t *testing.T) {
	got, err := acpCommandArgv([]string{" claw ", " --acp ", ""})
	if err != nil {
		t.Fatalf("parse acp command: %v", err)
	}
	want := []string{"claw", "--acp"}
	if !stringSlicesEqual(got, want) {
		t.Fatalf("argv = %+v, want %+v", got, want)
	}
}

func TestACPCommandArgvRequiresCommand(t *testing.T) {
	if _, err := acpCommandArgv(nil); err == nil {
		t.Fatal("expected missing command error")
	}
}

func TestProcessSpecFromConfigDefaultsToHTTPTransport(t *testing.T) {
	workDir := t.TempDir()
	spec := processSpecFromConfig(config.Config{
		ClawPortStart:  8101,
		ClawPortEnd:    8102,
		GatewayWorkDir: workDir,
	}, "inst_http", "agent_1", 8101)

	if spec.Transport != "http" {
		t.Fatalf("transport = %q, want http", spec.Transport)
	}
	if spec.BaseURL != "http://127.0.0.1:8101" {
		t.Fatalf("baseURL = %q, want http instance URL", spec.BaseURL)
	}
	if spec.DefaultProjectRoot != "" {
		t.Fatalf("default project root = %q, want empty (set by skill service)", spec.DefaultProjectRoot)
	}
}
