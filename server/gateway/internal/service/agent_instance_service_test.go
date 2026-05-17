package service

import (
	"context"
	"testing"

	"icoo_claw/server/gateway/internal/config"
	"icoo_claw/server/gateway/internal/dto"
	"icoo_claw/server/gateway/internal/model"
)

type instanceAgentRepo struct{}

func (r instanceAgentRepo) Create(context.Context, model.AgentProfile) error { return nil }
func (r instanceAgentRepo) Get(context.Context, string) (*model.AgentProfile, error) {
	return &model.AgentProfile{ID: "agent_1", Name: "Default"}, nil
}
func (r instanceAgentRepo) List(context.Context) ([]model.AgentProfile, error) { return nil, nil }
func (r instanceAgentRepo) Update(context.Context, model.AgentProfile) error   { return nil }
func (r instanceAgentRepo) Delete(context.Context, string) error               { return nil }

type memoryInstanceRepo struct {
	instances []model.AgentInstance
}

func (r *memoryInstanceRepo) Create(_ context.Context, instance model.AgentInstance) error {
	r.instances = append(r.instances, instance)
	return nil
}
func (r *memoryInstanceRepo) Get(_ context.Context, id string) (*model.AgentInstance, error) {
	for i := range r.instances {
		if r.instances[i].ID == id {
			return &r.instances[i], nil
		}
	}
	return nil, nil
}
func (r *memoryInstanceRepo) List(context.Context) ([]model.AgentInstance, error) {
	return append([]model.AgentInstance(nil), r.instances...), nil
}
func (r *memoryInstanceRepo) Update(_ context.Context, instance model.AgentInstance) error {
	for i := range r.instances {
		if r.instances[i].ID == instance.ID {
			r.instances[i] = instance
		}
	}
	return nil
}

type memorySupervisor struct{}

func (s memorySupervisor) Start(context.Context, StartAgentInstanceSpec) (*AgentProcess, error) {
	return &AgentProcess{PID: 42}, nil
}
func (s memorySupervisor) Stop(context.Context, model.AgentInstance) error  { return nil }
func (s memorySupervisor) Probe(context.Context, model.AgentInstance) error { return nil }

func TestAgentInstanceServiceStartAllocatesPort(t *testing.T) {
	repo := &memoryInstanceRepo{}
	svc := NewAgentInstanceService(
		config.Config{ClawPortStart: 8101, ClawPortEnd: 8102, MaxAgentInstances: 2},
		instanceAgentRepo{},
		repo,
		memorySupervisor{},
	)

	first, err := svc.Start(context.Background(), dto.StartAgentInstanceRequest{AgentID: "agent_1"})
	if err != nil {
		t.Fatalf("start first: %v", err)
	}
	second, err := svc.Start(context.Background(), dto.StartAgentInstanceRequest{AgentID: "agent_1"})
	if err != nil {
		t.Fatalf("start second: %v", err)
	}
	if first.Port != 8101 || second.Port != 8102 {
		t.Fatalf("ports = %d, %d", first.Port, second.Port)
	}
}
