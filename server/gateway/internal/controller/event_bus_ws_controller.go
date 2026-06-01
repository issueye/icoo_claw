package controller

import (
	"context"
	"net/http"

	"icoo_claw/server/gateway/internal/dto"
	"icoo_claw/server/gateway/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type EventSubscriber interface {
	Subscribe(ctx context.Context, filter service.EventBusFilter) (<-chan dto.EventBusEvent, func())
}

type EventBusWSController struct {
	bus      EventSubscriber
	upgrader websocket.Upgrader
}

func NewEventBusWSController(bus EventSubscriber) *EventBusWSController {
	return &EventBusWSController{
		bus: bus,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (c *EventBusWSController) Serve(ctx *gin.Context) {
	conn, err := c.upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	events, unsubscribe := c.bus.Subscribe(ctx.Request.Context(), service.EventBusFilter{
		Protocol: ctx.Query("protocol"),
	})
	defer unsubscribe()

	for event := range events {
		if err := conn.WriteJSON(event); err != nil {
			return
		}
	}
}
