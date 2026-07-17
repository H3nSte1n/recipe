package router

import (
	"os"
	"strings"
	"time"

	"github.com/H3nSte1n/recipe/internal/handler"
	"github.com/H3nSte1n/recipe/internal/middleware"
	"github.com/H3nSte1n/recipe/pkg/config"
	"github.com/H3nSte1n/recipe/pkg/signedurl"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// Rate limits for the public auth endpoints, keyed per client IP (see middleware.RateLimit).
// These endpoints have no auth token to key on, so IP is the only signal available; limits are
// generous enough for normal use (typos, retries) but bound how fast an attacker can guess
// passwords, spam registrations, or trigger password-reset emails against one IP.
var (
	loginRateLimit          = rate.Every(12 * time.Second) // 5 requests/min
	loginRateBurst          = 5
	registerRateLimit       = rate.Every(20 * time.Second) // 3 requests/min
	registerRateBurst       = 3
	forgotPasswordRateLimit = rate.Every(20 * time.Second) // 3 requests/min
	forgotPasswordRateBurst = 3
)

// defaultTrustedProxies is used when TRUSTED_PROXIES is unset. The production deployment plan
// puts this app behind a local nginx reverse proxy on the same host, so loopback is the correct
// default; override via TRUSTED_PROXIES (comma-separated IPs/CIDRs) if that topology changes.
var defaultTrustedProxies = []string{"127.0.0.1"}

// trustedProxiesFromEnv returns the proxy IPs/CIDRs Gin should trust when reading
// X-Forwarded-For/X-Real-IP. By default Gin trusts every proxy, which lets any client forge
// those headers; this restricts trust to known proxies only.
func trustedProxiesFromEnv() []string {
	raw := os.Getenv("TRUSTED_PROXIES")
	if raw == "" {
		return defaultTrustedProxies
	}

	proxies := make([]string, 0)
	for _, p := range strings.Split(raw, ",") {
		if p = strings.TrimSpace(p); p != "" {
			proxies = append(proxies, p)
		}
	}
	if len(proxies) == 0 {
		return defaultTrustedProxies
	}
	return proxies
}

type Router struct {
	engine   *gin.Engine
	handlers *handler.Handlers
	auth     *middleware.AuthMiddleware
	config   config.Config
	logger   *zap.Logger
}

func NewRouter(handlers *handler.Handlers, config config.Config, logger *zap.Logger, revocations middleware.TokenRevocationChecker) *Router {
	engine := gin.Default()

	// Gin trusts every proxy by default, which makes X-Forwarded-For/X-Real-IP
	// attacker-controlled from any client. Restrict trust to known proxies so those headers
	// (and IP-based rate limiting built on top of them) reflect the real client.
	if err := engine.SetTrustedProxies(trustedProxiesFromEnv()); err != nil {
		logger.Warn("invalid TRUSTED_PROXIES, falling back to no trusted proxies", zap.Error(err))
	}

	// Bound the in-memory portion of multipart parsing; larger parts spill to
	// temp files and per-handler MaxBytesReader enforces hard size limits.
	engine.MaxMultipartMemory = 10 << 20 // 10 MiB

	engine.Use(middleware.BodySizeLimit(middleware.MaxJSONBodyBytes))
	engine.Use(middleware.CORS(config.CORS.AllowedOrigins))

	return &Router{
		engine:   engine,
		handlers: handlers,
		auth:     middleware.NewAuthMiddleware(config.JWT.Secret, config.JWT.Issuer, config.JWT.Audience, revocations),
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
		// Login/register/forgot-password are rate-limited per client IP: they're the
		// endpoints an attacker would hit to brute-force credentials, mass-register, or spam
		// password-reset emails, and (unlike protected routes) have no user identity to key
		// on until after they succeed.
		auth.POST("/register", middleware.RateLimit(registerRateLimit, registerRateBurst), r.handlers.UserHandler.Register)
		auth.POST("/login", middleware.RateLimit(loginRateLimit, loginRateBurst), r.handlers.UserHandler.Login)
		auth.POST("/forgot-password", middleware.RateLimit(forgotPasswordRateLimit, forgotPasswordRateBurst), r.handlers.UserHandler.ForgotPassword)
		auth.POST("/reset-password", r.handlers.UserHandler.ResetPassword)
	}
}

// setupProtectedRoutes handles all routes that require authentication
func (r *Router) setupProtectedRoutes(rg *gin.RouterGroup) {
	users := rg.Group("/users")
	{
		users.GET("", r.handlers.ProfileHandler.Get)
		users.PUT("", r.handlers.ProfileHandler.Update)
		users.DELETE("/me", r.handlers.UserHandler.DeleteAccount)
		users.GET("/list", r.handlers.UserHandler.ListAll)
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

	shoppingLists := rg.Group("/shopping-lists")
	{
		shoppingLists.POST("", r.handlers.ShoppingListHandler.Create)
		shoppingLists.GET("", r.handlers.ShoppingListHandler.List)
		shoppingLists.GET("/:id", r.handlers.ShoppingListHandler.Get)
		shoppingLists.PUT("/:id", r.handlers.ShoppingListHandler.Update)
		shoppingLists.DELETE("/:id", r.handlers.ShoppingListHandler.Delete)

		shoppingLists.POST("/:id/items", r.handlers.ShoppingListHandler.AddItem)
		shoppingLists.PUT("/:id/items/:itemId", r.handlers.ShoppingListHandler.UpdateItem)
		shoppingLists.DELETE("/:id/items/:itemId", r.handlers.ShoppingListHandler.DeleteItem)
		shoppingLists.PATCH("/:id/items/:itemId/toggle", r.handlers.ShoppingListHandler.ToggleItem)

		shoppingLists.POST("/:id/add-recipe", r.handlers.ShoppingListHandler.AddRecipe)
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
