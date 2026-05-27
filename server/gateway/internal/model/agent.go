package model

import "time"

type AgentProfile struct {
	ID                string `gorm:"primaryKey;size:64"`
	Name              string `gorm:"size:128;not null"`
	ProviderID        string `gorm:"size:64;index"`
	ModelProvider     string `gorm:"size:32;not null"`
	ModelName         string `gorm:"size:128"`
	BaseURL           string `gorm:"size:512"`
	Transport         string `gorm:"size:32;not null;default:http"`
	CommandArgsJSON   string `gorm:"column:command_args;type:text"`
	SystemPrompt      string `gorm:"type:text"`
	MaxIterations     int
	ToolWhitelistJSON string `gorm:"column:tool_whitelist;type:text"`
	NetworkAllowJSON  string `gorm:"column:network_allow;type:text"`
	MCPServerIDsJSON  string `gorm:"column:mcp_server_ids;type:text"`
	SkillNamesJSON    string `gorm:"column:skill_names;type:text"`
	Enabled           bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
