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
	cancel                context.CancelFunc
	cwd                   string
	additionalDirectories []string
	mcpServers            []acp.McpServer
	modeID                string
	configOptions         map[string]any
	permissionRules       map[permissionRuleKey]bool
	meta                  map[string]any
	updatedAt             time.Time
	title                 string
}

type permissionRuleKey struct {
	toolName string
	target   string
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
				Close:                 &acp.SessionCloseCapabilities{},
				List:                  &acp.SessionListCapabilities{},
				Resume:                &acp.SessionResumeCapabilities{},
				AdditionalDirectories: &acp.SessionAdditionalDirectoriesCapabilities{},
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
	a.sessions[sid] = &sessionState{
		cwd:                   params.Cwd,
		additionalDirectories: cloneStringSlice(params.AdditionalDirectories),
		mcpServers:            cloneMCPServers(params.McpServers),
		meta:                  cloneMap(params.Meta),
		updatedAt:             time.Now().UTC(),
	}
	a.mu.Unlock()
	return acp.NewSessionResponse{SessionId: acp.SessionId(sid)}, nil
}

func (a *Agent) LoadSession(ctx context.Context, params acp.LoadSessionRequest) (acp.LoadSessionResponse, error) {
	sessionID := strings.TrimSpace(string(params.SessionId))
	if sessionID == "" {
		return acp.LoadSessionResponse{}, fmt.Errorf("sessionId is required")
	}
	a.mu.Lock()
	a.sessions[sessionID] = &sessionState{
		cwd:                   params.Cwd,
		additionalDirectories: cloneStringSlice(params.AdditionalDirectories),
		mcpServers:            cloneMCPServers(params.McpServers),
		meta:                  cloneMap(params.Meta),
		updatedAt:             time.Now().UTC(),
	}
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
	state.updatedAt = time.Now().UTC()
	if strings.TrimSpace(req.Prompt) != "" && strings.TrimSpace(state.title) == "" {
		state.title = summarizePromptTitle(req.Prompt)
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

func (a *Agent) ListSessions(ctx context.Context, params acp.ListSessionsRequest) (acp.ListSessionsResponse, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	sessions := make([]acp.SessionInfo, 0, len(a.sessions))
	for id, state := range a.sessions {
		if !matchesSessionListFilters(state, params) {
			continue
		}
		updatedAt := state.updatedAt.Format(time.RFC3339)
		if state.updatedAt.IsZero() {
			updatedAt = time.Now().UTC().Format(time.RFC3339)
		}
		var title *string
		if strings.TrimSpace(state.title) != "" {
			value := state.title
			title = &value
		}
		sessions = append(sessions, acp.SessionInfo{
			SessionId:             acp.SessionId(id),
			Cwd:                   state.cwd,
			AdditionalDirectories: cloneStringSlice(state.additionalDirectories),
			Title:                 title,
			UpdatedAt:             &updatedAt,
		})
	}
	return acp.ListSessionsResponse{Sessions: sessions}, nil
}

func (a *Agent) ResumeSession(ctx context.Context, params acp.ResumeSessionRequest) (acp.ResumeSessionResponse, error) {
	sessionID := strings.TrimSpace(string(params.SessionId))
	if sessionID == "" {
		return acp.ResumeSessionResponse{}, fmt.Errorf("sessionId is required")
	}
	a.mu.Lock()
	state, ok := a.sessions[sessionID]
	if !ok {
		state = &sessionState{}
		a.sessions[sessionID] = state
	}
	state.cwd = params.Cwd
	state.additionalDirectories = cloneStringSlice(params.AdditionalDirectories)
	state.mcpServers = cloneMCPServers(params.McpServers)
	state.meta = cloneMap(params.Meta)
	state.updatedAt = time.Now().UTC()
	modes := sessionModes(state.modeID)
	a.mu.Unlock()

	return acp.ResumeSessionResponse{Modes: modes}, nil
}

func (a *Agent) SetSessionConfigOption(ctx context.Context, params acp.SetSessionConfigOptionRequest) (acp.SetSessionConfigOptionResponse, error) {
	sessionID, configID, value := sessionConfigOptionValue(params)
	if sessionID == "" || configID == "" {
		return acp.SetSessionConfigOptionResponse{}, fmt.Errorf("sessionId and configId are required")
	}
	state := a.getOrCreateSession(sessionID)
	if state == nil {
		return acp.SetSessionConfigOptionResponse{}, fmt.Errorf("session %s not found", sessionID)
	}
	a.mu.Lock()
	if state.configOptions == nil {
		state.configOptions = map[string]any{}
	}
	state.configOptions[configID] = value
	state.updatedAt = time.Now().UTC()
	a.mu.Unlock()
	return acp.SetSessionConfigOptionResponse{ConfigOptions: []acp.SessionConfigOption{}}, nil
}

func (a *Agent) SetSessionMode(ctx context.Context, params acp.SetSessionModeRequest) (acp.SetSessionModeResponse, error) {
	sessionID := strings.TrimSpace(string(params.SessionId))
	if sessionID == "" {
		return acp.SetSessionModeResponse{}, fmt.Errorf("sessionId is required")
	}
	state := a.getOrCreateSession(sessionID)
	if state == nil {
		return acp.SetSessionModeResponse{}, fmt.Errorf("session %s not found", sessionID)
	}
	a.mu.Lock()
	state.modeID = strings.TrimSpace(string(params.ModeId))
	state.updatedAt = time.Now().UTC()
	a.mu.Unlock()
	return acp.SetSessionModeResponse{}, nil
}

func (a *Agent) PromptPermission(ctx context.Context, req api.PermissionRequest) (bool, error) {
	if a == nil || a.conn == nil {
		return false, fmt.Errorf("acp permission prompt unavailable")
	}
	sessionID := firstNonEmpty(api.SessionIDFromContext(ctx), "default")
	ruleKey := permissionKey(req)
	if allowed, ok := a.rememberedPermission(sessionID, ruleKey); ok {
		return allowed, nil
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
		SessionId: acp.SessionId(sessionID),
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
			{Kind: acp.PermissionOptionKindAllowOnce, Name: "允许本次", OptionId: acp.PermissionOptionId("allow_once")},
			{Kind: acp.PermissionOptionKindAllowAlways, Name: "当前会话始终允许", OptionId: acp.PermissionOptionId("allow_always")},
			{Kind: acp.PermissionOptionKindRejectOnce, Name: "拒绝本次", OptionId: acp.PermissionOptionId("reject_once")},
			{Kind: acp.PermissionOptionKindRejectAlways, Name: "当前会话始终拒绝", OptionId: acp.PermissionOptionId("reject_always")},
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
	case acp.PermissionOptionId("allow_always"):
		a.rememberPermission(sessionID, ruleKey, true)
		return true, nil
	case acp.PermissionOptionId("reject_always"):
		a.rememberPermission(sessionID, ruleKey, false)
		return false, nil
	case acp.PermissionOptionId("allow_once"):
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
		if len(state.additionalDirectories) > 0 {
			out["additional_directories"] = cloneStringSlice(state.additionalDirectories)
		}
		if len(state.mcpServers) > 0 {
			out["mcp_servers"] = state.mcpServers
		}
		if strings.TrimSpace(state.modeID) != "" {
			out["acp_session_mode"] = state.modeID
		}
		if len(state.configOptions) > 0 {
			out["acp_config_options"] = cloneAnyMap(state.configOptions)
		}
		for key, value := range state.meta {
			if _, ok := out[key]; !ok {
				out[key] = value
			}
		}
	}
	return out
}

func (a *Agent) rememberedPermission(sessionID string, key permissionRuleKey) (bool, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	state := a.sessions[sessionID]
	if state == nil || state.permissionRules == nil {
		return false, false
	}
	allowed, ok := state.permissionRules[key]
	return allowed, ok
}

func (a *Agent) rememberPermission(sessionID string, key permissionRuleKey, allowed bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	state := a.sessions[sessionID]
	if state == nil {
		state = &sessionState{}
		a.sessions[sessionID] = state
	}
	if state.permissionRules == nil {
		state.permissionRules = map[permissionRuleKey]bool{}
	}
	state.permissionRules[key] = allowed
	state.updatedAt = time.Now().UTC()
}

func permissionKey(req api.PermissionRequest) permissionRuleKey {
	target := strings.TrimSpace(req.Target)
	if target == "" {
		target = strings.TrimSpace(req.Rule)
	}
	if target == "" && len(req.Arguments) > 0 {
		if payload, err := json.Marshal(req.Arguments); err == nil {
			target = string(payload)
		}
	}
	return permissionRuleKey{
		toolName: strings.TrimSpace(req.ToolName),
		target:   target,
	}
}

func matchesSessionListFilters(state *sessionState, params acp.ListSessionsRequest) bool {
	if state == nil {
		return false
	}
	if params.Cwd != nil && strings.TrimSpace(*params.Cwd) != strings.TrimSpace(state.cwd) {
		return false
	}
	if len(params.AdditionalDirectories) > 0 && !equalStringSlices(params.AdditionalDirectories, state.additionalDirectories) {
		return false
	}
	return true
}

func sessionConfigOptionValue(params acp.SetSessionConfigOptionRequest) (string, string, any) {
	if params.Boolean != nil {
		return strings.TrimSpace(string(params.Boolean.SessionId)), strings.TrimSpace(string(params.Boolean.ConfigId)), params.Boolean.Value
	}
	if params.ValueId != nil {
		return strings.TrimSpace(string(params.ValueId.SessionId)), strings.TrimSpace(string(params.ValueId.ConfigId)), string(params.ValueId.Value)
	}
	return "", "", nil
}

func sessionModes(current string) *acp.SessionModeState {
	current = strings.TrimSpace(current)
	if current == "" {
		return nil
	}
	return &acp.SessionModeState{
		CurrentModeId: acp.SessionModeId(current),
		AvailableModes: []acp.SessionMode{{
			Id:   acp.SessionModeId(current),
			Name: current,
		}},
	}
}

func summarizePromptTitle(prompt string) string {
	prompt = strings.TrimSpace(strings.ReplaceAll(prompt, "\n", " "))
	if len(prompt) <= 64 {
		return prompt
	}
	return strings.TrimSpace(prompt[:64])
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

func cloneAnyMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]any, len(src))
	for key, value := range src {
		out[key] = value
	}
	return out
}

func cloneStringSlice(src []string) []string {
	if len(src) == 0 {
		return nil
	}
	return append([]string(nil), src...)
}

func cloneMCPServers(src []acp.McpServer) []acp.McpServer {
	if len(src) == 0 {
		return nil
	}
	return append([]acp.McpServer(nil), src...)
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if strings.TrimSpace(a[i]) != strings.TrimSpace(b[i]) {
			return false
		}
	}
	return true
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
