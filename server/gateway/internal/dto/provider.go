package dto

import "time"

type ProviderProfile struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Type          string    `json:"type"`
	BaseURL       string    `json:"base_url,omitempty"`
	DefaultModel  string    `json:"default_model,omitempty"`
	Enabled       bool      `json:"enabled"`
	APIKeySet     bool      `json:"api_key_set"`
	APIKeyPreview string    `json:"api_key_preview,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CreateProviderRequest struct {
	ID           string `json:"id"`
	Name         string `json:"name" binding:"required"`
	Type         string `json:"type" binding:"required"`
	BaseURL      string `json:"base_url"`
	APIKey       string `json:"api_key"`
	DefaultModel string `json:"default_model"`
	Enabled      *bool  `json:"enabled"`
}

type UpdateProviderRequest struct {
	Name         *string `json:"name"`
	Type         *string `json:"type"`
	BaseURL      *string `json:"base_url"`
	APIKey       *string `json:"api_key"`
	DefaultModel *string `json:"default_model"`
	Enabled      *bool   `json:"enabled"`
}
