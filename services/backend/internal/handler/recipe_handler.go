package handler

import (
	"encoding/json"
	"github.com/H3nSte1n/recipe/internal/domain"
	"github.com/H3nSte1n/recipe/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
	"strconv"
	"strings"
)

type RecipeHandler struct {
	recipeService service.RecipeService
	logger        *zap.Logger
}

func NewRecipeHandler(recipeService service.RecipeService, logger *zap.Logger) *RecipeHandler {
	return &RecipeHandler{
		recipeService: recipeService,
		logger:        logger,
	}
}

func (h *RecipeHandler) Create(c *gin.Context) {
	var req domain.CreateRecipeRequest

	contentType := c.GetHeader("Content-Type")

	if strings.Contains(contentType, "multipart/form-data") {
		recipeJSON := c.PostForm("recipe")
		if err := json.Unmarshal([]byte(recipeJSON), &req); err != nil {
			h.logger.Error("failed to parse recipe json", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if file, err := c.FormFile("image"); err == nil {
			req.Image = file
		}
	} else {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	userID, _ := c.Get("user_id")
	recipe, err := h.recipeService.Create(c.Request.Context(), userID.(string), &req)
	if err != nil {
		h.logger.Error("failed to create recipe", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create recipe"})
		return
	}

	c.JSON(http.StatusCreated, recipe)
}

func (h *RecipeHandler) Update(c *gin.Context) {
	var req domain.CreateRecipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	recipeID := c.Param("id")

	recipe, err := h.recipeService.Update(c.Request.Context(), userID.(string), recipeID, &req)
	if err != nil {
		h.logger.Error("failed to update recipe", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update recipe"})
		return
	}

	c.JSON(http.StatusOK, recipe)
}

func (h *RecipeHandler) Delete(c *gin.Context) {
	userID, _ := c.Get("user_id")
	recipeID := c.Param("id")

	if err := h.recipeService.Delete(c.Request.Context(), userID.(string), recipeID); err != nil {
		h.logger.Error("failed to delete recipe", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete recipe"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "recipe deleted"})
}

func (h *RecipeHandler) Get(c *gin.Context) {
	userID, _ := c.Get("user_id")
	recipeID := c.Param("id")

	nutritionLevel := domain.NutritionDetailLevel(c.DefaultQuery("nutrition_level", string(domain.NutritionDetailBase)))

	switch nutritionLevel {
	case domain.NutritionDetailBase, domain.NutritionDetailMacro, domain.NutritionDetailMicro:
	default:
		nutritionLevel = domain.NutritionDetailBase
	}

	recipe, err := h.recipeService.GetByID(c.Request.Context(), userID.(string), recipeID, nutritionLevel)
	if err != nil {
		h.logger.Error("failed to get recipe", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get recipe"})
		return
	}

	c.JSON(http.StatusOK, recipe)
}

func (h *RecipeHandler) ListMine(c *gin.Context) {
	userID, _ := c.Get("user_id")

	recipes, err := h.recipeService.ListUserRecipes(c.Request.Context(), userID.(string))
	if err != nil {
		h.logger.Error("failed to list user recipes", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list recipes"})
		return
	}

	c.JSON(http.StatusOK, recipes)
}

func (h *RecipeHandler) ListPublic(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	recipes, total, err := h.recipeService.ListPublicRecipes(c.Request.Context(), page, pageSize)
	if err != nil {
		h.logger.Error("failed to list public recipes", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list recipes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"recipes": recipes,
		"total":   total,
		"page":    page,
		"size":    pageSize,
	})
}

func (h *RecipeHandler) ImportFromURL(c *gin.Context) {
	var req domain.ImportURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	recipe, err := h.recipeService.ImportFromURL(c.Request.Context(), userID.(string), &req)
	if err != nil {
		h.logger.Error("failed to import recipe from URL", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to import recipe"})
		return
	}

	c.JSON(http.StatusOK, recipe)
}

func (h *RecipeHandler) ImportFromPDF(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no file provided"})
		return
	}

	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
		return
	}
	defer f.Close()

	fileBytes := make([]byte, file.Size)
	if _, err := f.Read(fileBytes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
		return
	}

	var req domain.ImportPDFRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	recipe, err := h.recipeService.ImportFromPDF(c.Request.Context(), userID.(string), &req, fileBytes)
	if err != nil {
		h.logger.Error("failed to import recipe from PDF", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to import recipe"})
		return
	}

	c.JSON(http.StatusOK, recipe)
}

func (h *RecipeHandler) ParsePlainTextInstructions(c *gin.Context) {
	var req domain.ParsePlainTextInstructionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	instructions, err := h.recipeService.ParsePlainTextInstructions(c.Request.Context(), userID.(string), &req)
	if err != nil {
		h.logger.Error("failed to parse plain text instructions", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse instructions"})
		return
	}

	c.JSON(http.StatusOK, instructions)
}
