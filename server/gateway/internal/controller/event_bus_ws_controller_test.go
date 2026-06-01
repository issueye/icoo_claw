package controller

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"icoo_claw/server/gateway/internal/dto"
	"icoo_claw/server/gateway/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func TestEventBusWSControllerStreamsFilteredEvents(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	bus := service.NewEventBus(10)
	_ = bus.Publish(context.Background(), dto.EventBusEvent{Protocol: "acp", Type: "session/update"})

	engine := gin.New()
	engine.GET("/v1/ws/events", NewEventBusWSController(bus).Serve)
	server := httptest.NewServer(engine)
	defer server.Close()

	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/v1/ws/events?protocol=acp", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	var event dto.EventBusEvent
	if err := conn.ReadJSON(&event); err != nil {
		t.Fatalf("read event: %v", err)
	}
	if event.Protocol != "acp" || event.Type != "session/update" {
		t.Fatalf("event = %+v", event)
	}
}
