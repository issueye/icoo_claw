package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"icoo_claw/common/id"
	"icoo_claw/server/gateway/internal/client"
	"icoo_claw/server/gateway/internal/dto"
	"icoo_claw/server/gateway/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type ChatStreamer interface {
	StreamMessage(ctx context.Context, conversationID string, req dto.SendMessageRequest) (<-chan client.StreamEvent, error)
}

type ChatWSController struct {
	chat     ChatStreamer
	sync     service.SyncPublisher
	upgrader websocket.Upgrader
}

const syncPublishTimeout = 500 * time.Millisecond

func NewChatWSController(chat ChatStreamer, syncPublisher ...service.SyncPublisher) *ChatWSController {
	publisher := service.NewNoopSyncPublisher()
	if len(syncPublisher) > 0 && syncPublisher[0] != nil {
		publisher = syncPublisher[0]
	}
	return &ChatWSController{
		chat: chat,
		sync: publisher,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (c *ChatWSController) Serve(ctx *gin.Context) {
	conn, err := c.upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		return
	}

	session := &chatWSSession{
		chat: c.chat,
		sync: c.sync,
		conn: conn,
	}
	session.run(ctx.Request.Context())
}

type chatWSSession struct {
	chat ChatStreamer
	sync service.SyncPublisher
	conn *websocket.Conn

	writeMu sync.Mutex
	stateMu sync.Mutex
	active  *activeStream
}

type activeStream struct {
	conversationID string
	requestID      string
	cancel         context.CancelFunc
	permissionMu   sync.Mutex
	permissions    map[string]chan client.PermissionVote
}

func (s *chatWSSession) run(ctx context.Context) {
	defer s.close()

	s.conn.SetReadLimit(1 << 20)
	for {
		_, payload, err := s.conn.ReadMessage()
		if err != nil {
			s.cancelActive()
			return
		}

		var req dto.ChatWSRequest
		if err := json.Unmarshal(payload, &req); err != nil {
			if writeErr := s.writeJSON(dto.ChatWSResponse{
				Type:  "session/error",
				Code:  "bad_request",
				Error: "invalid websocket payload",
			}); writeErr != nil {
				return
			}
			continue
		}

		s.publishRequest(ctx, req)
		if err := s.handleRequest(ctx, req); err != nil {
			if writeErr := s.writeErrorResponse(req, err); writeErr != nil {
				return
			}
		}
	}
}

func (s *chatWSSession) publishRequest(ctx context.Context, req dto.ChatWSRequest) {
	if s.sync == nil {
		return
	}
	publishCtx, cancel := syncContext(ctx)
	defer cancel()
	_ = s.sync.Publish(publishCtx, dto.SyncEvent{
		ID:             "sync_" + id.Random(),
		Time:           time.Now().UTC(),
		Source:         "gateway-ws",
		Protocol:       "acp",
		Direction:      "outbound",
		Type:           req.Type,
		ConversationID: req.ConversationID,
		RequestID:      req.RequestID,
		Payload:        req,
	})
}

func (s *chatWSSession) handleRequest(ctx context.Context, req dto.ChatWSRequest) error {
	switch req.Type {
	case "ping":
		return s.writeJSON(dto.ChatWSResponse{Type: "pong"})
	case "chat.start":
		return s.startStream(ctx, req)
	case "chat.cancel":
		s.cancelMatching(req.RequestID)
		return s.writeJSON(dto.ChatWSResponse{
			Type:           "cancel.accepted",
			ConversationID: req.ConversationID,
			RequestID:      req.RequestID,
		})
	case "chat.permission_decision":
		return s.handlePermissionDecision(req)
	default:
		return &client.HTTPError{
			Service:    "gateway",
			Method:     "WS",
			Path:       "/v1/ws/chat",
			StatusCode: http.StatusBadRequest,
			Code:       "unsupported_type",
			Message:    "unsupported websocket message type",
		}
	}
}

func (s *chatWSSession) startStream(ctx context.Context, req dto.ChatWSRequest) error {
	req.ConversationID = strings.TrimSpace(req.ConversationID)
	req.Prompt = strings.TrimSpace(req.Prompt)
	req.RequestID = strings.TrimSpace(req.RequestID)

	if req.ConversationID == "" || req.Prompt == "" {
		return &client.HTTPError{
			Service:    "gateway",
			Method:     "WS",
			Path:       "/v1/ws/chat",
			StatusCode: http.StatusBadRequest,
			Code:       "bad_request",
			Message:    "conversation_id and prompt are required",
		}
	}

	s.stateMu.Lock()
	if s.active != nil {
		s.stateMu.Unlock()
		return &client.HTTPError{
			Service:    "gateway",
			Method:     "WS",
			Path:       "/v1/ws/chat",
			StatusCode: http.StatusConflict,
			Code:       "request_in_progress",
			Message:    "a chat request is already running on this socket",
		}
	}
	runCtx, cancel := context.WithCancel(ctx)
	events, err := s.chat.StreamMessage(runCtx, req.ConversationID, dto.SendMessageRequest{
		Prompt:    req.Prompt,
		RequestID: req.RequestID,
		Metadata:  req.Metadata,
	})
	if err != nil {
		s.stateMu.Unlock()
		cancel()
		return err
	}
	state := &activeStream{
		conversationID: req.ConversationID,
		requestID:      req.RequestID,
		cancel:         cancel,
		permissions:    make(map[string]chan client.PermissionVote),
	}
	s.active = state
	s.stateMu.Unlock()

	if err := s.writeJSON(dto.ChatWSResponse{
		Type:           "session/accepted",
		ConversationID: req.ConversationID,
		RequestID:      req.RequestID,
	}); err != nil {
		s.clearActive(state)
		cancel()
		return err
	}

	go s.forwardEvents(runCtx, state, events)
	return nil
}

func (s *chatWSSession) forwardEvents(ctx context.Context, state *activeStream, events <-chan client.StreamEvent) {
	defer s.clearActive(state)

	completed := false
	sessionID := ""
	requestID := state.requestID
	for event := range events {
		if event.SessionID != "" {
			sessionID = event.SessionID
		}
		if event.RequestID != "" {
			requestID = event.RequestID
		}

		switch event.Type {
		case "session/error":
			message := ""
			code := "stream_error"
			if event.Error != nil {
				message = event.Error.Message
				if event.Error.Code != "" {
					code = event.Error.Code
				}
			}
			_ = s.writeJSON(dto.ChatWSResponse{
				Type:           "session/error",
				ConversationID: state.conversationID,
				SessionID:      sessionID,
				RequestID:      requestID,
				Code:           code,
				Error:          defaultOutput(message, "stream error"),
			})
			return
		case "session/completed":
			if !completed {
				_ = s.writeJSON(dto.ChatWSResponse{
					Type:           "session/completed",
					ConversationID: state.conversationID,
					SessionID:      sessionID,
					RequestID:      requestID,
					StopReason:     defaultOutput(event.StopReason, "end_turn"),
				})
				completed = true
			}
		case "session/update":
			_ = s.writeJSON(dto.ChatWSResponse{
				Type:           "session/update",
				ConversationID: state.conversationID,
				SessionID:      sessionID,
				RequestID:      requestID,
				Update:         event.Update,
			})
		case "session/request_permission":
			if event.Permission == nil || event.PermissionDecision == nil {
				continue
			}
			permissionID := event.Permission.ID
			if permissionID == "" {
				permissionID = event.Permission.ToolCall.ToolCallID
			}
			if permissionID == "" {
				event.PermissionDecision <- client.PermissionVote{Outcome: "cancelled"}
				continue
			}
			state.trackPermission(permissionID, event.PermissionDecision)
			if err := s.writeJSON(dto.ChatWSResponse{
				Type:           "session/request_permission",
				ConversationID: state.conversationID,
				SessionID:      sessionID,
				RequestID:      requestID,
				Permission:     event.Permission,
			}); err != nil {
				state.completePermission(permissionID, client.PermissionVote{ID: permissionID, Outcome: "cancelled"})
				return
			}
		}
	}

	if completed {
		return
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		_ = s.writeJSON(dto.ChatWSResponse{
			Type:           "session/completed",
			ConversationID: state.conversationID,
			SessionID:      sessionID,
			RequestID:      requestID,
			StopReason:     "cancelled",
		})
		return
	}
	_ = s.writeJSON(dto.ChatWSResponse{
		Type:           "session/error",
		ConversationID: state.conversationID,
		SessionID:      sessionID,
		RequestID:      requestID,
		Code:           "stream_closed",
		Error:          "agent stream closed before completion",
	})
}

func (s *chatWSSession) cancelMatching(requestID string) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	if s.active == nil {
		return
	}
	if requestID != "" && s.active.requestID != "" && s.active.requestID != requestID {
		return
	}
	s.active.cancelPermissions()
	s.active.cancel()
}

