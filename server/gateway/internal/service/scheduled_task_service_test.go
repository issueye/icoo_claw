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

type memoryScheduledTaskRepo struct {
	tasks []model.ScheduledTask
}

type memoryScheduledTaskRunRepo struct {
	runs []model.ScheduledTaskRun
}

func (r *memoryScheduledTaskRepo) Create(_ context.Context, task model.ScheduledTask) error {
	r.tasks = append(r.tasks, task)
	return nil
}
func (r *memoryScheduledTaskRepo) Get(_ context.Context, id string) (*model.ScheduledTask, error) {
	for i := range r.tasks {
		if r.tasks[i].ID == id {
			return &r.tasks[i], nil
		}
	}
	return nil, repository.ErrNotFound
}
func (r *memoryScheduledTaskRepo) List(context.Context) ([]model.ScheduledTask, error) {
	return append([]model.ScheduledTask(nil), r.tasks...), nil
}
func (r *memoryScheduledTaskRepo) ListDue(_ context.Context, now time.Time) ([]model.ScheduledTask, error) {
	var due []model.ScheduledTask
	for _, task := range r.tasks {
		if task.Enabled && task.NextRunAt != nil && !task.NextRunAt.After(now.UTC()) {
			due = append(due, task)
		}
	}
	return due, nil
}
func (r *memoryScheduledTaskRepo) Update(_ context.Context, task model.ScheduledTask) error {
	for i := range r.tasks {
		if r.tasks[i].ID == task.ID {
			r.tasks[i] = task
			return nil
		}
	}
	return repository.ErrNotFound
}
func (r *memoryScheduledTaskRepo) Delete(_ context.Context, id string) error {
	for i := range r.tasks {
		if r.tasks[i].ID == id {
			r.tasks = append(r.tasks[:i], r.tasks[i+1:]...)
			return nil
		}
	}
	return repository.ErrNotFound
}

func (r *memoryScheduledTaskRunRepo) Create(_ context.Context, run model.ScheduledTaskRun) error {
	r.runs = append(r.runs, run)
	return nil
}

func (r *memoryScheduledTaskRunRepo) ListByTaskID(_ context.Context, taskID string, limit int) ([]model.ScheduledTaskRun, error) {
	var runs []model.ScheduledTaskRun
	for _, run := range r.runs {
		if run.TaskID == taskID {
			runs = append(runs, run)
		}
	}
	if limit > 0 && len(runs) > limit {
		runs = runs[:limit]
	}
	return runs, nil
}

type scheduledTaskAgentRepo struct {
	agent model.AgentProfile
}

func (r scheduledTaskAgentRepo) Create(context.Context, model.AgentProfile) error { return nil }
func (r scheduledTaskAgentRepo) Get(context.Context, string) (*model.AgentProfile, error) {
	agent := r.agent
	return &agent, nil
}
func (r scheduledTaskAgentRepo) List(context.Context) ([]model.AgentProfile, error) { return nil, nil }
func (r scheduledTaskAgentRepo) Update(context.Context, model.AgentProfile) error   { return nil }
func (r scheduledTaskAgentRepo) Delete(context.Context, string) error               { return nil }

type scheduledTaskClaw struct {
	req client.RunRequest
}

func (c *scheduledTaskClaw) Run(_ context.Context, _ string, req client.RunRequest) (*client.RunResponse, error) {
	c.req = req
	return &client.RunResponse{SessionID: req.SessionID, RequestID: req.RequestID, Output: "task output", StopReason: "end_turn"}, nil
}

