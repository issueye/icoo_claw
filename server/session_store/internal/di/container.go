package di

import (
	"fmt"

	"icoo_claw/server/session_store/internal/config"
	"icoo_claw/server/session_store/internal/controller"
	"icoo_claw/server/session_store/internal/repository"
	"icoo_claw/server/session_store/internal/router"
	"icoo_claw/server/session_store/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type Container struct {
	Config config.Config
	Router *gin.Engine
	DB     *gorm.DB
}

func NewContainer(cfgPath string) (*Container, error) {
	cfg, err := config.LoadFile(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("load session store config: %w", err)
	}

	db, err := gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open session store sqlite: %w", err)
	}
	if err := repository.AutoMigrate(db); err != nil {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
		return nil, fmt.Errorf("migrate session store sqlite: %w", err)
	}

	sessionRepository := repository.NewGormSessionRepository(db)
	sessionService := service.NewSessionService(sessionRepository)
	healthController := controller.NewHealthController()
	sessionController := controller.NewSessionController(sessionService)
	engine := router.New(router.Controllers{
		Health:  healthController,
		Session: sessionController,
	})

	return &Container{
		Config: cfg,
		Router: engine,
		DB:     db,
	}, nil
}

func (c *Container) Run() error {
	return c.Router.Run(c.Config.HTTPAddr)
}

func (c *Container) Close() error {
	if c == nil || c.DB == nil {
		return nil
	}
	sqlDB, err := c.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
