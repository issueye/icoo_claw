package plugins

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type LoaderOptions struct {
	ProjectRoot string
	PluginDirs  []string
}

type Registration struct {
	Root     string
	Manifest Manifest
}

// LoadFromFS discovers plugin manifests from .agents/plugins and any explicit
// plugin directories. Broken plugins are reported as warnings without blocking
// valid plugin registrations.
func LoadFromFS(opts LoaderOptions) ([]Registration, []error) {
	var (
		registrations []Registration
		errs          []error
	)

	candidates := pluginRoots(opts)
	seenRoots := map[string]struct{}{}
	seenNames := map[string]string{}
	for _, root := range candidates {
		cleanRoot := filepath.Clean(root)
		if _, ok := seenRoots[cleanRoot]; ok {
			continue
		}
		seenRoots[cleanRoot] = struct{}{}

		manifest, err := Load(cleanRoot)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			errs = append(errs, fmt.Errorf("plugins: load %s: %w", cleanRoot, err))
			continue
		}
		name := strings.ToLower(strings.TrimSpace(manifest.Name))
		if prev, ok := seenNames[name]; ok {
			errs = append(errs, fmt.Errorf("plugins: duplicate plugin %q at %s (already from %s)", manifest.Name, cleanRoot, prev))
			continue
		}
		seenNames[name] = cleanRoot
		registrations = append(registrations, Registration{
			Root:     cleanRoot,
			Manifest: manifest,
		})
	}

	sort.Slice(registrations, func(i, j int) bool {
		return registrations[i].Manifest.Name < registrations[j].Manifest.Name
	})
	return registrations, errs
}

func pluginRoots(opts LoaderOptions) []string {
	var roots []string
	projectRoot := strings.TrimSpace(opts.ProjectRoot)
	if projectRoot != "" {
		pluginsDir := filepath.Join(projectRoot, ".agents", "plugins")
		entries, err := os.ReadDir(pluginsDir)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					roots = append(roots, filepath.Join(pluginsDir, entry.Name()))
				}
			}
		}
	}
	for _, dir := range opts.PluginDirs {
		if trimmed := strings.TrimSpace(dir); trimmed != "" {
			roots = append(roots, trimmed)
		}
	}
	return roots
}
