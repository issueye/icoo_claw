package service

import (
	"context"
	"strings"

	"icoo_claw/server/gateway/internal/dto"
	"icoo_claw/server/gateway/internal/model"
	"icoo_claw/server/gateway/internal/repository"
)

type ProviderService struct {
	repo repository.ProviderRepository
}

func NewProviderService(repo repository.ProviderRepository) *ProviderService {
	return &ProviderService{repo: repo}
}

func (s *ProviderService) Create(ctx context.Context, req dto.CreateProviderRequest) (*dto.ProviderProfile, error) {
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	provider := model.ProviderProfile{
		ID:           strings.TrimSpace(req.ID),
		Name:         strings.TrimSpace(req.Name),
		Type:         normalizeProviderType(req.Type),
		BaseURL:      strings.TrimSpace(req.BaseURL),
		APIKey:       strings.TrimSpace(req.APIKey),
		DefaultModel: strings.TrimSpace(req.DefaultModel),
		Enabled:      enabled,
	}
	if provider.ID == "" {
		provider.ID = provider.Type + "_" + randomID()
	}
	if err := s.repo.Create(ctx, provider); err != nil {
		return nil, err
	}
	return toProviderDTO(provider), nil
}

func (s *ProviderService) Get(ctx context.Context, id string) (*dto.ProviderProfile, error) {
	provider, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return toProviderDTO(*provider), nil
}

func (s *ProviderService) List(ctx context.Context) ([]dto.ProviderProfile, error) {
	providers, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]dto.ProviderProfile, len(providers))
	for i, provider := range providers {
		out[i] = *toProviderDTO(provider)
	}
	return out, nil
}

func (s *ProviderService) Update(ctx context.Context, id string, req dto.UpdateProviderRequest) (*dto.ProviderProfile, error) {
	provider, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		provider.Name = strings.TrimSpace(*req.Name)
	}
	if req.Type != nil {
		provider.Type = normalizeProviderType(*req.Type)
	}
	if req.BaseURL != nil {
		provider.BaseURL = strings.TrimSpace(*req.BaseURL)
	}
	if req.APIKey != nil {
		provider.APIKey = strings.TrimSpace(*req.APIKey)
	}
	if req.DefaultModel != nil {
		provider.DefaultModel = strings.TrimSpace(*req.DefaultModel)
	}
	if req.Enabled != nil {
		provider.Enabled = *req.Enabled
	}
	if err := s.repo.Update(ctx, *provider); err != nil {
		return nil, err
	}
	return toProviderDTO(*provider), nil
}

func (s *ProviderService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func toProviderDTO(provider model.ProviderProfile) *dto.ProviderProfile {
	return &dto.ProviderProfile{
		ID:            provider.ID,
		Name:          provider.Name,
		Type:          provider.Type,
		BaseURL:       provider.BaseURL,
		DefaultModel:  provider.DefaultModel,
		Enabled:       provider.Enabled,
		APIKeySet:     provider.APIKey != "",
		APIKeyPreview: previewSecret(provider.APIKey),
		CreatedAt:     provider.CreatedAt,
		UpdatedAt:     provider.UpdatedAt,
	}
}

func normalizeProviderType(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func previewSecret(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= 8 {
		return "****"
	}
	return value[:4] + "..." + value[len(value)-4:]
}
