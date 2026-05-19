package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultGatewayBaseURL = "http://127.0.0.1:8080"
	defaultGatewayPort    = 8080
	defaultSessionPort    = 8082
	defaultClawPortStart  = 8101
	defaultClawPortEnd    = 8108
	defaultInternalToken  = "dev-internal-token"
	defaultAgentID        = "agent_desktop_default"
)

type BundledGatewayManager struct {
	mu sync.Mutex
}

func NewBundledGatewayManager() *BundledGatewayManager {
	return &BundledGatewayManager{}
}

func (m *BundledGatewayManager) EnsureBundledGateway(baseURL, programPath, configPath string) (bool, error) {
	target := normalizeGatewayBaseURL(baseURL)
	appendBootstrapLog("", "ensure start", map[string]string{"base_url": target})
	if !isLocalGatewayBaseURL(target) {
		appendBootstrapLog("", "skip non-local base url", map[string]string{"base_url": target})
		return false, nil
	}
	if healthCheck(target) == nil {
		appendBootstrapLog("", "skip healthy gateway", map[string]string{"base_url": target})
		return false, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if healthCheck(target) == nil {
		appendBootstrapLog("", "skip healthy gateway after lock", map[string]string{"base_url": target})
		return false, nil
	}

	programPath = strings.TrimSpace(programPath)
	configPath = strings.TrimSpace(configPath)
	if programPath != "" {
		appendBootstrapLog("", "attempt custom gateway start", map[string]string{"program_path": programPath, "config_path": configPath})
		return ensureCustomGateway(target, programPath, configPath)
	}

	packageRoot, err := detectBundledPackageRoot()
	if err != nil {
		appendBootstrapLog("", "detect package root failed", map[string]string{"error": err.Error()})
		return false, err
	}
	if packageRoot == "" {
		appendBootstrapLog("", "skip no bundled package root", map[string]string{"base_url": target})
		return false, nil
	}
	appendBootstrapLog(packageRoot, "detected bundled package root", map[string]string{"package_root": packageRoot})

	cfg, err := buildRuntimeConfig(packageRoot, target)
	if err != nil {
		appendBootstrapLog(packageRoot, "build runtime config failed", map[string]string{"error": err.Error()})
		return false, err
	}
	if cfg == nil {
		appendBootstrapLog(packageRoot, "skip missing bundled binaries", nil)
		return false, nil
	}
	if err := cfg.prepareDirectories(); err != nil {
		appendBootstrapLog(packageRoot, "prepare directories failed", map[string]string{"error": err.Error()})
		return false, err
	}
	if err := cfg.stopManagedProcesses(); err != nil {
		appendBootstrapLog(packageRoot, "stop managed processes failed", map[string]string{"error": err.Error()})
		return false, err
	}
	if err := cfg.writeConfigFiles(); err != nil {
		appendBootstrapLog(packageRoot, "write config files failed", map[string]string{"error": err.Error()})
		return false, err
	}
	if err := cfg.startProcess("session_store", cfg.sessionBinaryPath, []string{"--config", cfg.sessionConfigPath}); err != nil {
		appendBootstrapLog(packageRoot, "start session_store failed", map[string]string{"error": err.Error()})
		return false, err
	}
	if err := cfg.startProcess("gateway", cfg.gatewayBinaryPath, []string{"--config", cfg.gatewayConfigPath}); err != nil {
		appendBootstrapLog(packageRoot, "start gateway failed", map[string]string{"error": err.Error()})
		return false, err
	}
	if err := waitForHealthy(target, 45*time.Second); err != nil {
		appendBootstrapLog(packageRoot, "wait healthy failed", map[string]string{"error": err.Error()})
		return false, err
	}
	if err := ensureDefaultAgent(target); err != nil {
		appendBootstrapLog(packageRoot, "ensure default agent failed", map[string]string{"error": err.Error()})
		return false, err
	}
	appendBootstrapLog(packageRoot, "ensure bundled gateway success", map[string]string{"base_url": target})
	return true, nil
}

type customGatewayRuntime struct {
	runtimeRoot string
	logDir      string
	runDir      string
	programPath string
	configPath  string
}

func ensureCustomGateway(baseURL, programPath, configPath string) (bool, error) {
	if _, err := os.Stat(programPath); err != nil {
		return false, fmt.Errorf("gateway program path is invalid: %w", err)
	}
	if configPath != "" {
		if _, err := os.Stat(configPath); err != nil {
			return false, fmt.Errorf("gateway config path is invalid: %w", err)
		}
	}

	runtimeRoot, err := managedRuntimeRoot()
	if err != nil {
		return false, err
	}
	rt := &customGatewayRuntime{
		runtimeRoot: runtimeRoot,
		logDir:      filepath.Join(runtimeRoot, "logs"),
		runDir:      filepath.Join(runtimeRoot, "run"),
		programPath: programPath,
		configPath:  configPath,
	}

	if err := rt.prepareDirectories(); err != nil {
		return false, err
	}
	if err := rt.stopManagedProcess(); err != nil {
		return false, err
	}
	args := []string{}
	if configPath != "" {
		args = []string{"--config", configPath}
	}
	if err := rt.startProcess("gateway", programPath, args); err != nil {
		return false, err
	}
	if err := waitForHealthy(baseURL, 45*time.Second); err != nil {
		return false, err
	}
	if err := ensureDefaultAgent(baseURL); err != nil {
		return false, err
	}
	return true, nil
}

type runtimeConfig struct {
	packageRoot      string
	binDir           string
	runtimeRoot      string
	configDir        string
	dataDir          string
	logDir           string
	runDir           string
	gatewayBaseURL   string
	gatewayPort      int
	sessionPort      int
	clawPortStart    int
	clawPortEnd      int
	internalToken    string
	agentID          string
	sessionBinaryPath string
	gatewayBinaryPath string
	clawBinaryPath    string
	sessionConfigPath string
	gatewayConfigPath string
}

func buildRuntimeConfig(packageRoot, baseURL string) (*runtimeConfig, error) {
	gatewayPort, err := gatewayPortFromBaseURL(baseURL)
	if err != nil {
		return nil, err
	}

	binDir := filepath.Join(packageRoot, "bin")
	cfg := &runtimeConfig{
		packageRoot:       packageRoot,
		binDir:            binDir,
		runtimeRoot:       filepath.Join(packageRoot, "runtime"),
		configDir:         filepath.Join(packageRoot, "runtime", "config"),
		dataDir:           filepath.Join(packageRoot, "runtime", "data"),
		logDir:            filepath.Join(packageRoot, "runtime", "logs"),
		runDir:            filepath.Join(packageRoot, "runtime", "run"),
		gatewayBaseURL:    normalizeGatewayBaseURL(baseURL),
		gatewayPort:       gatewayPort,
		sessionPort:       defaultSessionPort,
		clawPortStart:     defaultClawPortStart,
		clawPortEnd:       defaultClawPortEnd,
		internalToken:     defaultInternalToken,
		agentID:           defaultAgentID,
		sessionBinaryPath: filepath.Join(binDir, "session_store.exe"),
		gatewayBinaryPath: filepath.Join(binDir, "gateway.exe"),
		clawBinaryPath:    filepath.Join(binDir, "claw.exe"),
		sessionConfigPath: filepath.Join(packageRoot, "runtime", "config", "session_store.toml"),
		gatewayConfigPath: filepath.Join(packageRoot, "runtime", "config", "gateway.toml"),
	}

	for _, path := range []string{cfg.sessionBinaryPath, cfg.gatewayBinaryPath, cfg.clawBinaryPath} {
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			return nil, nil
		} else if err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

func managedRuntimeRoot() (string, error) {
	baseDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(baseDir, appSlug, "runtime"), nil
}

func (c *runtimeConfig) prepareDirectories() error {
	for _, path := range []string{c.binDir, c.configDir, c.dataDir, c.logDir, c.runDir} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func (c *runtimeConfig) stopManagedProcesses() error {
	if err := stopProcessByPIDFile(filepath.Join(c.runDir, "gateway.pid")); err != nil {
		return err
	}
	if err := stopProcessByPIDFile(filepath.Join(c.runDir, "session_store.pid")); err != nil {
		return err
	}
	return stopProcessesByPath("claw", c.clawBinaryPath)
}

func (c *runtimeConfig) writeConfigFiles() error {
	sessionConfig := fmt.Sprintf("http_addr = \"127.0.0.1:%d\"\ndb_path = \"%s\"\n",
		c.sessionPort,
		tomlPath(filepath.Join(c.dataDir, "session_store.sqlite")),
	)
	if err := writeUTF8NoBOM(c.sessionConfigPath, sessionConfig); err != nil {
		return err
	}

	gatewayConfig := strings.Join([]string{
		fmt.Sprintf("http_addr = \"127.0.0.1:%d\"", c.gatewayPort),
		fmt.Sprintf("db_path = \"%s\"", tomlPath(filepath.Join(c.dataDir, "gateway.sqlite"))),
		fmt.Sprintf("session_store_url = \"http://127.0.0.1:%d\"", c.sessionPort),
		fmt.Sprintf("internal_token = \"%s\"", c.internalToken),
		fmt.Sprintf("claw_binary_path = \"%s\"", tomlPath(c.clawBinaryPath)),
		fmt.Sprintf("claw_work_dir = \"%s\"", tomlPath(c.packageRoot)),
		fmt.Sprintf("claw_config_dir = \"%s\"", tomlPath(filepath.Join(c.dataDir, "claw-configs"))),
		"claw_runner_mode = \"fake\"",
		fmt.Sprintf("claw_port_start = %d", c.clawPortStart),
		fmt.Sprintf("claw_port_end = %d", c.clawPortEnd),
		"max_agent_instances = 2",
		"health_interval_seconds = 1",
		"shutdown_timeout_seconds = 2",
		"",
	}, "\n")

	return writeUTF8NoBOM(c.gatewayConfigPath, gatewayConfig)
}

func (c *runtimeConfig) startProcess(name, binaryPath string, args []string) error {
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = c.packageRoot
	configureBackgroundCommand(cmd)

	stdout, err := os.OpenFile(filepath.Join(c.logDir, name+".out.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer stdout.Close()

	stderr, err := os.OpenFile(filepath.Join(c.logDir, name+".err.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer stderr.Close()

	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(c.runDir, name+".pid"), []byte(strconv.Itoa(cmd.Process.Pid)), 0o644)
}

func (c *customGatewayRuntime) prepareDirectories() error {
	for _, path := range []string{c.runtimeRoot, c.logDir, c.runDir} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func (c *customGatewayRuntime) stopManagedProcess() error {
	return stopProcessByPIDFile(filepath.Join(c.runDir, "gateway.pid"))
}

func (c *customGatewayRuntime) startProcess(name, binaryPath string, args []string) error {
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = filepath.Dir(binaryPath)
	configureBackgroundCommand(cmd)

	stdout, err := os.OpenFile(filepath.Join(c.logDir, name+".out.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer stdout.Close()

	stderr, err := os.OpenFile(filepath.Join(c.logDir, name+".err.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer stderr.Close()

	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(c.runDir, name+".pid"), []byte(strconv.Itoa(cmd.Process.Pid)), 0o644)
}

func detectBundledPackageRoot() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	exeDir := filepath.Dir(exePath)
	packageRoot := filepath.Dir(exeDir)
	if !strings.EqualFold(filepath.Base(exeDir), "bin") {
		return "", nil
	}
	if _, err := os.Stat(filepath.Join(packageRoot, "bin", "gateway.exe")); errors.Is(err, os.ErrNotExist) {
		return "", nil
	} else if err != nil {
		return "", err
	}
	return packageRoot, nil
}

func ensureDefaultAgent(baseURL string) error {
	body := map[string]any{
		"id":             defaultAgentID,
		"name":           "Desktop Default Agent",
		"model_provider": "openai",
		"model_name":     "fake",
		"max_iterations": 1,
		"tool_whitelist": []string{},
		"enabled":        true,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, normalizeGatewayBaseURL(baseURL)+"/v1/agents", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient().Do(req)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusConflict {
			return nil
		}
	}

	getReq, err := http.NewRequestWithContext(context.Background(), http.MethodGet, normalizeGatewayBaseURL(baseURL)+"/v1/agents/"+defaultAgentID, nil)
	if err != nil {
		return err
	}
	getResp, err := httpClient().Do(getReq)
	if err != nil {
		return err
	}
	defer getResp.Body.Close()
	if getResp.StatusCode == http.StatusOK {
		return nil
	}

	if resp != nil {
		return fmt.Errorf("failed to ensure default agent: %s", resp.Status)
	}
	return fmt.Errorf("failed to ensure default agent: %w", err)
}

func waitForHealthy(baseURL string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := healthCheck(baseURL); err == nil {
			return nil
		}
		time.Sleep(300 * time.Millisecond)
	}
	return fmt.Errorf("gateway did not become healthy: %s", baseURL)
}

func healthCheck(baseURL string) error {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, normalizeGatewayBaseURL(baseURL)+"/health", nil)
	if err != nil {
		return err
	}
	resp, err := httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	return nil
}

func httpClient() *http.Client {
	return &http.Client{Timeout: 2 * time.Second}
}

func normalizeGatewayBaseURL(value string) string {
	value = strings.TrimSpace(strings.TrimRight(value, "/"))
	if value == "" {
		return defaultGatewayBaseURL
	}
	return value
}

func isLocalGatewayBaseURL(value string) bool {
	parsed, err := url.Parse(normalizeGatewayBaseURL(value))
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "127.0.0.1" || host == "localhost"
}

func gatewayPortFromBaseURL(value string) (int, error) {
	parsed, err := url.Parse(normalizeGatewayBaseURL(value))
	if err != nil {
		return 0, err
	}
	if port := parsed.Port(); port != "" {
		return strconv.Atoi(port)
	}
	return defaultGatewayPort, nil
}

func stopProcessByPIDFile(pidPath string) error {
	data, err := os.ReadFile(pidPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		_ = os.Remove(pidPath)
		return nil
	}
	process, err := os.FindProcess(pid)
	if err == nil {
		_ = process.Kill()
	}
	_ = os.Remove(pidPath)
	return nil
}

func stopProcessesByPath(processName, expectedPath string) error {
	cmd := exec.Command("powershell", "-NoProfile", "-Command",
		fmt.Sprintf(`$ErrorActionPreference = 'SilentlyContinue'; Get-Process %s | Where-Object { $_.Path -eq '%s' } | ForEach-Object { Stop-Process -Id $_.Id -Force }; exit 0`,
			processName,
			strings.ReplaceAll(expectedPath, "'", "''"),
		),
	)
	configureBackgroundCommand(cmd)
	return cmd.Run()
}

func writeUTF8NoBOM(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func tomlPath(path string) string {
	return filepath.ToSlash(path)
}

func appendBootstrapLog(packageRoot, message string, fields map[string]string) {
	root := packageRoot
	if root == "" {
		if detected, err := detectBundledPackageRoot(); err == nil && detected != "" {
			root = detected
		}
	}
	if root == "" {
		return
	}

	logDir := filepath.Join(root, "runtime", "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return
	}

	var builder strings.Builder
	builder.WriteString(time.Now().Format(time.RFC3339))
	builder.WriteString(" ")
	builder.WriteString(message)
	for key, value := range fields {
		builder.WriteString(" ")
		builder.WriteString(key)
		builder.WriteString("=")
		builder.WriteString(value)
	}
	builder.WriteString("\n")

	file, err := os.OpenFile(filepath.Join(logDir, "bootstrap.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer file.Close()
	_, _ = file.WriteString(builder.String())
}
