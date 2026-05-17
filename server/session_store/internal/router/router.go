package router

import (
	"icoo_claw/server/session_store/internal/controller"

	"github.com/gin-gonic/gin"
)

type Controllers struct {
	Health  *controller.HealthController
	Session *controller.SessionController
}

func New(controllers Controllers) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())

	engine.GET("/health", controllers.Health.Check)
	engine.POST("/v1/sessions", controllers.Session.Create)
	engine.GET("/v1/sessions/:session_id", controllers.Session.Get)
	engine.PATCH("/v1/sessions/:session_id", controllers.Session.Update)
	engine.DELETE("/v1/sessions/:session_id", controllers.Session.Delete)
	engine.GET("/v1/sessions/:session_id/messages", controllers.Session.ListMessages)
	engine.POST("/v1/sessions/:session_id/messages", controllers.Session.AppendMessages)
	engine.PUT("/v1/sessions/:session_id/messages/snapshot", controllers.Session.ReplaceMessages)

	return engine
}
