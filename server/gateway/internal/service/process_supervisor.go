package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"icoo_claw/common/agentproto"
	"icoo_claw/server/gateway/internal/client"
	"icoo_claw/server/gateway/internal/config"
	"icoo_claw/server/gateway/internal/model"
)

type StartAgentInstanceSpec struct {
	InstanceID         string
	AgentID            string
	Host               string
	Port               int
	BaseURL            string
	BinaryPath         string
	WorkDir            string
	SessionAPIURL      string
	InternalToken      string
	ConfigDir          string
	DefaultProjectRoot string
	RunnerMode         string
	Transport          string
	CommandArgs        []string
	Agent              agentproto.AgentLaunchConfig
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
	acp        *client.ACPRegistry
}

func NewLocalProcessSupervisor(acpRegistry ...*client.ACPRegistry) *LocalProcessSupervisor {
	var acp *client.ACPRegistry
	if len(acpRegistry) > 0 {
		acp = acpRegistry[0]
	}
	return &LocalProcessSupervisor{httpClient: &http.Client{Timeout: 2 * time.Second}, acp: acp}
}

func (s *LocalProcessSupervisor) Start(ctx context.Context, spec StartAgentInstanceSpec) (*AgentProcess, error) {
	transport := normalizeTransport(spec.Transport)
	if spec.BinaryPath == "" && transport != "acp" {
		return nil, errors.New("claw binary path is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	configPath, err := writeClawConfig(spec)
	if err != nil {
		return nil, err
	}

	var binaryPath string
	var args []string
	if transport == "acp" {
		command, err := acpCommandArgv(spec.CommandArgs)
		if err != nil {
			return nil, err
		}
		binaryPath, err = resolveExecutablePath(command[0], spec.WorkDir, executableDir(spec.BinaryPath))
		if err != nil {
			return nil, err
		}
		args = command[1:]
	} else {
		binaryPath, err = resolveExecutablePath(spec.BinaryPath, spec.WorkDir)
		if err != nil {
			return nil, err
		}
		args = []string{"--config", configPath}
		args = append(args, spec.CommandArgs...)
	}
	cmd := exec.Command(binaryPath, args...)
	if spec.WorkDir != "" {
		cmd.Dir = spec.WorkDir
	}
	cmd.Env = append(agentProcessEnv(os.Environ(), executableDir(spec.BinaryPath)), agentLaunchEnv(spec.Agent, configPath)...)
	if transport == "acp" {
		if s.acp == nil {
			return nil, errors.New("acp registry is required for acp transport")
		}
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return nil, fmt.Errorf("create acp stdin pipe: %w", err)
		}
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return nil, fmt.Errorf("create acp stdout pipe: %w", err)
		}
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			return nil, fmt.Errorf("start claw acp process: %w", err)
		}
		if err := s.acp.Register(ctx, spec.InstanceID, stdin, stdout, cmd.Process.Kill); err != nil {
			_ = cmd.Process.Kill()
			return nil, err
		}
		go func() {
			_ = cmd.Wait()
			s.acp.Remove(spec.InstanceID)
		}()
		return &AgentProcess{PID: cmd.Process.Pid}, nil
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start claw process: %w", err)
	}
	go func() { _ = cmd.Wait() }()
	return &AgentProcess{PID: cmd.Process.Pid}, nil
}

func (s *LocalProcessSupervisor) Stop(ctx context.Context, instance model.AgentInstance) error {
	if normalizeTransport(instance.Transport) == "acp" || client.IsACPBaseURL(instance.BaseURL) {
		if s.acp != nil {
			_ = s.acp.Close(instance.ID)
		}
	}
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
	if normalizeTransport(instance.Transport) == "acp" || client.IsACPBaseURL(instance.BaseURL) {
		if s.acp == nil {
			return errors.New("acp registry is not configured")
		}
		return s.acp.Probe(instance.ID)
	}
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

func acpCommandArgv(commandArgs []string) ([]string, error) {
	cleaned := make([]string, 0, len(commandArgs))
	for _, item := range commandArgs {
		item = strings.TrimSpace(item)
		if item != "" {
			cleaned = append(cleaned, item)
		}
	}
	if len(cleaned) == 0 {
		return nil, errors.New("acp command is required")
	}
	if len(cleaned) > 1 {
		return cleaned, nil
	}
	return splitCommandLine(cleaned[0])
}

func splitCommandLine(command string) ([]string, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return nil, errors.New("command is required")
	}
	var out []string
	var current strings.Builder
	var quote rune
	for _, ch := range command {
		if quote != 0 {
			if ch == quote {
				quote = 0
				continue
			}
			current.WriteRune(ch)
			continue
		}
		if ch == '\'' || ch == '"' {
			quote = ch
			continue
		}
		if ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n' {
			if current.Len() > 0 {
				out = append(out, current.String())
				current.Reset()
			}
			continue
		}
		current.WriteRune(ch)
	}
	if quote != 0 {
		return nil, errors.New("unterminated quote in command")
	}
	if current.Len() > 0 {
		out = append(out, current.String())
	}
	if len(out) == 0 {
		return nil, errors.New("command is required")
	}
	return out, nil
}

