package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"icoo_claw/server/gateway/internal/client"
	"icoo_claw/server/gateway/internal/config"
	"icoo_claw/server/gateway/internal/dto"
	"icoo_claw/server/gateway/internal/model"
	"icoo_claw/server/gateway/internal/repository"
)

type AgentInstanceService struct {
	cfg        config.Config
	agents     repository.AgentRepository
	providers  repository.ProviderRepository
	instances  repository.AgentInstanceRepository
	supervisor ProcessSupervisor
	skills     *SkillService
}

var ErrAgentInstanceActive = errors.New("agent instance is still active")

func NewAgentInstanceService(cfg config.Config, agents repository.AgentRepository, providers repository.ProviderRepository, instances repository.AgentInstanceRepository, supervisor ProcessSupervisor, skills ...*SkillService) *AgentInstanceService {
	var skillService *SkillService
	if len(skills) > 0 {
		skillService = skills[0]
	}
	return &AgentInstanceService{cfg: cfg, agents: agents, providers: providers, instances: instances, supervisor: supervisor, skills: skillService}
}

func (s *AgentInstanceService) Start(ctx context.Context, req dto.StartAgentInstanceRequest) (*dto.AgentInstance, error) {
	agent, err := s.agents.Get(ctx, req.AgentID)
	if err != nil {
		return nil, err
	}
	if err := EnsureAgentRunnable(agent); err != nil {
		return nil, err
	}
	launch, err := s.resolveLaunchConfig(ctx, agent)
	if err != nil {
		return nil, err
	}
	existing, err := s.instances.List(ctx)
	if err != nil {
		return nil, err
	}
	if activeCount(existing) >= s.cfg.MaxAgentInstances {
		return nil, fmt.Errorf("max agent instances reached")
	}

	port, err := s.allocatePort(existing)
	if err != nil {
		return nil, err
	}
	instanceID := "inst_" + randomID()
	spec := processSpecFromConfig(s.cfg, instanceID, req.AgentID, port)
	spec.Transport = normalizeTransport(agent.Transport)
	if req.Transport != "" {
		spec.Transport = normalizeTransport(req.Transport)
	}
	if spec.Transport == "acp" {
		spec.BaseURL = client.ACPBaseURL(instanceID)
	}
	spec.CommandArgs = parseStringSlice(agent.CommandArgsJSON)
	if req.CommandArgs != nil {
		spec.CommandArgs = cleanStringSlice(req.CommandArgs)
	}
	spec.Agent = launch
	if s.skills != nil {
		skillsRoot, err := s.skills.PublishForInstance(instanceID, agent.SkillNamesJSON)
		if err != nil {
			return nil, fmt.Errorf("publish skills for agent: %w", err)
		}
		if skillsRoot != "" {
			spec.DefaultProjectRoot = skillsRoot
		}
	}
	process, err := s.supervisor.Start(ctx, spec)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	instance := model.AgentInstance{
		ID:              instanceID,
		AgentID:         req.AgentID,
		Name:            req.Name,
		Status:          "starting",
		PID:             process.PID,
		Host:            spec.Host,
		Port:            spec.Port,
		BaseURL:         spec.BaseURL,
		Transport:       spec.Transport,
		CommandArgsJSON: mustJSON(spec.CommandArgs),
		ProviderID:      launch.ProviderID,
		ModelProvider:   launch.ModelProvider,
		ModelName:       launch.ModelName,
		ModelBaseURL:    launch.BaseURL,
		APIKeySet:       launch.APIKey != "",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	probeCtx, cancel := context.WithTimeout(ctx, startupTimeout(s.cfg))
	defer cancel()
	if err := s.waitUntilReady(probeCtx, instance); err != nil {
		instance.Status = "failed"
		instance.LastError = err.Error()
		_ = s.supervisor.Stop(context.Background(), instance)
		_ = s.instances.Create(ctx, instance)
		return nil, err
	}
	instance.Status = "ready"
	instance.LastHeartbeatAt = &now
	if err := s.instances.Create(ctx, instance); err != nil {
		return nil, err
	}
	return toAgentInstanceDTO(instance), nil
}

func (s *AgentInstanceService) waitUntilReady(ctx context.Context, instance model.AgentInstance) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		if err := s.supervisor.Probe(ctx, instance); err == nil {
			return nil
		} else if ctx.Err() != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func startupTimeout(cfg config.Config) time.Duration {
	if cfg.HealthInterval > 0 && cfg.HealthInterval < 5*time.Second {
		return 5 * time.Second
	}
	return 10 * time.Second
}

func (s *AgentInstanceService) List(ctx context.Context) ([]dto.AgentInstance, error) {
	if err := s.ProbeInstances(ctx); err != nil {
		return nil, err
	}
	instances, err := s.instances.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]dto.AgentInstance, len(instances))
	for i, instance := range instances {
		out[i] = *toAgentInstanceDTO(instance)
	}
	return out, nil
}

