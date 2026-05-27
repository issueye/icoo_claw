package service

import (
	"context"
	"os"
	"path/filepath"
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

func TestSkillServiceCreatePublishesActiveSkillRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".skills")
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

	activePath := filepath.Join(root, "active", ".agents", "skills", "doc-writer", "SKILL.md")
	if _, err := os.Stat(activePath); err != nil {
		t.Fatalf("active SKILL.md missing at %s: %v", activePath, err)
	}
	versionPath := filepath.Join(root, "versions", "doc-writer", "v2", "SKILL.md")
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

func TestSkillServiceEnsureLayoutCreatesSkillsStructure(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".skills")
	svc := NewSkillService(root, &memorySkillRepo{})

	if err := svc.EnsureLayout(); err != nil {
		t.Fatalf("ensure layout: %v", err)
	}
	for _, path := range []string{
		root,
		filepath.Join(root, "active"),
		filepath.Join(root, "active", ".agents", "skills"),
		filepath.Join(root, "versions"),
		filepath.Join(root, "agents"),
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

func TestSkillServicePublishForAgentFiltersBoundSkills(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".skills")
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

	agentRoot, err := svc.PublishForAgent("agent_1", `["doc-writer"]`)
	if err != nil {
		t.Fatalf("publish for agent: %v", err)
	}
	if _, err := os.Stat(filepath.Join(agentRoot, ".agents", "skills", "doc-writer", "SKILL.md")); err != nil {
		t.Fatalf("bound skill missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(agentRoot, ".agents", "skills", "docker-helper", "SKILL.md")); !os.IsNotExist(err) {
		t.Fatalf("unbound skill should not be published, stat err = %v", err)
	}
}

func TestSkillServiceCreatePublishesSupportFiles(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".skills")
	svc := NewSkillService(root, &memorySkillRepo{})

	if _, err := svc.Create(context.Background(), dto.CreateSkillRequest{
		ID:          "skill_1",
		Name:        "doc-writer",
		Description: "Write documents",
		Path:        "doc-writer",
		Files: []dto.SkillFile{
			{Path: "references/guide.md", Content: "guide"},
			{Path: "scripts/run.ps1", Content: "Write-Output ok"},
			{Path: "assets/sample.txt", Content: "asset"},
		},
	}); err != nil {
		t.Fatalf("create skill: %v", err)
	}

	for _, rel := range []string{"references/guide.md", "scripts/run.ps1", "assets/sample.txt"} {
		path := filepath.Join(root, "active", ".agents", "skills", "doc-writer", filepath.FromSlash(rel))
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
	root := filepath.Join(t.TempDir(), ".skills")
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
