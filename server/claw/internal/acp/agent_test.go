package acp

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"testing"

	acp "github.com/coder/acp-go-sdk"
	"icoo_claw/common/core/agent_sdk/api"
	sdkmessage "icoo_claw/common/core/agent_sdk/message"
	"icoo_claw/server/claw/pkg/agent_sdk"
)

type errorRunner struct{}

func (r errorRunner) Run(context.Context, agent_sdk.RunRequest) (*agent_sdk.RunResponse, error) {
	return nil, nil
}

func (r errorRunner) RunStream(context.Context, agent_sdk.RunRequest) (<-chan agent_sdk.StreamEvent, error) {
	events := make(chan agent_sdk.StreamEvent, 1)
	events <- agent_sdk.StreamEvent{Type: agent_sdk.StreamEventSessionError, Error: &agent_sdk.StreamError{Message: "boom"}}
	close(events)
	return events, nil
}

func TestPromptReturnsErrorForRuntimeSessionError(t *testing.T) {
	agent := NewAgent(errorRunner{})
	_, err := agent.Prompt(context.Background(), acp.PromptRequest{
		SessionId: acp.SessionId("sess_1"),
		Prompt:    []acp.ContentBlock{acp.TextBlock("hello")},
	})
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("error = %v, want runtime session error", err)
	}
}

func TestToACPUpdateSkipsUnsupportedUpdates(t *testing.T) {
	update, ok := toACPUpdate(&agent_sdk.SessionUpdate{SessionUpdate: "content_block_start"})
	if ok {
		t.Fatalf("ok = true for unsupported update: %+v", update)
	}
}

func TestToACPUpdateReturnsMarshalableAgentMessage(t *testing.T) {
	update, ok := toACPUpdate(&agent_sdk.SessionUpdate{
		SessionUpdate: "agent_message_chunk",
		Content:       &agent_sdk.ContentBlock{Type: "text", Text: "hello"},
	})
	if !ok {
		t.Fatal("ok = false, want supported update")
	}
	if _, err := json.Marshal(update); err != nil {
		t.Fatalf("marshal update: %v", err)
	}
}

type captureRunner struct {
	req agent_sdk.RunRequest
}

func (r *captureRunner) Run(context.Context, agent_sdk.RunRequest) (*agent_sdk.RunResponse, error) {
	return nil, nil
}

func (r *captureRunner) RunStream(_ context.Context, req agent_sdk.RunRequest) (<-chan agent_sdk.StreamEvent, error) {
	r.req = req
	events := make(chan agent_sdk.StreamEvent, 1)
	events <- agent_sdk.StreamEvent{Type: agent_sdk.StreamEventSessionCompleted, StopReason: "end_turn"}
	close(events)
	return events, nil
}

func TestLoadSessionStoresCwdForPromptMetadata(t *testing.T) {
	runner := &captureRunner{}
	agent := NewAgent(runner)
	_, err := agent.LoadSession(context.Background(), acp.LoadSessionRequest{
		SessionId:  "sess_loaded",
		Cwd:        "/tmp/project",
		McpServers: []acp.McpServer{},
	})
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	_, err = agent.Prompt(context.Background(), acp.PromptRequest{
		SessionId: "sess_loaded",
		Prompt:    []acp.ContentBlock{acp.TextBlock("hello")},
	})
	if err != nil {
		t.Fatalf("prompt: %v", err)
	}
	if runner.req.Metadata["cwd"] != "/tmp/project" || runner.req.Metadata["project_root"] != "/tmp/project" {
		t.Fatalf("metadata = %#v, want cwd/project_root from loaded session", runner.req.Metadata)
	}
}

type permissionClient struct {
	req      acp.RequestPermissionRequest
	optionID acp.PermissionOptionId
	calls    int
}

func (c *permissionClient) RequestPermission(_ context.Context, params acp.RequestPermissionRequest) (acp.RequestPermissionResponse, error) {
	c.req = params
	c.calls++
	return acp.RequestPermissionResponse{
		Outcome: acp.RequestPermissionOutcome{
			Selected: &acp.RequestPermissionOutcomeSelected{OptionId: c.optionID},
		},
	}, nil
}