func (s *AgentInstanceService) Stop(ctx context.Context, id string) error {
	instance, err := s.instances.Get(ctx, id)
	if err != nil {
		return err
	}
	if instance.Status != "stopped" {
		instance.Status = "draining"
		instance.UpdatedAt = time.Now().UTC()
		if err := s.instances.Update(ctx, *instance); err != nil {
			return err
		}
	}
	if err := s.waitForInflight(ctx, id); err != nil {
		return err
	}
	if err := s.supervisor.Stop(ctx, *instance); err != nil {
		return err
	}
	if s.skills != nil {
		_ = s.skills.CleanupInstance(id)
	}
	instance.Status = "stopped"
	instance.Inflight = 0
	instance.UpdatedAt = time.Now().UTC()
	return s.instances.Update(ctx, *instance)
}

func (s *AgentInstanceService) Restart(ctx context.Context, id string) (*dto.AgentInstance, error) {
	instance, err := s.instances.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	_ = s.supervisor.Stop(ctx, *instance)

	agent, err := s.agents.Get(ctx, instance.AgentID)
	if err != nil {
		return nil, err
	}
	if err := EnsureAgentRunnable(agent); err != nil {
		return nil, err
	}
	launch, err := s.resolveLaunchConfig(ctx, agent)
	if err != nil {
		return nil, err
	}

	existing, err := s.instances.List(ctx)
	if err != nil {
		return nil, err
	}
	var filtered []model.AgentInstance
	for _, inst := range existing {
		if inst.ID != id {
			filtered = append(filtered, inst)
		}
	}
	port, err := s.allocatePort(filtered)
	if err != nil {
		return nil, err
	}

	spec := processSpecFromConfig(s.cfg, id, instance.AgentID, port)
	spec.Transport = normalizeTransport(instance.Transport)
	if spec.Transport == "acp" {
		spec.BaseURL = client.ACPBaseURL(id)
	}
	spec.CommandArgs = parseStringSlice(instance.CommandArgsJSON)
	spec.Agent = launch
	if s.skills != nil {
		skillsRoot, err := s.skills.PublishForInstance(id, agent.SkillNamesJSON)
		if err != nil {
			return nil, fmt.Errorf("publish skills for agent: %w", err)
		}
		if skillsRoot != "" {
			spec.DefaultProjectRoot = skillsRoot
		}
	}

	process, err := s.supervisor.Start(ctx, spec)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	instance.PID = process.PID
	instance.Host = spec.Host
	instance.Port = spec.Port
	instance.BaseURL = spec.BaseURL
	instance.Status = "starting"
	instance.UpdatedAt = now

	probeCtx, cancel := context.WithTimeout(ctx, startupTimeout(s.cfg))
	defer cancel()
	if err := s.waitUntilReady(probeCtx, *instance); err != nil {
		instance.Status = "failed"
		instance.LastError = err.Error()
		_ = s.supervisor.Stop(context.Background(), *instance)
		_ = s.instances.Update(ctx, *instance)
		return nil, err
	}

	instance.Status = "ready"
	instance.LastHeartbeatAt = &now
	instance.LastError = ""
	if err := s.instances.Update(ctx, *instance); err != nil {
		return nil, err
	}

	dtoInst := toAgentInstanceDTO(*instance)
	return dtoInst, nil
}

func (s *AgentInstanceService) resolveLaunchConfig(ctx context.Context, agent *model.AgentProfile) (AgentLaunchConfig, error) {
	if agent == nil {
		return AgentLaunchConfig{}, nil
	}

	builder := NewAgentRuntimeProfileBuilder(s.providers)
	provider, err := builder.ResolveProvider(ctx, agent)
	if err != nil {
		return AgentLaunchConfig{}, err
	}
	return builder.BuildLaunchConfig(*agent, provider), nil
}

