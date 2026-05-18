package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session_store.toml")
	if err := os.WriteFile(path, []byte(`
http_addr = ":18082"
db_path = "./tmp/session_store.sqlite"
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatalf("load file: %v", err)
	}
	if cfg.HTTPAddr != ":18082" || cfg.DBPath != "./tmp/session_store.sqlite" {
		t.Fatalf("cfg = %+v", cfg)
	}
}
