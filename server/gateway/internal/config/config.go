package config

import "os"
import "strconv"

type Config struct {
	HTTPAddr          string
	DBPath            string
	ClawBinaryPath    string
	ClawWorkDir       string
	ClawPortStart     int
	ClawPortEnd       int
	MaxAgentInstances int
	SessionStoreURL   string
	InternalToken     string
}

func Load() Config {
	return Config{
		HTTPAddr:          env("GATEWAY_HTTP_ADDR", ":8080"),
		DBPath:            env("GATEWAY_DB_PATH", "gateway.sqlite"),
		ClawBinaryPath:    env("CLAW_BINARY_PATH", ""),
		ClawWorkDir:       env("CLAW_WORK_DIR", ""),
		ClawPortStart:     envInt("CLAW_PORT_START", 8101),
		ClawPortEnd:       envInt("CLAW_PORT_END", 8199),
		MaxAgentInstances: envInt("MAX_AGENT_INSTANCES", 4),
		SessionStoreURL:   env("SESSION_STORE_URL", "http://127.0.0.1:8082"),
		InternalToken:     os.Getenv("INTERNAL_TOKEN"),
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
