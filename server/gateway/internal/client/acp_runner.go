package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	acp "github.com/coder/acp-go-sdk"
	"icoo_claw/common/agentproto"
)

const acpBaseURLPrefix = "acp://"

type ACPRegistry struct {
	mu    sync.Mutex
	conns map[string]*ACPConnection
}

func NewACPRegistry() *ACPRegistry {
	return &ACPRegistry{conns: map[string]*ACPConnection{}}
}

func ACPBaseURL(instanceID string) string {
	return acpBaseURLPrefix + strings.TrimSpace(instanceID)
}

func IsACPBaseURL(baseURL string) bool {
	return strings.HasPrefix(strings.TrimSpace(baseURL), acpBaseURLPrefix)
}

func (r *ACPRegistry) Register(ctx context.Context, instanceID string, input io.Writer, output io.Reader, stop func() error) error {
	if r == nil {
		return errors.New("acp registry is not configured")
	}
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		return errors.New("acp instance id is required")
	}
	callbacks := &acpCallbacks{}
	conn := acp.NewClientSideConnection(callbacks, input, output)
	acpConn := &ACPConnection{
		instanceID: instanceID,
		conn:       conn,
		callbacks:  callbacks,
		stop:       stop,
	}
	callbacks.conn = acpConn

	if _, err := conn.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
		ClientCapabilities: acp.ClientCapabilities{
			Fs:       acp.FileSystemCapabilities{ReadTextFile: false, WriteTextFile: false},
			Terminal: false,
		},
		ClientInfo: &acp.Implementation{Name: "icoo-claw-gateway", Version: "dev"},
	}); err != nil {
		return fmt.Errorf("initialize acp agent: %w", err)
	}

	r.mu.Lock()
	if previous := r.conns[instanceID]; previous != nil {
		_ = previous.Close()
	}
	r.conns[instanceID] = acpConn
	r.mu.Unlock()
	return nil
}

func (r *ACPRegistry) Remove(instanceID string) {
	if r == nil {
		return
	}
	instanceID = strings.TrimSpace(instanceID)
	r.mu.Lock()
	delete(r.conns, instanceID)
	r.mu.Unlock()
}

func (r *ACPRegistry) Close(instanceID string) error {
	conn := r.lookup(instanceID)
	if conn == nil {
		return nil
	}
	r.Remove(instanceID)
	return conn.Close()
}

func (r *ACPRegistry) Probe(instanceID string) error {
	conn := r.lookup(instanceID)
	if conn == nil {
		return fmt.Errorf("acp instance %s is not connected", instanceID)
	}
	select {
	case <-conn.conn.Done():
		return fmt.Errorf("acp instance %s disconnected", instanceID)
	default:
		return nil
	}
}

func (r *ACPRegistry) Run(ctx context.Context, baseURL string, req RunRequest) (*RunResponse, error) {
	events, err := r.Stream(ctx, baseURL, req)
	if err != nil {
		return nil, err
	}
	collected, err := agentproto.CollectTextStream(events, req.SessionID, req.RequestID)
	if err != nil {
		return nil, err
	}
	return &RunResponse{
		SessionID:  collected.SessionID,
		RequestID:  collected.RequestID,
		Output:     collected.Output,
		StopReason: collected.StopReason,
	}, nil
}

func (r *ACPRegistry) Stream(ctx context.Context, baseURL string, req RunRequest) (<-chan StreamEvent, error) {
	instanceID := strings.TrimPrefix(strings.TrimSpace(baseURL), acpBaseURLPrefix)
	conn := r.lookup(instanceID)
	if conn == nil {
		return nil, &HTTPError{
			Service:    "acp",
			Method:     "POST",
			Path:       "/session/prompt",
			StatusCode: 503,
			Code:       "dependency_unavailable",
			Message:    "acp instance is not connected",
		}
	}
	return conn.Stream(ctx, req)
}

func (r *ACPRegistry) lookup(instanceID string) *ACPConnection {
	if r == nil {
		return nil
	}
	instanceID = strings.TrimSpace(instanceID)
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.conns[instanceID]
}

type ACPConnection struct {
	instanceID string
	conn       *acp.ClientSideConnection
	callbacks  *acpCallbacks
	stop       func() error

	mu       sync.Mutex
	active   *acpActiveStream
	sessions map[string]acp.SessionId
}

