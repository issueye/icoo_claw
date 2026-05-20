package di

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"icoo_claw/server/gateway/internal/client"
	"icoo_claw/server/gateway/internal/config"
	"icoo_claw/server/gateway/internal/controller"
	"icoo_claw/server/gateway/internal/model"
	"icoo_claw/server/gateway/internal/repository"
	"icoo_claw/server/gateway/internal/router"
	"icoo_claw/server/gateway/internal/service"
	sessionrepo "icoo_claw/server/gateway/internal/sessionstore/repository"
	sessionservice "icoo_claw/server/gateway/internal/sessionstore/service"

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

func NewContainer(cfgPath string) (*Container, error) {
	cfg, err := config.LoadFile(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("load gateway config: %w", err)
	}
	if err := ensureSQLiteParentDir(cfg.DBPath); err != nil {
		return nil, fmt.Errorf("prepare gateway db path: %w", err)
	}

	db, err := gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open gateway db: %w", err)
	}
	if err := db.AutoMigrate(&model.AgentProfile{}, &model.AgentInstance{}, &model.Conversation{}); err != nil {
		return nil, fmt.Errorf("migrate gateway db: %w", err)
	}
	if err := sessionrepo.AutoMigrate(db); err != nil {
		return nil, fmt.Errorf("migrate gateway session db: %w", err)
	}

	agentRepository := repository.NewGormAgentRepository(db)
	instanceRepository := repository.NewGormAgentInstanceRepository(db)
	conversationRepository := repository.NewGormConversationRepository(db)
	sessionRepository := sessionrepo.NewGormSessionRepository(db)
	sessionService := sessionservice.NewSessionService(sessionRepository)
	agentService := service.NewAgentService(agentRepository)
	instanceService := service.NewAgentInstanceService(cfg, agentRepository, instanceRepository, service.NewLocalProcessSupervisor())
	routerPolicy := service.NewDefaultRouterPolicy(conversationRepository, instanceRepository, instanceService)
	chatService := service.NewChatService(
		conversationRepository,
		agentRepository,
		routerPolicy,
		service.NewLocalSessionBackend(sessionService),
		client.NewClawClient(nil, cfg.InternalToken),
	)
	healthController := controller.NewHealthController()
	agentController := controller.NewAgentController(agentService)
	instanceController := controller.NewAgentInstanceController(instanceService)
	sessionController := controller.NewSessionController(sessionService)
	chatController := controller.NewChatController(chatService)
	chatWSController := controller.NewChatWSController(chatService)
	engine := router.New(router.Controllers{
		Health:        healthController,
		Agent:         agentController,
		AgentInstance: instanceController,
		Session:       sessionController,
		Chat:          chatController,
		ChatWS:        chatWSController,
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

func ensureSQLiteParentDir(dbPath string) error {
	path := strings.TrimSpace(dbPath)
	if path == "" || path == ":memory:" {
		return nil
	}
	if strings.HasPrefix(path, "file:") {
		path = strings.TrimPrefix(path, "file:")
		if queryIndex := strings.Index(path, "?"); queryIndex >= 0 {
			path = path[:queryIndex]
		}
		if path == "" || path == ":memory:" {
			return nil
		}
	}
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}
