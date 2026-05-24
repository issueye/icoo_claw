package service

import (
	"context"
	"testing"
	"time"

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
