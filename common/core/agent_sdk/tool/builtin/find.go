package toolbuiltin

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"icoo_claw/common/core/agent_sdk/gitignore"
	"icoo_claw/common/core/agent_sdk/sandbox"
	"icoo_claw/common/core/agent_sdk/tool"

	"github.com/bmatcuk/doublestar/v4"
)

const (
	findResultLimit = 100
	findToolDesc    = `Find files or directories by name, relative path, or glob pattern within the configured sandbox.
Plain patterns match case-insensitive substrings; glob patterns such as "*.go" or "src/**/*.ts" use path matching.
Results are relative to the sandbox root and limited to 100 entries.`
)

var (
	findSchema = &tool.JSONSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"pattern": map[string]interface{}{
				"type":        "string",
				"description": "Name, relative path substring, or glob pattern to find. Examples: README, *.go, src/**/*.ts.",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Directory to search in, relative to the sandbox root. Omit to search the root.",
			},
			"type": map[string]interface{}{
				"type":        "string",
				"description": "Which entries to return: any, file, or dir.",
				"enum":        []interface{}{"any", "file", "dir"},
				"default":     "any",
			},
			"max_results": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of results to return, capped at 100.",
			},
		},
		Required: []string{"pattern"},
	}
	errFindLimitReached = errors.New("find: result limit reached")
)

// FindTool looks up files and directories by path pattern.
type FindTool struct {
	policy           sandbox.FileSystemPolicy
	root             string
	maxResults       int
	respectGitignore bool
	gitignoreMatcher *gitignore.Matcher
}

// NewFindTool builds a FindTool rooted at the current directory.
func NewFindTool() *FindTool { return NewFindToolWithRoot("") }

// NewFindToolWithRoot builds a FindTool rooted at the provided directory.
func NewFindToolWithRoot(root string) *FindTool {
	resolved := resolveRoot(root)
	return &FindTool{
		policy:           sandbox.NewFileSystemAllowList(resolved),
		root:             resolved,
		maxResults:       findResultLimit,
		respectGitignore: true,
	}
}

// NewFindToolWithSandbox builds a FindTool using a custom sandbox.
func NewFindToolWithSandbox(root string, policy sandbox.FileSystemPolicy) *FindTool {
	resolved := resolveRoot(root)
	return &FindTool{
		policy:           policy,
		root:             resolved,
		maxResults:       findResultLimit,
		respectGitignore: true,
	}
}

// SetRespectGitignore configures whether the tool should respect .gitignore patterns.
func (f *FindTool) SetRespectGitignore(respect bool) {
	f.respectGitignore = respect
	if respect && f.gitignoreMatcher == nil {
		f.gitignoreMatcher, _ = gitignore.NewMatcher(f.root) //nolint:errcheck // best-effort gitignore
	}
}

func (f *FindTool) Name() string { return "find" }

func (f *FindTool) Description() string { return findToolDesc }

func (f *FindTool) Schema() *tool.JSONSchema { return findSchema }

func (f *FindTool) Metadata() tool.Metadata {
	return tool.Metadata{IsReadOnly: true, IsConcurrencySafe: true}
}

func (f *FindTool) Execute(ctx context.Context, params map[string]interface{}) (*tool.ToolResult, error) {
	if ctx == nil {
		return nil, errors.New("context is nil")
	}
	if f == nil {
		return nil, errors.New("find tool is not initialised")
	}

	pattern, err := parseFindPattern(params)
	if err != nil {
		return nil, err
	}
	entryType, err := parseFindType(params)
	if err != nil {
		return nil, err
	}
	limit, err := f.parseFindLimit(params)
	if err != nil {
		return nil, err
	}
	root, err := f.resolveSearchRoot(params)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if f.respectGitignore && f.gitignoreMatcher == nil {
		f.gitignoreMatcher, _ = gitignore.NewMatcher(f.root) //nolint:errcheck // best-effort gitignore
	}

	results := make([]string, 0, minInt(8, limit))
	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}

		clean := filepath.Clean(path)
		if f.policy != nil {
			if err := f.policy.Validate(clean); err != nil {
				return err
			}
		}

		relPath := displayPath(clean, f.root)
		if relPath == "." {
			return nil
		}
		isDir := d.IsDir()
		if f.respectGitignore && f.gitignoreMatcher != nil && f.gitignoreMatcher.Match(relPath, isDir) {
			if isDir {
				return filepath.SkipDir
			}
			return nil
		}
		if !matchesFindType(entryType, isDir) {
			return nil
		}
		if !matchesFindPattern(pattern, relPath) {
			return nil
		}

		results = append(results, relPath)
		if len(results) >= limit {
			return errFindLimitReached
		}
		return nil
	})

	truncated := errors.Is(walkErr, errFindLimitReached)
	if walkErr != nil && !truncated {
		return nil, fmt.Errorf("find failed: %w", walkErr)
	}

	return &tool.ToolResult{
		Success: true,
		Output:  formatFindOutput(results, truncated),
		Data: map[string]interface{}{
			"pattern":     pattern,
			"path":        displayPath(root, f.root),
			"type":        entryType,
			"matches":     results,
			"count":       len(results),
			"max_results": limit,
			"truncated":   truncated,
		},
	}, nil
}

