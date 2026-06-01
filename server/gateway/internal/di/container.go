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
	ConfigPath      string
	DB              *gorm.DB
	Router          *gin.Engine
	instanceService *service.AgentInstanceService
	taskService     *service.ScheduledTaskService
	syncPublisher   service.SyncPublisher
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
	if err := configureSQLite(db); err != nil {
		return nil, fmt.Errorf("configure gateway db: %w", err)
	}
	if err := db.AutoMigrate(&model.ProviderProfile{}, &model.AgentProfile{}, &model.SkillProfile{}, &model.AgentInstance{}, &model.Conversation{}, &model.ScheduledTask{}, &model.ScheduledTaskRun{}); err != nil {
		return nil, fmt.Errorf("migrate gateway db: %w", err)
	}
	if err := sessionrepo.AutoMigrate(db); err != nil {
		return nil, fmt.Errorf("migrate gateway session db: %w", err)
	}

	agentRepository := repository.NewGormAgentRepository(db)
	providerRepository := repository.NewGormProviderRepository(db)
	skillRepository := repository.NewGormSkillRepository(db)
	instanceRepository := repository.NewGormAgentInstanceRepository(db)
	conversationRepository := repository.NewGormConversationRepository(db)
	taskRepository := repository.NewGormScheduledTaskRepository(db)
	taskRunRepository := repository.NewGormScheduledTaskRunRepository(db)
	sessionRepository := sessionrepo.NewGormSessionRepository(db)
	sessionService := sessionservice.NewSessionService(sessionRepository)
	providerService := service.NewProviderService(providerRepository)
	agentService := service.NewAgentService(agentRepository)
	skillService := service.NewSkillService(cfg.GatewaySkillsRoot(), skillRepository)
	if err := skillService.EnsureLayout(); err != nil {
		return nil, fmt.Errorf("prepare gateway skills directory: %w", err)
	}
	acpRegistry := client.NewACPRegistry()
	agentRunner := client.NewAgentRunner(client.NewClawClient(nil, cfg.InternalToken), acpRegistry)
	instanceService := service.NewAgentInstanceService(cfg, agentRepository, providerRepository, instanceRepository, service.NewLocalProcessSupervisor(acpRegistry), skillService)
	taskService := service.NewScheduledTaskService(taskRepository, taskRunRepository, agentRepository, providerRepository, instanceRepository, instanceService, agentRunner, skillService)
	syncPublisher, err := service.NewGatewaySyncPublisher(cfg)
	if err != nil {
		return nil, fmt.Errorf("create mqtt sync service: %w", err)
	}
	routerPolicy := service.NewDefaultRouterPolicy(conversationRepository, instanceRepository, instanceService)
	chatService := service.NewChatService(
		conversationRepository,
		agentRepository,
		providerRepository,
		routerPolicy,
		service.NewLocalSessionBackend(sessionService),
		agentRunner,
		skillService,
	)
	healthController := controller.NewHealthController()
	providerController := controller.NewProviderController(providerService)
	skillController := controller.NewSkillController(skillService)
	agentController := controller.NewAgentController(agentService)
	instanceController := controller.NewAgentInstanceController(instanceService)
	taskController := controller.NewScheduledTaskController(taskService)
	sessionController := controller.NewSessionController(sessionService)
	chatController := controller.NewChatController(chatService)
	chatWSController := controller.NewChatWSController(chatService, syncPublisher)
	engine := router.New(router.Controllers{
		Health:        healthController,
		Provider:      providerController,
		Skill:         skillController,
		Agent:         agentController,
		AgentInstance: instanceController,
		ScheduledTask: taskController,
		Session:       sessionController,
		Chat:          chatController,
		ChatWS:        chatWSController,
	})

	return &Container{
		Config:          cfg,
		ConfigPath:      cfgPath,
		DB:              db,
		Router:          engine,
		instanceService: instanceService,
		taskService:     taskService,
		syncPublisher:   syncPublisher,
	}, nil
}

func (c *Container) Run() error {
	if c.instanceService != nil {
		c.instanceService.StartHealthLoop(context.Background())
	}
	if c.taskService != nil {
		c.taskService.StartLoop(context.Background())
	}
	if c.syncPublisher != nil {
		defer func() {
			_ = c.syncPublisher.Close()
		}()
	}
	c.printStartupBanner()
	return c.Router.Run(c.Config.HTTPAddr)
}

func (c *Container) printStartupBanner() {
	baseURL := publicBaseURL(c.Config.HTTPAddr)
	tokenState := "not set"
	if strings.TrimSpace(c.Config.InternalToken) != "" {
		tokenState = "set"
	}
	mqttState := "disabled"
	if c.Config.MQTT.Enabled {
		mqttState = displayValue(c.Config.MQTT.BrokerURL, "enabled")
	}

	fmt.Printf(`
  ___                  ___ _               
 |_ _|___ ___  ___    / __| |__ ___ __ __ 
  | |/ __/ _ \/ _ \  | (__| / _ \ V  V /
 |___\___\___/\___/   \___|_\___/ \_/\_/ 

 Icoo Claw Gateway
 ------------------------------------------------------------
 HTTP Listen        %s
 Base URL           %s
 Health             %s/health
 Chat WebSocket     %s/v1/ws/chat
 Config             %s
 Database           %s
 Session API        %s
 Claw runner        %s
 Claw binary        %s
 Claw work dir      %s
 Claw config dir    %s
 Gateway skills     %s
 Agent port range   %d-%d
 Max agents         %d
 Health interval    %s
 Shutdown timeout   %s
 Scheduler          enabled, scans every 30s
 Internal token     %s
 MQTT sync          %s
 ------------------------------------------------------------
 Press Ctrl+C to stop the gateway.
`,
		c.Config.HTTPAddr,
		baseURL,
		baseURL,
		strings.Replace(baseURL, "http", "ws", 1),
		displayValue(c.ConfigPath, "default"),
		displayValue(c.Config.DBPath, "gateway.sqlite"),
		displayValue(c.Config.SessionAPIURL, baseURL),
		displayValue(c.Config.ClawRunnerMode, "sdk"),
		displayValue(c.Config.ClawBinaryPath, "auto"),
		displayValue(c.Config.ClawWorkDir, "icoo_runtime"),
		displayValue(c.Config.ClawConfigDir, "icoo_runtime/claw_configs"),
		displayValue(c.Config.GatewaySkillsRoot(), "icoo_runtime/skills"),
		c.Config.ClawPortStart,
		c.Config.ClawPortEnd,
		c.Config.MaxAgentInstances,
		c.Config.HealthInterval,
		c.Config.ShutdownTimeout,
		tokenState,
		mqttState,
	)
}

func publicBaseURL(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return "http://127.0.0.1:8080"
	}
	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		return strings.TrimRight(addr, "/")
	}
	if strings.HasPrefix(addr, ":") {
		return "http://127.0.0.1" + addr
	}
	if strings.HasPrefix(addr, "0.0.0.0:") {
		return "http://127.0.0.1:" + strings.TrimPrefix(addr, "0.0.0.0:")
	}
	return "http://" + addr
}

func displayValue(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
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

func configureSQLite(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	for _, statement := range []string{
		"PRAGMA busy_timeout = 5000",
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA foreign_keys = ON",
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}
