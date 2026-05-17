package service

import (
	"context"
	"testing"
	"time"

	"icoo_claw/server/gateway/internal/client"
	"icoo_claw/server/gateway/internal/dto"
	"icoo_claw/server/gateway/internal/model"
	"icoo_claw/server/gateway/internal/repository"
)

type chatAgentRepo struct{}

func (r chatAgentRepo) Create(context.Context, model.AgentProfile) error { return nil }
func (r chatAgentRepo) Get(context.Context, string) (*model.AgentProfile, error) {
	return &model.AgentProfile{
		ID:                "agent_1",
		Name:              "Default",
		ModelProvider:     "openai",
		ToolWhitelistJSON: `["read"]`,
		MCPServerIDsJSON:  `[]`,
		SkillIDsJSON:      `[]`,
		Enabled:           true,
	}, nil
}
func (r chatAgentRepo) List(context.Context) ([]model.AgentProfile, error) { return nil, nil }
func (r chatAgentRepo) Update(context.Context, model.AgentProfile) error   { return nil }
func (r chatAgentRepo) Delete(context.Context, string) error               { return nil }

type chatConversationRepo struct {
	conversation *model.Conversation
}

func (r *chatConversationRepo) Create(_ context.Context, conversation model.Conversation) error {
	r.conversation = &conversation
	return nil
}
func (r *chatConversationRepo) Get(context.Context, string) (*model.Conversation, error) {
	if r.conversation == nil {
		return nil, repository.ErrNotFound
	}
	copy := *r.conversation
	return &copy, nil
}
func (r *chatConversationRepo) List(context.Context) ([]model.Conversation, error) {
	if r.conversation == nil {
		return nil, nil
	}
	return []model.Conversation{*r.conversation}, nil
}
func (r *chatConversationRepo) Update(_ context.Context, conversation model.Conversation) error {
	r.conversation = &conversation
	return nil
}
func (r *chatConversationRepo) Delete(context.Context, string) error { return nil }

type chatInstanceRepo struct {
	instance model.AgentInstance
}

func (r *chatInstanceRepo) Create(context.Context, model.AgentInstance) error { return nil }
func (r *chatInstanceRepo) Get(context.Context, string) (*model.AgentInstance, error) {
	copy := r.instance
	return &copy, nil
}
func (r *chatInstanceRepo) List(context.Context) ([]model.AgentInstance, error) {
	return []model.AgentInstance{r.instance}, nil
}
func (r *chatInstanceRepo) Update(_ context.Context, instance model.AgentInstance) error {
	r.instance = instance
	return nil
}

type chatSessionStore struct {
	created bool
}

func (s *chatSessionStore) CreateSession(context.Context, client.CreateSessionRequest) error {
	s.created = true
	return nil
}
func (s *chatSessionStore) ListMessages(context.Context, string) ([]dto.SessionMessage, error) {
	return []dto.SessionMessage{{Role: "user", Content: "hello", CreatedAt: time.Now()}}, nil
}

type chatClaw struct {
	baseURL string
	req     client.RunRequest
}

func (c *chatClaw) Run(_ context.Context, baseURL string, req client.RunRequest) (*client.RunResponse, error) {
	c.baseURL = baseURL
	c.req = req
	return &client.RunResponse{SessionID: req.SessionID, RequestID: req.RequestID, Output: "ok", StopReason: "end_turn"}, nil
}
func (c *chatClaw) Stream(context.Context, string, client.RunRequest) (<-chan client.StreamEvent, error) {
	out := make(chan client.StreamEvent)
	close(out)
	return out, nil
}

func TestChatServiceCreateAndSendMessage(t *testing.T) {
	conversations := &chatConversationRepo{}
	instances := &chatInstanceRepo{instance: model.AgentInstance{
		ID:      "inst_1",
		AgentID: "agent_1",
		Status:  "ready",
		BaseURL: "http://127.0.0.1:8101",
	}}
	sessionStore := &chatSessionStore{}
	claw := &chatClaw{}
	svc := NewChatService(conversations, chatAgentRepo{}, instances, sessionStore, claw)

	conv, err := svc.CreateConversation(context.Background(), dto.CreateConversationRequest{AgentID: "agent_1", Title: "Test"})
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	if !sessionStore.created {
		t.Fatal("expected session store create")
	}

	resp, err := svc.SendMessage(context.Background(), conv.ID, dto.SendMessageRequest{Prompt: "hello", RequestID: "req_1"})
	if err != nil {
		t.Fatalf("send message: %v", err)
	}
	if resp.Output != "ok" || claw.baseURL != "http://127.0.0.1:8101" {
		t.Fatalf("resp=%+v baseURL=%q", resp, claw.baseURL)
	}
	if claw.req.SessionID != conv.SessionID || claw.req.ToolWhitelist[0] != "read" {
		t.Fatalf("claw request = %+v", claw.req)
	}
	if instances.instance.Inflight != 0 {
		t.Fatalf("inflight = %d, want 0", instances.instance.Inflight)
	}
}
