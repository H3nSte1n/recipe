package main

import (
	"github.com/gin-gonic/gin"
	"github.com/yourusername/recipe-app/internal/handler"
	"github.com/yourusername/recipe-app/internal/repository"
	"github.com/yourusername/recipe-app/internal/router"
	"github.com/yourusername/recipe-app/internal/service"
	"github.com/yourusername/recipe-app/pkg/config"
	"github.com/yourusername/recipe-app/pkg/database"
	"go.uber.org/zap"
	"log"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig(".")
	if err != nil {
		log.Fatal("Cannot load config:", err)
	}

	if err := database.MigrateDB(&cfg); err != nil {
		log.Fatal("Could not migrate database:", err)
	}

	// Initialize logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Set Gin mode
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize database connection
	db, err := database.NewPostgresConnection(&cfg)
	if err != nil {
		logger.Fatal("Failed to connect to database:", zap.Error(err))
	}

	// Initialize repositories
	repos := repository.NewRepositories(db)

	// Initialize services
	services := service.NewServices(repos, cfg)

	// Initialize handlers
	handlers := handler.NewHandlers(services, logger)

	// Initialize router
	r := router.NewRouter(handlers, cfg.JWTSecret)
	r.SetupRoutes()

	// Start server
	logger.Info("Starting server on port " + cfg.AppPort)
	if err := r.Run(":" + cfg.AppPort); err != nil {
		logger.Fatal("Failed to start server:", zap.Error(err))
	}
}