func (c *permissionClient) SessionUpdate(context.Context, acp.SessionNotification) error { return nil }
func (c *permissionClient) ReadTextFile(context.Context, acp.ReadTextFileRequest) (acp.ReadTextFileResponse, error) {
	return acp.ReadTextFileResponse{}, nil
}
func (c *permissionClient) WriteTextFile(context.Context, acp.WriteTextFileRequest) (acp.WriteTextFileResponse, error) {
	return acp.WriteTextFileResponse{}, nil
}
func (c *permissionClient) CreateTerminal(context.Context, acp.CreateTerminalRequest) (acp.CreateTerminalResponse, error) {
	return acp.CreateTerminalResponse{}, nil
}
func (c *permissionClient) KillTerminal(context.Context, acp.KillTerminalRequest) (acp.KillTerminalResponse, error) {
	return acp.KillTerminalResponse{}, nil
}
func (c *permissionClient) TerminalOutput(context.Context, acp.TerminalOutputRequest) (acp.TerminalOutputResponse, error) {
	return acp.TerminalOutputResponse{}, nil
}
func (c *permissionClient) ReleaseTerminal(context.Context, acp.ReleaseTerminalRequest) (acp.ReleaseTerminalResponse, error) {
	return acp.ReleaseTerminalResponse{}, nil
}
func (c *permissionClient) WaitForTerminalExit(context.Context, acp.WaitForTerminalExitRequest) (acp.WaitForTerminalExitResponse, error) {
	return acp.WaitForTerminalExitResponse{}, nil
}

type loadReplayClient struct {
	mu      sync.Mutex
	updates []acp.SessionUpdate
}

func (c *loadReplayClient) SessionUpdate(_ context.Context, params acp.SessionNotification) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.updates = append(c.updates, params.Update)
	return nil
}

func (c *loadReplayClient) RequestPermission(context.Context, acp.RequestPermissionRequest) (acp.RequestPermissionResponse, error) {
	return acp.RequestPermissionResponse{Outcome: acp.RequestPermissionOutcome{Cancelled: &acp.RequestPermissionOutcomeCancelled{}}}, nil
}

func (c *loadReplayClient) ReadTextFile(context.Context, acp.ReadTextFileRequest) (acp.ReadTextFileResponse, error) {
	return acp.ReadTextFileResponse{}, nil
}
func (c *loadReplayClient) WriteTextFile(context.Context, acp.WriteTextFileRequest) (acp.WriteTextFileResponse, error) {
	return acp.WriteTextFileResponse{}, nil
}
func (c *loadReplayClient) CreateTerminal(context.Context, acp.CreateTerminalRequest) (acp.CreateTerminalResponse, error) {
	return acp.CreateTerminalResponse{}, nil
}
func (c *loadReplayClient) KillTerminal(context.Context, acp.KillTerminalRequest) (acp.KillTerminalResponse, error) {
	return acp.KillTerminalResponse{}, nil
}
func (c *loadReplayClient) TerminalOutput(context.Context, acp.TerminalOutputRequest) (acp.TerminalOutputResponse, error) {
	return acp.TerminalOutputResponse{}, nil
}
func (c *loadReplayClient) ReleaseTerminal(context.Context, acp.ReleaseTerminalRequest) (acp.ReleaseTerminalResponse, error) {
	return acp.ReleaseTerminalResponse{}, nil
}
func (c *loadReplayClient) WaitForTerminalExit(context.Context, acp.WaitForTerminalExitRequest) (acp.WaitForTerminalExitResponse, error) {
	return acp.WaitForTerminalExitResponse{}, nil
}

type staticHistoryLoader struct {
	sessionID string
	messages  []sdkmessage.Message
}

func (h *staticHistoryLoader) Load(_ context.Context, sessionID string) ([]sdkmessage.Message, error) {
	h.sessionID = sessionID
	return h.messages, nil
}

func TestLoadSessionReplaysHistoryBeforeReturning(t *testing.T) {
	clientRead, agentWrite := io.Pipe()
	agentRead, clientWrite := io.Pipe()
	t.Cleanup(func() {
		_ = clientRead.Close()
		_ = agentWrite.Close()
		_ = agentRead.Close()
		_ = clientWrite.Close()
	})

	agent := NewAgent(nil)
	history := &staticHistoryLoader{messages: []sdkmessage.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
	}}
	agent.SetHistoryLoader(history)
	agentConn := acp.NewAgentSideConnection(agent, agentWrite, agentRead)
	agent.SetAgentConnection(agentConn)
	go func() { <-agentConn.Done() }()

	client := &loadReplayClient{}
	clientConn := acp.NewClientSideConnection(client, clientWrite, clientRead)

	_, err := clientConn.LoadSession(context.Background(), acp.LoadSessionRequest{
		SessionId:  "sess_loaded",
		Cwd:        "/tmp/project",
		McpServers: []acp.McpServer{},
	})
	if err != nil {
		t.Fatalf("load session: %v", err)
	}

	client.mu.Lock()
	defer client.mu.Unlock()
	if len(client.updates) != 2 {
		t.Fatalf("updates = %+v, want user and assistant history chunks", client.updates)
	}
	if client.updates[0].UserMessageChunk == nil || client.updates[0].UserMessageChunk.Content.Text.Text != "hello" {
		t.Fatalf("first update = %+v, want user message hello", client.updates[0])
	}
	if client.updates[1].AgentMessageChunk == nil || client.updates[1].AgentMessageChunk.Content.Text.Text != "hi" {
		t.Fatalf("second update = %+v, want agent message hi", client.updates[1])
	}
}

