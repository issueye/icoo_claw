package router

import (
	"net/http"

	"icoo_claw/server/gateway/internal/controller"

	"github.com/gin-gonic/gin"
)

type Controllers struct {
	Health        *controller.HealthController
	Agent         *controller.AgentController
	AgentInstance *controller.AgentInstanceController
	Session       *controller.SessionController
	Chat          *controller.ChatController
	ChatWS        *controller.ChatWSController
}

func New(controllers Controllers) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(corsMiddleware())

	engine.GET("/health", controllers.Health.Check)
	engine.POST("/v1/agents", controllers.Agent.Create)
	engine.GET("/v1/agents", controllers.Agent.List)
	engine.GET("/v1/agents/:id", controllers.Agent.Get)
	engine.PATCH("/v1/agents/:id", controllers.Agent.Update)
	engine.DELETE("/v1/agents/:id", controllers.Agent.Delete)
	engine.POST("/v1/agent-instances", controllers.AgentInstance.Start)
	engine.GET("/v1/agent-instances", controllers.AgentInstance.List)
	engine.POST("/v1/agent-instances/:id/stop", controllers.AgentInstance.Stop)
	engine.POST("/v1/agent-instances/:id/restart", controllers.AgentInstance.Restart)
	engine.POST("/v1/agent-instances/:id/drain", controllers.AgentInstance.Drain)
	if controllers.Session != nil {
		engine.POST("/v1/sessions", controllers.Session.Create)
		engine.GET("/v1/sessions", controllers.Session.List)
		engine.GET("/v1/sessions/:session_id", controllers.Session.Get)
		engine.PATCH("/v1/sessions/:session_id", controllers.Session.Update)
		engine.DELETE("/v1/sessions/:session_id", controllers.Session.Delete)
		engine.GET("/v1/sessions/:session_id/messages", controllers.Session.ListMessages)
		engine.POST("/v1/sessions/:session_id/messages", controllers.Session.AppendMessages)
		engine.PUT("/v1/sessions/:session_id/messages/snapshot", controllers.Session.ReplaceMessages)
		engine.GET("/v1/sessions/:session_id/runs", controllers.Session.ListRuns)
		engine.POST("/v1/sessions/:session_id/runs", controllers.Session.AppendRuns)
		engine.GET("/v1/sessions/:session_id/runs/:run_id/events", controllers.Session.ListRunEvents)
		engine.POST("/v1/sessions/:session_id/runs/:run_id/events", controllers.Session.AppendRunEvents)
	}
	engine.POST("/v1/conversations", controllers.Chat.CreateConversation)
	engine.GET("/v1/conversations", controllers.Chat.ListConversations)
	engine.GET("/v1/conversations/:id/messages", controllers.Chat.ListMessages)
	engine.POST("/v1/conversations/:id/messages", controllers.Chat.SendMessage)
	if controllers.ChatWS != nil {
		engine.GET("/v1/ws/chat", controllers.ChatWS.Serve)
	}
	engine.DELETE("/v1/conversations/:id", controllers.Chat.DeleteConversation)

	return engine
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			origin = "*"
		}

		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization")
		c.Header("Access-Control-Max-Age", "86400")
		c.Header("Vary", "Origin")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
