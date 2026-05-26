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
	root := t.TempDir()
	repo := &memorySkillRepo{}
	svc := NewSkillService(SkillGatewayConfig{BaseDir: root}, repo)

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
	if summary.Path != filepath.Join(root, "active") {
		t.Fatalf("summary path = %q", summary.Path)
	}
	if len(summary.Skills) != 1 || summary.Skills[0].Name != "doc-writer" {
		t.Fatalf("summary skills = %+v", summary.Skills)
	}
}
