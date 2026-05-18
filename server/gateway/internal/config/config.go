package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	HTTPAddr          string
	DBPath            string
	ClawBinaryPath    string
	ClawWorkDir       string
	ClawConfigDir     string
	ClawRunnerMode    string
	ClawPortStart     int
	ClawPortEnd       int
	MaxAgentInstances int
	HealthInterval    time.Duration
	ShutdownTimeout   time.Duration
	SessionStoreURL   string
	InternalToken     string
}

type fileConfig struct {
	HTTPAddr           string `toml:"http_addr"`
	DBPath             string `toml:"db_path"`
	ClawBinaryPath     string `toml:"claw_binary_path"`
	ClawWorkDir        string `toml:"claw_work_dir"`
	ClawConfigDir      string `toml:"claw_config_dir"`
	ClawRunnerMode     string `toml:"claw_runner_mode"`
	ClawPortStart      int    `toml:"claw_port_start"`
	ClawPortEnd        int    `toml:"claw_port_end"`
	MaxAgentInstances  int    `toml:"max_agent_instances"`
	HealthIntervalSec  int    `toml:"health_interval_seconds"`
	ShutdownTimeoutSec int    `toml:"shutdown_timeout_seconds"`
	SessionStoreURL    string `toml:"session_store_url"`
	InternalToken      string `toml:"internal_token"`
}

func Load() Config {
	cfg, err := LoadFile(configPath("config/gateway.toml"))
	if err != nil {
		panic(err)
	}
	applyEnv(&cfg)
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
	applyFile(&cfg, file)
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
		HTTPAddr:          ":8080",
		DBPath:            "gateway.sqlite",
		ClawConfigDir:     "data/claw_configs",
		ClawRunnerMode:    "sdk",
		ClawPortStart:     8101,
		ClawPortEnd:       8199,
		MaxAgentInstances: 4,
		HealthInterval:    10 * time.Second,
		ShutdownTimeout:   10 * time.Second,
		SessionStoreURL:   "http://127.0.0.1:8082",
	}
}

func applyFile(cfg *Config, file fileConfig) {
	if file.HTTPAddr != "" {
		cfg.HTTPAddr = file.HTTPAddr
	}
	if file.DBPath != "" {
		cfg.DBPath = file.DBPath
	}
	if file.ClawBinaryPath != "" {
		cfg.ClawBinaryPath = file.ClawBinaryPath
	}
	if file.ClawWorkDir != "" {
		cfg.ClawWorkDir = file.ClawWorkDir
	}
	if file.ClawConfigDir != "" {
		cfg.ClawConfigDir = file.ClawConfigDir
	}
	if file.ClawRunnerMode != "" {
		cfg.ClawRunnerMode = file.ClawRunnerMode
	}
	if file.ClawPortStart != 0 {
		cfg.ClawPortStart = file.ClawPortStart
	}
	if file.ClawPortEnd != 0 {
		cfg.ClawPortEnd = file.ClawPortEnd
	}
	if file.MaxAgentInstances != 0 {
		cfg.MaxAgentInstances = file.MaxAgentInstances
	}
	if file.HealthIntervalSec > 0 {
		cfg.HealthInterval = time.Duration(file.HealthIntervalSec) * time.Second
	}
	if file.ShutdownTimeoutSec > 0 {
		cfg.ShutdownTimeout = time.Duration(file.ShutdownTimeoutSec) * time.Second
	}
	if file.SessionStoreURL != "" {
		cfg.SessionStoreURL = file.SessionStoreURL
	}
	if file.InternalToken != "" {
		cfg.InternalToken = file.InternalToken
	}
}

func applyEnv(cfg *Config) {
	if value := os.Getenv("GATEWAY_HTTP_ADDR"); value != "" {
		cfg.HTTPAddr = value
	}
	if value := os.Getenv("GATEWAY_DB_PATH"); value != "" {
		cfg.DBPath = value
	}
	if value := os.Getenv("CLAW_BINARY_PATH"); value != "" {
		cfg.ClawBinaryPath = value
	}
	if value := os.Getenv("CLAW_WORK_DIR"); value != "" {
		cfg.ClawWorkDir = value
	}
	if value := os.Getenv("CLAW_CONFIG_DIR"); value != "" {
		cfg.ClawConfigDir = value
	}
	if value := os.Getenv("CLAW_RUNNER_MODE"); value != "" {
		cfg.ClawRunnerMode = value
	}
	if value := envInt("CLAW_PORT_START"); value > 0 {
		cfg.ClawPortStart = value
	}
	if value := envInt("CLAW_PORT_END"); value > 0 {
		cfg.ClawPortEnd = value
	}
	if value := envInt("MAX_AGENT_INSTANCES"); value > 0 {
		cfg.MaxAgentInstances = value
	}
	if value := envInt("AGENT_HEALTH_INTERVAL_SECONDS"); value > 0 {
		cfg.HealthInterval = time.Duration(value) * time.Second
	}
	if value := envInt("AGENT_SHUTDOWN_TIMEOUT_SECONDS"); value > 0 {
		cfg.ShutdownTimeout = time.Duration(value) * time.Second
	}
	if value := os.Getenv("SESSION_STORE_URL"); value != "" {
		cfg.SessionStoreURL = value
	}
	if value := os.Getenv("INTERNAL_TOKEN"); value != "" {
		cfg.InternalToken = value
	}
}

func envInt(key string) int {
	value := os.Getenv(key)
	if value == "" {
		return 0
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return parsed
}
