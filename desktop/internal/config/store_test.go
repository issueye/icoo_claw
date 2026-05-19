package config

import (
	"path/filepath"
	"testing"
)

func TestStoreLoadReturnsDefaultsWhenFileIsMissing(t *testing.T) {
	t.Parallel()

	store := NewStore(filepath.Join(t.TempDir(), "settings.toml"))
	settings, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if settings.Gateway.BaseURL != "http://127.0.0.1:8080" {
		t.Fatalf("BaseURL = %q", settings.Gateway.BaseURL)
	}
	if settings.Gateway.ProgramPath != "" {
		t.Fatalf("ProgramPath = %q", settings.Gateway.ProgramPath)
	}
	if settings.Gateway.ConfigPath != "" {
		t.Fatalf("ConfigPath = %q", settings.Gateway.ConfigPath)
	}
	if !settings.UI.ShowTimestamps {
		t.Fatalf("ShowTimestamps = false, want true")
	}
}

func TestStoreSaveNormalizesAndRoundTrips(t *testing.T) {
	t.Parallel()

	store := NewStore(filepath.Join(t.TempDir(), "settings.toml"))
	_, err := store.Save(Settings{
		Gateway: GatewaySettings{
			BaseURL:        " http://localhost:8088/ ",
			DefaultAgentID: "  agent_main  ",
			ProgramPath:    "  C:/gateway/gateway.exe  ",
			ConfigPath:     "  C:/gateway/gateway.toml  ",
		},
		Workspace: WorkspaceSettings{
			RootDir: "  E:/workspace/demo  ",
		},
		UI: UISettings{
			ShowTimestamps: true,
		},
	})
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.Gateway.BaseURL != "http://localhost:8088" {
		t.Fatalf("BaseURL = %q", loaded.Gateway.BaseURL)
	}
	if loaded.Gateway.DefaultAgentID != "agent_main" {
		t.Fatalf("DefaultAgentID = %q", loaded.Gateway.DefaultAgentID)
	}
	if loaded.Gateway.ProgramPath != "C:/gateway/gateway.exe" {
		t.Fatalf("ProgramPath = %q", loaded.Gateway.ProgramPath)
	}
	if loaded.Gateway.ConfigPath != "C:/gateway/gateway.toml" {
		t.Fatalf("ConfigPath = %q", loaded.Gateway.ConfigPath)
	}
	if loaded.Workspace.RootDir != "E:/workspace/demo" {
		t.Fatalf("RootDir = %q", loaded.Workspace.RootDir)
	}
}
