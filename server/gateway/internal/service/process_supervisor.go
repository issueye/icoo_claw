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
	InstanceID      string
	AgentID         string
	Host            string
	Port            int
	BaseURL         string
	BinaryPath      string
	WorkDir         string
	SessionStoreURL string
	InternalToken   string
	ConfigDir       string
	RunnerMode      string
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
	configPath, err := writeClawConfig(spec)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(spec.BinaryPath, "--config", configPath)
	if spec.WorkDir != "" {
		cmd.Dir = spec.WorkDir
	}
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

func processSpecFromConfig(cfg config.Config, instanceID, agentID string, port int) StartAgentInstanceSpec {
	host := "127.0.0.1"
	baseURL := "http://" + host + ":" + strconv.Itoa(port)
	return StartAgentInstanceSpec{
		InstanceID:      instanceID,
		AgentID:         agentID,
		Host:            host,
		Port:            port,
		BaseURL:         baseURL,
		BinaryPath:      cfg.ClawBinaryPath,
		WorkDir:         cfg.ClawWorkDir,
		SessionStoreURL: cfg.SessionStoreURL,
		InternalToken:   cfg.InternalToken,
		ConfigDir:       cfg.ClawConfigDir,
		RunnerMode:      cfg.ClawRunnerMode,
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
	payload := fmt.Sprintf(
		"http_addr = %q\nsession_store_url = %q\ninternal_token = %q\nrunner_mode = %q\n",
		spec.Host+":"+strconv.Itoa(spec.Port),
		spec.SessionStoreURL,
		spec.InternalToken,
		spec.RunnerMode,
	)
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		return "", fmt.Errorf("write claw config: %w", err)
	}
	return path, nil
}
