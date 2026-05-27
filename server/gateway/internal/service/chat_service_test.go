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

type chatAgentRepoWithAgent struct {
	agent model.AgentProfile
}

func (r chatAgentRepoWithAgent) Create(context.Context, model.AgentProfile) error { return nil }
func (r chatAgentRepoWithAgent) Get(context.Context, string) (*model.AgentProfile, error) {
	agent := r.agent
	return &agent, nil
}
func (r chatAgentRepoWithAgent) List(context.Context) ([]model.AgentProfile, error) { return nil, nil }
func (r chatAgentRepoWithAgent) Update(context.Context, model.AgentProfile) error   { return nil }
func (r chatAgentRepoWithAgent) Delete(context.Context, string) error               { return nil }

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
func (r *chatInstanceRepo) Delete(context.Context, string) error { return nil }
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

type chatSessionBackend struct {
	created bool
}

func (s *chatSessionBackend) CreateSession(context.Context, SessionCreateRequest) error {
	s.created = true
	return nil
}
func (s *chatSessionBackend) ListMessages(context.Context, string) ([]dto.SessionMessage, error) {
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
func (c *chatClaw) Stream(_ context.Context, baseURL string, req client.RunRequest) (<-chan client.StreamEvent, error) {
	c.baseURL = baseURL
	c.req = req
	out := make(chan client.StreamEvent, 3)
	out <- client.StreamEvent{Type: "session/update", SessionID: req.SessionID, RequestID: req.RequestID, Update: &client.SessionUpdate{SessionUpdate: "agent_message_chunk", Content: &client.ContentBlock{Type: "text", Text: "o"}}}
	out <- client.StreamEvent{Type: "session/update", SessionID: req.SessionID, RequestID: req.RequestID, Update: &client.SessionUpdate{SessionUpdate: "agent_message_chunk", Content: &client.ContentBlock{Type: "text", Text: "k"}}}
	out <- client.StreamEvent{Type: "session/completed", SessionID: req.SessionID, RequestID: req.RequestID, StopReason: "end_turn"}
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
	sessionBackend := &chatSessionBackend{}
	claw := &chatClaw{}
	router := NewDefaultRouterPolicy(conversations, instances, nil)
	svc := NewChatService(conversations, chatAgentRepo{}, nil, router, sessionBackend, claw)

	conv, err := svc.CreateConversation(context.Background(), dto.CreateConversationRequest{AgentID: "agent_1", Title: "Test"})
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	if !sessionBackend.created {
		t.Fatal("expected session create")
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

func TestChatServicePassesForceSkills(t *testing.T) {
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
	svc := NewChatService(conversations, chatAgentRepo{}, nil, router, &chatSessionBackend{}, claw)

	if _, err := svc.SendMessage(context.Background(), "conv_1", dto.SendMessageRequest{Prompt: "hello", ForceSkills: []string{" doc-writer ", ""}}); err != nil {
		t.Fatalf("send message: %v", err)
	}
	if len(claw.req.ForceSkills) != 1 || claw.req.ForceSkills[0] != "doc-writer" {
		t.Fatalf("force skills = %+v", claw.req.ForceSkills)
	}
}

func TestCollectClawStreamErrorsWhenClosedBeforeCompletion(t *testing.T) {
	events := make(chan client.StreamEvent, 1)
	events <- client.StreamEvent{
		Type:      "session/update",
		SessionID: "sess_1",
		RequestID: "req_1",
		Update:    &client.SessionUpdate{SessionUpdate: "agent_message_chunk", Content: &client.ContentBlock{Type: "text", Text: "partial"}},
	}
	close(events)

	_, _, sessionID, requestID, err := collectClawStream(events, "sess_fallback", "req_fallback")
	if err == nil {
		t.Fatal("expected stream close error")
	}
	if sessionID != "sess_1" || requestID != "req_1" {
		t.Fatalf("ids = %q/%q, want stream ids", sessionID, requestID)
	}
	if err.Error() != "agent stream closed before completion" {
		t.Fatalf("error = %q", err.Error())
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
func (r *chatMultiInstanceRepo) Delete(context.Context, string) error { return nil }
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
	svc := NewChatService(conversations, chatAgentRepo{}, nil, router, &chatSessionBackend{}, claw)

	if _, err := svc.SendMessage(context.Background(), "conv_1", dto.SendMessageRequest{Prompt: "hello"}); err != nil {
		t.Fatalf("send message: %v", err)
	}
	if claw.baseURL != "http://127.0.0.1:8102" {
		t.Fatalf("baseURL = %q, want sticky instance", claw.baseURL)
	}
}

func TestChatServiceDefaultsToBuiltinToolsWhenWhitelistEmpty(t *testing.T) {
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
	agents := chatAgentRepoWithAgent{agent: model.AgentProfile{
		ID:                "agent_1",
		Name:              "Default",
		ModelProvider:     "openai",
		ToolWhitelistJSON: `[]`,
		NetworkAllowJSON:  `[]`,
		MCPServerIDsJSON:  `[]`,
		SkillIDsJSON:      `[]`,
		Enabled:           true,
	}}
	svc := NewChatService(conversations, agents, nil, router, &chatSessionBackend{}, claw)

	_, err := svc.SendMessage(context.Background(), "conv_1", dto.SendMessageRequest{
		Prompt: "hello",
		Metadata: map[string]any{
			"project_root": "E:/codes/icoo_claw",
		},
	})
	if err != nil {
		t.Fatalf("send message: %v", err)
	}
	if _, ok := claw.req.Agent["enabled_builtin_tools"]; ok {
		t.Fatalf("enabled_builtin_tools should be omitted when whitelist is empty: %+v", claw.req.Agent)
	}
	if got := claw.req.Agent["project_root"]; got != "E:/codes/icoo_claw" {
		t.Fatalf("project_root = %q, want desktop project root", got)
	}
	if len(claw.req.ToolWhitelist) != 0 {
		t.Fatalf("tool whitelist = %+v, want no restriction", claw.req.ToolWhitelist)
	}
}

func TestChatServiceAppliesMinimumMaxIterations(t *testing.T) {
	payload := agentProfileMap(model.AgentProfile{
		ID:                "agent_1",
		ModelProvider:     "openai",
		MaxIterations:     1,
		ToolWhitelistJSON: `[]`,
		NetworkAllowJSON:  `[]`,
		MCPServerIDsJSON:  `[]`,
	}, nil, nil)

	if got := payload["max_iterations"]; got != 4 {
		t.Fatalf("max_iterations = %v, want minimum 4", got)
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
		nil,
		instances,
		memorySupervisor{},
	)
	router := NewDefaultRouterPolicy(conversations, instances, starter)
	claw := &chatClaw{}
	svc := NewChatService(conversations, chatAgentRepo{}, nil, router, &chatSessionBackend{}, claw)

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
		nil,
		instances,
		&failingOnceSupervisor{failed: map[string]bool{}},
	)
	router := NewDefaultRouterPolicy(conversations, instances, starter)
	claw := &chatClaw{}
	svc := NewChatService(conversations, chatAgentRepo{}, nil, router, &chatSessionBackend{}, claw)

	if _, err := svc.SendMessage(context.Background(), "conv_1", dto.SendMessageRequest{Prompt: "hello"}); err != nil {
		t.Fatalf("send message: %v", err)
	}
	if claw.baseURL != "http://127.0.0.1:8101" {
		t.Fatalf("baseURL = %q, want reused failed instance port", claw.baseURL)
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
