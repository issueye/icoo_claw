package dto

import "time"

type SkillProfile struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Path        string         `json:"path"`
	Version     string         `json:"version"`
	Status      string         `json:"status"`
	Source      string         `json:"source,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type CreateSkillRequest struct {
	ID          string         `json:"id"`
	Name        string         `json:"name" binding:"required"`
	Description string         `json:"description" binding:"required"`
	Path        string         `json:"path" binding:"required"`
	Content     string         `json:"content"`
	Version     string         `json:"version"`
	Source      string         `json:"source"`
	Metadata    map[string]any `json:"metadata"`
}

type UpdateSkillRequest struct {
	Name        *string        `json:"name"`
	Description *string        `json:"description"`
	Path        *string        `json:"path"`
	Content     *string        `json:"content"`
	Version     *string        `json:"version"`
	Status      *string        `json:"status"`
	Source      *string        `json:"source"`
	Metadata    map[string]any `json:"metadata"`
}

type SkillSummary struct {
	Path   string             `json:"path"`
	Skills []SkillSummaryItem `json:"skills"`
}

type SkillSummaryItem struct {
	Name        string `json:"name"`
	Description string `json:"Description"`
	Version     string `json:"version"`
}
