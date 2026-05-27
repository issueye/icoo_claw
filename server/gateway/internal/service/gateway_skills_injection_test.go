package service

import (
	"context"
	"testing"

	"icoo_claw/server/gateway/internal/dto"
	"icoo_claw/server/gateway/internal/model"
)

func TestChatServiceDoesNotInjectGatewaySkillsSummary(t *testing.T) {
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
	skills := NewSkillService(t.TempDir(), &memorySkillRepo{items: []model.SkillProfile{
		{ID: "skill_1", Name: "doc-writer", Description: "Write docs", Path: "docs/doc-writer", Version: "v1", Status: "active"},
	}})
	svc := NewChatService(conversations, chatAgentRepo{}, nil, router, &chatSessionBackend{}, claw, skills)

	if _, err := svc.SendMessage(context.Background(), "conv_1", dto.SendMessageRequest{Prompt: "hello"}); err != nil {
		t.Fatalf("send message: %v", err)
	}
	if _, ok := claw.req.Agent["gateway_skills"]; ok {
		t.Fatalf("gateway_skills should not be injected per request: %+v", claw.req.Agent)
	}
	if _, ok := claw.req.Agent["project_root"]; ok {
		t.Fatalf("project_root should be omitted unless supplied by request metadata: %+v", claw.req.Agent)
	}
}
