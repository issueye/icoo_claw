package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"icoo_claw/server/gateway/internal/config"
	"icoo_claw/server/gateway/internal/controller"
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

type fakeSupervisor struct{}

func (f fakeSupervisor) Start(context.Context, service.StartAgentInstanceSpec) (*service.AgentProcess, error) {
	return &service.AgentProcess{PID: 1}, nil
}
func (f fakeSupervisor) Stop(context.Context, model.AgentInstance) error  { return nil }
func (f fakeSupervisor) Probe(context.Context, model.AgentInstance) error { return nil }

func TestHealthRoute(t *testing.T) {
	agentRepo := fakeAgentRepo{}
	engine := New(Controllers{
		Health: controller.NewHealthController(),
		Agent:  controller.NewAgentController(service.NewAgentService(agentRepo)),
		AgentInstance: controller.NewAgentInstanceController(service.NewAgentInstanceService(
			config.Config{ClawPortStart: 8101, ClawPortEnd: 8102, MaxAgentInstances: 2},
			agentRepo,
			fakeInstanceRepo{},
			fakeSupervisor{},
		)),
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
