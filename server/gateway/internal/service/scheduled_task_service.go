package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"maps"
	"strings"
	"time"

	"icoo_claw/server/gateway/internal/client"
	"icoo_claw/server/gateway/internal/dto"
	"icoo_claw/server/gateway/internal/model"
	"icoo_claw/server/gateway/internal/repository"
)

type ScheduledTaskService struct {
	repo      repository.ScheduledTaskRepository
	runRepo   repository.ScheduledTaskRunRepository
	agents    repository.AgentRepository
	providers repository.ProviderRepository
	instances repository.AgentInstanceRepository
	skills    *SkillService
	starter   AgentInstanceStarter
	claw      scheduledTaskRunner
}

type scheduledTaskRunner interface {
	Run(ctx context.Context, baseURL string, req client.RunRequest) (*client.RunResponse, error)
}

func NewScheduledTaskService(
	repo repository.ScheduledTaskRepository,
	runRepo repository.ScheduledTaskRunRepository,
	agents repository.AgentRepository,
	providers repository.ProviderRepository,
	instances repository.AgentInstanceRepository,
	starter AgentInstanceStarter,
	claw scheduledTaskRunner,
	skills ...*SkillService,
) *ScheduledTaskService {
	var skillService *SkillService
	if len(skills) > 0 {
		skillService = skills[0]
	}
	return &ScheduledTaskService{
		repo:      repo,
		runRepo:   runRepo,
		agents:    agents,
		providers: providers,
		instances: instances,
		skills:    skillService,
		starter:   starter,
		claw:      claw,
	}
}

func (s *ScheduledTaskService) Create(ctx context.Context, req dto.CreateScheduledTaskRequest) (*dto.ScheduledTask, error) {
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	now := time.Now().UTC()
	task := model.ScheduledTask{
		ID:            strings.TrimSpace(req.ID),
		Name:          strings.TrimSpace(req.Name),
		Description:   strings.TrimSpace(req.Description),
		AgentID:       strings.TrimSpace(req.AgentID),
		ScheduleType:  normalizeScheduleType(req.ScheduleType),
		ScheduleValue: strings.TrimSpace(req.ScheduleValue),
		ActionType:    normalizeActionType(req.ActionType),
		Enabled:       enabled,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if task.ID == "" {
		task.ID = "task_" + randomID()
	}
	payload := req.Payload
	if len(req.ForceSkills) > 0 {
		payload = clonePayload(payload)
		payload["force_skills"] = cleanStringSlice(req.ForceSkills)
	}
	payloadJSON, err := encodePayload(payload)
	if err != nil {
		return nil, err
	}
	task.PayloadJSON = payloadJSON
	if err := s.applySchedule(&task, now); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, task); err != nil {
		return nil, err
	}
	return toScheduledTaskDTO(task), nil
}

func (s *ScheduledTaskService) Get(ctx context.Context, id string) (*dto.ScheduledTask, error) {
	task, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return toScheduledTaskDTO(*task), nil
}

func (s *ScheduledTaskService) List(ctx context.Context) ([]dto.ScheduledTask, error) {
	tasks, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]dto.ScheduledTask, len(tasks))
	for i, task := range tasks {
		out[i] = *toScheduledTaskDTO(task)
	}
	return out, nil
}

func (s *ScheduledTaskService) Update(ctx context.Context, id string, req dto.UpdateScheduledTaskRequest) (*dto.ScheduledTask, error) {
	task, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		task.Name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		task.Description = strings.TrimSpace(*req.Description)
	}
	if req.AgentID != nil {
		task.AgentID = strings.TrimSpace(*req.AgentID)
	}
	if req.ScheduleType != nil {
		task.ScheduleType = normalizeScheduleType(*req.ScheduleType)
	}
	if req.ScheduleValue != nil {
		task.ScheduleValue = strings.TrimSpace(*req.ScheduleValue)
	}
	if req.ActionType != nil {
		task.ActionType = normalizeActionType(*req.ActionType)
	}
	if req.Payload != nil {
		payloadJSON, err := encodePayload(*req.Payload)
		if err != nil {
			return nil, err
		}
		task.PayloadJSON = payloadJSON
	}
	if req.Enabled != nil {
		task.Enabled = *req.Enabled
	}
	if req.ForceSkills != nil {
		payload := decodePayload(task.PayloadJSON)
		payload["force_skills"] = cleanStringSlice(req.ForceSkills)
		encoded, err := encodePayload(payload)
		if err != nil {
			return nil, err
		}
		task.PayloadJSON = encoded
	}
	task.UpdatedAt = time.Now().UTC()
	if err := s.applySchedule(task, task.UpdatedAt); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, *task); err != nil {
		return nil, err
	}
	return toScheduledTaskDTO(*task), nil
}

