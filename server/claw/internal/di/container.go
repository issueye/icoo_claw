package di

import (
	"fmt"
	"strings"

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

func NewContainer(cfgPath string) (*Container, error) {
	cfg, err := config.LoadFile(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("load claw config: %w", err)
	}

	sessionClient := sessionstore.NewClient(cfg.SessionStoreURL, nil)
	historyAdapter := agent_sdk.NewHistoryAdapter(sessionClient)
	runner := agent_sdk.Runner(agent_sdk.NewFakeRunner(historyAdapter))
	if strings.ToLower(strings.TrimSpace(cfg.RunnerMode)) != "fake" {
		runtimeFactory := agent_sdk.NewRuntimeFactory(historyAdapter, nil)
		runner = agent_sdk.NewSDKRunner(runtimeFactory, historyAdapter)
	}
	agentService := service.NewAgentService(runner)
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
