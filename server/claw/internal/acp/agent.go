package acp

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	acp "github.com/coder/acp-go-sdk"
	"icoo_claw/server/claw/pkg/agent_sdk"
)

type Agent struct {
	runner agent_sdk.Runner
	conn   *acp.AgentSideConnection

	mu       sync.Mutex
	sessions map[string]*sessionState
}

type sessionState struct {
	cancel context.CancelFunc
}

func NewAgent(runner agent_sdk.Runner) *Agent {
	return &Agent{
		runner:   runner,
		sessions: make(map[string]*sessionState),
	}
}

func (a *Agent) SetAgentConnection(conn *acp.AgentSideConnection) {
	a.conn = conn
}

func (a *Agent) Initialize(ctx context.Context, _ acp.InitializeRequest) (acp.InitializeResponse, error) {
	return acp.InitializeResponse{
		ProtocolVersion: acp.ProtocolVersionNumber,
		AgentCapabilities: acp.AgentCapabilities{
			LoadSession: false,
			SessionCapabilities: acp.SessionCapabilities{
				Close: &acp.SessionCloseCapabilities{},
				List:  nil,
				Resume: nil,
			},
			PromptCapabilities: acp.PromptCapabilities{
				EmbeddedContext: true,
			},
		},
		AuthMethods: []acp.AuthMethod{},
		AgentInfo: &acp.Implementation{
			Name:    "claw",
			Version: "dev",
		},
	}, nil
}

func (a *Agent) Authenticate(ctx context.Context, _ acp.AuthenticateRequest) (acp.AuthenticateResponse, error) {
	return acp.AuthenticateResponse{}, nil
}

func (a *Agent) NewSession(ctx context.Context, params acp.NewSessionRequest) (acp.NewSessionResponse, error) {
	sid := "sess_" + randomID()
	a.mu.Lock()
	a.sessions[sid] = &sessionState{}
	a.mu.Unlock()
	return acp.NewSessionResponse{SessionId: acp.SessionId(sid)}, nil
}

func (a *Agent) Prompt(ctx context.Context, params acp.PromptRequest) (acp.PromptResponse, error) {
	sessionID := string(params.SessionId)
	state := a.getOrCreateSession(sessionID)
	if state == nil {
		return acp.PromptResponse{}, fmt.Errorf("session %s not found", sessionID)
	}
	if state.cancel != nil {
		state.cancel()
	}
	runCtx, cancel := context.WithCancel(ctx)
	state.cancel = cancel

	req := agent_sdk.RunRequest{
		SessionID: sessionID,
		RequestID: firstNonEmpty(stringPtr(params.MessageId), "req_"+randomID()),
		Prompt:    promptText(params.Prompt),
	}
	events, err := a.runner.RunStream(runCtx, req)
	if err != nil {
		state.cancel = nil
		return acp.PromptResponse{}, err
	}

	var stopReason acp.StopReason = acp.StopReasonEndTurn
	for event := range events {
		switch event.Type {
		case agent_sdk.StreamEventSessionUpdate:
			if event.Update == nil {
				continue
			}
			if a.conn != nil {
				if err := a.conn.SessionUpdate(runCtx, acp.SessionNotification{
					SessionId: acp.SessionId(sessionID),
					Update:    toACPUpdate(event.Update),
				}); err != nil {
					state.cancel = nil
					return acp.PromptResponse{}, err
				}
			}
		case agent_sdk.StreamEventSessionCompleted:
			stopReason = acp.StopReasonEndTurn
		case agent_sdk.StreamEventSessionError:
			stopReason = acp.StopReasonCancelled
		}
	}

	state.cancel = nil
	return acp.PromptResponse{StopReason: stopReason}, nil
}

func (a *Agent) Cancel(ctx context.Context, params acp.CancelNotification) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if state, ok := a.sessions[string(params.SessionId)]; ok && state.cancel != nil {
		state.cancel()
	}
	return nil
}

