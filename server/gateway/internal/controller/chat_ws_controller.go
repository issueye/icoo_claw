package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"

	"icoo_claw/server/gateway/internal/client"
	"icoo_claw/server/gateway/internal/dto"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type ChatStreamer interface {
	StreamMessage(ctx context.Context, conversationID string, req dto.SendMessageRequest) (<-chan client.StreamEvent, error)
}

type ChatWSController struct {
	chat     ChatStreamer
	upgrader websocket.Upgrader
}

func NewChatWSController(chat ChatStreamer) *ChatWSController {
	return &ChatWSController{
		chat: chat,
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
		conn: conn,
	}
	session.run(ctx.Request.Context())
}

type chatWSSession struct {
	chat ChatStreamer
	conn *websocket.Conn

	writeMu sync.Mutex
	stateMu sync.Mutex
	active  *activeStream
}

type activeStream struct {
	conversationID string
	requestID      string
	cancel         context.CancelFunc
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
				Type:  "message.error",
				Code:  "bad_request",
				Error: "invalid websocket payload",
			}); writeErr != nil {
				return
			}
			continue
		}

		if err := s.handleRequest(ctx, req); err != nil {
			if writeErr := s.writeErrorResponse(req, err); writeErr != nil {
				return
			}
		}
	}
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
	}
	s.active = state
	s.stateMu.Unlock()

	if err := s.writeJSON(dto.ChatWSResponse{
		Type:           "session.accepted",
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
	for event := range events {
		requestID := state.requestID
		if event.RequestID != "" {
			requestID = event.RequestID
		}

		switch event.Type {
		case "error":
			_ = s.writeJSON(dto.ChatWSResponse{
				Type:           "message.error",
				ConversationID: state.conversationID,
				SessionID:      event.SessionID,
				RequestID:      requestID,
				Code:           "stream_error",
				Error:          defaultOutput(event.Output, "stream error"),
			})
			return
		case "message_stop", "agent_stop":
			if !completed {
				_ = s.writeJSON(dto.ChatWSResponse{
					Type:           "message.completed",
					ConversationID: state.conversationID,
					SessionID:      event.SessionID,
					RequestID:      requestID,
					StopReason:     "end_turn",
				})
				completed = true
			}
		default:
			if event.Output != "" {
				_ = s.writeJSON(dto.ChatWSResponse{
					Type:           "message.delta",
					ConversationID: state.conversationID,
					SessionID:      event.SessionID,
					RequestID:      requestID,
					Output:         event.Output,
				})
			}
		}
	}

	if completed {
		return
	}
	stopReason := "stream_closed"
	if errors.Is(ctx.Err(), context.Canceled) {
		stopReason = "cancelled"
	}
	_ = s.writeJSON(dto.ChatWSResponse{
		Type:           "message.completed",
		ConversationID: state.conversationID,
		RequestID:      state.requestID,
		StopReason:     stopReason,
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
	s.active.cancel()
}

func (s *chatWSSession) cancelActive() {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	if s.active != nil {
		s.active.cancel()
	}
}

func (s *chatWSSession) clearActive(state *activeStream) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	if s.active == state {
		s.active = nil
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
		Type:           "message.error",
		ConversationID: req.ConversationID,
		RequestID:      req.RequestID,
		Code:           code,
		Error:          message,
	})
}

func (s *chatWSSession) writeJSON(payload dto.ChatWSResponse) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.conn.WriteJSON(payload)
}

func defaultOutput(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