func parseFindPattern(params map[string]interface{}) (string, error) {
	if params == nil {
		return "", errors.New("params is nil")
	}
	raw, ok := params["pattern"]
	if !ok {
		return "", errors.New("pattern is required")
	}
	pattern, err := coerceString(raw)
	if err != nil {
		return "", fmt.Errorf("pattern must be string: %w", err)
	}
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return "", errors.New("pattern cannot be empty")
	}
	return filepath.ToSlash(pattern), nil
}

func parseFindType(params map[string]interface{}) (string, error) {
	if params == nil {
		return "any", nil
	}
	raw, ok := params["type"]
	if !ok || raw == nil {
		return "any", nil
	}
	value, err := coerceString(raw)
	if err != nil {
		return "", fmt.Errorf("type must be string: %w", err)
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "any", "all":
		return "any", nil
	case "file", "files":
		return "file", nil
	case "dir", "dirs", "directory", "directories":
		return "dir", nil
	default:
		return "", fmt.Errorf("type must be one of any, file, dir")
	}
}

func (f *FindTool) parseFindLimit(params map[string]interface{}) (int, error) {
	limit := f.maxResults
	if limit <= 0 {
		limit = findResultLimit
	}
	if params == nil {
		return limit, nil
	}
	raw, ok := params["max_results"]
	if !ok || raw == nil {
		return limit, nil
	}
	value, err := intFromParam(raw)
	if err != nil {
		return 0, fmt.Errorf("max_results must be an integer: %w", err)
	}
	if value <= 0 {
		return 0, errors.New("max_results must be > 0")
	}
	if value > limit {
		return limit, nil
	}
	return value, nil
}

func (f *FindTool) resolveSearchRoot(params map[string]interface{}) (string, error) {
	dir := f.root
	if params != nil {
		if raw, ok := params["path"]; ok && raw != nil {
			value, err := coerceString(raw)
			if err != nil {
				return "", fmt.Errorf("path must be string: %w", err)
			}
			value = strings.TrimSpace(value)
			if value != "" {
				dir = value
			}
		}
	}
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(f.root, dir)
	}
	dir = filepath.Clean(dir)
	if f.policy != nil {
		if err := f.policy.Validate(dir); err != nil {
			return "", err
		}
	}
	info, err := os.Stat(dir)
	if err != nil {
		return "", fmt.Errorf("stat dir: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s is not a directory", dir)
	}
	return dir, nil
}

func matchesFindType(entryType string, isDir bool) bool {
	switch entryType {
	case "file":
		return !isDir
	case "dir":
		return isDir
	default:
		return true
	}
}

func matchesFindPattern(pattern, relPath string) bool {
	rel := filepath.ToSlash(relPath)
	base := filepath.Base(rel)
	if strings.ContainsAny(pattern, "*?[") {
		if ok, _ := doublestar.PathMatch(pattern, rel); ok {
			return true
		}
		if !strings.Contains(pattern, "/") {
			if ok, _ := doublestar.PathMatch(pattern, base); ok {
				return true
			}
			ok, _ := doublestar.PathMatch("**/"+pattern, rel)
			return ok
		}
		return false
	}
	needle := strings.ToLower(pattern)
	return strings.Contains(strings.ToLower(rel), needle) || strings.Contains(strings.ToLower(base), needle)
}

func formatFindOutput(matches []string, truncated bool) string {
	if len(matches) == 0 {
		return "no matches"
	}
	output := strings.Join(matches, "\n")
	if truncated {
		output += fmt.Sprintf("\n... truncated to %d results", len(matches))
	}
	return output
}