type acpActiveStream struct {
	sessionID string
	requestID string
	events    chan StreamEvent
}

func (c *ACPConnection) Stream(ctx context.Context, req RunRequest) (<-chan StreamEvent, error) {
	if c == nil || c.conn == nil {
		return nil, errors.New("acp connection is not configured")
	}
	out := make(chan StreamEvent, 128)
	active := &acpActiveStream{
		sessionID: req.SessionID,
		requestID: req.RequestID,
		events:    out,
	}

	c.mu.Lock()
	if c.active != nil {
		c.mu.Unlock()
		return nil, &HTTPError{
			Service:    "acp",
			Method:     "POST",
			Path:       "/session/prompt",
			StatusCode: 409,
			Code:       "session_busy",
			Message:    "acp instance is already running a prompt",
		}
	}
	c.active = active
	c.mu.Unlock()

	go func() {
		defer close(out)
		defer c.clearActive(active)

		done := make(chan struct{})
		go func() {
			select {
			case <-ctx.Done():
				if sessionID := c.lookupSession(req.SessionID); strings.TrimSpace(string(sessionID)) != "" {
					_ = c.conn.Cancel(context.Background(), acp.CancelNotification{SessionId: sessionID})
				}
			case <-done:
			}
		}()
		defer close(done)

		acpSessionID, err := c.ensureSession(ctx, req.SessionID, req.Metadata)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			sendACPEvent(ctx, out, StreamEvent{
				Type:      "session/error",
				SessionID: req.SessionID,
				RequestID: req.RequestID,
				Error:     &StreamError{Message: err.Error(), Code: "acp_error"},
			})
			return
		}

		messageID := req.RequestID
		promptReq := acp.PromptRequest{
			SessionId: acpSessionID,
			Prompt:    []acp.ContentBlock{acp.TextBlock(req.Prompt)},
			Meta: map[string]any{
				"gateway_session_id": req.SessionID,
				"request_id":         req.RequestID,
				"agent":              req.Agent,
				"tool_whitelist":     req.ToolWhitelist,
				"metadata":           req.Metadata,
			},
		}
		if strings.TrimSpace(messageID) != "" {
			promptReq.MessageId = &messageID
		}
		resp, err := c.conn.Prompt(ctx, promptReq)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			if isACPResourceNotFound(err) {
				c.forgetSession(req.SessionID)
				if retrySessionID, retryErr := c.ensureSession(ctx, req.SessionID, req.Metadata); retryErr == nil {
					promptReq.SessionId = retrySessionID
					resp, err = c.conn.Prompt(ctx, promptReq)
				}
			}
		}
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			sendACPEvent(ctx, out, StreamEvent{
				Type:      "session/error",
				SessionID: req.SessionID,
				RequestID: req.RequestID,
				Error:     &StreamError{Message: err.Error(), Code: "acp_error"},
			})
			return
		}
		sendACPEvent(ctx, out, StreamEvent{
			Type:       "session/completed",
			SessionID:  req.SessionID,
			RequestID:  req.RequestID,
			StopReason: normalizeACPStopReason(resp.StopReason),
		})
	}()

	return out, nil
}

func (c *ACPConnection) ensureSession(ctx context.Context, gatewaySessionID string, metadata map[string]any) (acp.SessionId, error) {
	gatewaySessionID = strings.TrimSpace(gatewaySessionID)
	if gatewaySessionID == "" {
		gatewaySessionID = "default"
	}
	c.mu.Lock()
	if c.sessions == nil {
		c.sessions = make(map[string]acp.SessionId)
	}
	if sessionID := c.sessions[gatewaySessionID]; strings.TrimSpace(string(sessionID)) != "" {
		c.mu.Unlock()
		return sessionID, nil
	}
	c.mu.Unlock()

	resp, err := c.conn.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        acpSessionCwd(metadata),
		McpServers: []acp.McpServer{},
	})
	if err != nil {
		return "", fmt.Errorf("create acp session: %w", err)
	}
	if strings.TrimSpace(string(resp.SessionId)) == "" {
		return "", errors.New("create acp session: empty session id")
	}

	c.mu.Lock()
	if c.sessions == nil {
		c.sessions = make(map[string]acp.SessionId)
	}
	c.sessions[gatewaySessionID] = resp.SessionId
	c.mu.Unlock()
	return resp.SessionId, nil
}

