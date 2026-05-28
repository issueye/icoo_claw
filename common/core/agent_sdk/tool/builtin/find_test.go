package toolbuiltin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindToolMatchesFilesAndRespectsGitignore(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, ".gitignore"), "ignored/\n")
	mustWriteFile(t, filepath.Join(root, "src", "main.go"), "package main\n")
	mustWriteFile(t, filepath.Join(root, "src", "notes.md"), "notes\n")
	mustWriteFile(t, filepath.Join(root, "ignored", "skip.go"), "package ignored\n")

	find := NewFindToolWithRoot(root)
	result, err := find.Execute(context.Background(), map[string]interface{}{
		"pattern": "*.go",
		"type":    "file",
	})
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	matches := normalizedFindMatches(t, result.Data)
	if !containsString(matches, "src/main.go") {
		t.Fatalf("matches = %v, want src/main.go", matches)
	}
	if containsString(matches, "ignored/skip.go") {
		t.Fatalf("matches = %v, did not expect ignored/skip.go", matches)
	}
}

func TestFindToolPlainPatternAndScopedPath(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "src", "notes.md"), "notes\n")
	mustWriteFile(t, filepath.Join(root, "docs", "notes.md"), "docs\n")

	find := NewFindToolWithRoot(root)
	result, err := find.Execute(context.Background(), map[string]interface{}{
		"pattern": "notes",
		"path":    "src",
		"type":    "file",
	})
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	matches := normalizedFindMatches(t, result.Data)
	if len(matches) != 1 || matches[0] != "src/notes.md" {
		t.Fatalf("matches = %v, want only src/notes.md", matches)
	}
}

func TestFindToolRejectsPathOutsideRoot(t *testing.T) {
	root := t.TempDir()
	find := NewFindToolWithRoot(root)
	_, err := find.Execute(context.Background(), map[string]interface{}{
		"pattern": "*",
		"path":    filepath.Dir(root),
	})
	if err == nil {
		t.Fatal("find outside root succeeded, want error")
	}
}

func mustWriteFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func normalizedFindMatches(t *testing.T, data interface{}) []string {
	t.Helper()
	payload, ok := data.(map[string]interface{})
	if !ok {
		t.Fatalf("data = %T, want map[string]interface{}", data)
	}
	raw, ok := payload["matches"].([]string)
	if !ok {
		t.Fatalf("matches = %T, want []string", payload["matches"])
	}
	out := make([]string, len(raw))
	for i, match := range raw {
		out[i] = filepath.ToSlash(match)
	}
	return out
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(value, target) {
			return true
		}
	}
	return false
}
