package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"icoo_claw/server/gateway/internal/client"
	"icoo_claw/server/gateway/internal/dto"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type fakeChatStreamer struct {
	stream func(context.Context, string, dto.SendMessageRequest) (<-chan client.StreamEvent, error)
}

func (f fakeChatStreamer) StreamMessage(ctx context.Context, conversationID string, req dto.SendMessageRequest) (<-chan client.StreamEvent, error) {
	return f.stream(ctx, conversationID, req)
}

type recordingEventPublisher struct {
	mu     sync.Mutex
	events []dto.EventBusEvent
}

func (p *recordingEventPublisher) Publish(_ context.Context, event dto.EventBusEvent) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, event)
	return nil
}

func (p *recordingEventPublisher) snapshot() []dto.EventBusEvent {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]dto.EventBusEvent(nil), p.events...)
}

func TestChatWSControllerStreamsEvents(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.GET("/v1/ws/chat", NewChatWSController(fakeChatStreamer{
		stream: func(ctx context.Context, conversationID string, req dto.SendMessageRequest) (<-chan client.StreamEvent, error) {
			out := make(chan client.StreamEvent, 2)
			out <- client.StreamEvent{
				Type:      "session/update",
				SessionID: "sess_1",
				RequestID: req.RequestID,
				Update:    &client.SessionUpdate{SessionUpdate: "agent_message_chunk", Content: &client.ContentBlock{Type: "text", Text: "hello"}},
			}
			out <- client.StreamEvent{Type: "session/completed", SessionID: "sess_1", RequestID: req.RequestID, StopReason: "end_turn"}
			close(out)
			return out, nil
		},
	}).Serve)

	server := httptest.NewServer(engine)
	defer server.Close()

	conn := mustDialWS(t, server.URL+"/v1/ws/chat")
	defer conn.Close()

	if err := conn.WriteJSON(dto.ChatWSRequest{
		Type:           "chat.start",
		ConversationID: "conv_1",
		RequestID:      "req_1",
		Prompt:         "hello",
	}); err != nil {
		t.Fatalf("write request: %v", err)
	}

	accepted := mustReadWSMessage(t, conn)
	if accepted.Type != "session/accepted" || accepted.ConversationID != "conv_1" || accepted.RequestID != "req_1" {
		t.Fatalf("accepted = %+v", accepted)
	}

	delta := mustReadWSMessage(t, conn)
	if delta.Type != "session/update" || delta.Update == nil || delta.Update.Content == nil || delta.Update.Content.Text != "hello" || delta.SessionID != "sess_1" {
		t.Fatalf("delta = %+v", delta)
	}

	completed := mustReadWSMessage(t, conn)
	if completed.Type != "session/completed" || completed.StopReason != "end_turn" {
		t.Fatalf("completed = %+v", completed)
	}
}

func TestChatWSControllerPublishesEventBusEvents(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	publisher := &recordingEventPublisher{}
	engine := gin.New()
	engine.GET("/v1/ws/chat", NewChatWSController(fakeChatStreamer{
		stream: func(ctx context.Context, conversationID string, req dto.SendMessageRequest) (<-chan client.StreamEvent, error) {
			out := make(chan client.StreamEvent, 1)
			out <- client.StreamEvent{Type: "session/completed", SessionID: "sess_1", RequestID: req.RequestID, StopReason: "end_turn"}
			close(out)
			return out, nil
		},
	}, publisher).Serve)

	server := httptest.NewServer(engine)
	defer server.Close()

	conn := mustDialWS(t, server.URL+"/v1/ws/chat")
	defer conn.Close()

	if err := conn.WriteJSON(dto.ChatWSRequest{
		Type:           "chat.start",
		ConversationID: "conv_1",
		RequestID:      "req_sync",
		Prompt:         "hello",
	}); err != nil {
		t.Fatalf("write request: %v", err)
	}

	_ = mustReadWSMessage(t, conn)
	_ = mustReadWSMessage(t, conn)

	events := publisher.snapshot()
	if len(events) != 3 {
		t.Fatalf("event bus events len = %d, events = %+v", len(events), events)
	}
	if events[0].Direction != "outbound" || events[0].Type != "chat.start" || events[0].ConversationID != "conv_1" {
		t.Fatalf("request event = %+v", events[0])
	}
	if events[1].Direction != "inbound" || events[1].Type != "session/accepted" {
		t.Fatalf("accepted event = %+v", events[1])
	}
	if events[2].Direction != "inbound" || events[2].Type != "session/completed" || events[2].SessionID != "sess_1" {
		t.Fatalf("completed event = %+v", events[2])
	}
}

