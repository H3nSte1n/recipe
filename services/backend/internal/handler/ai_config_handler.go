package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/yourusername/recipe-app/internal/domain"
	"github.com/yourusername/recipe-app/internal/service"
	"net/http"
)

type AIConfigHandler struct {
	aiConfigService service.AIConfigService
}

func NewAIConfigHandler(aiConfigService service.AIConfigService) *AIConfigHandler {
	return &AIConfigHandler{
		aiConfigService: aiConfigService,
	}
}

func (h *AIConfigHandler) Create(c *gin.Context) {
	var req domain.CreateUserAIConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	config, err := h.aiConfigService.Create(c.Request.Context(), userID.(string), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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

	userID, _ := c.Get("user_id")
	configID := c.Param("id")

	config, err := h.aiConfigService.Update(c.Request.Context(), userID.(string), configID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

func (h *AIConfigHandler) Get(c *gin.Context) {
	userID, _ := c.Get("user_id")
	configID := c.Param("id")

	config, err := h.aiConfigService.GetByID(c.Request.Context(), userID.(string), configID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "AI configuration not found"})
		return
	}

	c.JSON(http.StatusOK, config)
}

func (h *AIConfigHandler) List(c *gin.Context) {
	userID, _ := c.Get("user_id")

	configs, err := h.aiConfigService.List(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, configs)
}

func (h *AIConfigHandler) Delete(c *gin.Context) {
	userID, _ := c.Get("user_id")
	configID := c.Param("id")

	if err := h.aiConfigService.Delete(c.Request.Context(), userID.(string), configID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "AI configuration deleted"})
}

func (h *AIConfigHandler) ListModels(c *gin.Context) {
	models, err := h.aiConfigService.ListAIModels(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, models)
}

func (h *AIConfigHandler) SetDefault(c *gin.Context) {
	userID, _ := c.Get("user_id")
	configID := c.Param("id")

	if err := h.aiConfigService.SetDefault(c.Request.Context(), userID.(string), configID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Default AI configuration updated"})
}

func (h *AIConfigHandler) GetDefault(c *gin.Context) {
	userID, _ := c.Get("user_id")

	config, err := h.aiConfigService.GetDefaultConfig(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No default AI configuration found"})
		return
	}

	c.JSON(http.StatusOK, config)
}
