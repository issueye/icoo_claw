package plugins

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const ManifestRelativePath = ".codex-plugin/plugin.json"

var pluginNameRegexp = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,62}[a-z0-9])?$`)

// Manifest describes an installable agent SDK plugin. It is intentionally
// declarative: runtime loaders decide which capabilities to enable.
type Manifest struct {
	Name         string       `json:"name"`
	Version      string       `json:"version,omitempty"`
	Description  string       `json:"description,omitempty"`
	Author       string       `json:"author,omitempty"`
	Homepage     string       `json:"homepage,omitempty"`
	Capabilities Capabilities `json:"capabilities,omitempty"`
}

// Capabilities lists plugin-owned resources using paths relative to the plugin
// root. Paths are kept generic so skills, subagents, and MCP config can evolve
// independently without changing the manifest envelope.
type Capabilities struct {
	Skills    []string `json:"skills,omitempty"`
	Subagents []string `json:"subagents,omitempty"`
	MCP       []string `json:"mcp,omitempty"`
	Tools     []string `json:"tools,omitempty"`
}

// Load reads and validates a plugin manifest from a plugin root directory.
func Load(root string) (Manifest, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return Manifest{}, errors.New("plugins: root is required")
	}
	data, err := os.ReadFile(filepath.Join(root, ManifestRelativePath))
	if err != nil {
		return Manifest{}, fmt.Errorf("plugins: read manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("plugins: decode manifest: %w", err)
	}
	if err := manifest.Validate(); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

// Validate checks cheap manifest invariants before a plugin is admitted.
func (m Manifest) Validate() error {
	name := strings.TrimSpace(m.Name)
	if name == "" {
		return errors.New("plugins: name is required")
	}
	if !pluginNameRegexp.MatchString(name) {
		return fmt.Errorf("plugins: invalid name %q (must be 1-64 chars, lowercase alphanumeric + hyphens, cannot start/end with hyphen)", m.Name)
	}
	if strings.TrimSpace(m.Version) == "" {
		return errors.New("plugins: version is required")
	}
	if err := validateCapabilityPaths("skills", m.Capabilities.Skills); err != nil {
		return err
	}
	if err := validateCapabilityPaths("subagents", m.Capabilities.Subagents); err != nil {
		return err
	}
	if err := validateCapabilityPaths("mcp", m.Capabilities.MCP); err != nil {
		return err
	}
	if err := validateCapabilityPaths("tools", m.Capabilities.Tools); err != nil {
		return err
	}
	return nil
}

func validateCapabilityPaths(field string, paths []string) error {
	seen := map[string]struct{}{}
	for i, path := range paths {
		trimmed := strings.TrimSpace(path)
		if trimmed == "" {
			return fmt.Errorf("plugins: capabilities.%s[%d] is empty", field, i)
		}
		if filepath.IsAbs(trimmed) || strings.HasPrefix(filepath.Clean(trimmed), "..") {
			return fmt.Errorf("plugins: capabilities.%s[%d] must be relative to plugin root", field, i)
		}
		key := filepath.ToSlash(filepath.Clean(trimmed))
		if _, ok := seen[key]; ok {
			return fmt.Errorf("plugins: capabilities.%s[%d] duplicates %q", field, i, key)
		}
		seen[key] = struct{}{}
	}
	return nil
}
