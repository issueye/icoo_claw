package di

import (
	"fmt"
	"strings"

	"icoo_claw/common/core/agent_sdk/api"
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
	Runner agent_sdk.Runner
	RuntimeFactory *agent_sdk.RuntimeFactory
	HistoryAdapter *agent_sdk.HistoryAdapter
}

func NewContainer(cfgPath string) (*Container, error) {
	cfg, err := config.LoadFile(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("load claw config: %w", err)
	}

	sessionClient := sessionstore.NewClient(cfg.SessionAPIURL, nil)
	historyAdapter := agent_sdk.NewHistoryAdapter(sessionClient)
	runner := agent_sdk.Runner(agent_sdk.NewFakeRunner(historyAdapter))
	var runtimeFactory *agent_sdk.RuntimeFactory
	if strings.ToLower(strings.TrimSpace(cfg.RunnerMode)) != "fake" {
		runtimeFactory = agent_sdk.NewRuntimeFactory(historyAdapter, nil)
		runtimeFactory.SetDefaultProjectRoot(cfg.DefaultProjectRoot)
		runner = agent_sdk.NewSDKRunner(runtimeFactory, historyAdapter)
	}
	agentService := service.NewAgentService(runner)
	healthController := controller.NewHealthController()
	agentController := controller.NewAgentController(agentService)

	engine := router.New(router.Controllers{
		Health: healthController,
		Agent:  agentController,
	}, cfg.InternalToken)

	return &Container{Config: cfg, Router: engine, Runner: runner, RuntimeFactory: runtimeFactory, HistoryAdapter: historyAdapter}, nil
}

func (c *Container) NewRunnerWithPermissionPrompter(prompter api.PermissionPrompter) agent_sdk.Runner {
	if c == nil || c.RuntimeFactory == nil {
		return c.Runner
	}
	factory := agent_sdk.NewRuntimeFactory(c.HistoryAdapter, nil)
	factory.SetDefaultProjectRoot(c.Config.DefaultProjectRoot)
	factory.SetPermissionPrompter(prompter)
	return agent_sdk.NewSDKRunner(factory, c.HistoryAdapter)
}

func (c *Container) Run() error {
	return c.Router.Run(c.Config.HTTPAddr)
}
