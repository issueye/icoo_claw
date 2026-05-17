package service

import (
	"context"
	"fmt"
	"time"

	"icoo_claw/server/gateway/internal/config"
	"icoo_claw/server/gateway/internal/dto"
	"icoo_claw/server/gateway/internal/model"
	"icoo_claw/server/gateway/internal/repository"
)

type AgentInstanceService struct {
	cfg        config.Config
	agents     repository.AgentRepository
	instances  repository.AgentInstanceRepository
	supervisor ProcessSupervisor
}

func NewAgentInstanceService(cfg config.Config, agents repository.AgentRepository, instances repository.AgentInstanceRepository, supervisor ProcessSupervisor) *AgentInstanceService {
	return &AgentInstanceService{cfg: cfg, agents: agents, instances: instances, supervisor: supervisor}
}

func (s *AgentInstanceService) Start(ctx context.Context, req dto.StartAgentInstanceRequest) (*dto.AgentInstance, error) {
	if _, err := s.agents.Get(ctx, req.AgentID); err != nil {
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
	process, err := s.supervisor.Start(ctx, spec)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	instance := model.AgentInstance{
		ID:        instanceID,
		AgentID:   req.AgentID,
		Name:      req.Name,
		Status:    "starting",
		PID:       process.PID,
		Host:      spec.Host,
		Port:      spec.Port,
		BaseURL:   spec.BaseURL,
		CreatedAt: now,
		UpdatedAt: now,
	}
	probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := s.supervisor.Probe(probeCtx, instance); err != nil {
		instance.Status = "failed"
		instance.LastError = err.Error()
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

func (s *AgentInstanceService) List(ctx context.Context) ([]dto.AgentInstance, error) {
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
	if err := s.supervisor.Stop(ctx, *instance); err != nil {
		return err
	}
	instance.Status = "stopped"
	instance.UpdatedAt = time.Now().UTC()
	return s.instances.Update(ctx, *instance)
}

func (s *AgentInstanceService) Restart(ctx context.Context, id string) (*dto.AgentInstance, error) {
	instance, err := s.instances.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	_ = s.supervisor.Stop(ctx, *instance)
	started, err := s.Start(ctx, dto.StartAgentInstanceRequest{AgentID: instance.AgentID, Name: instance.Name})
	if err != nil {
		return nil, err
	}
	instance.Status = "stopped"
	instance.UpdatedAt = time.Now().UTC()
	_ = s.instances.Update(ctx, *instance)
	return started, nil
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

func (s *AgentInstanceService) allocatePort(instances []model.AgentInstance) (int, error) {
	used := map[int]struct{}{}
	for _, instance := range instances {
		if instance.Status == "ready" || instance.Status == "starting" || instance.Status == "draining" {
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
		if instance.Status == "ready" || instance.Status == "starting" || instance.Status == "draining" {
			count++
		}
	}
	return count
}

func toAgentInstanceDTO(instance model.AgentInstance) *dto.AgentInstance {
	return &dto.AgentInstance{
		ID:              instance.ID,
		AgentID:         instance.AgentID,
		Status:          instance.Status,
		PID:             instance.PID,
		Host:            instance.Host,
		Port:            instance.Port,
		BaseURL:         instance.BaseURL,
		LastHeartbeatAt: instance.LastHeartbeatAt,
		LastError:       instance.LastError,
		Inflight:        instance.Inflight,
		CreatedAt:       instance.CreatedAt,
		UpdatedAt:       instance.UpdatedAt,
	}
}
