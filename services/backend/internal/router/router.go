package router

import (
	"github.com/H3nSte1n/recipe/internal/handler"
	"github.com/H3nSte1n/recipe/internal/middleware"
	"github.com/H3nSte1n/recipe/pkg/config"
	"github.com/H3nSte1n/recipe/pkg/signedurl"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Router struct {
	engine   *gin.Engine
	handlers *handler.Handlers
	auth     *middleware.AuthMiddleware
	config   config.Config
	logger   *zap.Logger
}

func NewRouter(handlers *handler.Handlers, config config.Config, logger *zap.Logger) *Router {
	engine := gin.Default()

	// Bound the in-memory portion of multipart parsing; larger parts spill to
	// temp files and per-handler MaxBytesReader enforces hard size limits.
	engine.MaxMultipartMemory = 10 << 20 // 10 MiB

	engine.Use(middleware.CORS(config.CORS.AllowedOrigins))

	return &Router{
		engine:   engine,
		handlers: handlers,
		auth:     middleware.NewAuthMiddleware(config.JWT.Secret),
		config:   config,
		logger:   logger,
	}
}

func (r *Router) SetupRoutes() *gin.Engine {
	// API v1 routes
	v1 := r.engine.Group("/api/v1")

	if r.config.Storage.Type == "local" {
		// Serve uploads through a signature-checked handler (not a public static
		// mount) so files require a valid short-lived link and are sent with
		// nosniff + attachment.
		signer := signedurl.NewSigner(r.config.JWT.Secret, signedurl.DefaultTTL)
		uploads := handler.NewUploadsHandler(r.config.Storage.LocalPath, signer, r.logger)
		r.engine.GET("/uploads/:filename", uploads.Serve)
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
		auth.POST("/verify-email", r.handlers.UserHandler.VerifyEmail)
		auth.POST("/resend-verification", r.handlers.UserHandler.ResendVerification)
	}
}

// setupProtectedRoutes handles all routes that require authentication
func (r *Router) setupProtectedRoutes(rg *gin.RouterGroup) {
	// Gates state-changing (write) routes behind email verification.
	// Read-only routes stay open to unverified users so a mail outage or
	// unconfigured SMTP provider doesn't lock people out of the app entirely
	// — see registration/email-verification remediation notes.
	requireVerified := middleware.RequireVerified(r.handlers.UserHandler.IsEmailVerified)

	users := rg.Group("/users")
	{
		users.GET("", r.handlers.ProfileHandler.Get)
		users.PUT("", requireVerified, r.handlers.ProfileHandler.Update)
		users.DELETE("/me", requireVerified, r.handlers.UserHandler.DeleteAccount)
		users.GET("/list", r.handlers.UserHandler.ListAll)
	}

	aiConfigs := rg.Group("/ai-configs")
	{
		aiConfigs.GET("", r.handlers.AIConfigHandler.List)
		aiConfigs.POST("", requireVerified, r.handlers.AIConfigHandler.Create)
		aiConfigs.GET("/:id", r.handlers.AIConfigHandler.Get)
		aiConfigs.PUT("/:id", requireVerified, r.handlers.AIConfigHandler.Update)
		aiConfigs.DELETE("/:id", requireVerified, r.handlers.AIConfigHandler.Delete)

		aiConfigs.GET("/default", r.handlers.AIConfigHandler.GetDefault)
		aiConfigs.POST("/:id/set-default", requireVerified, r.handlers.AIConfigHandler.SetDefault)

		aiConfigs.GET("/models", r.handlers.AIConfigHandler.ListModels)
	}

	recipes := rg.Group("/recipes")
	{
		recipes.POST("", requireVerified, r.handlers.RecipeHandler.Create)
		recipes.GET("/:id", r.handlers.RecipeHandler.Get)
		recipes.PUT("/:id", requireVerified, r.handlers.RecipeHandler.Update)
		recipes.DELETE("/:id", requireVerified, r.handlers.RecipeHandler.Delete)

		recipes.GET("", r.handlers.RecipeHandler.ListMine)
		recipes.GET("/public", r.handlers.RecipeHandler.ListPublic)

		imports := recipes.Group("/import")
		{
			imports.POST("/url", requireVerified, r.handlers.RecipeHandler.ImportFromURL)
			imports.POST("/pdf", requireVerified, r.handlers.RecipeHandler.ImportFromPDF)
		}

		parser := recipes.Group("/parser")
		{
			parser.POST("instructions", requireVerified, r.handlers.RecipeHandler.ParsePlainTextInstructions)
		}
	}

	shoppingLists := rg.Group("/shopping-lists")
	{
		shoppingLists.POST("", requireVerified, r.handlers.ShoppingListHandler.Create)
		shoppingLists.GET("", r.handlers.ShoppingListHandler.List)
		shoppingLists.GET("/:id", r.handlers.ShoppingListHandler.Get)
		shoppingLists.PUT("/:id", requireVerified, r.handlers.ShoppingListHandler.Update)
		shoppingLists.DELETE("/:id", requireVerified, r.handlers.ShoppingListHandler.Delete)

		shoppingLists.POST("/:id/items", requireVerified, r.handlers.ShoppingListHandler.AddItem)
		shoppingLists.PUT("/:id/items/:itemId", requireVerified, r.handlers.ShoppingListHandler.UpdateItem)
		shoppingLists.DELETE("/:id/items/:itemId", requireVerified, r.handlers.ShoppingListHandler.DeleteItem)
		shoppingLists.PATCH("/:id/items/:itemId/toggle", requireVerified, r.handlers.ShoppingListHandler.ToggleItem)

		shoppingLists.POST("/:id/add-recipe", requireVerified, r.handlers.ShoppingListHandler.AddRecipe)
		shoppingLists.GET("/:id/sorted", r.handlers.ShoppingListHandler.SortByStore)
	}

	storeChains := rg.Group("/store-chains")
	{
		storeChains.GET("", r.handlers.StoreChainHandler.List)
		storeChains.GET("/:id", r.handlers.StoreChainHandler.Get)
	}
}

func (r *Router) Run(addr string) error {
	return r.engine.Run(addr)
}
