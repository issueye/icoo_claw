package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

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
		ModelProvider:     defaultString(req.ModelProvider, "openai"),
		ModelName:         strings.TrimSpace(req.ModelName),
		BaseURL:           strings.TrimSpace(req.BaseURL),
		Transport:         normalizeTransport(req.Transport),
		SystemPrompt:      strings.TrimSpace(req.SystemPrompt),
		MaxIterations:     req.MaxIterations,
		ToolWhitelistJSON: mustJSON(req.ToolWhitelist),
		NetworkAllowJSON:  mustJSON(req.NetworkAllow),
		MCPServerIDsJSON:  mustJSON(req.MCPServerIDs),
		SkillIDsJSON:      mustJSON(req.SkillIDs),
		Enabled:           enabled,
	}
	if agent.ID == "" {
		agent.ID = "agent_" + randomID()
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
		agent.ModelProvider = defaultString(*req.ModelProvider, "openai")
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
	if req.SystemPrompt != nil {
		agent.SystemPrompt = strings.TrimSpace(*req.SystemPrompt)
	}
	if req.MaxIterations != nil {
		agent.MaxIterations = *req.MaxIterations
	}
	if req.ToolWhitelist != nil {
		agent.ToolWhitelistJSON = mustJSON(req.ToolWhitelist)
	}
	if req.NetworkAllow != nil {
		agent.NetworkAllowJSON = mustJSON(req.NetworkAllow)
	}
	if req.MCPServerIDs != nil {
		agent.MCPServerIDsJSON = mustJSON(req.MCPServerIDs)
	}
	if req.SkillIDs != nil {
		agent.SkillIDsJSON = mustJSON(req.SkillIDs)
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
		SystemPrompt:  agent.SystemPrompt,
		MaxIterations: agent.MaxIterations,
		ToolWhitelist: parseStringSlice(agent.ToolWhitelistJSON),
		NetworkAllow:  parseStringSlice(agent.NetworkAllowJSON),
		MCPServerIDs:  parseStringSlice(agent.MCPServerIDsJSON),
		SkillIDs:      parseStringSlice(agent.SkillIDsJSON),
		Enabled:       agent.Enabled,
		CreatedAt:     agent.CreatedAt,
		UpdatedAt:     agent.UpdatedAt,
	}
}

func mustJSON(values []string) string {
	if values == nil {
		values = []string{}
	}
	payload, _ := json.Marshal(values)
	return string(payload)
}

func parseStringSlice(raw string) []string {
	if raw == "" {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return []string{}
	}
	return out
}

func defaultString(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func randomID() string {
	var buf [12]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format("20060102150405.000000000")))
	}
	return hex.EncodeToString(buf[:])
}
