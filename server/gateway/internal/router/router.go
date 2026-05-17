package router

import (
	"icoo_claw/server/gateway/internal/controller"

	"github.com/gin-gonic/gin"
)

type Controllers struct {
	Health        *controller.HealthController
	Agent         *controller.AgentController
	AgentInstance *controller.AgentInstanceController
	Chat          *controller.ChatController
}

func New(controllers Controllers) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())

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
	engine.POST("/v1/conversations", controllers.Chat.CreateConversation)
	engine.GET("/v1/conversations", controllers.Chat.ListConversations)
	engine.GET("/v1/conversations/:id/messages", controllers.Chat.ListMessages)
	engine.POST("/v1/conversations/:id/messages", controllers.Chat.SendMessage)
	engine.POST("/v1/conversations/:id/stream", controllers.Chat.StreamMessage)
	engine.DELETE("/v1/conversations/:id", controllers.Chat.DeleteConversation)

	return engine
}
