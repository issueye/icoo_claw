package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"icoo_claw/server/gateway/internal/client"
	"icoo_claw/server/gateway/internal/config"
	"icoo_claw/server/gateway/internal/controller"
	"icoo_claw/server/gateway/internal/dto"
	"icoo_claw/server/gateway/internal/model"
	"icoo_claw/server/gateway/internal/repository"
	"icoo_claw/server/gateway/internal/service"
)

type fakeAgentRepo struct{}

func (f fakeAgentRepo) Create(context.Context, model.AgentProfile) error { return nil }
func (f fakeAgentRepo) Get(context.Context, string) (*model.AgentProfile, error) {
	return nil, repository.ErrNotFound
}
func (f fakeAgentRepo) List(context.Context) ([]model.AgentProfile, error) { return nil, nil }
func (f fakeAgentRepo) Update(context.Context, model.AgentProfile) error   { return nil }
func (f fakeAgentRepo) Delete(context.Context, string) error               { return nil }

type fakeInstanceRepo struct{}

func (f fakeInstanceRepo) Create(context.Context, model.AgentInstance) error { return nil }
func (f fakeInstanceRepo) Get(context.Context, string) (*model.AgentInstance, error) {
	return nil, repository.ErrNotFound
}
func (f fakeInstanceRepo) List(context.Context) ([]model.AgentInstance, error) { return nil, nil }
func (f fakeInstanceRepo) Update(context.Context, model.AgentInstance) error   { return nil }
func (f fakeInstanceRepo) AdjustInflight(context.Context, string, int) error   { return nil }

type fakeConversationRepo struct{}

func (f fakeConversationRepo) Create(context.Context, model.Conversation) error { return nil }
func (f fakeConversationRepo) Get(context.Context, string) (*model.Conversation, error) {
	return nil, repository.ErrNotFound
}
func (f fakeConversationRepo) List(context.Context) ([]model.Conversation, error) { return nil, nil }
func (f fakeConversationRepo) Update(context.Context, model.Conversation) error   { return nil }
func (f fakeConversationRepo) Delete(context.Context, string) error               { return nil }

type fakeSessionStore struct{}

func (f fakeSessionStore) CreateSession(context.Context, client.CreateSessionRequest) error {
	return nil
}
func (f fakeSessionStore) ListMessages(context.Context, string) ([]dto.SessionMessage, error) {
	return nil, nil
}

type fakeClawRunner struct{}

func (f fakeClawRunner) Run(context.Context, string, client.RunRequest) (*client.RunResponse, error) {
	return &client.RunResponse{Output: "ok"}, nil
}
func (f fakeClawRunner) Stream(context.Context, string, client.RunRequest) (<-chan client.StreamEvent, error) {
	out := make(chan client.StreamEvent)
	close(out)
	return out, nil
}

type fakeSupervisor struct{}

func (f fakeSupervisor) Start(context.Context, service.StartAgentInstanceSpec) (*service.AgentProcess, error) {
	return &service.AgentProcess{PID: 1}, nil
}
func (f fakeSupervisor) Stop(context.Context, model.AgentInstance) error  { return nil }
func (f fakeSupervisor) Probe(context.Context, model.AgentInstance) error { return nil }

func TestHealthRoute(t *testing.T) {
	agentRepo := fakeAgentRepo{}
	instanceRepo := fakeInstanceRepo{}
	conversationRepo := fakeConversationRepo{}
	instanceService := service.NewAgentInstanceService(
		config.Config{ClawPortStart: 8101, ClawPortEnd: 8102, MaxAgentInstances: 2},
		agentRepo,
		instanceRepo,
		fakeSupervisor{},
	)
	engine := New(Controllers{
		Health:        controller.NewHealthController(),
		Agent:         controller.NewAgentController(service.NewAgentService(agentRepo)),
		AgentInstance: controller.NewAgentInstanceController(instanceService),
		Chat: controller.NewChatController(service.NewChatService(
			conversationRepo,
			agentRepo,
			service.NewDefaultRouterPolicy(conversationRepo, instanceRepo, instanceService),
			fakeSessionStore{},
			fakeClawRunner{},
		)),
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
