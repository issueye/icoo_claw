package config

import "os"

type Config struct {
	HTTPAddr        string
	SessionStoreURL string
	InternalToken   string
}

func Load() Config {
	return Config{
		HTTPAddr:        env("CLAW_HTTP_ADDR", ":8081"),
		SessionStoreURL: env("SESSION_STORE_URL", "http://127.0.0.1:8082"),
		InternalToken:   os.Getenv("INTERNAL_TOKEN"),
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
