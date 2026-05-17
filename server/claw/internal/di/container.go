package di

import (
	"icoo_claw/server/claw/internal/config"
	"icoo_claw/server/claw/internal/controller"
	"icoo_claw/server/claw/internal/router"
	"icoo_claw/server/claw/internal/service"
	"icoo_claw/server/claw/pkg/agent_sdk"
	"icoo_claw/server/claw/pkg/sessionstore"

	"github.com/gin-gonic/gin"
)

type Container struct {
	Config config.Config
	Router *gin.Engine
}

func NewContainer() (*Container, error) {
	cfg := config.Load()

	sessionClient := sessionstore.NewClient(cfg.SessionStoreURL, nil)
	historyAdapter := agent_sdk.NewHistoryAdapter(sessionClient)
	runtimeFactory := agent_sdk.NewRuntimeFactory(historyAdapter, nil)
	agentService := service.NewAgentService(agent_sdk.NewSDKRunner(runtimeFactory, historyAdapter))
	healthController := controller.NewHealthController()
	agentController := controller.NewAgentController(agentService)

	engine := router.New(router.Controllers{
		Health: healthController,
		Agent:  agentController,
	}, cfg.InternalToken)

	return &Container{Config: cfg, Router: engine}, nil
}

func (c *Container) Run() error {
	return c.Router.Run(c.Config.HTTPAddr)
}
