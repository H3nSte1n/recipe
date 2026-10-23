package handler

import (
	stderrors "errors"
	"net/http"
	"strconv"

	"github.com/H3nSte1n/recipe/internal/domain"
	"github.com/H3nSte1n/recipe/internal/errors"
	"github.com/H3nSte1n/recipe/internal/middleware"
	"github.com/H3nSte1n/recipe/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	defaultPlannedMealPage     = 1
	defaultPlannedMealPageSize = 100
)

type PlannedMealHandler struct {
	service service.PlannedMealService
	logger  *zap.Logger
}

func NewPlannedMealHandler(service service.PlannedMealService, logger *zap.Logger) *PlannedMealHandler {
	return &PlannedMealHandler{
		service: service,
		logger:  logger,
	}
}

// fail answers validation errors (400) and missing meals (404) with safe messages; anything
// else is logged and returned as a generic 500 so storage details never reach the client.
func (h *PlannedMealHandler) fail(c *gin.Context, err error, action string) {
	switch errors.StatusCode(err) {
	case http.StatusBadRequest:
		var appErr *errors.AppError
		message := "invalid request"
		if stderrors.As(err, &appErr) {
			message = appErr.Message
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
	case http.StatusNotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": "planned meal not found"})
	default:
		h.logger.Error("failed to "+action, zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to " + action})
	}
}

func (h *PlannedMealHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req domain.PlannedMealRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Binding errors name Go types and fields, so clients only get a generic message.
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	meal, err := h.service.Create(c.Request.Context(), userID, &req)
	if err != nil {
		h.fail(c, err, "create planned meal")
		return
	}

	c.JSON(http.StatusCreated, meal)
}

func (h *PlannedMealHandler) Get(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	meal, err := h.service.GetByID(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		h.fail(c, err, "get planned meal")
		return
	}

	c.JSON(http.StatusOK, meal)
}

func (h *PlannedMealHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	page, pageSize := defaultPlannedMealPage, defaultPlannedMealPageSize
	if raw, ok := c.GetQuery("page"); ok {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page"})
			return
		}
		page = parsed
	}
	if raw, ok := c.GetQuery("page_size"); ok {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page_size"})
			return
		}
		pageSize = parsed
	}

	result, err := h.service.List(c.Request.Context(), userID, c.Query("from"), c.Query("to"), page, pageSize)
	if err != nil {
		h.fail(c, err, "list planned meals")
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *PlannedMealHandler) Update(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req domain.PlannedMealRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	meal, err := h.service.Update(c.Request.Context(), userID, c.Param("id"), &req)
	if err != nil {
		h.fail(c, err, "update planned meal")
		return
	}

	c.JSON(http.StatusOK, meal)
}

func (h *PlannedMealHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := h.service.Delete(c.Request.Context(), userID, c.Param("id")); err != nil {
		h.fail(c, err, "delete planned meal")
		return
	}

	c.Status(http.StatusNoContent)
}