func resolveExecutablePath(path string, workDir string, extraDirs ...string) (string, error) {
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
	for _, dir := range extraDirs {
		dir = strings.TrimSpace(dir)
		if dir != "" {
			candidates = append(candidates, filepath.Join(dir, path))
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(cwd, path))
	}

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		for _, expanded := range executablePathCandidates(candidate) {
			abs, err := filepath.Abs(expanded)
			if err != nil {
				continue
			}
			if info, err := os.Stat(abs); err == nil && !info.IsDir() {
				return abs, nil
			}
		}
	}

	found, err := exec.LookPath(path)
	if err != nil {
		return "", err
	}
	return filepath.Abs(found)
}

func executablePathCandidates(path string) []string {
	if runtime.GOOS != "windows" || filepath.Ext(path) != "" {
		return []string{path}
	}
	extensions := []string{".exe", ".cmd", ".bat", ".com"}
	if value := strings.TrimSpace(os.Getenv("PATHEXT")); value != "" {
		extensions = nil
		for _, ext := range strings.Split(value, ";") {
			ext = strings.TrimSpace(ext)
			if ext != "" {
				extensions = append(extensions, ext)
			}
		}
	}
	out := make([]string, 0, len(extensions)+1)
	out = append(out, path)
	for _, ext := range extensions {
		out = append(out, path+ext)
	}
	return out
}

func executableDir(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if !filepath.IsAbs(path) {
		if abs, err := filepath.Abs(path); err == nil {
			path = abs
		}
	}
	return filepath.Dir(path)
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
		Transport:     "http",
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
		"http_addr = %q\nsession_api_url = %q\ninternal_token = %q\nrunner_mode = %q\n\n[agent]\nagent_id = %q\nprovider_id = %q\nmodel_provider = %q\nmodel_name = %q\nbase_url = %q\napi_key_set = %t\n\n[gateway_skills]\npath = %q\n",
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
		spec.DefaultProjectRoot,
	)
	if err := os.WriteFile(absPath, []byte(payload), 0o600); err != nil {
		return "", fmt.Errorf("write claw config: %w", err)
	}
	return absPath, nil
}

func agentLaunchEnv(agent agentproto.AgentLaunchConfig, configPath string) []string {
	return []string{
		"ICOO_AGENT_PROVIDER_ID=" + agent.ProviderID,
		"ICOO_AGENT_MODEL_PROVIDER=" + agent.ModelProvider,
		"ICOO_AGENT_MODEL_NAME=" + agent.ModelName,
		"ICOO_AGENT_BASE_URL=" + agent.BaseURL,
		"ICOO_AGENT_API_KEY=" + agent.APIKey,
		"ICOO_CLAW_CONFIG=" + configPath,
	}
}

func agentProcessEnv(base []string, binDir string) []string {
	binDir = strings.TrimSpace(binDir)
	if binDir == "" {
		return append([]string(nil), base...)
	}
	out := append([]string(nil), base...)
	pathKey := "PATH"
	if runtime.GOOS == "windows" {
		pathKey = "Path"
	}
	prefix := binDir + string(os.PathListSeparator)
	for i, entry := range out {
		key, value, ok := strings.Cut(entry, "=")
		if ok && strings.EqualFold(key, "PATH") {
			out[i] = pathKey + "=" + prefix + value
			return out
		}
	}
	return append(out, pathKey+"="+binDir)
}

func normalizeTransport(value string) string {
	return agentproto.NormalizeTransport(value)
}