func (s *ScheduledTaskService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *ScheduledTaskService) RunDue(ctx context.Context, now time.Time) error {
	tasks, err := s.repo.ListDue(ctx, now.UTC())
	if err != nil {
		return err
	}
	for _, task := range tasks {
		if err := s.markExecuted(ctx, task, now.UTC()); err != nil {
			return err
		}
	}
	return nil
}

func (s *ScheduledTaskService) ListRuns(ctx context.Context, taskID string, limit int) ([]dto.ScheduledTaskRun, error) {
	if s.runRepo == nil {
		return []dto.ScheduledTaskRun{}, nil
	}
	runs, err := s.runRepo.ListByTaskID(ctx, taskID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]dto.ScheduledTaskRun, len(runs))
	for i, run := range runs {
		out[i] = dto.ScheduledTaskRun(run)
	}
	return out, nil
}

func (s *ScheduledTaskService) ListTaskRuns(ctx context.Context, taskID string, limit int) ([]dto.ScheduledTaskRun, error) {
	return s.ListRuns(ctx, taskID, limit)
}

func (s *ScheduledTaskService) StartLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				if err := s.RunDue(ctx, now); err != nil {
					log.Printf("scheduled task scan failed: %v", err)
				}
			}
		}
	}()
}

func (s *ScheduledTaskService) markExecuted(ctx context.Context, task model.ScheduledTask, now time.Time) error {
	execStatus := "completed"
	execSummary, execErr := s.executeTask(ctx, task, now)
	if execErr != nil {
		execStatus = "error"
	}

	task.LastRunAt = &now
	task.RunCount++
	task.LastError = ""
	if execErr != nil {
		task.LastError = execErr.Error()
	}
	if task.ScheduleType == "once" {
		task.Enabled = false
		task.Status = "completed"
		task.NextRunAt = nil
	} else {
		next, err := nextRunAt(task.ScheduleType, task.ScheduleValue, now)
		if err != nil {
			task.Enabled = false
			task.Status = "error"
			task.LastError = err.Error()
			task.NextRunAt = nil
		} else {
			task.Status = "active"
			task.NextRunAt = &next
		}
	}
	task.UpdatedAt = now
	if err := s.repo.Update(ctx, task); err != nil {
		return err
	}
	return s.recordRun(ctx, task, now, execStatus, execSummary, errString(execErr))
}

func (s *ScheduledTaskService) applySchedule(task *model.ScheduledTask, now time.Time) error {
	if task.Name == "" {
		return fmt.Errorf("task name is required")
	}
	if task.ScheduleType == "" {
		task.ScheduleType = "interval"
	}
	if task.ActionType == "" {
		task.ActionType = "webhook"
	}
	if task.ActionType == "agent_prompt" && strings.TrimSpace(task.AgentID) == "" {
		return fmt.Errorf("agent_id is required for agent_prompt tasks")
	}
	if task.Enabled {
		next, err := nextRunAt(task.ScheduleType, task.ScheduleValue, now)
		if err != nil {
			return err
		}
		task.NextRunAt = &next
		task.Status = "active"
		return nil
	}
	task.NextRunAt = nil
	task.Status = "paused"
	return nil
}

