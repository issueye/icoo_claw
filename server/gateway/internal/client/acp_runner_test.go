package client

import (
	"context"
	"io"
	"testing"

	acp "github.com/coder/acp-go-sdk"
)

type strictACPAgent struct {
	conn              *acp.AgentSideConnection
	newSessionCalls   int
	promptSessionID   acp.SessionId
	promptGatewayID   string
	promptRequestID   string
	knownACPSessionID acp.SessionId
	requestPermission bool
	permissionOption  acp.PermissionOptionId
}

func (a *strictACPAgent) Initialize(context.Context, acp.InitializeRequest) (acp.InitializeResponse, error) {
	return acp.InitializeResponse{ProtocolVersion: acp.ProtocolVersionNumber}, nil
}

func (a *strictACPAgent) Authenticate(context.Context, acp.AuthenticateRequest) (acp.AuthenticateResponse, error) {
	return acp.AuthenticateResponse{}, nil
}

func (a *strictACPAgent) NewSession(context.Context, acp.NewSessionRequest) (acp.NewSessionResponse, error) {
	a.newSessionCalls++
	a.knownACPSessionID = acp.SessionId("external-session")
	return acp.NewSessionResponse{SessionId: a.knownACPSessionID}, nil
}

func (a *strictACPAgent) Prompt(ctx context.Context, params acp.PromptRequest) (acp.PromptResponse, error) {
	if params.SessionId != a.knownACPSessionID {
		return acp.PromptResponse{}, &acp.RequestError{Code: -32002, Message: "Resource not found"}
	}
	a.promptSessionID = params.SessionId
	if value, ok := params.Meta["gateway_session_id"].(string); ok {
		a.promptGatewayID = value
	}
	if value, ok := params.Meta["request_id"].(string); ok {
		a.promptRequestID = value
	}
	if a.conn != nil {
		if a.requestPermission {
			resp, err := a.conn.RequestPermission(ctx, acp.RequestPermissionRequest{
				SessionId: params.SessionId,
				ToolCall: acp.ToolCallUpdate{
					ToolCallId: acp.ToolCallId("tool_1"),
					Title:      acp.Ptr("Sensitive action"),
				},
				Options: []acp.PermissionOption{
					{Kind: acp.PermissionOptionKindAllowOnce, Name: "Allow", OptionId: acp.PermissionOptionId("allow_once")},
					{Kind: acp.PermissionOptionKindRejectOnce, Name: "Reject", OptionId: acp.PermissionOptionId("reject_once")},
				},
			})
			if err != nil {
				return acp.PromptResponse{}, err
			}
			if resp.Outcome.Selected != nil {
				a.permissionOption = resp.Outcome.Selected.OptionId
			}
		}
		if err := a.conn.SessionUpdate(ctx, acp.SessionNotification{
			SessionId: params.SessionId,
			Update:    acp.UpdateAgentMessageText("ok"),
		}); err != nil {
			return acp.PromptResponse{}, err
		}
	}
	return acp.PromptResponse{StopReason: acp.StopReasonEndTurn}, nil
}

func TestACPConnectionEmitsAgentPermissionRequestInProtocol(t *testing.T) {
	clientRead, agentWrite := io.Pipe()
	agentRead, clientWrite := io.Pipe()
	t.Cleanup(func() {
		_ = clientRead.Close()
		_ = agentWrite.Close()
		_ = agentRead.Close()
		_ = clientWrite.Close()
	})

	agent := &strictACPAgent{requestPermission: true}
	agentConn := acp.NewAgentSideConnection(agent, agentWrite, agentRead)
	agent.conn = agentConn
	go func() { <-agentConn.Done() }()

	registry := NewACPRegistry()
	if err := registry.Register(context.Background(), "inst_perm", clientWrite, clientRead, nil); err != nil {
		t.Fatalf("register: %v", err)
	}
	t.Cleanup(func() { _ = registry.Close("inst_perm") })

	events, err := registry.Stream(context.Background(), ACPBaseURL("inst_perm"), RunRequest{
		SessionID: "gateway-session",
		RequestID: "req_perm",
		Prompt:    "hello",
	})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	for event := range events {
		if event.Type == "session/error" {
			t.Fatalf("unexpected stream error: %+v", event.Error)
		}
		if event.Type == "session/request_permission" {
			if event.Permission == nil || event.Permission.ToolCall.ToolCallID != "tool_1" {
				t.Fatalf("permission event = %+v", event.Permission)
			}
			event.PermissionDecision <- PermissionVote{ID: event.Permission.ID, Outcome: "selected", OptionID: "allow_once"}
		}
	}
	if agent.permissionOption != acp.PermissionOptionId("allow_once") {
		t.Fatalf("permission option = %q, want allow_once", agent.permissionOption)
	}
}

