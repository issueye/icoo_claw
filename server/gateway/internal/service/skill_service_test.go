package service

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"icoo_claw/server/gateway/internal/dto"
	"icoo_claw/server/gateway/internal/model"
	"icoo_claw/server/gateway/internal/repository"
)

type memorySkillRepo struct {
	items []model.SkillProfile
}

func (r *memorySkillRepo) Create(_ context.Context, skill model.SkillProfile) error {
	r.items = append(r.items, skill)
	return nil
}

func (r *memorySkillRepo) Get(_ context.Context, id string) (*model.SkillProfile, error) {
	for i := range r.items {
		if r.items[i].ID == id {
			copy := r.items[i]
			return &copy, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (r *memorySkillRepo) GetByName(_ context.Context, name string) (*model.SkillProfile, error) {
	for i := range r.items {
		if r.items[i].Name == name {
			copy := r.items[i]
			return &copy, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (r *memorySkillRepo) List(context.Context) ([]model.SkillProfile, error) {
	return append([]model.SkillProfile(nil), r.items...), nil
}

func (r *memorySkillRepo) ListActive(context.Context) ([]model.SkillProfile, error) {
	var out []model.SkillProfile
	for _, item := range r.items {
		if item.Status == "active" {
			out = append(out, item)
		}
	}
	return out, nil
}

func (r *memorySkillRepo) Update(_ context.Context, skill model.SkillProfile) error {
	for i := range r.items {
		if r.items[i].ID == skill.ID {
			r.items[i] = skill
			return nil
		}
	}
	return repository.ErrNotFound
}

func (r *memorySkillRepo) Delete(_ context.Context, id string) error {
	for i := range r.items {
		if r.items[i].ID == id {
			r.items = append(r.items[:i], r.items[i+1:]...)
			return nil
		}
	}
	return repository.ErrNotFound
}

func TestSkillServiceCreatePublishesVersionedSkillRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "icoo_runtime", "skills")
	repo := &memorySkillRepo{}
	svc := NewSkillService(root, repo)

	skill, err := svc.Create(context.Background(), dto.CreateSkillRequest{
		ID:          "skill_1",
		Name:        "doc-writer",
		Description: "Write documents",
		Path:        "docs/doc-writer",
		Version:     "v2",
	})
	if err != nil {
		t.Fatalf("create skill: %v", err)
	}

	versionPath := filepath.Join(root, "doc-writer", "v2", "SKILL.md")
	if _, err := os.Stat(versionPath); err != nil {
		t.Fatalf("version SKILL.md missing at %s: %v", versionPath, err)
	}
	if skill.Path != "docs/doc-writer" || skill.Version != "v2" {
		t.Fatalf("skill dto = %+v", skill)
	}

	summary, err := svc.SyncSummary(context.Background())
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	if len(summary.Skills) != 1 || summary.Skills[0].Name != "doc-writer" {
		t.Fatalf("summary skills = %+v", summary.Skills)
	}
}

func TestSkillServiceListImportsFilesystemSkillCreatedByRuntime(t *testing.T) {
	root := filepath.Join(t.TempDir(), "icoo_runtime", "skills")
	repo := &memorySkillRepo{}
	svc := NewSkillService(root, repo)
	writeRuntimeSkill(t, root, "weather", "v1", "Query weather", "Use weather data to answer.")

	list, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("list skills: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("skills = %+v, want one imported skill", list)
	}
	if list[0].Name != "weather" || list[0].Version != "v1" || list[0].Source != "filesystem" {
		t.Fatalf("imported skill = %+v", list[0])
	}
	if list[0].Content != "Use weather data to answer." {
		t.Fatalf("content = %q", list[0].Content)
	}

	stored, err := repo.GetByName(context.Background(), "weather")
	if err != nil {
		t.Fatalf("stored skill missing: %v", err)
	}
	if stored.ID == "" || stored.Path != "weather" {
		t.Fatalf("stored skill = %+v", stored)
	}
}

func TestSkillServiceSyncSummaryImportsFilesystemSkill(t *testing.T) {
	root := filepath.Join(t.TempDir(), "icoo_runtime", "skills")
	svc := NewSkillService(root, &memorySkillRepo{})
	writeRuntimeSkill(t, root, "weather", "v1", "Query weather", "Use weather data to answer.")

	summary, err := svc.SyncSummary(context.Background())
	if err != nil {
		t.Fatalf("sync summary: %v", err)
	}
	if len(summary.Skills) != 1 || summary.Skills[0].Name != "weather" {
		t.Fatalf("summary skills = %+v", summary.Skills)
	}
}

func TestSkillServiceRuntimeRootImportsAndValidatesFilesystemSkill(t *testing.T) {
	root := filepath.Join(t.TempDir(), "icoo_runtime", "skills")
	svc := NewSkillService(root, &memorySkillRepo{})
	writeRuntimeSkill(t, root, "weather", "v1", "Query weather", "Use weather data to answer.")

	runtimeRoot, err := svc.RuntimeRoot(context.Background(), `["weather"]`)
	if err != nil {
		t.Fatalf("runtime root: %v", err)
	}
	if runtimeRoot != root {
		t.Fatalf("runtime root = %q, want %q", runtimeRoot, root)
	}
}

func TestSkillServiceCreateGeneratesTimestampVersionWhenMissing(t *testing.T) {
	root := filepath.Join(t.TempDir(), "icoo_runtime", "skills")
	svc := NewSkillService(root, &memorySkillRepo{})

	skill, err := svc.Create(context.Background(), dto.CreateSkillRequest{
		ID:          "skill_1",
		Name:        "weather",
		Description: "Query weather",
		Path:        "tools/weather",
	})
	if err != nil {
		t.Fatalf("create skill: %v", err)
	}
	if !regexp.MustCompile(`^\d{14}$`).MatchString(skill.Version) {
		t.Fatalf("version = %q, want timestamp", skill.Version)
	}
	if _, err := os.Stat(filepath.Join(root, "weather", skill.Version, "SKILL.md")); err != nil {
		t.Fatalf("timestamp version SKILL.md missing: %v", err)
	}
}

func TestSkillServiceEnsureLayoutCreatesCanonicalSkillRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "icoo_runtime", "skills")
	svc := NewSkillService(root, &memorySkillRepo{})

	if err := svc.EnsureLayout(); err != nil {
		t.Fatalf("ensure layout: %v", err)
	}
	info, err := os.Stat(root)
	if err != nil {
		t.Fatalf("stat %s: %v", root, err)
	}
	if !info.IsDir() {
		t.Fatalf("%s is not a directory", root)
	}
	if _, err := os.Stat(filepath.Join(root, "instances")); !os.IsNotExist(err) {
		t.Fatalf("instances directory should not be created, stat err = %v", err)
	}
}

func TestSkillServiceRejectsInvalidSkillName(t *testing.T) {
	svc := NewSkillService(t.TempDir(), &memorySkillRepo{})

	_, err := svc.Create(context.Background(), dto.CreateSkillRequest{
		ID:          "skill_1",
		Name:        "Doc Writer",
		Description: "Write documents",
		Path:        "docs/doc-writer",
	})
	if err == nil {
		t.Fatal("expected invalid skill name error")
	}
}

func TestSkillServiceRuntimeRootValidatesBoundSkillsWithoutCopying(t *testing.T) {
	root := filepath.Join(t.TempDir(), "icoo_runtime", "skills")
	repo := &memorySkillRepo{}
	svc := NewSkillService(root, repo)

	for _, req := range []dto.CreateSkillRequest{
		{ID: "skill_1", Name: "doc-writer", Description: "Write documents", Path: "docs/doc-writer"},
		{ID: "skill_2", Name: "docker-helper", Description: "Docker help", Path: "tools/docker"},
	} {
		if _, err := svc.Create(context.Background(), req); err != nil {
			t.Fatalf("create skill %s: %v", req.Name, err)
		}
	}

	runtimeRoot, err := svc.RuntimeRoot(context.Background(), `["doc-writer"]`)
	if err != nil {
		t.Fatalf("runtime root: %v", err)
	}
	if runtimeRoot != root {
		t.Fatalf("runtime root = %q, want %q", runtimeRoot, root)
	}
}

func TestSkillServiceRuntimeRootUsesCanonicalRootWhenUnbound(t *testing.T) {
	root := filepath.Join(t.TempDir(), "icoo_runtime", "skills")
	svc := NewSkillService(root, &memorySkillRepo{})

	runtimeRoot, err := svc.RuntimeRoot(context.Background(), "")
	if err != nil {
		t.Fatalf("runtime root: %v", err)
	}
	if runtimeRoot != root {
		t.Fatalf("runtime root = %q, want %q", runtimeRoot, root)
	}
}

func TestSkillServiceCreatePublishesSupportFiles(t *testing.T) {
	root := filepath.Join(t.TempDir(), "icoo_runtime", "skills")
	svc := NewSkillService(root, &memorySkillRepo{})

	if _, err := svc.Create(context.Background(), dto.CreateSkillRequest{
		ID:          "skill_1",
		Name:        "doc-writer",
		Description: "Write documents",
		Path:        "doc-writer",
		Version:     "20260530210600",
		Files: []dto.SkillFile{
			{Path: "references/guide.md", Content: "guide"},
			{Path: "scripts/run.ps1", Content: "Write-Output ok"},
			{Path: "assets/sample.txt", Content: "asset"},
		},
	}); err != nil {
		t.Fatalf("create skill: %v", err)
	}

	for _, rel := range []string{"references/guide.md", "scripts/run.ps1", "assets/sample.txt"} {
		path := filepath.Join(root, "doc-writer", "20260530210600", filepath.FromSlash(rel))
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read support file %s: %v", rel, err)
		}
		if len(data) == 0 {
			t.Fatalf("support file %s was empty", rel)
		}
	}
}

func TestSkillServiceRejectsEscapingSupportFilePath(t *testing.T) {
	root := filepath.Join(t.TempDir(), "icoo_runtime", "skills")
	svc := NewSkillService(root, &memorySkillRepo{})

	_, err := svc.Create(context.Background(), dto.CreateSkillRequest{
		ID:          "skill_1",
		Name:        "doc-writer",
		Description: "Write documents",
		Path:        "doc-writer",
		Files:       []dto.SkillFile{{Path: "../outside.txt", Content: "nope"}},
	})
	if err == nil {
		t.Fatal("expected invalid support file path error")
	}
}

func writeRuntimeSkill(t *testing.T, root, name, version, description, body string) {
	t.Helper()
	dir := filepath.Join(root, name, version)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	content := strings.Join([]string{
		"---",
		`name: "` + name + `"`,
		`description: "` + description + `"`,
		"metadata:",
		`  version: "` + version + `"`,
		"---",
		body,
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o600); err != nil {
		t.Fatalf("write skill: %v", err)
	}
}
