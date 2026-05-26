package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	acp "github.com/coder/acp-go-sdk"
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

	var output strings.Builder
	stopReason := "stream_closed"
	for event := range events {
		switch event.Type {
		case "session/update":
			if event.Update != nil && event.Update.SessionUpdate == "agent_message_chunk" && event.Update.Content != nil {
				output.WriteString(event.Update.Content.Text)
			}
		case "session/completed":
			stopReason = defaultString(event.StopReason, "end_turn")
			return &RunResponse{
				SessionID:  defaultString(event.SessionID, req.SessionID),
				RequestID:  defaultString(event.RequestID, req.RequestID),
				Output:     output.String(),
				StopReason: stopReason,
			}, nil
		case "session/error":
			message := "acp stream error"
			if event.Error != nil && strings.TrimSpace(event.Error.Message) != "" {
				message = event.Error.Message
			}
			return nil, errors.New(message)
		}
	}

	return nil, errors.New("acp stream closed before completion")
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

	mu     sync.Mutex
	active *acpActiveStream
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
				_ = c.conn.Cancel(context.Background(), acp.CancelNotification{SessionId: acp.SessionId(req.SessionID)})
			case <-done:
			}
		}()
		defer close(done)

		messageID := req.RequestID
		promptReq := acp.PromptRequest{
			SessionId: acp.SessionId(req.SessionID),
			Prompt:    []acp.ContentBlock{acp.TextBlock(req.Prompt)},
			Meta: map[string]any{
				"request_id":     req.RequestID,
				"agent":          req.Agent,
				"tool_whitelist": req.ToolWhitelist,
				"force_skills":   req.ForceSkills,
				"metadata":       req.Metadata,
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

func defaultString(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
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

func (c *acpCallbacks) RequestPermission(context.Context, acp.RequestPermissionRequest) (acp.RequestPermissionResponse, error) {
	return acp.RequestPermissionResponse{}, errors.New("gateway acp client cannot grant permissions")
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
