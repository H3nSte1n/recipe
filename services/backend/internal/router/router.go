// internal/router/router.go
package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yourusername/recipe-app/internal/handler"
	"github.com/yourusername/recipe-app/internal/middleware"
	"github.com/yourusername/recipe-app/pkg/config"
)

type Router struct {
	engine   *gin.Engine
	handlers *handler.Handlers
	auth     *middleware.AuthMiddleware
	config   config.Config
}

func NewRouter(handlers *handler.Handlers, config config.Config) *Router {
	engine := gin.Default()

	// Add basic middleware
	engine.Use(gin.Recovery())
	engine.Use(gin.Logger())

	return &Router{
		engine:   engine,
		handlers: handlers,
		auth:     middleware.NewAuthMiddleware(config.JWT.Secret),
		config:   config,
	}
}

func (r *Router) SetupRoutes() *gin.Engine {
	// API v1 routes
	v1 := r.engine.Group("/api/v1")

	if r.config.Storage.Type == "local" {
		r.engine.Static("/uploads", r.config.Storage.LocalPath)
	}

	// Public routes (no authentication required)
	r.setupPublicRoutes(v1)

	// Protected routes (authentication required)
	protected := v1.Group("")
	protected.Use(r.auth.AuthRequired())
	r.setupProtectedRoutes(protected)

	return r.engine
}

// setupPublicRoutes handles public routes that don't require authentication
func (r *Router) setupPublicRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	{
		auth.POST("/register", r.handlers.UserHandler.Register)
		auth.POST("/login", r.handlers.UserHandler.Login)
		// Could add other public routes like:
		auth.POST("/forgot-password", r.handlers.UserHandler.ForgotPassword)
		auth.POST("/reset-password", r.handlers.UserHandler.ResetPassword)
	}
}

// setupProtectedRoutes handles all routes that require authentication
func (r *Router) setupProtectedRoutes(rg *gin.RouterGroup) {
	users := rg.Group("/users")
	{
		users.GET("", r.handlers.ProfileHandler.Get)
		users.PUT("", r.handlers.ProfileHandler.Update)
		users.DELETE("/me", r.handlers.UserHandler.DeleteAccount) // or /account
	}

	aiConfigs := rg.Group("/ai-configs")
	{
		aiConfigs.GET("", r.handlers.AIConfigHandler.List)
		aiConfigs.POST("", r.handlers.AIConfigHandler.Create)
		aiConfigs.GET("/:id", r.handlers.AIConfigHandler.Get)
		aiConfigs.PUT("/:id", r.handlers.AIConfigHandler.Update)
		aiConfigs.DELETE("/:id", r.handlers.AIConfigHandler.Delete)

		aiConfigs.GET("/default", r.handlers.AIConfigHandler.GetDefault)
		aiConfigs.POST("/:id/set-default", r.handlers.AIConfigHandler.SetDefault)

		aiConfigs.GET("/models", r.handlers.AIConfigHandler.ListModels)
	}

	recipes := rg.Group("/recipes")
	{
		recipes.POST("", r.handlers.RecipeHandler.Create)
		recipes.GET("/:id", r.handlers.RecipeHandler.Get)
		recipes.PUT("/:id", r.handlers.RecipeHandler.Update)
		recipes.DELETE("/:id", r.handlers.RecipeHandler.Delete)

		recipes.GET("", r.handlers.RecipeHandler.ListMine)
		recipes.GET("/public", r.handlers.RecipeHandler.ListPublic)

		imports := recipes.Group("/import")
		{
			imports.POST("/url", r.handlers.RecipeHandler.ImportFromURL)
			imports.POST("/pdf", r.handlers.RecipeHandler.ImportFromPDF)
		}

		parser := recipes.Group("/parser")
		{
			parser.POST("instructions", r.handlers.RecipeHandler.ParsePlainTextInstructions)
		}
	}
}

func (r *Router) Run(addr string) error {
	return r.engine.Run(addr)
}
