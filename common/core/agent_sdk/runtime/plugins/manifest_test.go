package plugins

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManifestValidateAcceptsMinimalPlugin(t *testing.T) {
	manifest := Manifest{
		Name:    "workspace-helper",
		Version: "0.1.0",
		Capabilities: Capabilities{
			Skills:    []string{"skills/review"},
			Subagents: []string{"subagents/explorer"},
			MCP:       []string{"mcp/devtools.json"},
			Tools:     []string{"tools/tool.json"},
		},
	}

	if err := manifest.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
}

func TestManifestValidateRejectsInvalidPlugin(t *testing.T) {
	cases := []Manifest{
		{Name: "", Version: "0.1.0"},
		{Name: "Bad_Name", Version: "0.1.0"},
		{Name: "valid-name", Version: ""},
		{Name: "valid-name", Version: "0.1.0", Capabilities: Capabilities{Skills: []string{"../escape"}}},
		{Name: "valid-name", Version: "0.1.0", Capabilities: Capabilities{MCP: []string{"", "mcp/dev.json"}}},
		{Name: "valid-name", Version: "0.1.0", Capabilities: Capabilities{Tools: []string{"tools/a.json", "tools/a.json"}}},
	}

	for _, tc := range cases {
		if err := tc.Validate(); err == nil {
			t.Fatalf("expected validation error for %#v", tc)
		}
	}
}

func TestLoadReadsManifestFromPluginRoot(t *testing.T) {
	root := t.TempDir()
	manifestDir := filepath.Join(root, ".codex-plugin")
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatalf("mkdir manifest dir: %v", err)
	}
	content := `{
  "name": "workspace-helper",
  "version": "0.1.0",
  "description": "Helpful plugin",
  "capabilities": {
    "skills": ["skills/review"],
    "subagents": ["subagents/explorer"]
  }
}`
	if err := os.WriteFile(filepath.Join(manifestDir, "plugin.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	manifest, err := Load(root)
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	if manifest.Name != "workspace-helper" || manifest.Version != "0.1.0" {
		t.Fatalf("manifest = %#v", manifest)
	}
	if len(manifest.Capabilities.Skills) != 1 || manifest.Capabilities.Skills[0] != "skills/review" {
		t.Fatalf("skills = %#v", manifest.Capabilities.Skills)
	}
}
