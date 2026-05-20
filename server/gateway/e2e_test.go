package gateway_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

type childProcess struct {
	name string
	cmd  *exec.Cmd
	logs *bytes.Buffer
	done chan error
}

func TestGatewayConversationE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e process test in short mode")
	}

	tmp := t.TempDir()
	binDir := filepath.Join(tmp, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("create bin dir: %v", err)
	}

	gatewayBin := filepath.Join(binDir, exeName("gateway"))
	clawBin := filepath.Join(binDir, exeName("claw"))
	buildBinary(t, gatewayBin, "./cmd/gateway")
	buildBinary(t, clawBin, "../claw/cmd/claw")

	gatewayPort := freePort(t)
	clawPort := freePort(t)
	token := "e2e-token"

	gatewayConfig := filepath.Join(tmp, "gateway.toml")
	writeFile(t, gatewayConfig, fmt.Sprintf(`
http_addr = "127.0.0.1:%d"
db_path = %q
session_api_url = "http://127.0.0.1:%d"
internal_token = %q
claw_binary_path = %q
claw_config_dir = %q
claw_runner_mode = "fake"
claw_port_start = %d
claw_port_end = %d
max_agent_instances = 1
health_interval_seconds = 1
shutdown_timeout_seconds = 1
`, gatewayPort, slashPath(filepath.Join(tmp, "gateway.sqlite")), gatewayPort, token, slashPath(clawBin), slashPath(filepath.Join(tmp, "claw-configs")), clawPort, clawPort))

	gatewayProc := startProcess(t, "gateway", gatewayBin, "--config", gatewayConfig)
	defer gatewayProc.stop()
	gatewayURL := fmt.Sprintf("http://127.0.0.1:%d", gatewayPort)
	defer stopAgentInstances(gatewayURL)
	waitHealth(t, gatewayProc, gatewayURL+"/health")

	doJSON(t, http.MethodPost, gatewayURL+"/v1/agents", map[string]any{
		"id":             "agent_e2e",
		"name":           "E2E Agent",
		"model_provider": "openai",
		"model_name":     "fake",
		"max_iterations": 1,
		"tool_whitelist": []string{},
		"enabled":        true,
	}, http.StatusCreated, nil)

	var conversation struct {
		ID        string `json:"id"`
		SessionID string `json:"session_id"`
	}
	doJSON(t, http.MethodPost, gatewayURL+"/v1/conversations", map[string]any{
		"agent_id": "agent_e2e",
		"title":    "E2E Conversation",
	}, http.StatusCreated, &conversation)
	if conversation.ID == "" || conversation.SessionID == "" {
		t.Fatalf("conversation missing ids: %+v", conversation)
	}

	var chatResp struct {
		Output string `json:"output"`
	}
	doJSON(t, http.MethodPost, gatewayURL+"/v1/conversations/"+conversation.ID+"/messages", map[string]any{
		"prompt":     "hello e2e",
		"request_id": "req_e2e",
	}, http.StatusOK, &chatResp)
	if chatResp.Output != "fake agent response: hello e2e" {
		t.Fatalf("output = %q", chatResp.Output)
	}

	var messages struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	doJSON(t, http.MethodGet, gatewayURL+"/v1/conversations/"+conversation.ID+"/messages", nil, http.StatusOK, &messages)
	if len(messages.Messages) != 2 {
		t.Fatalf("messages = %+v, want two messages", messages.Messages)
	}
	if messages.Messages[0].Role != "user" || messages.Messages[0].Content != "hello e2e" {
		t.Fatalf("first message = %+v", messages.Messages[0])
	}
	if messages.Messages[1].Role != "assistant" || messages.Messages[1].Content != "fake agent response: hello e2e" {
		t.Fatalf("second message = %+v", messages.Messages[1])
	}
}

func stopAgentInstances(gatewayURL string) {
	var response struct {
		Instances []struct {
			ID string `json:"id"`
		} `json:"instances"`
	}
	if err := doJSONNoFail(http.MethodGet, gatewayURL+"/v1/agent-instances", nil, http.StatusOK, &response); err != nil {
		return
	}
	for _, instance := range response.Instances {
		if instance.ID == "" {
			continue
		}
		_ = doJSONNoFail(http.MethodPost, gatewayURL+"/v1/agent-instances/"+instance.ID+"/stop", map[string]any{}, http.StatusNoContent, nil)
	}
	time.Sleep(500 * time.Millisecond)
}

func buildBinary(t *testing.T, out string, pkg string) {
	t.Helper()
	cmd := exec.Command("go", "build", "-o", out, pkg)
	var logs bytes.Buffer
	cmd.Stdout = &logs
	cmd.Stderr = &logs
	if err := cmd.Run(); err != nil {
		t.Fatalf("build %s: %v\n%s", pkg, err, logs.String())
	}
}

func startProcess(t *testing.T, name string, bin string, args ...string) *childProcess {
	t.Helper()
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, bin, args...)
	logs := &bytes.Buffer{}
	cmd.Stdout = logs
	cmd.Stderr = logs
	if err := cmd.Start(); err != nil {
		t.Fatalf("start %s: %v", name, err)
	}
	proc := &childProcess{name: name, cmd: cmd, logs: logs, done: make(chan error, 1)}
	go func() { proc.done <- cmd.Wait() }()
	return proc
}

func (p *childProcess) stop() {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return
	}
	_ = p.cmd.Process.Kill()
	select {
	case <-p.done:
	case <-time.After(2 * time.Second):
	}
}

func waitHealth(t *testing.T, proc *childProcess, url string) {
	t.Helper()
	client := &http.Client{Timeout: 500 * time.Millisecond}
	deadline := time.After(20 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case err := <-proc.done:
			t.Fatalf("%s exited before healthy: %v\n%s", proc.name, err, proc.logs.String())
		case <-deadline:
			t.Fatalf("timeout waiting for %s health at %s\n%s", proc.name, url, proc.logs.String())
		case <-ticker.C:
			resp, err := client.Get(url)
			if err == nil {
				_ = resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return
				}
			}
		}
	}
}

func doJSON(t *testing.T, method string, url string, body any, expected int, out any) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer resp.Body.Close()
	payload, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != expected {
		t.Fatalf("%s %s status = %d, want %d: %s", method, url, resp.StatusCode, expected, strings.TrimSpace(string(payload)))
	}
	if out != nil {
		if err := json.Unmarshal(payload, out); err != nil {
			t.Fatalf("decode response: %v\n%s", err, payload)
		}
	}
}

func doJSONNoFail(method string, url string, body any, expected int, out any) error {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	payload, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != expected {
		return fmt.Errorf("%s %s status = %d, want %d: %s", method, url, resp.StatusCode, expected, strings.TrimSpace(string(payload)))
	}
	if out != nil {
		if err := json.Unmarshal(payload, out); err != nil {
			return err
		}
	}
	return nil
}

func freePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen free port: %v", err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func slashPath(path string) string {
	return filepath.ToSlash(path)
}

func exeName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}
