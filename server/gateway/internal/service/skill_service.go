package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"icoo_claw/common/id"
	"icoo_claw/common/jsonutil"
	"icoo_claw/server/gateway/internal/dto"
	"icoo_claw/server/gateway/internal/model"
	"icoo_claw/server/gateway/internal/repository"

	"github.com/goccy/go-yaml"
)

type SkillService struct {
	baseDir string
	repo    repository.SkillRepository
}

var skillNameRegexp = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,62}[a-z0-9])?$`)

func NewSkillService(baseDir string, repo repository.SkillRepository) *SkillService {
	return &SkillService{baseDir: filepath.Clean(strings.TrimSpace(baseDir)), repo: repo}
}

func (s *SkillService) EnsureLayout() error {
	if strings.TrimSpace(s.baseDir) == "" || s.baseDir == "." {
		return fmt.Errorf("gateway skills base dir is required")
	}
	if err := os.MkdirAll(s.baseDir, 0o755); err != nil {
		return err
	}
	return nil
}

func (s *SkillService) Create(ctx context.Context, req dto.CreateSkillRequest) (*dto.SkillProfile, error) {
	now := time.Now().UTC()
	skill := model.SkillProfile{
		ID:               strings.TrimSpace(req.ID),
		Name:             strings.TrimSpace(req.Name),
		Description:      strings.TrimSpace(req.Description),
		Path:             strings.TrimSpace(req.Path),
		Content:          strings.TrimSpace(req.Content),
		Version:          normalizeSkillVersion(req.Version),
		Status:           "active",
		Source:           strings.TrimSpace(req.Source),
		AllowedToolsJSON: mustStringSliceJSON(req.AllowedTools),
		MetadataJSON:     mustAnyJSON(req.Metadata),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if skill.ID == "" {
		skill.ID = "skill_" + id.Random()
	}
	if !isValidSkillName(skill.Name) {
		return nil, invalidSkillNameError(skill.Name)
	}
	if err := s.publishSkillFiles(skill, req.Files); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, skill); err != nil {
		return nil, err
	}
	return toSkillDTO(skill), nil
}

func (s *SkillService) Get(ctx context.Context, id string) (*dto.SkillProfile, error) {
	skill, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return toSkillDTO(*skill), nil
}

func (s *SkillService) List(ctx context.Context) ([]dto.SkillProfile, error) {
	if err := s.syncFilesystemSkills(ctx); err != nil {
		return nil, err
	}
	skills, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]dto.SkillProfile, len(skills))
	for i, skill := range skills {
		out[i] = *toSkillDTO(skill)
	}
	return out, nil
}

func (s *SkillService) Update(ctx context.Context, id string, req dto.UpdateSkillRequest) (*dto.SkillProfile, error) {
	skill, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		skill.Name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		skill.Description = strings.TrimSpace(*req.Description)
	}
	if req.Path != nil {
		skill.Path = strings.TrimSpace(*req.Path)
	}
	if req.Content != nil {
		skill.Content = strings.TrimSpace(*req.Content)
	}
	if req.Version != nil {
		skill.Version = normalizeSkillVersion(*req.Version)
	}
	if req.Status != nil {
		skill.Status = normalizeSkillStatus(*req.Status)
	}
	if req.Source != nil {
		skill.Source = strings.TrimSpace(*req.Source)
	}
	if req.AllowedTools != nil {
		skill.AllowedToolsJSON = mustStringSliceJSON(*req.AllowedTools)
	}
	if req.Metadata != nil {
		skill.MetadataJSON = mustAnyJSON(req.Metadata)
	}
	if !isValidSkillName(skill.Name) {
		return nil, invalidSkillNameError(skill.Name)
	}
	skill.UpdatedAt = time.Now().UTC()
	if err := s.publishSkillFiles(*skill, req.Files); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, *skill); err != nil {
		return nil, err
	}
	return toSkillDTO(*skill), nil
}

func (s *SkillService) Delete(ctx context.Context, id string) error {
	skill, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := s.removeSkillFiles(*skill); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

func (s *SkillService) Download(ctx context.Context, id string) ([]byte, string, error) {
	skill, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, "", err
	}
	path := filepath.Join(s.skillVersionPath(*skill), "SKILL.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	return data, skill.Name + "-SKILL.md", nil
}

func (s *SkillService) SyncSummary(ctx context.Context) (*dto.SkillSummary, error) {
	if err := s.syncFilesystemSkills(ctx); err != nil {
		return nil, err
	}
	skills, err := s.repo.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]dto.SkillSummaryItem, 0, len(skills))
	for _, skill := range skills {
		items = append(items, dto.SkillSummaryItem{
			Name:        skill.Name,
			Description: skill.Description,
			Version:     skill.Version,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return &dto.SkillSummary{
		Skills: items,
	}, nil
}

// RuntimeRoot validates agent skill bindings and returns the single gateway
// skill root used by both management APIs and Claw runtime loading.
func (s *SkillService) RuntimeRoot(ctx context.Context, skillNamesJSON string) (string, error) {
	if strings.TrimSpace(s.baseDir) == "" {
		return "", nil
	}
	if err := s.syncFilesystemSkills(ctx); err != nil {
		return "", err
	}
	names := jsonutil.CleanStringSlice(jsonutil.UnmarshalStringSlice(skillNamesJSON))
	for _, name := range names {
		if !isValidSkillName(name) {
			return "", invalidSkillNameError(name)
		}
	}
	if len(names) == 0 {
		return s.root(), nil
	}

	for _, name := range names {
		skill, err := s.repo.GetByName(ctx, name)
		if err != nil {
			return "", err
		}
		if skill.Status != "active" {
			return "", fmt.Errorf("skill %s is not active", name)
		}
	}
	return s.root(), nil
}

func (s *SkillService) publishSkillFiles(skill model.SkillProfile, files []dto.SkillFile) error {
	if strings.TrimSpace(s.baseDir) == "" {
		return nil
	}
	if strings.TrimSpace(skill.Name) == "" || strings.TrimSpace(skill.Path) == "" {
		return fmt.Errorf("skill name and path are required")
	}
	if skill.Status != "active" {
		return os.RemoveAll(s.skillVersionPath(skill))
	}
	return writeSkillPackage(s.skillVersionPath(skill), skill, files)
}

func writeSkillPackage(root string, skill model.SkillProfile, files []dto.SkillFile) error {
	if err := os.RemoveAll(root); err != nil {
		return err
	}
	if err := writeSkillFile(root, skill); err != nil {
		return err
	}
	for _, file := range files {
		if err := writeSupportFile(root, file); err != nil {
			return err
		}
	}
	return nil
}

func writeSkillFile(root string, skill model.SkillProfile) error {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString("name: ")
	sb.WriteString(escapeYAMLScalar(skill.Name))
	sb.WriteString("\ndescription: ")
	sb.WriteString(escapeYAMLScalar(skill.Description))
	sb.WriteByte('\n')
	tools := parseStringSliceRaw(skill.AllowedToolsJSON)
	if len(tools) > 0 {
		sb.WriteString("allowed-tools:\n")
		for _, t := range tools {
			sb.WriteString("  - ")
			sb.WriteString(escapeYAMLScalar(t))
			sb.WriteByte('\n')
		}
	}
	sb.WriteString("metadata:\n")
	sb.WriteString("  version: ")
	sb.WriteString(escapeYAMLScalar(skill.Version))
	sb.WriteString("\n  gateway_path: ")
	sb.WriteString(escapeYAMLScalar(skill.Path))
	sb.WriteString("\n---\n")
	sb.WriteString(defaultSkillBody(skill))
	sb.WriteByte('\n')
	return os.WriteFile(filepath.Join(root, "SKILL.md"), []byte(sb.String()), 0o600)
}

func writeSupportFile(root string, file dto.SkillFile) error {
	rel := filepath.Clean(filepath.FromSlash(strings.TrimSpace(file.Path)))
	if rel == "" || rel == "." || filepath.IsAbs(rel) || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		return fmt.Errorf("invalid skill support file path %q", file.Path)
	}
	if strings.EqualFold(rel, "SKILL.md") {
		return nil
	}
	top := strings.Split(filepath.ToSlash(rel), "/")[0]
	switch top {
	case "assets", "references", "scripts":
	default:
		return fmt.Errorf("unsupported skill support file path %q", file.Path)
	}
	target := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(target, []byte(file.Content), 0o600)
}

func (s *SkillService) removeSkillFiles(skill model.SkillProfile) error {
	if strings.TrimSpace(s.baseDir) == "" {
		return nil
	}
	return errorsJoin(
		os.RemoveAll(filepath.Join(s.baseDir, skill.Name)),
	)
}

func (s *SkillService) root() string {
	if strings.TrimSpace(s.baseDir) == "" {
		return ""
	}
	return filepath.Clean(s.baseDir)
}

func (s *SkillService) skillVersionPath(skill model.SkillProfile) string {
	return filepath.Clean(filepath.Join(s.baseDir, sanitizePathComponent(skill.Name), normalizeSkillVersion(skill.Version)))
}

func (s *SkillService) syncFilesystemSkills(ctx context.Context) error {
	if strings.TrimSpace(s.baseDir) == "" || s.baseDir == "." {
		return nil
	}
	skills, err := s.scanFilesystemSkills()
	if err != nil {
		return err
	}
	for _, skill := range skills {
		existing, err := s.repo.GetByName(ctx, skill.Name)
		if err == nil {
			if shouldUpdateSkillFromFS(*existing, skill) {
				skill.ID = existing.ID
				skill.Path = firstNonEmpty(existing.Path, skill.Path)
				skill.Source = firstNonEmpty(existing.Source, skill.Source)
				skill.CreatedAt = existing.CreatedAt
				if err := s.repo.Update(ctx, skill); err != nil {
					return err
				}
			}
			continue
		}
		if !errors.Is(err, repository.ErrNotFound) {
			return err
		}
		if err := s.repo.Create(ctx, skill); err != nil {
			created, lookupErr := s.repo.GetByName(ctx, skill.Name)
			if lookupErr == nil && created != nil {
				continue
			}
			return err
		}
	}
	return nil
}

func (s *SkillService) scanFilesystemSkills() ([]model.SkillProfile, error) {
	info, err := os.Stat(s.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("gateway skills base dir %s is not a directory", s.baseDir)
	}

	latest := map[string]model.SkillProfile{}
	if err := filepath.WalkDir(s.baseDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(s.baseDir, path)
		if err != nil {
			return err
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if entry.IsDir() {
			return nil
		}
		if entry.Name() != "SKILL.md" || len(parts) != 3 {
			return nil
		}
		skill, err := parseFilesystemSkill(path, parts[0], parts[1])
		if err != nil {
			return err
		}
		if !isValidSkillName(skill.Name) {
			return nil
		}
		if current, ok := latest[skill.Name]; !ok || skill.Version > current.Version {
			latest[skill.Name] = skill
		}
		return nil
	}); err != nil {
		return nil, err
	}

	out := make([]model.SkillProfile, 0, len(latest))
	for _, skill := range latest {
		out = append(out, skill)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

type skillFrontMatter struct {
	Name         string            `yaml:"name"`
	Description  string            `yaml:"description"`
	AllowedTools []string          `yaml:"allowed-tools"`
	Metadata     map[string]string `yaml:"metadata"`
}

func parseFilesystemSkill(path, dirName, dirVersion string) (model.SkillProfile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return model.SkillProfile{}, err
	}
	meta, body, err := parseSkillMarkdown(string(data))
	if err != nil {
		return model.SkillProfile{}, fmt.Errorf("parse skill %s: %w", path, err)
	}
	name := strings.TrimSpace(meta.Name)
	if name == "" {
		name = strings.TrimSpace(dirName)
	}
	if dirName != "" && name != strings.TrimSpace(dirName) {
		return model.SkillProfile{}, fmt.Errorf("skill name %q does not match directory %q in %s", name, dirName, path)
	}
	version := strings.TrimSpace(meta.Metadata["version"])
	if version == "" {
		version = strings.TrimSpace(dirVersion)
	}
	gatewayPath := strings.TrimSpace(meta.Metadata["gateway_path"])
	if gatewayPath == "" {
		gatewayPath = name
	}
	now := time.Now().UTC()
	return model.SkillProfile{
		ID:               "skill_" + id.Random(),
		Name:             name,
		Description:      strings.TrimSpace(meta.Description),
		Path:             gatewayPath,
		Content:          strings.TrimSpace(body),
		Version:          normalizeSkillVersion(version),
		Status:           "active",
		Source:           "filesystem",
		AllowedToolsJSON: mustStringSliceJSON(meta.AllowedTools),
		MetadataJSON:     mustAnyJSON(meta.Metadata),
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}

func parseSkillMarkdown(text string) (skillFrontMatter, string, error) {
	text = strings.TrimPrefix(text, "\uFEFF")
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return skillFrontMatter{}, "", errors.New("missing YAML frontmatter")
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end == -1 {
		return skillFrontMatter{}, "", errors.New("missing closing frontmatter separator")
	}
	var meta skillFrontMatter
	if err := yaml.Unmarshal([]byte(strings.Join(lines[1:end], "\n")), &meta); err != nil {
		return skillFrontMatter{}, "", err
	}
	return meta, strings.TrimPrefix(strings.Join(lines[end+1:], "\n"), "\n"), nil
}

func shouldUpdateSkillFromFS(existing, file model.SkillProfile) bool {
	return existing.Description != file.Description ||
		existing.Content != file.Content ||
		existing.Version != file.Version ||
		existing.Status != "active" ||
		existing.AllowedToolsJSON != file.AllowedToolsJSON ||
		existing.MetadataJSON != file.MetadataJSON
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func isValidSkillName(name string) bool {
	return skillNameRegexp.MatchString(strings.TrimSpace(name))
}

func invalidSkillNameError(name string) error {
	return fmt.Errorf("invalid skill name %q (must be 1-64 chars, lowercase alphanumeric + hyphens)", name)
}

func sanitizePathComponent(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "default"
	}
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "default"
	}
	return out
}

func normalizeSkillVersion(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
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

func defaultSkillBody(skill model.SkillProfile) string {
	if strings.TrimSpace(skill.Content) != "" {
		return strings.TrimSpace(skill.Content)
	}
	if strings.TrimSpace(skill.Description) == "" {
		return "No description provided."
	}
	return skill.Description
}

func toSkillDTO(skill model.SkillProfile) *dto.SkillProfile {
	return &dto.SkillProfile{
		ID:           skill.ID,
		Name:         skill.Name,
		Description:  skill.Description,
		Path:         skill.Path,
		Content:      skill.Content,
		Version:      skill.Version,
		Status:       skill.Status,
		Source:       skill.Source,
		AllowedTools: parseStringSliceRaw(skill.AllowedToolsJSON),
		Metadata:     parseAnyJSON(skill.MetadataJSON),
		CreatedAt:    skill.CreatedAt,
		UpdatedAt:    skill.UpdatedAt,
	}
}

func mustStringSliceJSON(values []string) string {
	if len(values) == 0 {
		return "[]"
	}
	payload, _ := json.Marshal(values)
	return string(payload)
}

func parseStringSliceRaw(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" || raw == "null" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func mustAnyJSON(values any) string {
	if values == nil {
		return "{}"
	}
	payload, _ := json.Marshal(values)
	return string(payload)
}

func escapeYAMLScalar(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, `"`, `\"`)
	value = strings.ReplaceAll(value, "\n", "\\n")
	return `"` + value + `"`
}

func errorsJoin(errs ...error) error {
	var parts []string
	for _, err := range errs {
		if err != nil {
			parts = append(parts, err.Error())
		}
	}
	if len(parts) == 0 {
		return nil
	}
	return errors.New(strings.Join(parts, "; "))
}

func parseAnyJSON(raw string) map[string]any {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return map[string]any{"raw": raw}
	}
	return out
}

func normalizeSkillStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "deleted":
		return "deleted"
	case "active":
		return "active"
	default:
		return "active"
	}
}
