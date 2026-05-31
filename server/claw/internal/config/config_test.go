package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "claw.toml")
	if err := os.WriteFile(path, []byte(`
http_addr = ":18101"
session_api_url = "http://127.0.0.1:18080"
internal_token = "token"

[gateway_skills]
path = "./skills/active"
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatalf("load file: %v", err)
	}
	if cfg.HTTPAddr != ":18101" || cfg.SessionAPIURL != "http://127.0.0.1:18080" || cfg.InternalToken != "token" {
		t.Fatalf("cfg = %+v", cfg)
	}
	if cfg.DefaultProjectRoot != filepath.Join(filepath.Dir(path), "skills", "active") {
		t.Fatalf("default project root = %q", cfg.DefaultProjectRoot)
	}
}

func TestLoadFileDefaultsUseIcooRuntimeSkills(t *testing.T) {
	dir := t.TempDir()
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

	cfg, err := LoadFile("")
	if err != nil {
		t.Fatalf("load defaults: %v", err)
	}
	want := filepath.Join(dir, "icoo_runtime", "skills")
	if cfg.DefaultProjectRoot != want {
		t.Fatalf("default project root = %q, want %q", cfg.DefaultProjectRoot, want)
	}
}