func (s *chatWSSession) cancelActive() {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	if s.active != nil {
		s.active.cancelPermissions()
		s.active.cancel()
	}
}

func (s *chatWSSession) clearActive(state *activeStream) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	if s.active == state {
		state.cancelPermissions()
		s.active = nil
	}
}

func (s *chatWSSession) handlePermissionDecision(req dto.ChatWSRequest) error {
	permissionID := strings.TrimSpace(req.PermissionID)
	if permissionID == "" {
		return &client.HTTPError{
			Service:    "gateway",
			Method:     "WS",
			Path:       "/v1/ws/chat",
			StatusCode: http.StatusBadRequest,
			Code:       "bad_request",
			Message:    "permission_id is required",
		}
	}

	s.stateMu.Lock()
	active := s.active
	s.stateMu.Unlock()
	if active == nil {
		return &client.HTTPError{
			Service:    "gateway",
			Method:     "WS",
			Path:       "/v1/ws/chat",
			StatusCode: http.StatusConflict,
			Code:       "no_active_stream",
			Message:    "no active stream is waiting for permission",
		}
	}

	if ok := active.completePermission(permissionID, client.PermissionVote{
		ID:       permissionID,
		Outcome:  strings.TrimSpace(req.Outcome),
		OptionID: strings.TrimSpace(req.OptionID),
	}); !ok {
		return &client.HTTPError{
			Service:    "gateway",
			Method:     "WS",
			Path:       "/v1/ws/chat",
			StatusCode: http.StatusNotFound,
			Code:       "permission_not_found",
			Message:    "permission request is no longer pending",
		}
	}

	return s.writeJSON(dto.ChatWSResponse{
		Type:           "chat.permission_decision.accepted",
		ConversationID: req.ConversationID,
		RequestID:      req.RequestID,
		Metadata:       map[string]any{"permission_id": permissionID},
	})
}

