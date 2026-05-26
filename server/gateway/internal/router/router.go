package router

import (
	"net/http"

	"icoo_claw/server/gateway/internal/controller"

	"github.com/gin-gonic/gin"
)

type Controllers struct {
	Health        *controller.HealthController
	Provider      *controller.ProviderController
	Skill         *controller.SkillController
	Agent         *controller.AgentController
	AgentInstance *controller.AgentInstanceController
	ScheduledTask *controller.ScheduledTaskController
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
	if controllers.Provider != nil {
		engine.POST("/v1/providers", controllers.Provider.Create)
		engine.GET("/v1/providers", controllers.Provider.List)
		engine.GET("/v1/providers/:id", controllers.Provider.Get)
		engine.PATCH("/v1/providers/:id", controllers.Provider.Update)
		engine.DELETE("/v1/providers/:id", controllers.Provider.Delete)
	}
	if controllers.Skill != nil {
		engine.POST("/v1/skills", controllers.Skill.Create)
		engine.GET("/v1/skills", controllers.Skill.List)
		engine.GET("/v1/skills/:id", controllers.Skill.Get)
		engine.GET("/v1/skills/:id/download", controllers.Skill.Download)
		engine.PATCH("/v1/skills/:id", controllers.Skill.Update)
		engine.DELETE("/v1/skills/:id", controllers.Skill.Delete)
		engine.POST("/v1/skills/install", controllers.Skill.Create)
		engine.POST("/v1/skills/sync", controllers.Skill.Sync)
	}
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
	engine.DELETE("/v1/agent-instances/:id", controllers.AgentInstance.Delete)
	if controllers.ScheduledTask != nil {
		engine.POST("/v1/scheduled-tasks", controllers.ScheduledTask.Create)
		engine.GET("/v1/scheduled-tasks", controllers.ScheduledTask.List)
		engine.GET("/v1/scheduled-tasks/:id", controllers.ScheduledTask.Get)
		engine.GET("/v1/scheduled-tasks/:id/runs", controllers.ScheduledTask.ListRuns)
		engine.PATCH("/v1/scheduled-tasks/:id", controllers.ScheduledTask.Update)
		engine.DELETE("/v1/scheduled-tasks/:id", controllers.ScheduledTask.Delete)
	}
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
