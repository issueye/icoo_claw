package plugins

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromFSDiscoversProjectPlugins(t *testing.T) {
	root := t.TempDir()
	writePluginManifest(t, filepath.Join(root, ".agents", "plugins", "beta"), `{
  "name": "beta-plugin",
  "version": "0.1.0"
}`)
	writePluginManifest(t, filepath.Join(root, ".agents", "plugins", "alpha"), `{
  "name": "alpha-plugin",
  "version": "0.1.0",
  "capabilities": {"skills": ["skills/review"]}
}`)

	registrations, errs := LoadFromFS(LoaderOptions{ProjectRoot: root})
	if len(errs) != 0 {
		t.Fatalf("errs = %v, want none", errs)
	}
	if len(registrations) != 2 {
		t.Fatalf("registrations len = %d, want 2", len(registrations))
	}
	if registrations[0].Manifest.Name != "alpha-plugin" || registrations[1].Manifest.Name != "beta-plugin" {
		t.Fatalf("registrations order = %#v", registrations)
	}
}

func TestLoadFromFSAggregatesInvalidAndDuplicatePlugins(t *testing.T) {
	root := t.TempDir()
	writePluginManifest(t, filepath.Join(root, ".agents", "plugins", "valid"), `{
  "name": "valid-plugin",
  "version": "0.1.0"
}`)
	writePluginManifest(t, filepath.Join(root, ".agents", "plugins", "duplicate"), `{
  "name": "valid-plugin",
  "version": "0.2.0"
}`)
	writePluginManifest(t, filepath.Join(root, ".agents", "plugins", "broken"), `{
  "name": "bad-plugin"
}`)

	registrations, errs := LoadFromFS(LoaderOptions{ProjectRoot: root})
	if len(registrations) != 1 {
		t.Fatalf("registrations len = %d, want 1", len(registrations))
	}
	if len(errs) != 2 {
		t.Fatalf("errs len = %d, want duplicate + invalid: %v", len(errs), errs)
	}
}

func TestLoadFromFSUsesExplicitPluginDirs(t *testing.T) {
	root := t.TempDir()
	explicit := filepath.Join(root, "external-plugin")
	writePluginManifest(t, explicit, `{
  "name": "external-plugin",
  "version": "1.0.0"
}`)

	registrations, errs := LoadFromFS(LoaderOptions{PluginDirs: []string{explicit}})
	if len(errs) != 0 {
		t.Fatalf("errs = %v, want none", errs)
	}
	if len(registrations) != 1 || registrations[0].Manifest.Name != "external-plugin" {
		t.Fatalf("registrations = %#v", registrations)
	}
}

func writePluginManifest(t *testing.T, root string, content string) {
	t.Helper()
	dir := filepath.Join(root, ".codex-plugin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plugin.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("write plugin manifest: %v", err)
	}
}
