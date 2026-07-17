package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/H3nSte1n/recipe/internal/domain"
	apperrors "github.com/H3nSte1n/recipe/internal/errors"
	"github.com/H3nSte1n/recipe/internal/middleware"
	"github.com/H3nSte1n/recipe/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"go.uber.org/zap"
)

const (
	// maxImageUploadBytes caps a recipe image upload; maxPDFUploadBytes caps an
	// imported PDF. Both bound memory/disk use against upload-DoS.
	maxImageUploadBytes = 10 << 20 // 10 MiB
	maxPDFUploadBytes   = 20 << 20 // 20 MiB
)

// parseRecipeMultipart enforces a body-size limit, parses the multipart form,
// unmarshals the JSON "recipe" part, runs its binding validators (json.Unmarshal
// alone skips them), and attaches the optional image. It writes the appropriate
// error response and returns false on failure.
func (h *RecipeHandler) parseRecipeMultipart(c *gin.Context, req *domain.CreateRecipeRequest) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxImageUploadBytes)
	if err := c.Request.ParseMultipartForm(maxImageUploadBytes); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "upload too large"})
			return false
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid multipart form"})
		return false
	}

	if err := json.Unmarshal([]byte(c.Request.FormValue("recipe")), req); err != nil {
		h.logger.Error("failed to parse recipe json", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return false
	}

	// json.Unmarshal bypasses the struct `binding:` validators that ShouldBindJSON
	// would run, so validate explicitly.
	if err := binding.Validator.ValidateStruct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return false
	}

	if file, err := c.FormFile("image"); err == nil {
		req.Image = file
	}
	return true
}

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

// respondError maps a service error to its HTTP status and writes the response. Known errors
// (not-found, cross-tenant/unauthorized) get their specific status and safe message; anything
// else is logged server-side with the real error and returns the generic fallback message so no
// internal detail reaches the client.
func (h *RecipeHandler) respondError(c *gin.Context, err error, fallback string) {
	status := apperrors.StatusCode(err)
	if status == http.StatusInternalServerError {
		h.logger.Error(fallback, zap.Error(err))
		c.JSON(status, gin.H{"error": fallback})
		return
	}
	c.JSON(status, gin.H{"error": err.Error()})
}

func (h *RecipeHandler) Create(c *gin.Context) {
	var req domain.CreateRecipeRequest

	contentType := c.GetHeader("Content-Type")

	if strings.Contains(contentType, "multipart/form-data") {
		if !h.parseRecipeMultipart(c, &req) {
			return
		}
	} else {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	recipe, err := h.recipeService.Create(c.Request.Context(), userID, &req)
	if err != nil {
		h.logger.Error("failed to create recipe", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create recipe"})
		return
	}

	c.JSON(http.StatusCreated, recipe)
}

func (h *RecipeHandler) Update(c *gin.Context) {
	var req domain.CreateRecipeRequest

	contentType := c.GetHeader("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		if !h.parseRecipeMultipart(c, &req) {
			return
		}
	} else {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	recipeID := c.Param("id")

	recipe, err := h.recipeService.Update(c.Request.Context(), userID, recipeID, &req)
	if err != nil {
		h.respondError(c, err, "failed to update recipe")
		return
	}

	c.JSON(http.StatusOK, recipe)
}

func (h *RecipeHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	recipeID := c.Param("id")

	if err := h.recipeService.Delete(c.Request.Context(), userID, recipeID); err != nil {
		h.respondError(c, err, "failed to delete recipe")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "recipe deleted"})
}

func (h *RecipeHandler) Get(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"errors": "unauthorized"})
		return
	}
	recipeID := c.Param("id")

	nutritionLevel := domain.NutritionDetailLevel(c.DefaultQuery("nutrition_level", string(domain.NutritionDetailBase)))

	switch nutritionLevel {
	case domain.NutritionDetailBase, domain.NutritionDetailMacro, domain.NutritionDetailMicro:
	default:
		nutritionLevel = domain.NutritionDetailBase
	}

	recipe, err := h.recipeService.GetByID(c.Request.Context(), userID, recipeID, nutritionLevel)
	if err != nil {
		h.respondError(c, err, "failed to get recipe")
		return
	}

	c.JSON(http.StatusOK, recipe)
}

func (h *RecipeHandler) ListMine(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"errors": "unauthorized"})
		return
	}

	recipes, err := h.recipeService.ListUserRecipes(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("failed to list user recipes", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list recipes"})
		return
	}

	c.JSON(http.StatusOK, recipes)
}

// maxPublicRecipePageSize caps how many rows a single /recipes/public request can pull,
// regardless of what the caller asks for.
const maxPublicRecipePageSize = 100

func (h *RecipeHandler) ListPublic(c *gin.Context) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "page must be a positive integer"})
		return
	}

	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if err != nil || pageSize < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "page_size must be a positive integer"})
		return
	}
	if pageSize > maxPublicRecipePageSize {
		pageSize = maxPublicRecipePageSize
	}

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
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req domain.ImportURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	recipe, err := h.recipeService.ImportFromURL(c.Request.Context(), userID, &req)
	if err != nil {
		h.logger.Error("failed to import recipe from URL", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to import recipe"})
		return
	}

	c.JSON(http.StatusOK, recipe)
}

func (h *RecipeHandler) ImportFromPDF(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxPDFUploadBytes)

	file, err := c.FormFile("file")
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "PDF too large"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "no file provided"})
		return
	}
	if file.Size > maxPDFUploadBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "PDF too large"})
		return
	}

	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
		return
	}
	defer f.Close()

	// io.ReadAll reads the whole file (fixing the previous single Read that could
	// short-read), and the LimitReader caps it to defend against oversized input.
	fileBytes, err := io.ReadAll(io.LimitReader(f, maxPDFUploadBytes+1))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
		return
	}
	if len(fileBytes) > maxPDFUploadBytes {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "PDF too large"})
		return
	}

	var req domain.ImportPDFRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	recipe, err := h.recipeService.ImportFromPDF(c.Request.Context(), userID, &req, fileBytes)
	if err != nil {
		h.logger.Error("failed to import recipe from PDF", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to import recipe"})
		return
	}

	c.JSON(http.StatusOK, recipe)
}

func (h *RecipeHandler) ParsePlainTextInstructions(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req domain.ParsePlainTextInstructionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	instructions, err := h.recipeService.ParsePlainTextInstructions(c.Request.Context(), userID, &req)
	if err != nil {
		h.logger.Error("failed to parse plain text instructions", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse instructions"})
		return
	}

	c.JSON(http.StatusOK, instructions)
}
