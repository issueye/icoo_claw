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

	"icoo_claw/server/gateway/internal/dto"
	"icoo_claw/server/gateway/internal/model"
	"icoo_claw/server/gateway/internal/repository"
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
	for _, dir := range []string{
		s.baseDir,
		s.ActiveSkillPath(),
		s.activeSkillsRoot(),
		filepath.Join(s.baseDir, "versions"),
		filepath.Join(s.baseDir, "agents"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func (s *SkillService) Create(ctx context.Context, req dto.CreateSkillRequest) (*dto.SkillProfile, error) {
	now := time.Now().UTC()
	skill := model.SkillProfile{
		ID:           strings.TrimSpace(req.ID),
		Name:         strings.TrimSpace(req.Name),
		Description:  strings.TrimSpace(req.Description),
		Path:         strings.TrimSpace(req.Path),
		Content:      strings.TrimSpace(req.Content),
		Version:      defaultString(req.Version, "v1"),
		Status:       "active",
		Source:       strings.TrimSpace(req.Source),
		MetadataJSON: mustAnyJSON(req.Metadata),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if skill.ID == "" {
		skill.ID = "skill_" + randomID()
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
		skill.Version = defaultString(*req.Version, skill.Version)
	}
	if req.Status != nil {
		skill.Status = normalizeSkillStatus(*req.Status)
	}
	if req.Source != nil {
		skill.Source = strings.TrimSpace(*req.Source)
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
	path := filepath.Join(s.activeSkillsRoot(), skill.Name, "SKILL.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	return data, skill.Name + "-SKILL.md", nil
}

func (s *SkillService) SyncSummary(ctx context.Context) (*dto.SkillSummary, error) {
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

func (s *SkillService) PublishForAgent(agentID, skillIDsJSON string) (string, error) {
	if strings.TrimSpace(s.baseDir) == "" {
		return "", nil
	}
	names := cleanStringSlice(parseStringSlice(skillIDsJSON))
	if len(names) == 0 {
		return s.ActiveSkillPath(), nil
	}
	for _, name := range names {
		if !isValidSkillName(name) {
			return "", invalidSkillNameError(name)
		}
	}

	root := s.agentSkillPath(agentID)
	if err := os.RemoveAll(root); err != nil {
		return "", err
	}
	for _, name := range names {
		source := filepath.Join(s.activeSkillsRoot(), name)
		target := filepath.Join(root, ".agents", "skills", name)
		if err := copyDir(source, target); err != nil {
			return "", fmt.Errorf("publish skill %s for agent %s: %w", name, agentID, err)
		}
	}
	return root, nil
}

func (s *SkillService) publishSkillFiles(skill model.SkillProfile, files []dto.SkillFile) error {
	if strings.TrimSpace(s.baseDir) == "" {
		return nil
	}
	if strings.TrimSpace(skill.Name) == "" || strings.TrimSpace(skill.Path) == "" {
		return fmt.Errorf("skill name and path are required")
	}
	versionRoot := filepath.Join(s.baseDir, "versions", skill.Name, skill.Version)
	if err := writeSkillPackage(versionRoot, skill, files); err != nil {
		return err
	}
	if skill.Status != "active" {
		return os.RemoveAll(filepath.Join(s.activeSkillsRoot(), skill.Name))
	}
	return writeSkillPackage(filepath.Join(s.activeSkillsRoot(), skill.Name), skill, files)
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
	content := fmt.Sprintf("---\nname: %s\ndescription: %s\nmetadata:\n  version: %s\n  gateway_path: %s\n---\n%s\n",
		escapeYAMLScalar(skill.Name), escapeYAMLScalar(skill.Description), escapeYAMLScalar(skill.Version), escapeYAMLScalar(skill.Path), defaultSkillBody(skill))
	return os.WriteFile(filepath.Join(root, "SKILL.md"), []byte(content), 0o600)
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
		os.RemoveAll(filepath.Join(s.baseDir, "versions", skill.Name)),
		os.RemoveAll(filepath.Join(s.activeSkillsRoot(), skill.Name)),
	)
}

func (s *SkillService) ActiveSkillPath() string {
	if strings.TrimSpace(s.baseDir) == "" {
		return ""
	}
	return filepath.Clean(filepath.Join(s.baseDir, "active"))
}

func (s *SkillService) activeSkillsRoot() string {
	return filepath.Join(s.ActiveSkillPath(), ".agents", "skills")
}

func (s *SkillService) agentSkillPath(agentID string) string {
	agentID = sanitizePathComponent(agentID)
	return filepath.Clean(filepath.Join(s.baseDir, "agents", agentID))
}

func copyFile(source, target string) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(target, data, 0o600)
}

func copyDir(source, target string) error {
	info, err := os.Stat(source)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", source)
	}
	if err := os.RemoveAll(target); err != nil {
		return err
	}
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(target, 0o755)
		}
		dest := filepath.Join(target, rel)
		if entry.IsDir() {
			return os.MkdirAll(dest, 0o755)
		}
		return copyFile(path, dest)
	})
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
		ID:          skill.ID,
		Name:        skill.Name,
		Description: skill.Description,
		Path:        skill.Path,
		Version:     skill.Version,
		Status:      skill.Status,
		Source:      skill.Source,
		Metadata:    parseAnyJSON(skill.MetadataJSON),
		CreatedAt:   skill.CreatedAt,
		UpdatedAt:   skill.UpdatedAt,
	}
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
