package acp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	acp "github.com/coder/acp-go-sdk"
	"icoo_claw/common/agentproto"
	"icoo_claw/common/agentproto/agentruntime"
	"icoo_claw/common/core/agent_sdk/api"
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
	cwd    string
	meta   map[string]any
}

func NewAgent(runner agent_sdk.Runner) *Agent {
	return &Agent{
		runner:   runner,
		sessions: make(map[string]*sessionState),
	}
}

func (a *Agent) SetRunner(runner agent_sdk.Runner) {
	a.runner = runner
}

func (a *Agent) SetAgentConnection(conn *acp.AgentSideConnection) {
	a.conn = conn
}

func (a *Agent) Initialize(ctx context.Context, _ acp.InitializeRequest) (acp.InitializeResponse, error) {
	return acp.InitializeResponse{
		ProtocolVersion: acp.ProtocolVersionNumber,
		AgentCapabilities: acp.AgentCapabilities{
			LoadSession: true,
			SessionCapabilities: acp.SessionCapabilities{
				Close:  &acp.SessionCloseCapabilities{},
				List:   nil,
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
	a.sessions[sid] = &sessionState{cwd: params.Cwd, meta: cloneMap(params.Meta)}
	a.mu.Unlock()
	return acp.NewSessionResponse{SessionId: acp.SessionId(sid)}, nil
}

func (a *Agent) LoadSession(ctx context.Context, params acp.LoadSessionRequest) (acp.LoadSessionResponse, error) {
	sessionID := strings.TrimSpace(string(params.SessionId))
	if sessionID == "" {
		return acp.LoadSessionResponse{}, fmt.Errorf("sessionId is required")
	}
	a.mu.Lock()
	a.sessions[sessionID] = &sessionState{cwd: params.Cwd, meta: cloneMap(params.Meta)}
	a.mu.Unlock()
	return acp.LoadSessionResponse{}, nil
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
		SessionID:     sessionID,
		RequestID:     firstNonEmpty(metaString(params.Meta, "request_id"), stringPtr(params.MessageId), "req_"+randomID()),
		Prompt:        promptText(params.Prompt),
		Agent:         metaAgentProfile(params.Meta, "agent"),
		ToolWhitelist: metaStringSlice(params.Meta, "tool_whitelist"),
		Metadata:      runMetadata(params.Meta, state),
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
			update, ok := toACPUpdate(event.Update)
			if !ok {
				continue
			}
			if a.conn != nil {
				if err := a.conn.SessionUpdate(runCtx, acp.SessionNotification{
					SessionId: acp.SessionId(sessionID),
					Update:    update,
				}); err != nil {
					state.cancel = nil
					return acp.PromptResponse{}, err
				}
			}
		case agent_sdk.StreamEventSessionCompleted:
			stopReason = acp.StopReasonEndTurn
		case agent_sdk.StreamEventSessionError:
			state.cancel = nil
			if event.Error != nil && strings.TrimSpace(event.Error.Message) != "" {
				return acp.PromptResponse{}, fmt.Errorf("agent stream error: %s", event.Error.Message)
			}
			return acp.PromptResponse{}, fmt.Errorf("agent stream error")
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

func (a *Agent) PromptPermission(ctx context.Context, req api.PermissionRequest) (bool, error) {
	if a == nil || a.conn == nil {
		return false, fmt.Errorf("acp permission prompt unavailable")
	}
	toolCallID := strings.TrimSpace(req.ToolCallID)
	if toolCallID == "" {
		toolCallID = "permission_" + randomID()
	}
	kind := acp.ToolKind(agentruntime.ToolKind(req.ToolName))
	if kind == "" {
		kind = acp.ToolKindOther
	}
	status := acp.ToolCallStatusPending
	title := permissionTitle(req)
	locations := permissionLocations(req.Target)

	resp, err := a.conn.RequestPermission(ctx, acp.RequestPermissionRequest{
		SessionId: acp.SessionId(firstNonEmpty(api.SessionIDFromContext(ctx), "default")),
		ToolCall: acp.ToolCallUpdate{
			ToolCallId: acp.ToolCallId(toolCallID),
			Title:      &title,
			Kind:       &kind,
			Status:     &status,
			Locations:  locations,
			RawInput:   req.Arguments,
			Meta: map[string]any{
				"toolName": req.ToolName,
				"target":   req.Target,
				"rule":     req.Rule,
				"mode":     req.Mode,
			},
		},
		Options: []acp.PermissionOption{
			{Kind: acp.PermissionOptionKindAllowOnce, Name: "Allow once", OptionId: acp.PermissionOptionId("allow_once")},
			{Kind: acp.PermissionOptionKindRejectOnce, Name: "Reject once", OptionId: acp.PermissionOptionId("reject_once")},
		},
	})
	if err != nil {
		return false, err
	}
	if resp.Outcome.Cancelled != nil {
		return false, context.Canceled
	}
	if resp.Outcome.Selected == nil {
		return false, nil
	}
	switch resp.Outcome.Selected.OptionId {
	case acp.PermissionOptionId("allow_once"), acp.PermissionOptionId("allow_always"):
		return true, nil
	default:
		return false, nil
	}
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

func runMetadata(meta map[string]any, state *sessionState) map[string]any {
	out := metaMap(meta, "metadata")
	if out == nil {
		out = map[string]any{}
	}
	if state != nil {
		if strings.TrimSpace(state.cwd) != "" {
			if _, ok := out["cwd"]; !ok {
				out["cwd"] = state.cwd
			}
			if _, ok := out["project_root"]; !ok {
				out["project_root"] = state.cwd
			}
		}
		for key, value := range state.meta {
			if _, ok := out[key]; !ok {
				out[key] = value
			}
		}
	}
	return out
}

func cloneMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]any, len(src))
	for key, value := range src {
		out[key] = value
	}
	return out
}

func permissionTitle(req api.PermissionRequest) string {
	toolName := strings.TrimSpace(req.ToolName)
	if toolName == "" {
		toolName = "tool"
	}
	target := strings.TrimSpace(req.Target)
	if target == "" {
		return fmt.Sprintf("Run %s", toolName)
	}
	return fmt.Sprintf("Run %s on %s", toolName, target)
}

func permissionLocations(target string) []acp.ToolCallLocation {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil
	}
	if filepath.IsAbs(target) || strings.ContainsAny(target, `/\`) {
		return []acp.ToolCallLocation{{Path: target}}
	}
	return nil
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

func toACPUpdate(update *agent_sdk.SessionUpdate) (acp.SessionUpdate, bool) {
	switch update.SessionUpdate {
	case "agent_message_chunk":
		if update.Content != nil {
			return acp.UpdateAgentMessageText(update.Content.Text), true
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
		return acp.StartToolCall(acp.ToolCallId(update.ToolCallID), update.Title, opts...), true
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
		return acp.UpdateToolCall(acp.ToolCallId(update.ToolCallID), opts...), true
	case "usage_update":
		if update.Usage != nil {
			return acp.SessionUpdate{
				UsageUpdate: &acp.SessionUsageUpdate{
					SessionUpdate: "usage_update",
					Size:          update.Usage.TotalTokens,
					Used:          update.Usage.InputTokens + update.Usage.OutputTokens,
				},
			}, true
		}
	}
	return acp.SessionUpdate{}, false
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

func metaString(meta map[string]any, key string) string {
	if meta == nil {
		return ""
	}
	value, _ := meta[key].(string)
	return strings.TrimSpace(value)
}

func metaMap(meta map[string]any, key string) map[string]any {
	if meta == nil {
		return nil
	}
	raw, ok := meta[key].(map[string]any)
	if !ok {
		return nil
	}
	out := make(map[string]any, len(raw))
	for k, v := range raw {
		out[k] = v
	}
	return out
}

func metaAgentProfile(meta map[string]any, key string) *agentproto.AgentRuntimeProfile {
	raw := metaMap(meta, key)
	if raw == nil {
		return nil
	}
	payload, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var profile agentproto.AgentRuntimeProfile
	if err := json.Unmarshal(payload, &profile); err != nil {
		return nil
	}
	return &profile
}

func metaStringSlice(meta map[string]any, key string) []string {
	if meta == nil {
		return nil
	}
	switch raw := meta[key].(type) {
	case []string:
		return append([]string(nil), raw...)
	case []any:
		out := make([]string, 0, len(raw))
		for _, item := range raw {
			if value, ok := item.(string); ok && strings.TrimSpace(value) != "" {
				out = append(out, strings.TrimSpace(value))
			}
		}
		return out
	default:
		return nil
	}
}

func randomID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func init() {
	_ = os.Getenv
}
