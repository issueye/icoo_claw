package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"icoo_claw/common/runtimepath"

	"github.com/pelletier/go-toml/v2"
)

const DefaultConfigPath = runtimepath.DirName + "/config/claw.toml"

type Config struct {
	HTTPAddr           string
	SessionAPIURL      string
	InternalToken      string
	RunnerMode         string
	DefaultProjectRoot string
}

type fileConfig struct {
	HTTPAddr      string                  `toml:"http_addr"`
	SessionAPIURL string                  `toml:"session_api_url"`
	InternalToken string                  `toml:"internal_token"`
	RunnerMode    string                  `toml:"runner_mode"`
	GatewaySkills gatewaySkillsFileConfig `toml:"gateway_skills"`
}

type gatewaySkillsFileConfig struct {
	Path *string `toml:"path"`
}

func Load() Config {
	cfg, err := LoadFile(configPath(DefaultConfigPath))
	if err != nil {
		panic(err)
	}
	return cfg
}

func LoadFile(path string) (Config, error) {
	cfg := defaults()
	if path == "" {
		return cfg, nil
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}
	var file fileConfig
	if err := toml.Unmarshal(payload, &file); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}
	if file.HTTPAddr != "" {
		cfg.HTTPAddr = file.HTTPAddr
	}
	if file.SessionAPIURL != "" {
		cfg.SessionAPIURL = file.SessionAPIURL
	}
	if file.InternalToken != "" {
		cfg.InternalToken = file.InternalToken
	}
	if file.RunnerMode != "" {
		cfg.RunnerMode = file.RunnerMode
	}
	if file.GatewaySkills.Path != nil {
		cfg.DefaultProjectRoot = *file.GatewaySkills.Path
	}
	resolveRelativePaths(&cfg, path)
	return cfg, nil
}

func configPath(fallback string) string {
	args := os.Args[1:]
	for i, arg := range args {
		if arg == "--config" && i+1 < len(args) {
			return args[i+1]
		}
		if value, ok := strings.CutPrefix(arg, "--config="); ok {
			return value
		}
	}
	return fallback
}

func defaults() Config {
	return Config{
		HTTPAddr:           ":8081",
		SessionAPIURL:      "http://127.0.0.1:8080",
		RunnerMode:         "sdk",
		DefaultProjectRoot: filepath.Join(runtimepath.Root(), "skills"),
	}
}

func resolveRelativePaths(cfg *Config, configFilePath string) {
	cfg.DefaultProjectRoot = resolveConfigDataPath(cfg.DefaultProjectRoot, configBaseDir(configFilePath))
}

func configBaseDir(configFilePath string) string {
	if configFilePath == "" {
		return ""
	}
	abs, err := filepath.Abs(configFilePath)
	if err != nil {
		return filepath.Dir(configFilePath)
	}
	return filepath.Dir(abs)
}

func resolveConfigDataPath(value string, baseDir string) string {
	value = strings.TrimSpace(value)
	if value == "" || baseDir == "" {
		return value
	}
	if filepath.IsAbs(value) {
		return filepath.Clean(value)
	}
	return filepath.Clean(filepath.Join(baseDir, value))
}