func nextRunAt(scheduleType, value string, now time.Time) (time.Time, error) {
	switch normalizeScheduleType(scheduleType) {
	case "interval":
		duration, err := time.ParseDuration(strings.TrimSpace(value))
		if err != nil || duration <= 0 {
			return time.Time{}, fmt.Errorf("interval schedule requires a duration like 5m or 1h")
		}
		return now.UTC().Add(duration), nil
	case "daily":
		hour, minute, err := parseDailyClock(value)
		if err != nil {
			return time.Time{}, err
		}
		next := time.Date(now.UTC().Year(), now.UTC().Month(), now.UTC().Day(), hour, minute, 0, 0, time.UTC)
		if !next.After(now.UTC()) {
			next = next.Add(24 * time.Hour)
		}
		return next, nil
	case "once":
		next, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
		if err != nil {
			return time.Time{}, fmt.Errorf("once schedule requires RFC3339 time")
		}
		if !next.After(now.UTC()) {
			return time.Time{}, fmt.Errorf("once schedule must be in the future")
		}
		return next.UTC(), nil
	default:
		return time.Time{}, fmt.Errorf("unsupported schedule type %q", scheduleType)
	}
}

func parseDailyClock(value string) (int, int, error) {
	parsed, err := time.Parse("15:04", strings.TrimSpace(value))
	if err != nil {
		return 0, 0, fmt.Errorf("daily schedule requires HH:mm")
	}
	return parsed.Hour(), parsed.Minute(), nil
}

func normalizeScheduleType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "interval"
	}
	return value
}

func normalizeActionType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "webhook"
	}
	return value
}