func TestPromptUsesGatewaySessionIDFromMeta(t *testing.T) {
	runner := &captureRunner{}
	agent := NewAgent(runner)
	_, err := agent.NewSession(context.Background(), acp.NewSessionRequest{
		Cwd: "/tmp/project",
	})
	if err != nil {
		t.Fatalf("new session: %v", err)
	}
	sessions, err := agent.ListSessions(context.Background(), acp.ListSessionsRequest{})
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions.Sessions) != 1 {
		t.Fatalf("sessions = %+v, want one ACP session", sessions.Sessions)
	}

	_, err = agent.Prompt(context.Background(), acp.PromptRequest{
		SessionId: sessions.Sessions[0].SessionId,
		Prompt:    []acp.ContentBlock{acp.TextBlock("hello")},
		Meta: map[string]any{
			"gateway_session_id": "gateway_sess_1",
			"request_id":         "req_1",
		},
	})
	if err != nil {
		t.Fatalf("prompt: %v", err)
	}
	if runner.req.SessionID != "gateway_sess_1" {
		t.Fatalf("runner session id = %q, want gateway session id", runner.req.SessionID)
	}
	if runner.req.RequestID != "req_1" {
		t.Fatalf("runner request id = %q, want req_1", runner.req.RequestID)
	}
}

func TestPromptPermissionBridgesToACPClient(t *testing.T) {
	clientRead, agentWrite := io.Pipe()
	agentRead, clientWrite := io.Pipe()
	t.Cleanup(func() {
		_ = clientRead.Close()
		_ = agentWrite.Close()
		_ = agentRead.Close()
		_ = clientWrite.Close()
	})

	agent := NewAgent(nil)
	agentConn := acp.NewAgentSideConnection(agent, agentWrite, agentRead)
	agent.SetAgentConnection(agentConn)
	go func() { <-agentConn.Done() }()

	client := &permissionClient{optionID: acp.PermissionOptionId("allow_once")}
	_ = acp.NewClientSideConnection(client, clientWrite, clientRead)

	ctx := api.WithSessionID(context.Background(), "sess_perm")
	allowed, err := agent.PromptPermission(ctx, api.PermissionRequest{
		ToolCallID: "tool_1",
		ToolName:   "write",
		Arguments:  map[string]any{"path": "/tmp/file.txt"},
		Target:     "/tmp/file.txt",
		Mode:       "ask",
	})
	if err != nil {
		t.Fatalf("prompt permission: %v", err)
	}
	if !allowed {
		t.Fatal("allowed = false, want true")
	}
	if client.req.SessionId != "sess_perm" || client.req.ToolCall.ToolCallId != "tool_1" {
		t.Fatalf("permission request = %+v", client.req)
	}
	if len(client.req.Options) != 4 {
		t.Fatalf("options = %+v, want allow/reject once/always", client.req.Options)
	}
}

func TestPromptPermissionRejectsSelectedDeny(t *testing.T) {
	clientRead, agentWrite := io.Pipe()
	agentRead, clientWrite := io.Pipe()
	t.Cleanup(func() {
		_ = clientRead.Close()
		_ = agentWrite.Close()
		_ = agentRead.Close()
		_ = clientWrite.Close()
	})

	agent := NewAgent(nil)
	agentConn := acp.NewAgentSideConnection(agent, agentWrite, agentRead)
	agent.SetAgentConnection(agentConn)
	go func() { <-agentConn.Done() }()

	client := &permissionClient{optionID: acp.PermissionOptionId("reject_once")}
	_ = acp.NewClientSideConnection(client, clientWrite, clientRead)

	allowed, err := agent.PromptPermission(api.WithSessionID(context.Background(), "sess_perm"), api.PermissionRequest{
		ToolCallID: "tool_2",
		ToolName:   "bash",
	})
	if err != nil {
		t.Fatalf("prompt permission: %v", err)
	}
	if allowed {
		t.Fatal("allowed = true, want false")
	}
}

