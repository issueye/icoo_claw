package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"icoo_claw/server/gateway/internal/config"
	"icoo_claw/server/gateway/internal/model"
)

type StartAgentInstanceSpec struct {
	InstanceID    string
	AgentID       string
	Host          string
	Port          int
	BaseURL       string
	BinaryPath    string
	WorkDir       string
	SessionAPIURL string
	InternalToken string
	ConfigDir     string
	RunnerMode    string
	Agent         AgentLaunchConfig
}

type AgentLaunchConfig struct {
	ProviderID    string
	ModelProvider string
	ModelName     string
	APIKey        string
	BaseURL       string
}

type AgentProcess struct {
	PID int
}

type ProcessSupervisor interface {
	Start(ctx context.Context, spec StartAgentInstanceSpec) (*AgentProcess, error)
	Stop(ctx context.Context, instance model.AgentInstance) error
	Probe(ctx context.Context, instance model.AgentInstance) error
}

type LocalProcessSupervisor struct {
	httpClient *http.Client
}

func NewLocalProcessSupervisor() *LocalProcessSupervisor {
	return &LocalProcessSupervisor{httpClient: &http.Client{Timeout: 2 * time.Second}}
}

func (s *LocalProcessSupervisor) Start(ctx context.Context, spec StartAgentInstanceSpec) (*AgentProcess, error) {
	if spec.BinaryPath == "" {
		return nil, errors.New("claw binary path is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	binaryPath, err := resolveExecutablePath(spec.BinaryPath, spec.WorkDir)
	if err != nil {
		return nil, err
	}
	configPath, err := writeClawConfig(spec)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(binaryPath, "--config", configPath)
	if spec.WorkDir != "" {
		cmd.Dir = spec.WorkDir
	}
	cmd.Env = append(os.Environ(), agentLaunchEnv(spec.Agent)...)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start claw process: %w", err)
	}
	go func() { _ = cmd.Wait() }()
	return &AgentProcess{PID: cmd.Process.Pid}, nil
}

func (s *LocalProcessSupervisor) Stop(ctx context.Context, instance model.AgentInstance) error {
	process, err := os.FindProcess(instance.PID)
	if err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- process.Kill() }()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}

func (s *LocalProcessSupervisor) Probe(ctx context.Context, instance model.AgentInstance) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, instance.BaseURL+"/health", nil)
	if err != nil {
		return err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health status %d", resp.StatusCode)
	}
	return nil
}

func resolveExecutablePath(path string, workDir string) (string, error) {
	if filepath.IsAbs(path) {
		return path, nil
	}

	candidates := []string{path}
	if executable, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(executable), path))
	}
	if workDir != "" {
		candidates = append(candidates, filepath.Join(workDir, path))
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(cwd, path))
	}

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		abs, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		if info, err := os.Stat(abs); err == nil && !info.IsDir() {
			return abs, nil
		}
	}

	found, err := exec.LookPath(path)
	if err != nil {
		return "", err
	}
	return filepath.Abs(found)
}

func processSpecFromConfig(cfg config.Config, instanceID, agentID string, port int) StartAgentInstanceSpec {
	host := "127.0.0.1"
	baseURL := "http://" + host + ":" + strconv.Itoa(port)
	return StartAgentInstanceSpec{
		InstanceID:    instanceID,
		AgentID:       agentID,
		Host:          host,
		Port:          port,
		BaseURL:       baseURL,
		BinaryPath:    cfg.ClawBinaryPath,
		WorkDir:       cfg.ClawWorkDir,
		SessionAPIURL: cfg.SessionAPIURL,
		InternalToken: cfg.InternalToken,
		ConfigDir:     cfg.ClawConfigDir,
		RunnerMode:    cfg.ClawRunnerMode,
	}
}

func writeClawConfig(spec StartAgentInstanceSpec) (string, error) {
	dir := spec.ConfigDir
	if dir == "" {
		dir = filepath.Join(os.TempDir(), "icoo_claw")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create claw config dir: %w", err)
	}
	path := filepath.Join(dir, spec.InstanceID+".toml")
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve claw config path: %w", err)
	}
	payload := fmt.Sprintf(
		"http_addr = %q\nsession_api_url = %q\ninternal_token = %q\nrunner_mode = %q\n\n[agent]\nagent_id = %q\nprovider_id = %q\nmodel_provider = %q\nmodel_name = %q\nbase_url = %q\napi_key_set = %t\n",
		spec.Host+":"+strconv.Itoa(spec.Port),
		spec.SessionAPIURL,
		spec.InternalToken,
		spec.RunnerMode,
		spec.AgentID,
		spec.Agent.ProviderID,
		spec.Agent.ModelProvider,
		spec.Agent.ModelName,
		spec.Agent.BaseURL,
		spec.Agent.APIKey != "",
	)
	if err := os.WriteFile(absPath, []byte(payload), 0o600); err != nil {
		return "", fmt.Errorf("write claw config: %w", err)
	}
	return absPath, nil
}

func agentLaunchEnv(agent AgentLaunchConfig) []string {
	return []string{
		"ICOO_AGENT_PROVIDER_ID=" + agent.ProviderID,
		"ICOO_AGENT_MODEL_PROVIDER=" + agent.ModelProvider,
		"ICOO_AGENT_MODEL_NAME=" + agent.ModelName,
		"ICOO_AGENT_BASE_URL=" + agent.BaseURL,
		"ICOO_AGENT_API_KEY=" + agent.APIKey,
	}
}
