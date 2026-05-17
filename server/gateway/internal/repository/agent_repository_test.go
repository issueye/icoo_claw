package repository

import (
	"context"
	"testing"

	"icoo_claw/server/gateway/internal/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestGormAgentRepositoryCRUD(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.AgentProfile{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := NewGormAgentRepository(db)
	ctx := context.Background()
	agent := model.AgentProfile{
		ID:                "agent_1",
		Name:              "Default",
		ModelProvider:     "openai",
		ToolWhitelistJSON: "[]",
		MCPServerIDsJSON:  "[]",
		SkillIDsJSON:      "[]",
		Enabled:           true,
	}
	if err := repo.Create(ctx, agent); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.Get(ctx, "agent_1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "Default" {
		t.Fatalf("name = %q", got.Name)
	}

	got.Name = "Updated"
	if err := repo.Update(ctx, *got); err != nil {
		t.Fatalf("update: %v", err)
	}
	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].Name != "Updated" {
		t.Fatalf("list = %+v", list)
	}

	if err := repo.Delete(ctx, "agent_1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := repo.Get(ctx, "agent_1"); err != ErrNotFound {
		t.Fatalf("get deleted error = %v", err)
	}
}