func TestChatWSControllerPropagatesBusyError(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.GET("/v1/ws/chat", NewChatWSController(fakeChatStreamer{
		stream: func(ctx context.Context, conversationID string, req dto.SendMessageRequest) (<-chan client.StreamEvent, error) {
			return nil, &client.HTTPError{
				Service:    "claw",
				Method:     http.MethodPost,
				Path:       "/internal/agent/run/stream",
				StatusCode: http.StatusConflict,
				Code:       "session_busy",
				Message:    "session is already running",
			}
		},
	}).Serve)

	server := httptest.NewServer(engine)
	defer server.Close()

	conn := mustDialWS(t, server.URL+"/v1/ws/chat")
	defer conn.Close()

	if err := conn.WriteJSON(dto.ChatWSRequest{
		Type:           "chat.start",
		ConversationID: "conv_1",
		RequestID:      "req_busy",
		Prompt:         "hello",
	}); err != nil {
		t.Fatalf("write request: %v", err)
	}

	msg := mustReadWSMessage(t, conn)
	if msg.Type != "session/error" || msg.Code != "session_busy" {
		t.Fatalf("message = %+v", msg)
	}
}

func TestChatWSControllerReportsStreamClosedBeforeCompletion(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.GET("/v1/ws/chat", NewChatWSController(fakeChatStreamer{
		stream: func(ctx context.Context, conversationID string, req dto.SendMessageRequest) (<-chan client.StreamEvent, error) {
			out := make(chan client.StreamEvent, 1)
			out <- client.StreamEvent{
				Type:      "session/update",
				SessionID: "sess_1",
				RequestID: req.RequestID,
				Update:    &client.SessionUpdate{SessionUpdate: "agent_message_chunk", Content: &client.ContentBlock{Type: "text", Text: "partial"}},
			}
			close(out)
			return out, nil
		},
	}).Serve)

	server := httptest.NewServer(engine)
	defer server.Close()

	conn := mustDialWS(t, server.URL+"/v1/ws/chat")
	defer conn.Close()

	if err := conn.WriteJSON(dto.ChatWSRequest{
		Type:           "chat.start",
		ConversationID: "conv_1",
		RequestID:      "req_closed",
		Prompt:         "hello",
	}); err != nil {
		t.Fatalf("write request: %v", err)
	}

	accepted := mustReadWSMessage(t, conn)
	if accepted.Type != "session/accepted" {
		t.Fatalf("accepted = %+v", accepted)
	}

	update := mustReadWSMessage(t, conn)
	if update.Type != "session/update" {
		t.Fatalf("update = %+v", update)
	}

	errFrame := mustReadWSMessage(t, conn)
	if errFrame.Type != "session/error" || errFrame.Code != "stream_closed" || errFrame.SessionID != "sess_1" || errFrame.RequestID != "req_closed" {
		t.Fatalf("error frame = %+v", errFrame)
	}
}

func TestChatWSControllerRespondsToPing(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.GET("/v1/ws/chat", NewChatWSController(fakeChatStreamer{
		stream: func(ctx context.Context, conversationID string, req dto.SendMessageRequest) (<-chan client.StreamEvent, error) {
			ch := make(chan client.StreamEvent)
			close(ch)
			return ch, nil
		},
	}).Serve)

	server := httptest.NewServer(engine)
	defer server.Close()

	conn := mustDialWS(t, server.URL+"/v1/ws/chat")
	defer conn.Close()

	if err := conn.WriteJSON(dto.ChatWSRequest{Type: "ping"}); err != nil {
		t.Fatalf("write ping: %v", err)
	}

	msg := mustReadWSMessage(t, conn)
	if msg.Type != "pong" {
		t.Fatalf("message = %+v", msg)
	}
}

