package service

import (
	"context"
	"errors"
	"testing"
	"time"

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
func (r *memoryInstanceRepo) AdjustInflight(_ context.Context, id string, delta int) error {
	for i := range r.instances {
		if r.instances[i].ID == id {
			r.instances[i].Inflight += delta
			if r.instances[i].Inflight < 0 {
				r.instances[i].Inflight = 0
			}
			return nil
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

type probeSupervisor struct {
	err error
}

func (s probeSupervisor) Start(context.Context, StartAgentInstanceSpec) (*AgentProcess, error) {
	return &AgentProcess{PID: 42}, nil
}
func (s probeSupervisor) Stop(context.Context, model.AgentInstance) error { return nil }
func (s probeSupervisor) Probe(context.Context, model.AgentInstance) error {
	return s.err
}

type stopSupervisor struct {
	stopped bool
}

func (s *stopSupervisor) Start(context.Context, StartAgentInstanceSpec) (*AgentProcess, error) {
	return &AgentProcess{PID: 42}, nil
}
func (s *stopSupervisor) Stop(context.Context, model.AgentInstance) error {
	s.stopped = true
	return nil
}
func (s *stopSupervisor) Probe(context.Context, model.AgentInstance) error { return nil }

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

func TestAgentInstanceServiceProbeInstancesUpdatesStatus(t *testing.T) {
	now := time.Now().UTC()
	repo := &memoryInstanceRepo{instances: []model.AgentInstance{
		{ID: "inst_1", AgentID: "agent_1", Status: "starting", BaseURL: "http://127.0.0.1:8101", CreatedAt: now, UpdatedAt: now},
	}}
	svc := NewAgentInstanceService(
		config.Config{},
		instanceAgentRepo{},
		repo,
		probeSupervisor{},
	)

	if err := svc.ProbeInstances(context.Background()); err != nil {
		t.Fatalf("probe: %v", err)
	}
	if repo.instances[0].Status != "ready" {
		t.Fatalf("status = %q, want ready", repo.instances[0].Status)
	}
	if repo.instances[0].LastHeartbeatAt == nil {
		t.Fatal("expected heartbeat")
	}
}

func TestAgentInstanceServiceProbeInstancesMarksFailed(t *testing.T) {
	probeErr := errors.New("down")
	repo := &memoryInstanceRepo{instances: []model.AgentInstance{
		{ID: "inst_1", AgentID: "agent_1", Status: "ready", BaseURL: "http://127.0.0.1:8101"},
	}}
	svc := NewAgentInstanceService(
		config.Config{},
		instanceAgentRepo{},
		repo,
		probeSupervisor{err: probeErr},
	)

	if err := svc.ProbeInstances(context.Background()); err != nil {
		t.Fatalf("probe: %v", err)
	}
	if repo.instances[0].Status != "failed" || repo.instances[0].LastError != "down" {
		t.Fatalf("instance = %+v, want failed with error", repo.instances[0])
	}
}

func TestAgentInstanceServiceStopWaitsForInflight(t *testing.T) {
	repo := &memoryInstanceRepo{instances: []model.AgentInstance{
		{ID: "inst_1", AgentID: "agent_1", Status: "ready", PID: 42, Inflight: 1},
	}}
	supervisor := &stopSupervisor{}
	svc := NewAgentInstanceService(
		config.Config{ShutdownTimeout: time.Second},
		instanceAgentRepo{},
		repo,
		supervisor,
	)

	done := make(chan error, 1)
	go func() {
		done <- svc.Stop(context.Background(), "inst_1")
	}()

	time.Sleep(20 * time.Millisecond)
	if repo.instances[0].Status != "draining" {
		t.Fatalf("status = %q, want draining", repo.instances[0].Status)
	}
	repo.instances[0].Inflight = 0

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("stop: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("stop did not finish")
	}
	if !supervisor.stopped {
		t.Fatal("expected supervisor stop")
	}
	if repo.instances[0].Status != "stopped" {
		t.Fatalf("status = %q, want stopped", repo.instances[0].Status)
	}
}
