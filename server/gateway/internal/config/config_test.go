package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gateway.toml")
	if err := os.WriteFile(path, []byte(`
http_addr = ":18080"
db_path = "./tmp/gateway.sqlite"
claw_binary_path = "./bin/claw.exe"
claw_work_dir = "."
claw_config_dir = "./tmp/claw_configs"
claw_port_start = 9101
claw_port_end = 9199
max_agent_instances = 8
health_interval_seconds = 3
shutdown_timeout_seconds = 4
session_api_url = "http://127.0.0.1:18080"
internal_token = "token"

[mqtt]
enabled = true
broker_url = "tcp://127.0.0.1:1883"
client_id = "gateway-test"
username = "user"
password = "pass"
topic_prefix = "icoo/test"
qos = 1
retained = true
connect_timeout_seconds = 2
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatalf("load file: %v", err)
	}
	wantDBPath := filepath.Join(filepath.Dir(path), "tmp", "gateway.sqlite")
	if cfg.HTTPAddr != ":18080" || cfg.DBPath != wantDBPath {
		t.Fatalf("cfg = %+v", cfg)
	}
	if cfg.ClawBinaryPath != filepath.Join(filepath.Dir(path), "bin", "claw.exe") {
		t.Fatalf("ClawBinaryPath = %q", cfg.ClawBinaryPath)
	}
	if cfg.ClawWorkDir != filepath.Dir(path) || cfg.ClawConfigDir != filepath.Join(filepath.Dir(path), "tmp", "claw_configs") {
		t.Fatalf("path config = %+v", cfg)
	}
	if cfg.GatewaySkillsRoot() != filepath.Join(cfg.GatewayWorkDir, "skills") {
		t.Fatalf("gateway skills root = %q, work dir = %q", cfg.GatewaySkillsRoot(), cfg.GatewayWorkDir)
	}
	if cfg.ClawPortStart != 9101 || cfg.ClawPortEnd != 9199 || cfg.MaxAgentInstances != 8 {
		t.Fatalf("ports/instances = %+v", cfg)
	}
	if cfg.HealthInterval != 3*time.Second || cfg.ShutdownTimeout != 4*time.Second {
		t.Fatalf("durations = %s %s", cfg.HealthInterval, cfg.ShutdownTimeout)
	}
	if cfg.SessionAPIURL != "http://127.0.0.1:18080" || cfg.InternalToken != "token" {
		t.Fatalf("remote config = %+v", cfg)
	}
	if !cfg.MQTT.Enabled || cfg.MQTT.BrokerURL != "tcp://127.0.0.1:1883" || cfg.MQTT.ClientID != "gateway-test" {
		t.Fatalf("mqtt config = %+v", cfg.MQTT)
	}
	if cfg.MQTT.Username != "user" || cfg.MQTT.Password != "pass" || cfg.MQTT.TopicPrefix != "icoo/test" {
		t.Fatalf("mqtt credentials/topic = %+v", cfg.MQTT)
	}
	if cfg.MQTT.QoS != 1 || !cfg.MQTT.Retained || cfg.MQTT.ConnectTimeout != 2*time.Second {
		t.Fatalf("mqtt publish config = %+v", cfg.MQTT)
	}
}

func TestLoadFileIgnoresRemovedGatewaySkillsDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gateway.toml")
	if err := os.WriteFile(path, []byte(`gateway_skills_dir = "./legacy_skills"`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatalf("load file: %v", err)
	}
	if strings.Contains(cfg.GatewaySkillsRoot(), "legacy_skills") {
		t.Fatalf("gateway skills root = %q, removed config should be ignored", cfg.GatewaySkillsRoot())
	}
}

func TestLoadFileDefaultsUseIcooRuntime(t *testing.T) {
	dir := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	}()

	cfg, err := LoadFile("")
	if err != nil {
		t.Fatalf("load defaults: %v", err)
	}
	runtimeRoot := filepath.Join(dir, "icoo_runtime")
	if cfg.DBPath != filepath.Join(runtimeRoot, "gateway.sqlite") {
		t.Fatalf("db path = %q", cfg.DBPath)
	}
	if cfg.ClawConfigDir != filepath.Join(runtimeRoot, "claw_configs") {
		t.Fatalf("claw config dir = %q", cfg.ClawConfigDir)
	}
	if cfg.ClawWorkDir != runtimeRoot {
		t.Fatalf("claw work dir = %q", cfg.ClawWorkDir)
	}
	if cfg.GatewaySkillsRoot() != filepath.Join(runtimeRoot, "skills") {
		t.Fatalf("gateway skills root = %q", cfg.GatewaySkillsRoot())
	}
}

func TestLoadFileLeavesBareExecutableOnPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gateway.toml")
	if err := os.WriteFile(path, []byte(`claw_binary_path = "claw"`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatalf("load file: %v", err)
	}
	if cfg.ClawBinaryPath != "claw" {
		t.Fatalf("ClawBinaryPath = %q, want bare executable unchanged", cfg.ClawBinaryPath)
	}
}
