package di

import (
	"context"
	"fmt"

	"icoo_claw/server/gateway/internal/client"
	"icoo_claw/server/gateway/internal/config"
	"icoo_claw/server/gateway/internal/controller"
	"icoo_claw/server/gateway/internal/model"
	"icoo_claw/server/gateway/internal/repository"
	"icoo_claw/server/gateway/internal/router"
	"icoo_claw/server/gateway/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type Container struct {
	Config          config.Config
	DB              *gorm.DB
	Router          *gin.Engine
	instanceService *service.AgentInstanceService
}

func NewContainer() (*Container, error) {
	cfg := config.Load()

	db, err := gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open gateway db: %w", err)
	}
	if err := db.AutoMigrate(&model.AgentProfile{}, &model.AgentInstance{}, &model.Conversation{}); err != nil {
		return nil, fmt.Errorf("migrate gateway db: %w", err)
	}

	agentRepository := repository.NewGormAgentRepository(db)
	instanceRepository := repository.NewGormAgentInstanceRepository(db)
	conversationRepository := repository.NewGormConversationRepository(db)
	agentService := service.NewAgentService(agentRepository)
	instanceService := service.NewAgentInstanceService(cfg, agentRepository, instanceRepository, service.NewLocalProcessSupervisor())
	routerPolicy := service.NewDefaultRouterPolicy(conversationRepository, instanceRepository, instanceService)
	chatService := service.NewChatService(
		conversationRepository,
		agentRepository,
		routerPolicy,
		client.NewSessionStoreClient(cfg.SessionStoreURL, nil),
		client.NewClawClient(nil, cfg.InternalToken),
	)
	healthController := controller.NewHealthController()
	agentController := controller.NewAgentController(agentService)
	instanceController := controller.NewAgentInstanceController(instanceService)
	chatController := controller.NewChatController(chatService)
	engine := router.New(router.Controllers{
		Health:        healthController,
		Agent:         agentController,
		AgentInstance: instanceController,
		Chat:          chatController,
	})

	return &Container{
		Config:          cfg,
		DB:              db,
		Router:          engine,
		instanceService: instanceService,
	}, nil
}

func (c *Container) Run() error {
	if c.instanceService != nil {
		c.instanceService.StartHealthLoop(context.Background())
	}
	return c.Router.Run(c.Config.HTTPAddr)
}
