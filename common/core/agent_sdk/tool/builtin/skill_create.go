package toolbuiltin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"icoo_claw/common/core/agent_sdk/runtime/skills"
	"icoo_claw/common/core/agent_sdk/tool"
)

var skillCreateSchema = &tool.JSONSchema{
	Type: "object",
	Properties: map[string]interface{}{
		"name": map[string]interface{}{
			"type":        "string",
			"description": "The skill name, lowercase alphanumeric plus hyphens.",
		},
		"description": map[string]interface{}{
			"type":        "string",
			"description": "A concise description of what the skill does.",
		},
		"content": map[string]interface{}{
			"type":        "string",
			"description": "The body of SKILL.md after YAML frontmatter.",
		},
		"version": map[string]interface{}{
			"type":        "string",
			"description": "Optional version. If omitted, a UTC timestamp is used.",
		},
		"allowed_tools": map[string]interface{}{
			"type":        "array",
			"description": "Optional tool names the skill may use.",
		},
		"files": map[string]interface{}{
			"type":        "array",
			"description": "Optional support files under assets/, references/, or scripts/.",
		},
	},
	Required: []string{"name", "description", "content"},
}

var skillCreateNameRegexp = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,62}[a-z0-9])?$`)

// SkillCreateTool writes a filesystem skill into the runtime project root.
type SkillCreateTool struct {
	root     string
	registry *skills.Registry
}

func NewSkillCreateTool(root string, reg *skills.Registry) *SkillCreateTool {
	return &SkillCreateTool{root: filepath.Clean(strings.TrimSpace(root)), registry: reg}
}

func (s *SkillCreateTool) Name() string { return "skill_create" }

func (s *SkillCreateTool) Description() string {
	return "Create or update a skill under the current skills root using skills/<name>/<version>/SKILL.md layout. If version is omitted, generate a timestamp version."
}

func (s *SkillCreateTool) Schema() *tool.JSONSchema { return skillCreateSchema }

func (s *SkillCreateTool) Execute(_ context.Context, params map[string]interface{}) (*tool.ToolResult, error) {
	if s == nil {
		return nil, errors.New("skill_create is not initialised")
	}
	if strings.TrimSpace(s.root) == "" || s.root == "." {
		return nil, errors.New("skill_create requires a project root")
	}
	req, err := parseSkillCreateParams(params)
	if err != nil {
		return nil, err
	}

	version := normalizeSkillCreateVersion(req.version)
	req.version = version
	root := filepath.Join(s.root, req.name, version)
	if err := writeCreatedSkill(root, req); err != nil {
		return nil, err
	}
	for _, file := range req.files {
		if err := writeCreatedSkillSupportFile(root, file); err != nil {
			return nil, err
		}
	}
	if s.registry != nil {
		reg, errs := skills.LoadFromFS(skills.LoaderOptions{ProjectRoot: s.root})
		if len(errs) > 0 {
			return nil, errs[0]
		}
		refreshed, err := registerLoadedSkills(reg)
		if err != nil {
			return nil, err
		}
		s.registry.ReplaceWith(refreshed)
	}

	skillPath := filepath.Join(root, "SKILL.md")
	return &tool.ToolResult{
		Success: true,
		Output:  fmt.Sprintf("skill %s created at %s", req.name, skillPath),
		Data: map[string]any{
			"name":    req.name,
			"version": version,
			"path":    skillPath,
		},
	}, nil
}

type skillCreateRequest struct {
	name         string
	description  string
	content      string
	version      string
	allowedTools []string
	files        []skillCreateFile
}

type skillCreateFile struct {
	path    string
	content string
}

func parseSkillCreateParams(params map[string]interface{}) (skillCreateRequest, error) {
	req := skillCreateRequest{
		name:        strings.ToLower(strings.TrimSpace(fmt.Sprint(params["name"]))),
		description: strings.TrimSpace(fmt.Sprint(params["description"])),
		content:     strings.TrimSpace(fmt.Sprint(params["content"])),
		version:     strings.TrimSpace(fmt.Sprint(params["version"])),
	}
	if !skillCreateNameRegexp.MatchString(req.name) {
		return req, fmt.Errorf("invalid skill name %q (must be 1-64 chars, lowercase alphanumeric + hyphens)", req.name)
	}
	if req.description == "" {
		return req, errors.New("skill description is required")
	}
	if req.content == "" {
		return req, errors.New("skill content is required")
	}
	req.allowedTools = skillCreateStringList(params["allowed_tools"])
	files, err := skillCreateFiles(params["files"])
	if err != nil {
		return req, err
	}
	req.files = files
	return req, nil
}

func writeCreatedSkill(root string, req skillCreateRequest) error {
	if err := os.RemoveAll(root); err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("name: ")
	b.WriteString(quoteSkillYAML(req.name))
	b.WriteString("\ndescription: ")
	b.WriteString(quoteSkillYAML(req.description))
	b.WriteByte('\n')
	if len(req.allowedTools) > 0 {
		b.WriteString("allowed-tools:\n")
		for _, name := range req.allowedTools {
			b.WriteString("  - ")
			b.WriteString(quoteSkillYAML(name))
			b.WriteByte('\n')
		}
	}
	b.WriteString("metadata:\n")
	b.WriteString("  version: ")
	b.WriteString(quoteSkillYAML(normalizeSkillCreateVersion(req.version)))
	b.WriteString("\n---\n")
	b.WriteString(req.content)
	b.WriteByte('\n')
	return os.WriteFile(filepath.Join(root, "SKILL.md"), []byte(b.String()), 0o600)
}

func writeCreatedSkillSupportFile(root string, file skillCreateFile) error {
	rel := filepath.Clean(filepath.FromSlash(strings.TrimSpace(file.path)))
	if rel == "" || rel == "." || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("invalid skill support file path %q", file.path)
	}
	if strings.EqualFold(rel, "SKILL.md") {
		return nil
	}
	top := strings.Split(filepath.ToSlash(rel), "/")[0]
	switch top {
	case "assets", "references", "scripts":
	default:
		return fmt.Errorf("unsupported skill support file path %q", file.path)
	}
	target := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(target, []byte(file.content), 0o600)
}

func normalizeSkillCreateVersion(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "<nil>" {
		return time.Now().UTC().Format("20060102150405")
	}
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-_.")
	if out == "" {
		return time.Now().UTC().Format("20060102150405")
	}
	return out
}

func skillCreateStringList(value any) []string {
	switch v := value.(type) {
	case []string:
		return cleanSkillCreateStrings(v)
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			out = append(out, fmt.Sprint(item))
		}
		return cleanSkillCreateStrings(out)
	case string:
		return cleanSkillCreateStrings(strings.Split(v, ","))
	default:
		return nil
	}
}

func cleanSkillCreateStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := values[:0]
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func skillCreateFiles(value any) ([]skillCreateFile, error) {
	items, ok := value.([]any)
	if !ok || len(items) == 0 {
		return nil, nil
	}
	out := make([]skillCreateFile, 0, len(items))
	for _, item := range items {
		obj, ok := item.(map[string]any)
		if !ok {
			return nil, errors.New("skill files must be objects")
		}
		out = append(out, skillCreateFile{
			path:    strings.TrimSpace(fmt.Sprint(obj["path"])),
			content: fmt.Sprint(obj["content"]),
		})
	}
	return out, nil
}

func registerLoadedSkills(regs []skills.SkillRegistration) (*skills.Registry, error) {
	reg := skills.NewRegistry()
	for _, item := range regs {
		if err := reg.Register(item.Definition, item.Handler); err != nil {
			return nil, err
		}
	}
	return reg, nil
}

func quoteSkillYAML(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, `"`, `\"`)
	value = strings.ReplaceAll(value, "\n", "\\n")
	return `"` + value + `"`
}