func (s *AgentInstanceService) Drain(ctx context.Context, id string) (*dto.AgentInstance, error) {
	instance, err := s.instances.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	instance.Status = "draining"
	instance.UpdatedAt = time.Now().UTC()
	if err := s.instances.Update(ctx, *instance); err != nil {
		return nil, err
	}
	return toAgentInstanceDTO(*instance), nil
}

func (s *AgentInstanceService) Remove(ctx context.Context, id string) error {
	instance, err := s.instances.Get(ctx, id)
	if err != nil {
		return err
	}
	if instance == nil {
		return repository.ErrNotFound
	}
	if instanceUsesPort(*instance) {
		return ErrAgentInstanceActive
	}
	if s.skills != nil {
		_ = s.skills.CleanupInstance(id)
	}
	return s.instances.Delete(ctx, id)
}

func (s *AgentInstanceService) ProbeInstances(ctx context.Context) error {
	instances, err := s.instances.List(ctx)
	if err != nil {
		return err
	}
	for _, instance := range instances {
		if instance.Status != "ready" && instance.Status != "starting" && instance.Status != "draining" {
			continue
		}

		probeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		err := s.supervisor.Probe(probeCtx, instance)
		cancel()

		now := time.Now().UTC()
		if err != nil {
			instance.Status = "failed"
			instance.LastError = err.Error()
			instance.UpdatedAt = now
		} else {
			if instance.Status == "starting" {
				instance.Status = "ready"
			}
			instance.LastError = ""
			instance.LastHeartbeatAt = &now
			instance.UpdatedAt = now
		}
		if updateErr := s.instances.Update(ctx, instance); updateErr != nil {
			return updateErr
		}
	}
	return nil
}

func (s *AgentInstanceService) StartHealthLoop(ctx context.Context) {
	interval := s.cfg.HealthInterval
	if interval <= 0 {
		interval = 10 * time.Second
	}
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := s.ProbeInstances(ctx); err != nil {
					log.Printf("agent instance health probe failed: %v", err)
				}
			}
		}
	}()
}

func (s *AgentInstanceService) waitForInflight(ctx context.Context, id string) error {
	timeout := s.cfg.ShutdownTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		instance, err := s.instances.Get(ctx, id)
		if err != nil {
			return err
		}
		if instance.Inflight <= 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return fmt.Errorf("timeout waiting for instance %s inflight requests to finish", id)
		case <-ticker.C:
		}
	}
}

func (s *AgentInstanceService) allocatePort(instances []model.AgentInstance) (int, error) {
	used := map[int]struct{}{}
	for _, instance := range instances {
		if instanceUsesPort(instance) {
			used[instance.Port] = struct{}{}
		}
	}
	for port := s.cfg.ClawPortStart; port <= s.cfg.ClawPortEnd; port++ {
		if _, ok := used[port]; !ok {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available claw port")
}

func activeCount(instances []model.AgentInstance) int {
	var count int
	for _, instance := range instances {
		if instanceUsesPort(instance) {
			count++
		}
	}
	return count
}

func instanceUsesPort(instance model.AgentInstance) bool {
	return instance.Status == "ready" || instance.Status == "starting" || instance.Status == "draining"
}

func toAgentInstanceDTO(instance model.AgentInstance) *dto.AgentInstance {
	return &dto.AgentInstance{
		ID:              instance.ID,
		AgentID:         instance.AgentID,
		Name:            instance.Name,
		Status:          instance.Status,
		PID:             instance.PID,
		Host:            instance.Host,
		Port:            instance.Port,
		BaseURL:         instance.BaseURL,
		Transport:       defaultString(instance.Transport, "http"),
		CommandArgs:     parseStringSlice(instance.CommandArgsJSON),
		ProviderID:      instance.ProviderID,
		ModelProvider:   instance.ModelProvider,
		ModelName:       instance.ModelName,
		ModelBaseURL:    instance.ModelBaseURL,
		APIKeySet:       instance.APIKeySet,
		LastHeartbeatAt: instance.LastHeartbeatAt,
		LastError:       instance.LastError,
		Inflight:        instance.Inflight,
		CreatedAt:       instance.CreatedAt,
		UpdatedAt:       instance.UpdatedAt,
	}
}
