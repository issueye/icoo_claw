package model

import "time"

type AgentProfile struct {
	ID                string `gorm:"primaryKey;size:64"`
	Name              string `gorm:"size:128;not null"`
	ModelProvider     string `gorm:"size:32;not null"`
	ModelName         string `gorm:"size:128"`
	BaseURL           string `gorm:"size:512"`
	SystemPrompt      string `gorm:"type:text"`
	MaxIterations     int
	ToolWhitelistJSON string `gorm:"column:tool_whitelist;type:text"`
	NetworkAllowJSON  string `gorm:"column:network_allow;type:text"`
	MCPServerIDsJSON  string `gorm:"column:mcp_server_ids;type:text"`
	SkillIDsJSON      string `gorm:"column:skill_ids;type:text"`
	Enabled           bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
