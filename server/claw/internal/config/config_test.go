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
session_store_url = "http://127.0.0.1:18082"
internal_token = "token"
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatalf("load file: %v", err)
	}
	if cfg.HTTPAddr != ":18101" || cfg.SessionStoreURL != "http://127.0.0.1:18082" || cfg.InternalToken != "token" {
		t.Fatalf("cfg = %+v", cfg)
	}
}
