// internal/router/router.go
package router

import (
	"github.com/gin-gonic/gin"
	"github.com/yourusername/recipe-app/internal/handler"
	"github.com/yourusername/recipe-app/internal/middleware"
)

type Router struct {
	engine   *gin.Engine
	handlers *handler.Handlers
	auth     *middleware.AuthMiddleware
}

func NewRouter(handlers *handler.Handlers, jwtSecret string) *Router {
	engine := gin.Default()

	// Add basic middleware
	engine.Use(gin.Recovery())
	engine.Use(gin.Logger())

	return &Router{
		engine:   engine,
		handlers: handlers,
		auth:     middleware.NewAuthMiddleware(jwtSecret),
	}
}

func (r *Router) SetupRoutes() *gin.Engine {
	// API v1 routes
	v1 := r.engine.Group("/api/v1")

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
		auth.POST("/register", r.handlers.AuthHandler.Register)
		auth.POST("/login", r.handlers.AuthHandler.Login)
		// Could add other public routes like:
		auth.POST("/forgot-password", r.handlers.AuthHandler.ForgotPassword)
		auth.POST("/reset-password", r.handlers.AuthHandler.ResetPassword)
	}
}

// setupProtectedRoutes handles all routes that require authentication
func (r *Router) setupProtectedRoutes(rg *gin.RouterGroup) {
}

func (r *Router) Run(addr string) error {
	return r.engine.Run(addr)
}
