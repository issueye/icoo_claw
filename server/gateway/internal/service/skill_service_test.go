package service

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
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

func TestSkillServiceEnsureLayoutCreatesSkillsStructure(t *testing.T) {
	root := filepath.Join(t.TempDir(), "icoo_runtime", "skills")
	svc := NewSkillService(root, &memorySkillRepo{})

	if err := svc.EnsureLayout(); err != nil {
		t.Fatalf("ensure layout: %v", err)
	}
	for _, path := range []string{
		root,
		filepath.Join(root, "instances"),
	} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat %s: %v", path, err)
		}
		if !info.IsDir() {
			t.Fatalf("%s is not a directory", path)
		}
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

func TestSkillServicePublishForInstanceFiltersBoundSkills(t *testing.T) {
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

	instanceRoot, err := svc.PublishForInstance("inst_1", `["doc-writer"]`)
	if err != nil {
		t.Fatalf("publish for instance: %v", err)
	}
	docSkill := repo.items[0]
	if _, err := os.Stat(filepath.Join(instanceRoot, "doc-writer", docSkill.Version, "SKILL.md")); err != nil {
		t.Fatalf("bound skill missing: %v", err)
	}
	dockerSkill := repo.items[1]
	if _, err := os.Stat(filepath.Join(instanceRoot, "docker-helper", dockerSkill.Version, "SKILL.md")); !os.IsNotExist(err) {
		t.Fatalf("unbound skill should not be published, stat err = %v", err)
	}
}

func TestSkillServicePublishForInstanceUsesGlobalRootWhenUnbound(t *testing.T) {
	root := filepath.Join(t.TempDir(), "icoo_runtime", "skills")
	svc := NewSkillService(root, &memorySkillRepo{})

	instanceRoot, err := svc.PublishForInstance("inst_1", "")
	if err != nil {
		t.Fatalf("publish for instance: %v", err)
	}
	if instanceRoot != root {
		t.Fatalf("instance root = %q, want global root %q", instanceRoot, root)
	}
	if _, err := os.Stat(filepath.Join(root, "instances", "inst_1")); !os.IsNotExist(err) {
		t.Fatalf("unbound instance dir should not be created, stat err = %v", err)
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
