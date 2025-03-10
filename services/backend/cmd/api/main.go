package main

import (
	"github.com/gin-gonic/gin"
	"github.com/yourusername/recipe-app/internal/handler"
	"github.com/yourusername/recipe-app/internal/repository"
	"github.com/yourusername/recipe-app/internal/router"
	"github.com/yourusername/recipe-app/internal/service"
	"github.com/yourusername/recipe-app/pkg/ai"
	"github.com/yourusername/recipe-app/pkg/config"
	"github.com/yourusername/recipe-app/pkg/database"
	"github.com/yourusername/recipe-app/pkg/storage"
	"go.uber.org/zap"
	"log"
	"os"
)

func main() {
	// Load configuration
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	cfg, err := config.LoadConfig(env)
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
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize database connection
	db, err := database.NewPostgresConnection(&cfg)
	if err != nil {
		logger.Fatal("Failed to connect to database:", zap.Error(err))
	}

	// Initialize file store
	fileStore, err := storage.NewFileStore(&cfg)
	if err != nil {
		logger.Fatal("Failed to initialize file store:", zap.Error(err))
	}

	// Inizilize AI model factory
	aiModelFactory := ai.NewModelFactory(&cfg, logger)

	// Initialize repositories
	repos := repository.NewRepositories(db)

	// Initialize services
	services := service.NewServices(repos, cfg, fileStore, logger, *aiModelFactory)

	// Initialize handlers
	handlers := handler.NewHandlers(services, logger)

	// Initialize router
	r := router.NewRouter(handlers, cfg)
	r.SetupRoutes()

	// Start server
	logger.Info("Starting server on port " + cfg.App.Port)
	if err := r.Run(":" + cfg.App.Port); err != nil {
		logger.Fatal("Failed to start server:", zap.Error(err))
	}
}
