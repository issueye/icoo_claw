package di

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureSQLiteParentDirCreatesMissingDirectory(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "data", "gateway.sqlite")
	if err := ensureSQLiteParentDir(dbPath); err != nil {
		t.Fatalf("ensureSQLiteParentDir() error = %v", err)
	}
	if info, err := os.Stat(filepath.Dir(dbPath)); err != nil || !info.IsDir() {
		t.Fatalf("parent dir was not created: info=%v err=%v", info, err)
	}
}

func TestEnsureSQLiteParentDirIgnoresMemoryDatabase(t *testing.T) {
	t.Parallel()

	for _, dbPath := range []string{":memory:", "file::memory:?cache=shared"} {
		if err := ensureSQLiteParentDir(dbPath); err != nil {
			t.Fatalf("ensureSQLiteParentDir(%q) error = %v", dbPath, err)
		}
	}
}
