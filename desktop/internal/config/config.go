package config

import "strings"

type Settings struct {
	Gateway   GatewaySettings   `json:"gateway" toml:"gateway"`
	Workspace WorkspaceSettings `json:"workspace" toml:"workspace"`
	UI        UISettings        `json:"ui" toml:"ui"`
}

type GatewaySettings struct {
	BaseURL        string `json:"baseUrl" toml:"base_url"`
	DefaultAgentID string `json:"defaultAgentId" toml:"default_agent_id"`
}

type WorkspaceSettings struct {
	RootDir string `json:"rootDir" toml:"root_dir"`
}

type UISettings struct {
	ShowTimestamps bool `json:"showTimestamps" toml:"show_timestamps"`
}

func DefaultSettings() Settings {
	return Settings{
		Gateway: GatewaySettings{
			BaseURL:        "http://127.0.0.1:8080",
			DefaultAgentID: "",
		},
		Workspace: WorkspaceSettings{},
		UI: UISettings{
			ShowTimestamps: true,
		},
	}
}

func (s Settings) Normalize() Settings {
	normalized := DefaultSettings()
	normalized.Gateway.BaseURL = normalizeBaseURL(s.Gateway.BaseURL)
	normalized.Gateway.DefaultAgentID = strings.TrimSpace(s.Gateway.DefaultAgentID)
	normalized.Workspace.RootDir = strings.TrimSpace(s.Workspace.RootDir)
	normalized.UI.ShowTimestamps = s.UI.ShowTimestamps
	return normalized
}

func normalizeBaseURL(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimRight(value, "/")
	if value == "" {
		return DefaultSettings().Gateway.BaseURL
	}
	return value
}