func encodePayload(payload map[string]any) (string, error) {
	if payload == nil {
		payload = map[string]any{}
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func decodePayload(raw string) map[string]any {
	var payload map[string]any
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return map[string]any{"raw": raw}
	}
	return payload
}

func toScheduledTaskDTO(task model.ScheduledTask) *dto.ScheduledTask {
	return &dto.ScheduledTask{
		ID:            task.ID,
		Name:          task.Name,
		Description:   task.Description,
		AgentID:       task.AgentID,
		ScheduleType:  task.ScheduleType,
		ScheduleValue: task.ScheduleValue,
		ActionType:    task.ActionType,
		Payload:       decodePayload(task.PayloadJSON),
		ForceSkills:   payloadForceSkills(decodePayload(task.PayloadJSON)),
		Enabled:       task.Enabled,
		Status:        task.Status,
		LastRunAt:     task.LastRunAt,
		NextRunAt:     task.NextRunAt,
		RunCount:      task.RunCount,
		LastError:     task.LastError,
		CreatedAt:     task.CreatedAt,
		UpdatedAt:     task.UpdatedAt,
	}
}

func (s *ScheduledTaskService) recordRun(ctx context.Context, task model.ScheduledTask, now time.Time, status, summary, errText string) error {
	if s.runRepo == nil {
		return nil
	}
	status = strings.TrimSpace(status)
	if status == "" {
		status = "unknown"
	}
	summary = strings.TrimSpace(summary)
	if summary == "" {
		summary = strings.TrimSpace(task.Description)
	}
	if summary == "" {
		summary = strings.TrimSpace(task.Name)
	}
	run := model.ScheduledTaskRun{
		ID:         "run_" + randomID(),
		TaskID:     task.ID,
		AgentID:    task.AgentID,
		Status:     status,
		Summary:    summary,
		Error:      strings.TrimSpace(errText),
		ExecutedAt: now,
	}
	return s.runRepo.Create(ctx, run)
}

func (s *ScheduledTaskService) executeTask(ctx context.Context, task model.ScheduledTask, now time.Time) (string, error) {
	switch task.ActionType {
	case "agent_prompt":
		return s.executeAgentTask(ctx, task, now)
	default:
		summary := strings.TrimSpace(task.Description)
		if summary == "" {
			summary = strings.TrimSpace(task.Name)
		}
		return summary, nil
	}
}

func (s *ScheduledTaskService) executeAgentTask(ctx context.Context, task model.ScheduledTask, now time.Time) (string, error) {
	if s.agents == nil || s.claw == nil {
		return "", fmt.Errorf("agent task execution is unavailable")
	}
	agentID := strings.TrimSpace(task.AgentID)
	if agentID == "" {
		return "", fmt.Errorf("agent_id is required for agent_prompt tasks")
	}

	payload := decodePayload(task.PayloadJSON)
	prompt := firstTaskPrompt(payload)
	if prompt == "" {
		prompt = strings.TrimSpace(task.Description)
	}
	if prompt == "" {
		prompt = strings.TrimSpace(task.Name)
	}
	if prompt == "" {
		return "", fmt.Errorf("agent_prompt task requires a prompt")
	}

	metadata := map[string]any{
		"scheduled_task_id":   task.ID,
		"scheduled_task_name": task.Name,
		"scheduled_task_type": task.ScheduleType,
		"scheduled_action":    task.ActionType,
	}
	if extra := payloadMetadata(payload); len(extra) > 0 {
		maps.Copy(metadata, extra)
	}
	if projectRoot := firstTaskString(payload, "project_root"); projectRoot != "" {
		metadata["project_root"] = projectRoot
	}

	executor := NewGatewayAgentExecutor(GatewayAgentExecutorConfig{
		Agents:    s.agents,
		Providers: s.providers,
		Instances: s.instances,
		Starter:   s.starter,
	})
	execCtx, err := executor.Prepare(ctx, AgentExecutionRequest{
		AgentID:      agentID,
		SessionID:    taskSessionID(task.ID, now),
		Prompt:       prompt,
		RequestID:    "req_" + randomID(),
		ForceSkills:  payloadForceSkills(payload),
		Metadata:     metadata,
		InstanceName: task.Name,
	})
	if err != nil {
		return "", err
	}
	if err := executor.markInflight(ctx, execCtx.Instance.ID, 1); err != nil {
		return "", err
	}
	defer func() { _ = executor.markInflight(context.Background(), execCtx.Instance.ID, -1) }()
	resp, err := s.claw.Run(ctx, execCtx.Instance.BaseURL, execCtx.Request)
	if err != nil {
		return "", err
	}
	if resp == nil {
		return "", nil
	}
	summary := strings.TrimSpace(resp.Output)
	if summary == "" {
		summary = prompt
	}
	return summary, nil
}

func firstTaskPrompt(payload map[string]any) string {
	if text, ok := anyToString(payload["prompt"]); ok {
		return strings.TrimSpace(text)
	}
	if text, ok := anyToString(payload["instruction"]); ok {
		return strings.TrimSpace(text)
	}
	if text, ok := anyToString(payload["message"]); ok {
		return strings.TrimSpace(text)
	}
	return ""
}

func firstTaskString(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	if text, ok := anyToString(payload[key]); ok {
		return strings.TrimSpace(text)
	}
	return ""
}

func payloadMetadata(payload map[string]any) map[string]any {
	if payload == nil {
		return nil
	}
	if raw, ok := payload["metadata"].(map[string]any); ok {
		out := make(map[string]any, len(raw))
		maps.Copy(out, raw)
		return out
	}
	return nil
}

func clonePayload(payload map[string]any) map[string]any {
	out := make(map[string]any, len(payload)+1)
	maps.Copy(out, payload)
	return out
}

func payloadForceSkills(payload map[string]any) []string {
	if payload == nil {
		return nil
	}
	switch raw := payload["force_skills"].(type) {
	case []string:
		return cleanStringSlice(raw)
	case []any:
		values := make([]string, 0, len(raw))
		for _, item := range raw {
			if text, ok := item.(string); ok {
				values = append(values, text)
			}
		}
		return cleanStringSlice(values)
	default:
		return nil
	}
}

func taskSessionID(taskID string, now time.Time) string {
	return "sess_task_" + taskID + "_" + now.UTC().Format("20060102150405")
}

func anyToString(value any) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, true
	case fmt.Stringer:
		return v.String(), true
	default:
		return "", false
	}
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
