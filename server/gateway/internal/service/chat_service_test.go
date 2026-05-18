package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"icoo_claw/server/gateway/internal/client"
	"icoo_claw/server/gateway/internal/config"
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
		NetworkAllowJSON:  `["example.com"]`,
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
func (r *chatInstanceRepo) AdjustInflight(_ context.Context, id string, delta int) error {
	if r.instance.ID != id {
		return repository.ErrNotFound
	}
	r.instance.Inflight += delta
	if r.instance.Inflight < 0 {
		r.instance.Inflight = 0
	}
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
	router := NewDefaultRouterPolicy(conversations, instances, nil)
	svc := NewChatService(conversations, chatAgentRepo{}, router, sessionStore, claw)

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
	if got := claw.req.Agent["network_allow"].([]string); len(got) != 1 || got[0] != "example.com" {
		t.Fatalf("network_allow = %+v", claw.req.Agent["network_allow"])
	}
	if instances.instance.Inflight != 0 {
		t.Fatalf("inflight = %d, want 0", instances.instance.Inflight)
	}
}

type chatMultiInstanceRepo struct {
	instances []model.AgentInstance
}

func (r *chatMultiInstanceRepo) Create(context.Context, model.AgentInstance) error { return nil }
func (r *chatMultiInstanceRepo) Get(_ context.Context, id string) (*model.AgentInstance, error) {
	for i := range r.instances {
		if r.instances[i].ID == id {
			copy := r.instances[i]
			return &copy, nil
		}
	}
	return nil, repository.ErrNotFound
}
func (r *chatMultiInstanceRepo) List(context.Context) ([]model.AgentInstance, error) {
	return append([]model.AgentInstance(nil), r.instances...), nil
}
func (r *chatMultiInstanceRepo) Update(_ context.Context, instance model.AgentInstance) error {
	for i := range r.instances {
		if r.instances[i].ID == instance.ID {
			r.instances[i] = instance
			return nil
		}
	}
	return repository.ErrNotFound
}
func (r *chatMultiInstanceRepo) AdjustInflight(_ context.Context, id string, delta int) error {
	for i := range r.instances {
		if r.instances[i].ID == id {
			r.instances[i].Inflight += delta
			if r.instances[i].Inflight < 0 {
				r.instances[i].Inflight = 0
			}
			return nil
		}
	}
	return repository.ErrNotFound
}

func TestChatServiceUsesStickyInstance(t *testing.T) {
	conversations := &chatConversationRepo{conversation: &model.Conversation{
		ID:               "conv_1",
		SessionID:        "sess_1",
		AgentID:          "agent_1",
		Status:           "active",
		StickyInstanceID: "inst_2",
	}}
	instances := &chatMultiInstanceRepo{instances: []model.AgentInstance{
		{ID: "inst_1", AgentID: "agent_1", Status: "ready", BaseURL: "http://127.0.0.1:8101", Inflight: 0},
		{ID: "inst_2", AgentID: "agent_1", Status: "ready", BaseURL: "http://127.0.0.1:8102", Inflight: 9},
	}}
	claw := &chatClaw{}
	router := NewDefaultRouterPolicy(conversations, instances, nil)
	svc := NewChatService(conversations, chatAgentRepo{}, router, &chatSessionStore{}, claw)

	if _, err := svc.SendMessage(context.Background(), "conv_1", dto.SendMessageRequest{Prompt: "hello"}); err != nil {
		t.Fatalf("send message: %v", err)
	}
	if claw.baseURL != "http://127.0.0.1:8102" {
		t.Fatalf("baseURL = %q, want sticky instance", claw.baseURL)
	}
}

func TestChatServiceStartsInstanceWhenNoneReady(t *testing.T) {
	conversations := &chatConversationRepo{conversation: &model.Conversation{
		ID:        "conv_1",
		SessionID: "sess_1",
		AgentID:   "agent_1",
		Status:    "active",
	}}
	instances := &memoryInstanceRepo{}
	starter := NewAgentInstanceService(
		config.Config{ClawPortStart: 8101, ClawPortEnd: 8102, MaxAgentInstances: 1},
		instanceAgentRepo{},
		instances,
		memorySupervisor{},
	)
	router := NewDefaultRouterPolicy(conversations, instances, starter)
	claw := &chatClaw{}
	svc := NewChatService(conversations, chatAgentRepo{}, router, &chatSessionStore{}, claw)

	if _, err := svc.SendMessage(context.Background(), "conv_1", dto.SendMessageRequest{Prompt: "hello"}); err != nil {
		t.Fatalf("send message: %v", err)
	}
	if len(instances.instances) != 1 {
		t.Fatalf("instances = %d, want 1", len(instances.instances))
	}
	if claw.baseURL != "http://127.0.0.1:8101" {
		t.Fatalf("baseURL = %q, want auto-started instance", claw.baseURL)
	}
	if conversations.conversation.StickyInstanceID != instances.instances[0].ID {
		t.Fatalf("sticky = %q, want %q", conversations.conversation.StickyInstanceID, instances.instances[0].ID)
	}
	if instances.instances[0].Inflight != 0 {
		t.Fatalf("inflight = %d, want 0", instances.instances[0].Inflight)
	}
}

func TestChatServiceRefreshesAndSkipsFailedStickyInstance(t *testing.T) {
	conversations := &chatConversationRepo{conversation: &model.Conversation{
		ID:               "conv_1",
		SessionID:        "sess_1",
		AgentID:          "agent_1",
		Status:           "active",
		StickyInstanceID: "inst_1",
	}}
	instances := &memoryInstanceRepo{instances: []model.AgentInstance{
		{ID: "inst_1", AgentID: "agent_1", Status: "ready", BaseURL: "http://127.0.0.1:8101", Port: 8101},
	}}
	starter := NewAgentInstanceService(
		config.Config{ClawPortStart: 8101, ClawPortEnd: 8102, MaxAgentInstances: 2},
		instanceAgentRepo{},
		instances,
		&failingOnceSupervisor{failed: map[string]bool{}},
	)
	router := NewDefaultRouterPolicy(conversations, instances, starter)
	claw := &chatClaw{}
	svc := NewChatService(conversations, chatAgentRepo{}, router, &chatSessionStore{}, claw)

	if _, err := svc.SendMessage(context.Background(), "conv_1", dto.SendMessageRequest{Prompt: "hello"}); err != nil {
		t.Fatalf("send message: %v", err)
	}
	if claw.baseURL != "http://127.0.0.1:8102" {
		t.Fatalf("baseURL = %q, want fallback instance", claw.baseURL)
	}
	if conversations.conversation.StickyInstanceID == "inst_1" {
		t.Fatalf("sticky instance was not rebound: %+v", conversations.conversation)
	}
}

type failingOnceSupervisor struct {
	failed map[string]bool
}

func (s *failingOnceSupervisor) Start(context.Context, StartAgentInstanceSpec) (*AgentProcess, error) {
	return &AgentProcess{PID: 42}, nil
}
func (s *failingOnceSupervisor) Stop(context.Context, model.AgentInstance) error { return nil }
func (s *failingOnceSupervisor) Probe(_ context.Context, instance model.AgentInstance) error {
	if instance.ID == "inst_1" && !s.failed[instance.ID] {
		s.failed[instance.ID] = true
		return errors.New("health unavailable")
	}
	return nil
}