func TestPromptPermissionRemembersAllowAlwaysForSession(t *testing.T) {
	clientRead, agentWrite := io.Pipe()
	agentRead, clientWrite := io.Pipe()
	t.Cleanup(func() {
		_ = clientRead.Close()
		_ = agentWrite.Close()
		_ = agentRead.Close()
		_ = clientWrite.Close()
	})

	agent := NewAgent(nil)
	agentConn := acp.NewAgentSideConnection(agent, agentWrite, agentRead)
	agent.SetAgentConnection(agentConn)
	go func() { <-agentConn.Done() }()

	client := &permissionClient{optionID: acp.PermissionOptionId("allow_always")}
	_ = acp.NewClientSideConnection(client, clientWrite, clientRead)

	ctx := api.WithSessionID(context.Background(), "sess_perm")
	req := api.PermissionRequest{ToolCallID: "tool_3", ToolName: "write", Target: "/tmp/file.txt"}
	allowed, err := agent.PromptPermission(ctx, req)
	if err != nil {
		t.Fatalf("prompt permission: %v", err)
	}
	if !allowed {
		t.Fatal("allowed = false, want true")
	}

	client.optionID = acp.PermissionOptionId("reject_once")
	req.ToolCallID = "tool_4"
	allowed, err = agent.PromptPermission(ctx, req)
	if err != nil {
		t.Fatalf("prompt permission remembered: %v", err)
	}
	if !allowed {
		t.Fatal("remembered allowed = false, want true")
	}
	if client.calls != 1 {
		t.Fatalf("permission calls = %d, want one prompt before remembered decision", client.calls)
	}
}

func TestPromptPermissionRemembersRejectAlwaysForSession(t *testing.T) {
	clientRead, agentWrite := io.Pipe()
	agentRead, clientWrite := io.Pipe()
	t.Cleanup(func() {
		_ = clientRead.Close()
		_ = agentWrite.Close()
		_ = agentRead.Close()
		_ = clientWrite.Close()
	})

	agent := NewAgent(nil)
	agentConn := acp.NewAgentSideConnection(agent, agentWrite, agentRead)
	agent.SetAgentConnection(agentConn)
	go func() { <-agentConn.Done() }()

	client := &permissionClient{optionID: acp.PermissionOptionId("reject_always")}
	_ = acp.NewClientSideConnection(client, clientWrite, clientRead)

	ctx := api.WithSessionID(context.Background(), "sess_perm")
	req := api.PermissionRequest{ToolCallID: "tool_5", ToolName: "bash", Target: "rm -rf /tmp/demo"}
	allowed, err := agent.PromptPermission(ctx, req)
	if err != nil {
		t.Fatalf("prompt permission: %v", err)
	}
	if allowed {
		t.Fatal("allowed = true, want false")
	}

	client.optionID = acp.PermissionOptionId("allow_once")
	req.ToolCallID = "tool_6"
	allowed, err = agent.PromptPermission(ctx, req)
	if err != nil {
		t.Fatalf("prompt permission remembered: %v", err)
	}
	if allowed {
		t.Fatal("remembered allowed = true, want false")
	}
	if client.calls != 1 {
		t.Fatalf("permission calls = %d, want one prompt before remembered decision", client.calls)
	}
}

