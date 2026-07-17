package main

import (
	"github.com/H3nSte1n/recipe/internal/handler"
	"github.com/H3nSte1n/recipe/internal/repository"
	"github.com/H3nSte1n/recipe/internal/router"
	"github.com/H3nSte1n/recipe/internal/service"
	"github.com/H3nSte1n/recipe/pkg/ai"
	"github.com/H3nSte1n/recipe/pkg/config"
	"github.com/H3nSte1n/recipe/pkg/crypto"
	"github.com/H3nSte1n/recipe/pkg/database"
	"github.com/H3nSte1n/recipe/pkg/storage"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"log"
	"os"
)

func main() {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	cfg, err := config.LoadConfig(env)
	if err != nil {
		log.Fatal("Cannot load config:", err)
	}

	// Fail closed on a weak/default/missing JWT signing secret.
	if err := cfg.Validate(); err != nil {
		log.Fatal("Invalid configuration: ", err)
	}

	if err := database.MigrateDB(&cfg); err != nil {
		log.Fatal("Could not migrate database:", err)
	}

	var logger *zap.Logger
	if cfg.App.Env == "production" {
		logger, _ = zap.NewProduction()
		gin.SetMode(gin.ReleaseMode)
	} else {
		logger, _ = zap.NewDevelopment()
	}
	defer func() { _ = logger.Sync() }()

	db, err := database.NewPostgresConnection(&cfg)
	if err != nil {
		logger.Fatal("Failed to connect to database:", zap.Error(err))
	}

	fileStore, err := storage.NewFileStore(&cfg)
	if err != nil {
		logger.Fatal("Failed to initialize file store:", zap.Error(err))
	}

	aiModelFactory := ai.NewModelFactory(&cfg, logger)

	// Fail closed: secrets at rest (user AI API keys) require an encryption key.
	cipher, err := crypto.NewCipher(cfg.Security.EncryptionKey)
	if err != nil {
		logger.Fatal("Failed to initialize encryption (set SECURITY_ENCRYPTION_KEY):", zap.Error(err))
	}

	repos := repository.NewRepositories(db)

	services := service.NewServices(repos, cfg, fileStore, logger, *aiModelFactory, cipher)

	handlers := handler.NewHandlers(services, logger)

	r := router.NewRouter(handlers, cfg, logger, repos.UserRepository)
	r.SetupRoutes()

	logger.Info("Starting server on port " + cfg.App.Port)
	if err := r.Run(":" + cfg.App.Port); err != nil {
		logger.Fatal("Failed to start server:", zap.Error(err))
	}
}
