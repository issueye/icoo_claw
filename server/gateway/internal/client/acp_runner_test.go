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
		if err := a.conn.SessionUpdate(ctx, acp.SessionNotification{
			SessionId: params.SessionId,
			Update:    acp.UpdateAgentMessageText("ok"),
		}); err != nil {
			return acp.PromptResponse{}, err
		}
	}
	return acp.PromptResponse{StopReason: acp.StopReasonEndTurn}, nil
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
