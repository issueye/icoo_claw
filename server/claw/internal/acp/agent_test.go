package acp

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	acp "github.com/coder/acp-go-sdk"
	"icoo_claw/common/core/agent_sdk/api"
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
}

func (c *permissionClient) RequestPermission(_ context.Context, params acp.RequestPermissionRequest) (acp.RequestPermissionResponse, error) {
	c.req = params
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
	if len(client.req.Options) != 2 {
		t.Fatalf("options = %+v, want allow/reject", client.req.Options)
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