func TestACPConnectionPermissionRequestReturnsCancelledWhenContextCancels(t *testing.T) {
	conn := &ACPConnection{}
	events := make(chan StreamEvent, 1)
	conn.active = &acpActiveStream{
		sessionID: "gateway-session",
		requestID: "req_cancel",
		events:    events,
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	resp, err := conn.requestPermission(ctx, acp.RequestPermissionRequest{
		SessionId: acp.SessionId("external-session"),
		ToolCall: acp.ToolCallUpdate{
			ToolCallId: acp.ToolCallId("tool_1"),
			Title:      acp.Ptr("Sensitive action"),
		},
		Options: []acp.PermissionOption{
			{Kind: acp.PermissionOptionKindAllowOnce, Name: "Allow", OptionId: acp.PermissionOptionId("allow_once")},
		},
	})
	if err != nil {
		t.Fatalf("request permission: %v", err)
	}
	if resp.Outcome.Cancelled == nil {
		t.Fatalf("outcome = %+v, want cancelled", resp.Outcome)
	}
}

func (a *strictACPAgent) Cancel(context.Context, acp.CancelNotification) error { return nil }
func (a *strictACPAgent) CloseSession(context.Context, acp.CloseSessionRequest) (acp.CloseSessionResponse, error) {
	return acp.CloseSessionResponse{}, nil
}
func (a *strictACPAgent) ListSessions(context.Context, acp.ListSessionsRequest) (acp.ListSessionsResponse, error) {
	return acp.ListSessionsResponse{}, acp.NewMethodNotFound(acp.AgentMethodSessionList)
}
func (a *strictACPAgent) ResumeSession(context.Context, acp.ResumeSessionRequest) (acp.ResumeSessionResponse, error) {
	return acp.ResumeSessionResponse{}, acp.NewMethodNotFound(acp.AgentMethodSessionResume)
}
func (a *strictACPAgent) SetSessionConfigOption(context.Context, acp.SetSessionConfigOptionRequest) (acp.SetSessionConfigOptionResponse, error) {
	return acp.SetSessionConfigOptionResponse{}, acp.NewMethodNotFound(acp.AgentMethodSessionSetConfigOption)
}
func (a *strictACPAgent) SetSessionMode(context.Context, acp.SetSessionModeRequest) (acp.SetSessionModeResponse, error) {
	return acp.SetSessionModeResponse{}, acp.NewMethodNotFound(acp.AgentMethodSessionSetMode)
}

func TestACPConnectionCreatesSessionBeforePrompt(t *testing.T) {
	clientRead, agentWrite := io.Pipe()
	agentRead, clientWrite := io.Pipe()
	t.Cleanup(func() {
		_ = clientRead.Close()
		_ = agentWrite.Close()
		_ = agentRead.Close()
		_ = clientWrite.Close()
	})

	agent := &strictACPAgent{}
	agentConn := acp.NewAgentSideConnection(agent, agentWrite, agentRead)
	agent.conn = agentConn
	go func() { <-agentConn.Done() }()

	registry := NewACPRegistry()
	if err := registry.Register(context.Background(), "inst_1", clientWrite, clientRead, nil); err != nil {
		t.Fatalf("register: %v", err)
	}
	t.Cleanup(func() { _ = registry.Close("inst_1") })

	events, err := registry.Stream(context.Background(), ACPBaseURL("inst_1"), RunRequest{
		SessionID: "gateway-session",
		RequestID: "req_1",
		Prompt:    "hello",
	})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	var gotText, gotCompleted bool
	for event := range events {
		if event.Type == "session/error" {
			t.Fatalf("unexpected stream error: %+v", event.Error)
		}
		if event.SessionID != "gateway-session" || event.RequestID != "req_1" {
			t.Fatalf("event ids = %q/%q", event.SessionID, event.RequestID)
		}
		if event.Update != nil && event.Update.Content != nil && event.Update.Content.Text == "ok" {
			gotText = true
		}
		if event.Type == "session/completed" {
			gotCompleted = true
		}
	}
	if !gotText || !gotCompleted {
		t.Fatalf("events missing text/completion: text=%t completed=%t", gotText, gotCompleted)
	}
	if agent.newSessionCalls != 1 {
		t.Fatalf("new session calls = %d, want 1", agent.newSessionCalls)
	}
	if agent.promptSessionID != "external-session" {
		t.Fatalf("prompt session id = %q", agent.promptSessionID)
	}
	if agent.promptGatewayID != "gateway-session" || agent.promptRequestID != "req_1" {
		t.Fatalf("prompt meta gateway/request = %q/%q", agent.promptGatewayID, agent.promptRequestID)
	}
}