func TestScheduledTaskServiceCreateInterval(t *testing.T) {
	repo := &memoryScheduledTaskRepo{}
	svc := NewScheduledTaskService(repo, nil, nil, nil, nil, nil, nil)

	task, err := svc.Create(context.Background(), dto.CreateScheduledTaskRequest{
		Name:          "Heartbeat",
		ScheduleType:  "interval",
		ScheduleValue: "5m",
		ActionType:    "webhook",
		Payload:       map[string]any{"url": "http://127.0.0.1"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if task.ID == "" || task.Status != "active" || task.NextRunAt == nil {
		t.Fatalf("task = %+v, want active task with next run", task)
	}
	if task.Payload["url"] != "http://127.0.0.1" {
		t.Fatalf("payload = %+v", task.Payload)
	}
}

func TestScheduledTaskServiceCreateStoresForceSkills(t *testing.T) {
	repo := &memoryScheduledTaskRepo{}
	svc := NewScheduledTaskService(repo, nil, nil, nil, nil, nil, nil)

	task, err := svc.Create(context.Background(), dto.CreateScheduledTaskRequest{
		Name:          "Skill task",
		ScheduleType:  "interval",
		ScheduleValue: "5m",
		ActionType:    "agent_prompt",
		AgentID:       "agent_1",
		Payload:       map[string]any{"prompt": "hello"},
		ForceSkills:   []string{" doc-writer ", ""},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if len(task.ForceSkills) != 1 || task.ForceSkills[0] != "doc-writer" {
		t.Fatalf("force skills = %+v", task.ForceSkills)
	}
	if raw := task.Payload["force_skills"]; raw == nil {
		t.Fatalf("payload force_skills missing: %+v", task.Payload)
	}
}

func TestScheduledTaskServiceRunDueCompletesOnce(t *testing.T) {
	now := time.Now().UTC()
	repo := &memoryScheduledTaskRepo{tasks: []model.ScheduledTask{
		{
			ID:            "task_1",
			Name:          "Once",
			ScheduleType:  "once",
			ScheduleValue: now.Add(time.Hour).Format(time.RFC3339),
			ActionType:    "webhook",
			Enabled:       true,
			Status:        "active",
			NextRunAt:     &now,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}}
	runRepo := &memoryScheduledTaskRunRepo{}
	svc := NewScheduledTaskService(repo, runRepo, nil, nil, nil, nil, nil)

	if err := svc.RunDue(context.Background(), now); err != nil {
		t.Fatalf("run due: %v", err)
	}
	if repo.tasks[0].Enabled || repo.tasks[0].Status != "completed" || repo.tasks[0].RunCount != 1 {
		t.Fatalf("task after due = %+v", repo.tasks[0])
	}
	if len(runRepo.runs) != 1 || runRepo.runs[0].TaskID != "task_1" || runRepo.runs[0].Status != "completed" {
		t.Fatalf("runs = %+v, want completed run for task_1", runRepo.runs)
	}
}

func TestScheduledTaskServiceRejectsInvalidSchedule(t *testing.T) {
	svc := NewScheduledTaskService(&memoryScheduledTaskRepo{}, nil, nil, nil, nil, nil, nil)
	_, err := svc.Create(context.Background(), dto.CreateScheduledTaskRequest{
		Name:          "Bad",
		ScheduleType:  "interval",
		ScheduleValue: "soon",
		ActionType:    "webhook",
	})
	if err == nil {
		t.Fatal("expected invalid schedule error")
	}
}

func TestScheduledTaskServiceRejectsDisabledAgentTask(t *testing.T) {
	now := time.Now().UTC()
	task := model.ScheduledTask{
		ID:            "task_1",
		Name:          "Disabled agent task",
		ScheduleType:  "once",
		ScheduleValue: now.Add(time.Hour).Format(time.RFC3339),
		ActionType:    "agent_prompt",
		AgentID:       "agent_1",
		PayloadJSON:   `{"prompt":"hello"}`,
		Enabled:       true,
		Status:        "active",
		NextRunAt:     &now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	repo := &memoryScheduledTaskRepo{tasks: []model.ScheduledTask{task}}
	runRepo := &memoryScheduledTaskRunRepo{}
	svc := NewScheduledTaskService(
		repo,
		runRepo,
		scheduledTaskAgentRepo{agent: model.AgentProfile{ID: "agent_1", Name: "Disabled", Enabled: false}},
		nil,
		&memoryInstanceRepo{instances: []model.AgentInstance{{ID: "inst_1", AgentID: "agent_1", Status: "ready", BaseURL: "http://127.0.0.1:8101"}}},
		nil,
		&scheduledTaskClaw{},
	)

	if err := svc.RunDue(context.Background(), now); err != nil {
		t.Fatalf("run due: %v", err)
	}
	if repo.tasks[0].Status != "completed" || repo.tasks[0].LastError != ErrAgentDisabled.Error() {
		t.Fatalf("task after run = %+v, want completed with disabled error", repo.tasks[0])
	}
	if len(runRepo.runs) != 1 || runRepo.runs[0].Status != "error" || runRepo.runs[0].Error != ErrAgentDisabled.Error() {
		t.Fatalf("runs = %+v, want disabled agent error run", runRepo.runs)
	}
}

func TestScheduledTaskServiceAgentTaskBuildsRuntimePayload(t *testing.T) {
	now := time.Now().UTC()
	task := model.ScheduledTask{
		ID:            "task_1",
		Name:          "Agent task",
		ScheduleType:  "once",
		ScheduleValue: now.Add(time.Hour).Format(time.RFC3339),
		ActionType:    "agent_prompt",
		AgentID:       "agent_1",
		PayloadJSON:   `{"prompt":"hello","project_root":"E:/project","force_skills":[" doc-writer ",""]}`,
		Enabled:       true,
		Status:        "active",
		NextRunAt:     &now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	repo := &memoryScheduledTaskRepo{tasks: []model.ScheduledTask{task}}
	claw := &scheduledTaskClaw{}
	svc := NewScheduledTaskService(
		repo,
		&memoryScheduledTaskRunRepo{},
		scheduledTaskAgentRepo{agent: model.AgentProfile{
			ID:                "agent_1",
			Name:              "Agent",
			ModelProvider:     "openai",
			ToolWhitelistJSON: `[]`,
			NetworkAllowJSON:  `["example.com"]`,
			MCPServerIDsJSON:  `["mcp_1"]`,
			Enabled:           true,
		}},
		nil,
		&memoryInstanceRepo{instances: []model.AgentInstance{{ID: "inst_1", AgentID: "agent_1", Status: "ready", BaseURL: "http://127.0.0.1:8101"}}},
		nil,
		claw,
	)

	if err := svc.RunDue(context.Background(), now); err != nil {
		t.Fatalf("run due: %v", err)
	}
	if claw.req.Prompt != "hello" {
		t.Fatalf("prompt = %q, want hello", claw.req.Prompt)
	}
	if got := claw.req.Agent.ProjectRoot; got != "E:/project" {
		t.Fatalf("project_root = %q", got)
	}
	if got := claw.req.Agent.NetworkAllow; len(got) != 1 || got[0] != "example.com" {
		t.Fatalf("network_allow = %+v", got)
	}
	if got := claw.req.Agent.MCPServers; len(got) != 1 || got[0] != "mcp_1" {
		t.Fatalf("mcp_servers = %+v", got)
	}
	if len(claw.req.Agent.EnabledBuiltinTools) != 0 {
		t.Fatalf("enabled_builtin_tools should be omitted when whitelist is empty: %+v", claw.req.Agent)
	}
	if len(claw.req.ForceSkills) != 1 || claw.req.ForceSkills[0] != "doc-writer" {
		t.Fatalf("force skills = %+v", claw.req.ForceSkills)
	}
}
