package service

import (
	"context"
	"testing"

	"icoo_claw/server/gateway/internal/dto"
	"icoo_claw/server/gateway/internal/model"
)

func TestChatServiceInjectsGatewaySkills(t *testing.T) {
	conversations := &chatConversationRepo{conversation: &model.Conversation{
		ID:        "conv_1",
		SessionID: "sess_1",
		AgentID:   "agent_1",
		Status:    "active",
	}}
	instances := &chatInstanceRepo{instance: model.AgentInstance{
		ID:      "inst_1",
		AgentID: "agent_1",
		Status:  "ready",
		BaseURL: "http://127.0.0.1:8101",
	}}
	claw := &chatClaw{}
	router := NewDefaultRouterPolicy(conversations, instances, nil)
	skills := NewSkillService(SkillGatewayConfig{BaseDir: t.TempDir()}, &memorySkillRepo{items: []model.SkillProfile{
		{ID: "skill_1", Name: "doc-writer", Description: "Write docs", Path: "docs/doc-writer", Version: "v1", Status: "active"},
	}})
	svc := NewChatService(conversations, chatAgentRepo{}, nil, router, &chatSessionBackend{}, claw, skills)

	if _, err := svc.SendMessage(context.Background(), "conv_1", dto.SendMessageRequest{Prompt: "hello"}); err != nil {
		t.Fatalf("send message: %v", err)
	}
	raw, ok := claw.req.Agent["gateway_skills"].(*dto.SkillSummary)
	if !ok {
		t.Fatalf("gateway_skills = %#v, want *dto.SkillSummary", claw.req.Agent["gateway_skills"])
	}
	if raw.Path == "" || len(raw.Skills) != 1 || raw.Skills[0].Name != "doc-writer" {
		t.Fatalf("gateway skills summary = %+v", raw)
	}
	if claw.req.Agent["project_root"] != raw.Path {
		t.Fatalf("project_root = %#v, want gateway skills path %q", claw.req.Agent["project_root"], raw.Path)
	}
}
