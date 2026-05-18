package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	HTTPAddr string
	DBPath   string
}

type fileConfig struct {
	HTTPAddr string `toml:"http_addr"`
	DBPath   string `toml:"db_path"`
}

func Load() Config {
	cfg, err := LoadFile(configPath("config/session_store.toml"))
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
	if file.HTTPAddr != "" {
		cfg.HTTPAddr = file.HTTPAddr
	}
	if file.DBPath != "" {
		cfg.DBPath = file.DBPath
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
		HTTPAddr: ":8082",
		DBPath:   "session_store.sqlite",
	}
}

func applyEnv(cfg *Config) {
	if value := os.Getenv("SESSION_STORE_HTTP_ADDR"); value != "" {
		cfg.HTTPAddr = value
	}
	if value := os.Getenv("SESSION_STORE_DB_PATH"); value != "" {
		cfg.DBPath = value
	}
}
