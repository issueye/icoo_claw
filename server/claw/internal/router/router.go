package router

import (
	"icoo_claw/server/claw/internal/controller"

	"github.com/gin-gonic/gin"
)

type Controllers struct {
	Health *controller.HealthController
	Agent  *controller.AgentController
}

func New(controllers Controllers) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())

	engine.GET("/health", controllers.Health.Check)
	engine.POST("/internal/agent/run", controllers.Agent.Run)
	engine.POST("/internal/agent/run/stream", controllers.Agent.RunStream)

	return engine
}