func (c *ACPConnection) lookupSession(gatewaySessionID string) acp.SessionId {
	gatewaySessionID = strings.TrimSpace(gatewaySessionID)
	if gatewaySessionID == "" {
		gatewaySessionID = "default"
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.sessions[gatewaySessionID]
}

func (c *ACPConnection) forgetSession(gatewaySessionID string) {
	gatewaySessionID = strings.TrimSpace(gatewaySessionID)
	if gatewaySessionID == "" {
		gatewaySessionID = "default"
	}
	c.mu.Lock()
	delete(c.sessions, gatewaySessionID)
	c.mu.Unlock()
}

func acpSessionCwd(metadata map[string]any) string {
	if len(metadata) > 0 {
		if value, ok := metadata["project_root"].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
		if value, ok := metadata["cwd"].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	if cwd, err := os.Getwd(); err == nil && strings.TrimSpace(cwd) != "" {
		return cwd
	}
	return "."
}

func isACPResourceNotFound(err error) bool {
	var requestErr *acp.RequestError
	if errors.As(err, &requestErr) {
		return requestErr.Code == -32002
	}
	return strings.Contains(err.Error(), "Resource not found")
}

func (c *ACPConnection) Close() error {
	if c == nil {
		return nil
	}
	if c.stop != nil {
		return c.stop()
	}
	return nil
}

func (c *ACPConnection) clearActive(active *acpActiveStream) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.active == active {
		c.active = nil
	}
}

func (c *ACPConnection) handleUpdate(ctx context.Context, params acp.SessionNotification) error {
	c.mu.Lock()
	active := c.active
	c.mu.Unlock()
	if active == nil {
		return nil
	}
	event := mapACPUpdate(params.Update, active.sessionID, active.requestID)
	if event.Type == "" {
		return nil
	}
	sendACPEvent(ctx, active.events, event)
	return nil
}

func sendACPEvent(ctx context.Context, events chan<- StreamEvent, event StreamEvent) {
	select {
	case <-ctx.Done():
	case events <- event:
	}
}

func mapACPUpdate(update acp.SessionUpdate, sessionID string, requestID string) StreamEvent {
	base := StreamEvent{
		Type:      "session/update",
		SessionID: sessionID,
		RequestID: requestID,
	}
	switch {
	case update.AgentMessageChunk != nil:
		base.Update = &SessionUpdate{
			SessionUpdate: "agent_message_chunk",
			Content:       acpContentBlock(update.AgentMessageChunk.Content),
		}
		if update.AgentMessageChunk.MessageId != nil {
			base.Update.MessageID = *update.AgentMessageChunk.MessageId
		}
	case update.ToolCall != nil:
		base.Update = &SessionUpdate{
			SessionUpdate: "tool_call",
			ToolCallID:    string(update.ToolCall.ToolCallId),
			Title:         update.ToolCall.Title,
			Kind:          string(update.ToolCall.Kind),
			Status:        string(update.ToolCall.Status),
			Locations:     acpLocations(update.ToolCall.Locations),
			RawInput:      update.ToolCall.RawInput,
			RawOutput:     update.ToolCall.RawOutput,
		}
	case update.ToolCallUpdate != nil:
		status := ""
		if update.ToolCallUpdate.Status != nil {
			status = string(*update.ToolCallUpdate.Status)
		}
		kind := ""
		if update.ToolCallUpdate.Kind != nil {
			kind = string(*update.ToolCallUpdate.Kind)
		}
		title := ""
		if update.ToolCallUpdate.Title != nil {
			title = *update.ToolCallUpdate.Title
		}
		base.Update = &SessionUpdate{
			SessionUpdate: "tool_call_update",
			ToolCallID:    string(update.ToolCallUpdate.ToolCallId),
			Title:         title,
			Kind:          kind,
			Status:        status,
			Locations:     acpLocations(update.ToolCallUpdate.Locations),
			RawInput:      update.ToolCallUpdate.RawInput,
			RawOutput:     update.ToolCallUpdate.RawOutput,
		}
	case update.UsageUpdate != nil:
		base.Update = &SessionUpdate{
			SessionUpdate: "usage_update",
			Usage:         &UsageUpdate{TotalTokens: update.UsageUpdate.Used},
		}
	case update.Plan != nil:
		base.Update = &SessionUpdate{SessionUpdate: "plan"}
	case update.AgentThoughtChunk != nil:
		base.Update = &SessionUpdate{SessionUpdate: "agent_thought_chunk", Content: acpContentBlock(update.AgentThoughtChunk.Content)}
	default:
		return StreamEvent{}
	}
	return base
}

func acpContentBlock(content acp.ContentBlock) *ContentBlock {
	if content.Text != nil {
		return &ContentBlock{Type: "text", Text: content.Text.Text}
	}
	if content.ResourceLink != nil {
		return &ContentBlock{Type: "resource_link", URI: content.ResourceLink.Uri, Mime: stringValuePtr(content.ResourceLink.MimeType)}
	}
	return &ContentBlock{Type: "unknown"}
}

func acpLocations(locations []acp.ToolCallLocation) []ToolCallLocation {
	if len(locations) == 0 {
		return nil
	}
	out := make([]ToolCallLocation, 0, len(locations))
	for _, location := range locations {
		var line *int
		if location.Line != nil {
			value := int(*location.Line)
			line = &value
		}
		out = append(out, ToolCallLocation{Path: location.Path, Line: line})
	}
	return out
}

func normalizeACPStopReason(value acp.StopReason) string {
	if strings.TrimSpace(string(value)) == "" {
		return "end_turn"
	}
	return string(value)
}

func stringValuePtr(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

type acpCallbacks struct {
	conn *ACPConnection
}

func (c *acpCallbacks) SessionUpdate(ctx context.Context, params acp.SessionNotification) error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.handleUpdate(ctx, params)
}

func (c *acpCallbacks) ReadTextFile(context.Context, acp.ReadTextFileRequest) (acp.ReadTextFileResponse, error) {
	return acp.ReadTextFileResponse{}, errors.New("gateway acp client does not expose filesystem reads")
}

func (c *acpCallbacks) WriteTextFile(context.Context, acp.WriteTextFileRequest) (acp.WriteTextFileResponse, error) {
	return acp.WriteTextFileResponse{}, errors.New("gateway acp client does not expose filesystem writes")
}

func (c *acpCallbacks) RequestPermission(ctx context.Context, params acp.RequestPermissionRequest) (acp.RequestPermissionResponse, error) {
	if c != nil && c.conn != nil {
		return c.conn.requestPermission(ctx, params)
	}
	return rejectACPPermission(params), nil
}

func (c *ACPConnection) requestPermission(ctx context.Context, params acp.RequestPermissionRequest) (acp.RequestPermissionResponse, error) {
	c.mu.Lock()
	active := c.active
	c.mu.Unlock()
	if active == nil || active.events == nil {
		return rejectACPPermission(params), nil
	}

	decision := make(chan PermissionVote, 1)
	permission := mapACPPermissionRequest(params, active.sessionID, active.requestID)
	sendACPEvent(ctx, active.events, StreamEvent{
		Type:               agentproto.StreamEventPermissionRequest,
		SessionID:          active.sessionID,
		RequestID:          active.requestID,
		Permission:         permission,
		PermissionDecision: decision,
	})

	select {
	case <-ctx.Done():
		return acp.RequestPermissionResponse{
			Outcome: acp.RequestPermissionOutcome{
				Cancelled: &acp.RequestPermissionOutcomeCancelled{},
			},
		}, nil
	case vote := <-decision:
		if strings.EqualFold(strings.TrimSpace(vote.Outcome), "selected") && strings.TrimSpace(vote.OptionID) != "" {
			if optionID, ok := matchingPermissionOption(params.Options, vote.OptionID); ok {
				return acp.RequestPermissionResponse{
					Outcome: acp.RequestPermissionOutcome{
						Selected: &acp.RequestPermissionOutcomeSelected{OptionId: optionID},
					},
				}, nil
			}
		}
		return acp.RequestPermissionResponse{
			Outcome: acp.RequestPermissionOutcome{
				Cancelled: &acp.RequestPermissionOutcomeCancelled{},
			},
		}, nil
	}
}

func rejectACPPermission(params acp.RequestPermissionRequest) acp.RequestPermissionResponse {
	for _, preferred := range []acp.PermissionOptionKind{
		acp.PermissionOptionKindRejectOnce,
		acp.PermissionOptionKindRejectAlways,
	} {
		for _, option := range params.Options {
			if option.Kind == preferred && strings.TrimSpace(string(option.OptionId)) != "" {
				return acp.RequestPermissionResponse{
					Outcome: acp.RequestPermissionOutcome{
						Selected: &acp.RequestPermissionOutcomeSelected{OptionId: option.OptionId},
					},
				}
			}
		}
	}
	return acp.RequestPermissionResponse{
		Outcome: acp.RequestPermissionOutcome{
			Cancelled: &acp.RequestPermissionOutcomeCancelled{},
		},
	}
}

func matchingPermissionOption(options []acp.PermissionOption, optionID string) (acp.PermissionOptionId, bool) {
	optionID = strings.TrimSpace(optionID)
	for _, option := range options {
		if strings.TrimSpace(string(option.OptionId)) == optionID {
			return option.OptionId, true
		}
	}
	return "", false
}

func mapACPPermissionRequest(params acp.RequestPermissionRequest, sessionID string, requestID string) *PermissionRequest {
	toolCall := params.ToolCall
	permissionID := strings.TrimSpace(requestID)
	if toolID := strings.TrimSpace(string(toolCall.ToolCallId)); toolID != "" {
		if permissionID != "" {
			permissionID += ":"
		}
		permissionID += toolID
	}
	if permissionID == "" {
		permissionID = strings.TrimSpace(string(params.SessionId))
	}
	options := make([]agentproto.PermissionOption, 0, len(params.Options))
	for _, option := range params.Options {
		options = append(options, agentproto.PermissionOption{
			OptionID: strings.TrimSpace(string(option.OptionId)),
			Name:     option.Name,
			Kind:     string(option.Kind),
			Metadata: option.Meta,
		})
	}

	return &PermissionRequest{
		ID:        permissionID,
		SessionID: defaultString(sessionID, string(params.SessionId)),
		ToolCall: agentproto.PermissionToolCall{
			ToolCallID: strings.TrimSpace(string(toolCall.ToolCallId)),
			Title:      stringValuePtr(toolCall.Title),
			Kind:       stringToolKindPtr(toolCall.Kind),
			Status:     stringToolStatusPtr(toolCall.Status),
			Locations:  acpLocations(toolCall.Locations),
			RawInput:   toolCall.RawInput,
			RawOutput:  toolCall.RawOutput,
		},
		Options:  options,
		Metadata: params.Meta,
	}
}

func stringToolKindPtr(value *acp.ToolKind) string {
	if value == nil {
		return ""
	}
	return string(*value)
}

func stringToolStatusPtr(value *acp.ToolCallStatus) string {
	if value == nil {
		return ""
	}
	return string(*value)
}

func defaultString(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func (c *acpCallbacks) CreateTerminal(context.Context, acp.CreateTerminalRequest) (acp.CreateTerminalResponse, error) {
	return acp.CreateTerminalResponse{}, errors.New("gateway acp client does not expose terminals")
}

func (c *acpCallbacks) KillTerminal(context.Context, acp.KillTerminalRequest) (acp.KillTerminalResponse, error) {
	return acp.KillTerminalResponse{}, nil
}

func (c *acpCallbacks) TerminalOutput(context.Context, acp.TerminalOutputRequest) (acp.TerminalOutputResponse, error) {
	return acp.TerminalOutputResponse{}, nil
}

func (c *acpCallbacks) ReleaseTerminal(context.Context, acp.ReleaseTerminalRequest) (acp.ReleaseTerminalResponse, error) {
	return acp.ReleaseTerminalResponse{}, nil
}

func (c *acpCallbacks) WaitForTerminalExit(context.Context, acp.WaitForTerminalExitRequest) (acp.WaitForTerminalExitResponse, error) {
	return acp.WaitForTerminalExitResponse{}, nil
}
