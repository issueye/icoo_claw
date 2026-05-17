package config

import "os"

type Config struct {
	HTTPAddr string
	RESPAddr string
	DBPath   string
}

func Load() Config {
	return Config{
		HTTPAddr: env("SESSION_STORE_HTTP_ADDR", ":8082"),
		RESPAddr: env("SESSION_STORE_RESP_ADDR", ":6380"),
		DBPath:   env("SESSION_STORE_DB_PATH", "session_store.sqlite"),
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
