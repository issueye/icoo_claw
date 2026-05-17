package router

import (
	"icoo_claw/server/claw/internal/controller"
	"icoo_claw/server/claw/internal/middleware"

	"github.com/gin-gonic/gin"
)

type Controllers struct {
	Health *controller.HealthController
	Agent  *controller.AgentController
}

func New(controllers Controllers, internalToken ...string) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())

	engine.GET("/health", controllers.Health.Check)
	protected := engine.Group("/internal")
	token := ""
	if len(internalToken) > 0 {
		token = internalToken[0]
	}
	protected.Use(middleware.InternalToken(token))
	protected.POST("/agent/run", controllers.Agent.Run)
	protected.POST("/agent/run/stream", controllers.Agent.RunStream)

	return engine
}
