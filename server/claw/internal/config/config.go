package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	HTTPAddr      string
	SessionAPIURL string
	InternalToken string
	RunnerMode    string
}

type fileConfig struct {
	HTTPAddr      string `toml:"http_addr"`
	SessionAPIURL string `toml:"session_api_url"`
	InternalToken string `toml:"internal_token"`
	RunnerMode    string `toml:"runner_mode"`
}

func Load() Config {
	cfg, err := LoadFile(configPath("config/claw.toml"))
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
		HTTPAddr:      ":8081",
		SessionAPIURL: "http://127.0.0.1:8080",
		RunnerMode:    "sdk",
	}
}
