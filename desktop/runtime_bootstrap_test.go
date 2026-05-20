package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindBundledPackageRootNear(t *testing.T) {
	t.Parallel()

	packageRoot := t.TempDir()
	binDir := filepath.Join(packageRoot, "bin")
	configDir := filepath.Join(packageRoot, "runtime", "config")
	for _, dir := range []string{binDir, configDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}
	}
	for _, name := range []string{"gateway.exe", "claw.exe"} {
		if err := os.WriteFile(filepath.Join(binDir, name), []byte("test"), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
	}
	gatewayConfigPath := filepath.Join(configDir, "gateway.toml")
	if err := os.WriteFile(gatewayConfigPath, []byte("http_addr = \"127.0.0.1:8080\""), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	tests := []string{
		packageRoot,
		binDir,
		filepath.Join(binDir, "gateway.exe"),
		gatewayConfigPath,
	}

	for _, input := range tests {
		input := input
		t.Run(input, func(t *testing.T) {
			root, ok, err := findBundledPackageRootNear(input)
			if err != nil {
				t.Fatalf("findBundledPackageRootNear() error = %v", err)
			}
			if !ok {
				t.Fatalf("findBundledPackageRootNear() ok = false")
			}
			if root != packageRoot {
				t.Fatalf("root = %q, want %q", root, packageRoot)
			}
		})
	}
}

func TestBuildRuntimeConfigDetectsCompleteBundle(t *testing.T) {
	t.Parallel()

	packageRoot := newTestBundle(t)
	cfg, err := buildRuntimeConfig(packageRoot, "http://127.0.0.1:8099")
	if err != nil {
		t.Fatalf("buildRuntimeConfig() error = %v", err)
	}
	if cfg == nil {
		t.Fatalf("buildRuntimeConfig() = nil")
	}
	if cfg.packageRoot != packageRoot {
		t.Fatalf("packageRoot = %q, want %q", cfg.packageRoot, packageRoot)
	}
	if cfg.gatewayPort != 8099 {
		t.Fatalf("gatewayPort = %d, want 8099", cfg.gatewayPort)
	}
}

func TestResolveConfiguredBundledPackageRootPrefersProgramPath(t *testing.T) {
	t.Parallel()

	packageRoot := newTestBundle(t)
	root, ok, err := resolveConfiguredBundledPackageRoot(filepath.Join(packageRoot, "bin", "gateway.exe"), "")
	if err != nil {
		t.Fatalf("resolveConfiguredBundledPackageRoot() error = %v", err)
	}
	if !ok {
		t.Fatalf("resolveConfiguredBundledPackageRoot() ok = false")
	}
	if root != packageRoot {
		t.Fatalf("root = %q, want %q", root, packageRoot)
	}
}

func TestResolveConfiguredBundledPackageRootUsesConfigOnlyWithoutProgramPath(t *testing.T) {
	t.Parallel()

	packageRoot := newTestBundle(t)
	configDir := filepath.Join(packageRoot, "runtime", "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	configPath := filepath.Join(configDir, "gateway.toml")
	if err := os.WriteFile(configPath, []byte("http_addr = \"127.0.0.1:8080\""), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	root, ok, err := resolveConfiguredBundledPackageRoot("", configPath)
	if err != nil {
		t.Fatalf("resolveConfiguredBundledPackageRoot() error = %v", err)
	}
	if !ok {
		t.Fatalf("resolveConfiguredBundledPackageRoot() ok = false")
	}
	if root != packageRoot {
		t.Fatalf("root = %q, want %q", root, packageRoot)
	}
}

func TestBuildRuntimeConfigReturnsNilWhenBundleIsIncomplete(t *testing.T) {
	t.Parallel()

	packageRoot := t.TempDir()
	binDir := filepath.Join(packageRoot, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "gateway.exe"), []byte("test"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := buildRuntimeConfig(packageRoot, "http://127.0.0.1:8080")
	if err != nil {
		t.Fatalf("buildRuntimeConfig() error = %v", err)
	}
	if cfg != nil {
		t.Fatalf("buildRuntimeConfig() = %#v, want nil", cfg)
	}
}

func TestFindBundledPackageRootNearReturnsFalseForStandaloneGateway(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	programPath := filepath.Join(dir, "gateway.exe")
	if err := os.WriteFile(programPath, []byte("test"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	root, ok, err := findBundledPackageRootNear(programPath)
	if err != nil {
		t.Fatalf("findBundledPackageRootNear() error = %v", err)
	}
	if ok {
		t.Fatalf("ok = true, root = %q", root)
	}
}

func newTestBundle(t *testing.T) string {
	t.Helper()

	packageRoot := t.TempDir()
	binDir := filepath.Join(packageRoot, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	for _, name := range []string{"gateway.exe", "claw.exe"} {
		if err := os.WriteFile(filepath.Join(binDir, name), []byte("test"), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
	}
	return packageRoot
}