func TestChatWSControllerForwardsPermissionDecision(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.GET("/v1/ws/chat", NewChatWSController(fakeChatStreamer{
		stream: func(ctx context.Context, conversationID string, req dto.SendMessageRequest) (<-chan client.StreamEvent, error) {
			out := make(chan client.StreamEvent, 3)
			go func() {
				decision := make(chan client.PermissionVote, 1)
				out <- client.StreamEvent{
					Type:      "session/request_permission",
					SessionID: "sess_1",
					RequestID: req.RequestID,
					Permission: &client.PermissionRequest{
						ID: "perm_1",
						ToolCall: client.PermissionToolCall{
							ToolCallID: "tool_1",
							Title:      "Sensitive action",
						},
						Options: []client.PermissionOption{
							{OptionID: "allow_once", Name: "Allow", Kind: "allow_once"},
						},
					},
					PermissionDecision: decision,
				}
				vote := <-decision
				out <- client.StreamEvent{
					Type:      "session/update",
					SessionID: "sess_1",
					RequestID: req.RequestID,
					Update: &client.SessionUpdate{
						SessionUpdate: "agent_message_chunk",
						Content:       &client.ContentBlock{Type: "text", Text: vote.OptionID},
					},
				}
				out <- client.StreamEvent{Type: "session/completed", SessionID: "sess_1", RequestID: req.RequestID, StopReason: "end_turn"}
				close(out)
			}()
			return out, nil
		},
	}).Serve)

	server := httptest.NewServer(engine)
	defer server.Close()

	conn := mustDialWS(t, server.URL+"/v1/ws/chat")
	defer conn.Close()

	if err := conn.WriteJSON(dto.ChatWSRequest{
		Type:           "chat.start",
		ConversationID: "conv_1",
		RequestID:      "req_perm",
		Prompt:         "hello",
	}); err != nil {
		t.Fatalf("write request: %v", err)
	}

	accepted := mustReadWSMessage(t, conn)
	if accepted.Type != "session/accepted" {
		t.Fatalf("accepted = %+v", accepted)
	}
	permission := mustReadWSMessage(t, conn)
	if permission.Type != "session/request_permission" || permission.Permission == nil || permission.Permission.ID != "perm_1" {
		t.Fatalf("permission = %+v", permission)
	}

	if err := conn.WriteJSON(dto.ChatWSRequest{
		Type:         "chat.permission_decision",
		RequestID:    "req_perm",
		PermissionID: "perm_1",
		Outcome:      "selected",
		OptionID:     "allow_once",
	}); err != nil {
		t.Fatalf("write permission decision: %v", err)
	}

	decisionAck := mustReadWSMessage(t, conn)
	if decisionAck.Type != "chat.permission_decision.accepted" {
		t.Fatalf("decision ack = %+v", decisionAck)
	}
	update := mustReadWSMessage(t, conn)
	if update.Type != "session/update" || update.Update == nil || update.Update.Content == nil || update.Update.Content.Text != "allow_once" {
		t.Fatalf("update = %+v", update)
	}
	completed := mustReadWSMessage(t, conn)
	if completed.Type != "session/completed" {
		t.Fatalf("completed = %+v", completed)
	}
}

func TestChatWSControllerRejectsMalformedPayload(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.GET("/v1/ws/chat", NewChatWSController(fakeChatStreamer{
		stream: func(ctx context.Context, conversationID string, req dto.SendMessageRequest) (<-chan client.StreamEvent, error) {
			ch := make(chan client.StreamEvent)
			close(ch)
			return ch, nil
		},
	}).Serve)

	server := httptest.NewServer(engine)
	defer server.Close()

	conn := mustDialWS(t, server.URL+"/v1/ws/chat")
	defer conn.Close()

	if err := conn.WriteMessage(websocket.TextMessage, []byte("{")); err != nil {
		t.Fatalf("write malformed frame: %v", err)
	}

	msg := mustReadWSMessage(t, conn)
	if msg.Type != "session/error" || msg.Code != "bad_request" {
		t.Fatalf("message = %+v", msg)
	}
}

func mustDialWS(t *testing.T, target string) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(target, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	return conn
}

func mustReadWSMessage(t *testing.T, conn *websocket.Conn) dto.ChatWSResponse {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	var msg dto.ChatWSResponse
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("read websocket json: %v", err)
	}
	return msg
}
