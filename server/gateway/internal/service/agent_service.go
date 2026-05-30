package service

import (
	"context"
	"strings"

	"icoo_claw/common/id"
	"icoo_claw/common/jsonutil"
	"icoo_claw/common/stringutil"
	"icoo_claw/server/gateway/internal/dto"
	"icoo_claw/server/gateway/internal/model"
	"icoo_claw/server/gateway/internal/repository"
)

type AgentService struct {
	repo repository.AgentRepository
}

func NewAgentService(repo repository.AgentRepository) *AgentService {
	return &AgentService{repo: repo}
}

func (s *AgentService) Create(ctx context.Context, req dto.CreateAgentRequest) (*dto.AgentProfile, error) {
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	agent := model.AgentProfile{
		ID:                strings.TrimSpace(req.ID),
		Name:              strings.TrimSpace(req.Name),
		ProviderID:        strings.TrimSpace(req.ProviderID),
		ModelProvider:     stringutil.Default(req.ModelProvider, "openai"),
		ModelName:         strings.TrimSpace(req.ModelName),
		BaseURL:           strings.TrimSpace(req.BaseURL),
		Transport:         normalizeTransport(req.Transport),
		CommandArgsJSON:   jsonutil.MarshalStringSlice(jsonutil.CleanStringSlice(req.CommandArgs)),
		SystemPrompt:      strings.TrimSpace(req.SystemPrompt),
		MaxIterations:     req.MaxIterations,
		ToolWhitelistJSON: jsonutil.MarshalStringSlice(req.ToolWhitelist),
		NetworkAllowJSON:  jsonutil.MarshalStringSlice(req.NetworkAllow),
		MCPServerIDsJSON:  jsonutil.MarshalStringSlice(req.MCPServerIDs),
		SkillNamesJSON:    jsonutil.MarshalStringSlice(req.SkillNames),
		Enabled:           enabled,
	}
	if agent.ID == "" {
		agent.ID = "agent_" + id.Random()
	}
	if err := s.repo.Create(ctx, agent); err != nil {
		return nil, err
	}
	return toAgentDTO(agent), nil
}

func (s *AgentService) Get(ctx context.Context, id string) (*dto.AgentProfile, error) {
	agent, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return toAgentDTO(*agent), nil
}

func (s *AgentService) List(ctx context.Context) ([]dto.AgentProfile, error) {
	agents, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]dto.AgentProfile, len(agents))
	for i, agent := range agents {
		out[i] = *toAgentDTO(agent)
	}
	return out, nil
}

func (s *AgentService) Update(ctx context.Context, id string, req dto.UpdateAgentRequest) (*dto.AgentProfile, error) {
	agent, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		agent.Name = strings.TrimSpace(*req.Name)
	}
	if req.ProviderID != nil {
		agent.ProviderID = strings.TrimSpace(*req.ProviderID)
	}
	if req.ModelProvider != nil {
		agent.ModelProvider = stringutil.Default(*req.ModelProvider, "openai")
	}
	if req.ModelName != nil {
		agent.ModelName = strings.TrimSpace(*req.ModelName)
	}
	if req.BaseURL != nil {
		agent.BaseURL = strings.TrimSpace(*req.BaseURL)
	}
	if req.Transport != nil {
		agent.Transport = normalizeTransport(*req.Transport)
	}
	if req.CommandArgs != nil {
		agent.CommandArgsJSON = jsonutil.MarshalStringSlice(jsonutil.CleanStringSlice(req.CommandArgs))
	}
	if req.SystemPrompt != nil {
		agent.SystemPrompt = strings.TrimSpace(*req.SystemPrompt)
	}
	if req.MaxIterations != nil {
		agent.MaxIterations = *req.MaxIterations
	}
	if req.ToolWhitelist != nil {
		agent.ToolWhitelistJSON = jsonutil.MarshalStringSlice(req.ToolWhitelist)
	}
	if req.NetworkAllow != nil {
		agent.NetworkAllowJSON = jsonutil.MarshalStringSlice(req.NetworkAllow)
	}
	if req.MCPServerIDs != nil {
		agent.MCPServerIDsJSON = jsonutil.MarshalStringSlice(req.MCPServerIDs)
	}
	if req.SkillNames != nil {
		agent.SkillNamesJSON = jsonutil.MarshalStringSlice(req.SkillNames)
	}
	if req.Enabled != nil {
		agent.Enabled = *req.Enabled
	}
	if err := s.repo.Update(ctx, *agent); err != nil {
		return nil, err
	}
	return toAgentDTO(*agent), nil
}

func (s *AgentService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func toAgentDTO(agent model.AgentProfile) *dto.AgentProfile {
	return &dto.AgentProfile{
		ID:            agent.ID,
		Name:          agent.Name,
		ProviderID:    agent.ProviderID,
		ModelProvider: agent.ModelProvider,
		ModelName:     agent.ModelName,
		BaseURL:       agent.BaseURL,
		Transport:     normalizeTransport(agent.Transport),
		CommandArgs:   jsonutil.UnmarshalStringSlice(agent.CommandArgsJSON),
		SystemPrompt:  agent.SystemPrompt,
		MaxIterations: agent.MaxIterations,
		ToolWhitelist: jsonutil.UnmarshalStringSlice(agent.ToolWhitelistJSON),
		NetworkAllow:  jsonutil.UnmarshalStringSlice(agent.NetworkAllowJSON),
		MCPServerIDs:  jsonutil.UnmarshalStringSlice(agent.MCPServerIDsJSON),
		SkillNames:    jsonutil.UnmarshalStringSlice(agent.SkillNamesJSON),
		Enabled:       agent.Enabled,
		CreatedAt:     agent.CreatedAt,
		UpdatedAt:     agent.UpdatedAt,
	}
}