func TestSessionMethodsStoreACPStateForPromptMetadata(t *testing.T) {
	runner := &captureRunner{}
	agent := NewAgent(runner)

	_, err := agent.LoadSession(context.Background(), acp.LoadSessionRequest{
		SessionId:             "sess_state",
		Cwd:                   "/tmp/project",
		AdditionalDirectories: []string{"/tmp/shared"},
		Meta:                  map[string]any{"request_source": "desktop"},
	})
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	_, err = agent.SetSessionMode(context.Background(), acp.SetSessionModeRequest{
		SessionId: "sess_state",
		ModeId:    acp.SessionModeId("plan"),
	})
	if err != nil {
		t.Fatalf("set mode: %v", err)
	}
	_, err = agent.SetSessionConfigOption(context.Background(), acp.SetSessionConfigOptionRequest{
		Boolean: &acp.SetSessionConfigOptionBoolean{
			SessionId: "sess_state",
			ConfigId:  acp.SessionConfigId("safe_mode"),
			Type:      "boolean",
			Value:     true,
		},
	})
	if err != nil {
		t.Fatalf("set config option: %v", err)
	}

	_, err = agent.Prompt(context.Background(), acp.PromptRequest{
		SessionId: "sess_state",
		Prompt:    []acp.ContentBlock{acp.TextBlock("hello")},
	})
	if err != nil {
		t.Fatalf("prompt: %v", err)
	}
	if runner.req.Metadata["acp_session_mode"] != "plan" {
		t.Fatalf("metadata mode = %#v, want plan", runner.req.Metadata["acp_session_mode"])
	}
	configOptions, ok := runner.req.Metadata["acp_config_options"].(map[string]any)
	if !ok || configOptions["safe_mode"] != true {
		t.Fatalf("metadata config = %#v, want safe_mode true", runner.req.Metadata["acp_config_options"])
	}
	additional, ok := runner.req.Metadata["additional_directories"].([]string)
	if !ok || len(additional) != 1 || additional[0] != "/tmp/shared" {
		t.Fatalf("metadata additional dirs = %#v", runner.req.Metadata["additional_directories"])
	}
	if runner.req.Metadata["request_source"] != "desktop" {
		t.Fatalf("metadata = %#v, want request_source", runner.req.Metadata)
	}
}

func TestSessionConfigOptionResponseReturnsCompleteState(t *testing.T) {
	agent := NewAgent(nil)
	resp, err := agent.NewSession(context.Background(), acp.NewSessionRequest{
		Cwd:        "/tmp/project",
		McpServers: []acp.McpServer{},
	})
	if err != nil {
		t.Fatalf("new session: %v", err)
	}
	if len(resp.ConfigOptions) == 0 || resp.ConfigOptions[0].Select == nil {
		t.Fatalf("config options = %+v, want mode select option", resp.ConfigOptions)
	}
	if resp.Modes == nil || resp.Modes.CurrentModeId != "ask" {
		t.Fatalf("modes = %+v, want ask mode", resp.Modes)
	}

	configResp, err := agent.SetSessionConfigOption(context.Background(), acp.SetSessionConfigOptionRequest{
		ValueId: &acp.SetSessionConfigOptionValueId{
			SessionId: resp.SessionId,
			ConfigId:  acp.SessionConfigId("mode"),
			Value:     acp.SessionConfigValueId("code"),
		},
	})
	if err != nil {
		t.Fatalf("set config option: %v", err)
	}
	if len(configResp.ConfigOptions) == 0 || configResp.ConfigOptions[0].Select == nil {
		t.Fatalf("config options = %+v, want complete mode select state", configResp.ConfigOptions)
	}
	if got := configResp.ConfigOptions[0].Select.CurrentValue; got != "code" {
		t.Fatalf("current value = %q, want code", got)
	}
	if len(*configResp.ConfigOptions[0].Select.Options.Ungrouped) < 3 {
		t.Fatalf("mode options = %+v, want full selectable state", configResp.ConfigOptions[0].Select.Options)
	}
}

func TestListAndResumeSessions(t *testing.T) {
	agent := NewAgent(nil)
	_, err := agent.NewSession(context.Background(), acp.NewSessionRequest{
		Cwd:                   "/tmp/project",
		AdditionalDirectories: []string{"/tmp/shared"},
		McpServers:            []acp.McpServer{},
	})
	if err != nil {
		t.Fatalf("new session: %v", err)
	}
	sessions, err := agent.ListSessions(context.Background(), acp.ListSessionsRequest{})
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions.Sessions) != 1 || sessions.Sessions[0].Cwd != "/tmp/project" {
		t.Fatalf("sessions = %+v, want created session", sessions.Sessions)
	}

	_, err = agent.ResumeSession(context.Background(), acp.ResumeSessionRequest{
		SessionId:             sessions.Sessions[0].SessionId,
		Cwd:                   "/tmp/resumed",
		AdditionalDirectories: []string{"/tmp/other"},
		McpServers:            []acp.McpServer{},
	})
	if err != nil {
		t.Fatalf("resume session: %v", err)
	}
	cwd := "/tmp/resumed"
	sessions, err = agent.ListSessions(context.Background(), acp.ListSessionsRequest{Cwd: &cwd})
	if err != nil {
		t.Fatalf("list resumed sessions: %v", err)
	}
	if len(sessions.Sessions) != 1 || sessions.Sessions[0].Cwd != "/tmp/resumed" {
		t.Fatalf("sessions = %+v, want resumed session", sessions.Sessions)
	}
}
