package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"icoo_claw/server/gateway/internal/config"
	"icoo_claw/server/gateway/internal/dto"
	"icoo_claw/server/gateway/internal/model"
)

type instanceAgentRepo struct {
	agent model.AgentProfile
}

func (r instanceAgentRepo) Create(context.Context, model.AgentProfile) error { return nil }
func (r instanceAgentRepo) Get(context.Context, string) (*model.AgentProfile, error) {
	if r.agent.ID == "" {
		return &model.AgentProfile{ID: "agent_1", Name: "Default"}, nil
	}
	agent := r.agent
	return &agent, nil
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
func (r *memoryInstanceRepo) Delete(_ context.Context, id string) error {
	for i := range r.instances {
		if r.instances[i].ID == id {
			r.instances = append(r.instances[:i], r.instances[i+1:]...)
			return nil
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

type captureSupervisor struct {
	spec StartAgentInstanceSpec
}

func (s *captureSupervisor) Start(_ context.Context, spec StartAgentInstanceSpec) (*AgentProcess, error) {
	s.spec = spec
	return &AgentProcess{PID: 42}, nil
}
func (s *captureSupervisor) Stop(context.Context, model.AgentInstance) error  { return nil }
func (s *captureSupervisor) Probe(context.Context, model.AgentInstance) error { return nil }

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
		nil,
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

func TestAgentInstanceServiceStartReusesFailedInstancePorts(t *testing.T) {
	repo := &memoryInstanceRepo{instances: []model.AgentInstance{
		{ID: "inst_failed_1", AgentID: "agent_1", Status: "failed", Port: 8101},
		{ID: "inst_failed_2", AgentID: "agent_1", Status: "failed", Port: 8102},
	}}
	svc := NewAgentInstanceService(
		config.Config{ClawPortStart: 8101, ClawPortEnd: 8102, MaxAgentInstances: 2},
		instanceAgentRepo{},
		nil,
		repo,
		memorySupervisor{},
	)

	started, err := svc.Start(context.Background(), dto.StartAgentInstanceRequest{AgentID: "agent_1"})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if started.Port != 8101 {
		t.Fatalf("port = %d, want failed port reused", started.Port)
	}
}

func TestAgentInstanceServiceStartUsesAgentTransport(t *testing.T) {
	repo := &memoryInstanceRepo{}
	supervisor := &captureSupervisor{}
	svc := NewAgentInstanceService(
		config.Config{ClawPortStart: 8101, ClawPortEnd: 8101, MaxAgentInstances: 1},
		instanceAgentRepo{agent: model.AgentProfile{ID: "agent_1", Name: "Default", Transport: "acp"}},
		nil,
		repo,
		supervisor,
	)

	started, err := svc.Start(context.Background(), dto.StartAgentInstanceRequest{AgentID: "agent_1"})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if started.Transport != "acp" || supervisor.spec.Transport != "acp" {
		t.Fatalf("transport = dto %q spec %q, want acp", started.Transport, supervisor.spec.Transport)
	}
	if started.BaseURL != "acp://"+started.ID {
		t.Fatalf("baseURL = %q, want acp instance URL", started.BaseURL)
	}
}

func TestAgentInstanceServiceStartRequestTransportOverridesAgent(t *testing.T) {
	repo := &memoryInstanceRepo{}
	supervisor := &captureSupervisor{}
	svc := NewAgentInstanceService(
		config.Config{ClawPortStart: 8101, ClawPortEnd: 8101, MaxAgentInstances: 1},
		instanceAgentRepo{agent: model.AgentProfile{ID: "agent_1", Name: "Default", Transport: "http"}},
		nil,
		repo,
		supervisor,
	)

	started, err := svc.Start(context.Background(), dto.StartAgentInstanceRequest{AgentID: "agent_1", Transport: "acp"})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if started.Transport != "acp" || supervisor.spec.Transport != "acp" {
		t.Fatalf("transport = dto %q spec %q, want acp", started.Transport, supervisor.spec.Transport)
	}
	if started.BaseURL != "acp://"+started.ID {
		t.Fatalf("baseURL = %q, want acp instance URL", started.BaseURL)
	}
}

func TestAgentInstanceServiceStartUsesAgentCommandArgs(t *testing.T) {
	repo := &memoryInstanceRepo{}
	supervisor := &captureSupervisor{}
	svc := NewAgentInstanceService(
		config.Config{ClawPortStart: 8101, ClawPortEnd: 8101, MaxAgentInstances: 1},
		instanceAgentRepo{agent: model.AgentProfile{
			ID:              "agent_1",
			Name:            "Default",
			CommandArgsJSON: `["--runner-mode","fake"]`,
		}},
		nil,
		repo,
		supervisor,
	)

	started, err := svc.Start(context.Background(), dto.StartAgentInstanceRequest{AgentID: "agent_1"})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	want := []string{"--runner-mode", "fake"}
	if !stringSlicesEqual(supervisor.spec.CommandArgs, want) {
		t.Fatalf("spec command args = %+v, want %+v", supervisor.spec.CommandArgs, want)
	}
	if !stringSlicesEqual(started.CommandArgs, want) {
		t.Fatalf("dto command args = %+v, want %+v", started.CommandArgs, want)
	}
}

func TestAgentInstanceServiceStartPublishesBoundSkillRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".skills")
	skillRepo := &memorySkillRepo{}
	skills := NewSkillService(root, skillRepo)
	if _, err := skills.Create(context.Background(), dto.CreateSkillRequest{
		ID:          "skill_1",
		Name:        "doc-writer",
		Description: "Write documents",
		Path:        "docs/doc-writer",
	}); err != nil {
		t.Fatalf("create skill: %v", err)
	}

	repo := &memoryInstanceRepo{}
	supervisor := &captureSupervisor{}
	svc := NewAgentInstanceService(
		config.Config{ClawPortStart: 8101, ClawPortEnd: 8101, MaxAgentInstances: 1},
		instanceAgentRepo{agent: model.AgentProfile{
			ID:           "agent_1",
			Name:         "Default",
			SkillIDsJSON: `["doc-writer"]`,
		}},
		nil,
		repo,
		supervisor,
		skills,
	)

	started, err := svc.Start(context.Background(), dto.StartAgentInstanceRequest{AgentID: "agent_1"})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	wantRoot := filepath.Join(root, "agents", started.ID)
	if supervisor.spec.DefaultProjectRoot != wantRoot {
		t.Fatalf("default project root = %q, want %q", supervisor.spec.DefaultProjectRoot, wantRoot)
	}
	if _, err := os.Stat(filepath.Join(wantRoot, ".agents", "skills", "doc-writer", "SKILL.md")); err != nil {
		t.Fatalf("published skill missing: %v", err)
	}
}

func TestAgentInstanceServiceStartDefaultsToActiveSkillsRoot(t *testing.T) {
	workDir := t.TempDir()
	repo := &memoryInstanceRepo{}
	supervisor := &captureSupervisor{}
	svc := NewAgentInstanceService(
		config.Config{ClawPortStart: 8101, ClawPortEnd: 8101, MaxAgentInstances: 1, GatewayWorkDir: workDir},
		instanceAgentRepo{agent: model.AgentProfile{ID: "agent_1", Name: "Default"}},
		nil,
		repo,
		supervisor,
	)

	if _, err := svc.Start(context.Background(), dto.StartAgentInstanceRequest{AgentID: "agent_1"}); err != nil {
		t.Fatalf("start: %v", err)
	}
	wantRoot := filepath.Join(workDir, ".skills", "active")
	if supervisor.spec.DefaultProjectRoot != wantRoot {
		t.Fatalf("default project root = %q, want %q", supervisor.spec.DefaultProjectRoot, wantRoot)
	}
}

func TestAgentInstanceServiceStartRequestCommandArgsOverrideAgent(t *testing.T) {
	repo := &memoryInstanceRepo{}
	supervisor := &captureSupervisor{}
	svc := NewAgentInstanceService(
		config.Config{ClawPortStart: 8101, ClawPortEnd: 8101, MaxAgentInstances: 1},
		instanceAgentRepo{agent: model.AgentProfile{
			ID:              "agent_1",
			Name:            "Default",
			CommandArgsJSON: `["--runner-mode","sdk"]`,
		}},
		nil,
		repo,
		supervisor,
	)

	started, err := svc.Start(context.Background(), dto.StartAgentInstanceRequest{
		AgentID:     "agent_1",
		CommandArgs: []string{" --runner-mode ", "fake", ""},
	})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	want := []string{"--runner-mode", "fake"}
	if !stringSlicesEqual(supervisor.spec.CommandArgs, want) {
		t.Fatalf("spec command args = %+v, want %+v", supervisor.spec.CommandArgs, want)
	}
	if !stringSlicesEqual(started.CommandArgs, want) {
		t.Fatalf("dto command args = %+v, want %+v", started.CommandArgs, want)
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
		nil,
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

func stringSlicesEqual(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestAgentInstanceServiceProbeInstancesMarksFailed(t *testing.T) {
	probeErr := errors.New("down")
	repo := &memoryInstanceRepo{instances: []model.AgentInstance{
		{ID: "inst_1", AgentID: "agent_1", Status: "ready", BaseURL: "http://127.0.0.1:8101"},
	}}
	svc := NewAgentInstanceService(
		config.Config{},
		instanceAgentRepo{},
		nil,
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
		nil,
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

func TestAgentInstanceServiceRemoveDeletesStoppedInstance(t *testing.T) {
	repo := &memoryInstanceRepo{instances: []model.AgentInstance{
		{ID: "inst_1", AgentID: "agent_1", Status: "stopped", Port: 8101},
		{ID: "inst_2", AgentID: "agent_1", Status: "failed", Port: 8102},
	}}
	svc := NewAgentInstanceService(config.Config{}, instanceAgentRepo{}, nil, repo, memorySupervisor{})

	if err := svc.Remove(context.Background(), "inst_1"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if len(repo.instances) != 1 || repo.instances[0].ID != "inst_2" {
		t.Fatalf("instances after remove = %+v", repo.instances)
	}
}

func TestAgentInstanceServiceRemoveRejectsActiveInstance(t *testing.T) {
	repo := &memoryInstanceRepo{instances: []model.AgentInstance{
		{ID: "inst_1", AgentID: "agent_1", Status: "ready", Port: 8101},
	}}
	svc := NewAgentInstanceService(config.Config{}, instanceAgentRepo{}, nil, repo, memorySupervisor{})

	err := svc.Remove(context.Background(), "inst_1")
	if !errors.Is(err, ErrAgentInstanceActive) {
		t.Fatalf("remove error = %v, want ErrAgentInstanceActive", err)
	}
	if len(repo.instances) != 1 {
		t.Fatalf("active instance was removed: %+v", repo.instances)
	}
}
