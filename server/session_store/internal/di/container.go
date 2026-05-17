package di

import (
	"fmt"

	"icoo_claw/server/session_store/internal/config"
	"icoo_claw/server/session_store/internal/controller"
	redkainternal "icoo_claw/server/session_store/internal/redka"
	"icoo_claw/server/session_store/internal/repository"
	"icoo_claw/server/session_store/internal/router"
	"icoo_claw/server/session_store/internal/service"

	"github.com/gin-gonic/gin"
)

type Container struct {
	Config      config.Config
	Router      *gin.Engine
	RedkaServer *redkainternal.Server
}

func NewContainer() (*Container, error) {
	cfg := config.Load()

	redkaServer, err := redkainternal.NewServer(cfg.DBPath, cfg.RESPAddr)
	if err != nil {
		return nil, fmt.Errorf("create redka server: %w", err)
	}

	if err := redkaServer.Start(); err != nil {
		_ = redkaServer.Close()
		return nil, fmt.Errorf("start redka server: %w", err)
	}

	sessionRepository := repository.NewRedkaSessionRepository(redkaServer.DB())
	sessionService := service.NewSessionService(sessionRepository)
	healthController := controller.NewHealthController()
	sessionController := controller.NewSessionController(sessionService)
	engine := router.New(router.Controllers{
		Health:  healthController,
		Session: sessionController,
	})

	return &Container{
		Config:      cfg,
		Router:      engine,
		RedkaServer: redkaServer,
	}, nil
}

func (c *Container) Run() error {
	return c.Router.Run(c.Config.HTTPAddr)
}

func (c *Container) Close() error {
	if c == nil || c.RedkaServer == nil {
		return nil
	}
	return c.RedkaServer.Close()
}