func (a *Agent) CloseSession(ctx context.Context, params acp.CloseSessionRequest) (acp.CloseSessionResponse, error) {
	_ = a.Cancel(ctx, acp.CancelNotification{SessionId: params.SessionId})
	a.mu.Lock()
	delete(a.sessions, string(params.SessionId))
	a.mu.Unlock()
	return acp.CloseSessionResponse{}, nil
}

func (a *Agent) ListSessions(ctx context.Context, _ acp.ListSessionsRequest) (acp.ListSessionsResponse, error) {
	return acp.ListSessionsResponse{}, acp.NewMethodNotFound(acp.AgentMethodSessionList)
}

func (a *Agent) ResumeSession(ctx context.Context, _ acp.ResumeSessionRequest) (acp.ResumeSessionResponse, error) {
	return acp.ResumeSessionResponse{}, acp.NewMethodNotFound(acp.AgentMethodSessionResume)
}

func (a *Agent) SetSessionConfigOption(ctx context.Context, _ acp.SetSessionConfigOptionRequest) (acp.SetSessionConfigOptionResponse, error) {
	return acp.SetSessionConfigOptionResponse{}, acp.NewMethodNotFound(acp.AgentMethodSessionSetConfigOption)
}

func (a *Agent) SetSessionMode(ctx context.Context, _ acp.SetSessionModeRequest) (acp.SetSessionModeResponse, error) {
	return acp.SetSessionModeResponse{}, acp.NewMethodNotFound(acp.AgentMethodSessionSetMode)
}

func (a *Agent) getOrCreateSession(sessionID string) *sessionState {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if state, ok := a.sessions[sessionID]; ok {
		return state
	}
	state := &sessionState{}
	a.sessions[sessionID] = state
	return state
}

func promptText(blocks []acp.ContentBlock) string {
	var b strings.Builder
	for _, block := range blocks {
		if block.Text != nil {
			b.WriteString(block.Text.Text)
			b.WriteString(" ")
		}
	}
	return strings.TrimSpace(b.String())
}

func toACPUpdate(update *agent_sdk.SessionUpdate) acp.SessionUpdate {
	switch update.SessionUpdate {
	case "agent_message_chunk":
		if update.Content != nil {
			return acp.UpdateAgentMessageText(update.Content.Text)
		}
	case "tool_call":
		opts := []acp.ToolCallStartOpt{
			acp.WithStartKind(acp.ToolKind(update.Kind)),
			acp.WithStartStatus(acp.ToolCallStatus(update.Status)),
		}
		if len(update.Locations) > 0 {
			locs := make([]acp.ToolCallLocation, 0, len(update.Locations))
			for _, location := range update.Locations {
				locs = append(locs, acp.ToolCallLocation{Path: location.Path})
			}
			opts = append(opts, acp.WithStartLocations(locs))
		}
		if update.RawInput != nil {
			opts = append(opts, acp.WithStartRawInput(update.RawInput))
		}
		if update.RawOutput != nil {
			opts = append(opts, acp.WithStartRawOutput(update.RawOutput))
		}
		return acp.StartToolCall(acp.ToolCallId(update.ToolCallID), update.Title, opts...)
	case "tool_call_update":
		opts := []acp.ToolCallUpdateOpt{
			acp.WithUpdateKind(acp.ToolKind(update.Kind)),
			acp.WithUpdateStatus(acp.ToolCallStatus(update.Status)),
		}
		if update.RawInput != nil {
			opts = append(opts, acp.WithUpdateRawInput(update.RawInput))
		}
		if update.RawOutput != nil {
			opts = append(opts, acp.WithUpdateRawOutput(update.RawOutput))
		}
		return acp.UpdateToolCall(acp.ToolCallId(update.ToolCallID), opts...)
	case "usage_update":
		if update.Usage != nil {
			return acp.SessionUpdate{
				UsageUpdate: &acp.SessionUsageUpdate{
					SessionUpdate: "usage_update",
					Size:          update.Usage.TotalTokens,
					Used:          update.Usage.InputTokens + update.Usage.OutputTokens,
				},
			}
		}
	}
	return acp.SessionUpdate{}
}

func stringPtr(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func randomID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func init() {
	_ = os.Getenv
}