func (a *activeStream) trackPermission(id string, decisions chan client.PermissionVote) {
	a.permissionMu.Lock()
	defer a.permissionMu.Unlock()
	if a.permissions == nil {
		a.permissions = make(map[string]chan client.PermissionVote)
	}
	a.permissions[id] = decisions
}

func (a *activeStream) completePermission(id string, vote client.PermissionVote) bool {
	a.permissionMu.Lock()
	decisions := a.permissions[id]
	if decisions != nil {
		delete(a.permissions, id)
	}
	a.permissionMu.Unlock()
	if decisions == nil {
		return false
	}
	if vote.ID == "" {
		vote.ID = id
	}
	if vote.Outcome == "" {
		vote.Outcome = "cancelled"
	}
	decisions <- vote
	return true
}

func (a *activeStream) cancelPermissions() {
	a.permissionMu.Lock()
	pending := a.permissions
	a.permissions = make(map[string]chan client.PermissionVote)
	a.permissionMu.Unlock()
	for id, decisions := range pending {
		decisions <- client.PermissionVote{ID: id, Outcome: "cancelled"}
	}
}

func (s *chatWSSession) close() {
	_ = s.conn.Close()
}

func (s *chatWSSession) writeErrorResponse(req dto.ChatWSRequest, err error) error {
	code := "store_error"
	message := err.Error()

	var downstream *client.HTTPError
	switch {
	case errors.As(err, &downstream):
		code = gatewayCodeForDownstream(downstream)
		message = downstream.Message
	case errors.Is(err, context.Canceled):
		code = "cancelled"
		message = "request cancelled"
	}

	return s.writeJSON(dto.ChatWSResponse{
		Type:           "session/error",
		ConversationID: req.ConversationID,
		RequestID:      req.RequestID,
		Code:           code,
		Error:          message,
	})
}

func (s *chatWSSession) writeJSON(payload dto.ChatWSResponse) error {
	s.publishResponse(payload)
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.conn.WriteJSON(payload)
}

func (s *chatWSSession) publishResponse(payload dto.ChatWSResponse) {
	if s.sync == nil {
		return
	}
	publishCtx, cancel := syncContext(context.Background())
	defer cancel()
	_ = s.sync.Publish(publishCtx, dto.SyncEvent{
		Time:           time.Now().UTC(),
		Source:         "gateway-ws",
		Protocol:       "acp",
		Direction:      "inbound",
		Type:           payload.Type,
		ConversationID: payload.ConversationID,
		SessionID:      payload.SessionID,
		RequestID:      payload.RequestID,
		Payload:        payload,
	})
}

func syncContext(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, syncPublishTimeout)
}

func defaultOutput(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
