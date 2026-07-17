package handler

import (
	"github.com/H3nSte1n/recipe/internal/domain"
	apperrors "github.com/H3nSte1n/recipe/internal/errors"
	"github.com/H3nSte1n/recipe/internal/middleware"
	"github.com/H3nSte1n/recipe/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
)

type AIConfigHandler struct {
	aiConfigService service.AIConfigService
	logger          *zap.Logger
}

func NewAIConfigHandler(aiConfigService service.AIConfigService, logger *zap.Logger) *AIConfigHandler {
	return &AIConfigHandler{
		aiConfigService: aiConfigService,
		logger:          logger,
	}
}

// respondError maps a service error to its HTTP status and writes the response. Known errors
// (not-found, cross-tenant/unauthorized) get their specific status and safe message; anything
// else — including raw GORM/Postgres driver errors, which must never reach the client — is
// logged server-side with the real error and returns a generic fallback message.
func (h *AIConfigHandler) respondError(c *gin.Context, err error, fallback string) {
	status := apperrors.StatusCode(err)
	if status == http.StatusInternalServerError {
		h.logger.Error(fallback, zap.Error(err))
		c.JSON(status, gin.H{"error": fallback})
		return
	}
	c.JSON(status, gin.H{"error": err.Error()})
}

func (h *AIConfigHandler) Create(c *gin.Context) {
	var req domain.CreateUserAIConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := middleware.GetUserID(c)
	config, err := h.aiConfigService.Create(c.Request.Context(), userID, &req)
	if err != nil {
		h.respondError(c, err, "failed to create AI configuration")
		return
	}

	c.JSON(http.StatusCreated, config)
}

func (h *AIConfigHandler) Update(c *gin.Context) {
	var req domain.UpdateUserAIConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := middleware.GetUserID(c)
	configID := c.Param("id")

	config, err := h.aiConfigService.Update(c.Request.Context(), userID, configID, &req)
	if err != nil {
		h.respondError(c, err, "failed to update AI configuration")
		return
	}

	c.JSON(http.StatusOK, config)
}

func (h *AIConfigHandler) Get(c *gin.Context) {
	userID := middleware.GetUserID(c)
	configID := c.Param("id")

	config, err := h.aiConfigService.GetByID(c.Request.Context(), userID, configID)
	if err != nil {
		h.respondError(c, err, "AI configuration not found")
		return
	}

	c.JSON(http.StatusOK, config)
}

func (h *AIConfigHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)

	configs, err := h.aiConfigService.List(c.Request.Context(), userID)
	if err != nil {
		h.respondError(c, err, "failed to list AI configurations")
		return
	}

	c.JSON(http.StatusOK, configs)
}

func (h *AIConfigHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	configID := c.Param("id")

	if err := h.aiConfigService.Delete(c.Request.Context(), userID, configID); err != nil {
		h.respondError(c, err, "failed to delete AI configuration")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "AI configuration deleted"})
}

func (h *AIConfigHandler) ListModels(c *gin.Context) {
	models, err := h.aiConfigService.ListAIModels(c.Request.Context())
	if err != nil {
		h.respondError(c, err, "failed to list AI models")
		return
	}

	c.JSON(http.StatusOK, models)
}

func (h *AIConfigHandler) SetDefault(c *gin.Context) {
	userID := middleware.GetUserID(c)
	configID := c.Param("id")

	if err := h.aiConfigService.SetDefault(c.Request.Context(), userID, configID); err != nil {
		h.respondError(c, err, "failed to set default AI configuration")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Default AI configuration updated"})
}

func (h *AIConfigHandler) GetDefault(c *gin.Context) {
	userID := middleware.GetUserID(c)

	config, err := h.aiConfigService.GetDefaultConfig(c.Request.Context(), userID)
	if err != nil {
		h.respondError(c, err, "No default AI configuration found")
		return
	}

	c.JSON(http.StatusOK, config)
}
