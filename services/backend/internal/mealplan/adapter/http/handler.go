package http

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/H3nSte1n/recipe/internal/mealplan/application"
	"github.com/H3nSte1n/recipe/internal/mealplan/domain"
	"github.com/H3nSte1n/recipe/internal/middleware"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type UseCases interface {
	Create(context.Context, string, domain.Fields) (domain.PlannedMeal, error)
	Replace(context.Context, string, string, domain.Fields) (domain.PlannedMeal, error)
	Get(context.Context, string, string) (domain.PlannedMeal, error)
	List(context.Context, string, domain.Date, domain.Date, int, int) (application.Page, error)
	Delete(context.Context, string, string) error
}

type Handler struct {
	service UseCases
	logger  *zap.Logger
}

func NewHandler(service UseCases, logger *zap.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

func (h *Handler) failure(c *gin.Context, err error) {
	switch {
	case errors.Is(err, application.ErrInvalid):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, application.ErrUnauthorized):
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
	case errors.Is(err, application.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	default:
		h.logger.Error("planned meal request failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}

func (h *Handler) Create(c *gin.Context) {
	f, err := decodeWrite(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	m, err := h.service.Create(c.Request.Context(), middleware.GetUserID(c), f)
	if err != nil {
		h.failure(c, err)
		return
	}
	c.JSON(http.StatusCreated, toResponse(m))
}

func (h *Handler) Replace(c *gin.Context) {
	f, err := decodeWrite(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	m, err := h.service.Replace(c.Request.Context(), middleware.GetUserID(c), c.Param("id"), f)
	if err != nil {
		h.failure(c, err)
		return
	}
	c.JSON(http.StatusOK, toResponse(m))
}

func (h *Handler) Get(c *gin.Context) {
	m, err := h.service.Get(c.Request.Context(), middleware.GetUserID(c), c.Param("id"))
	if err != nil {
		h.failure(c, err)
		return
	}
	c.JSON(http.StatusOK, toResponse(m))
}

func (h *Handler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), middleware.GetUserID(c), c.Param("id")); err != nil {
		h.failure(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) List(c *gin.Context) {
	from, err := domain.ParseDate(c.Query("from"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from date"})
		return
	}
	to, err := domain.ParseDate(c.Query("to"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid to date"})
		return
	}
	page, pageSize := 1, 100
	if raw, ok := c.GetQuery("page"); ok {
		page, err = strconv.Atoi(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page"})
			return
		}
	}
	if raw, ok := c.GetQuery("page_size"); ok {
		pageSize, err = strconv.Atoi(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page_size"})
			return
		}
	}
	result, err := h.service.List(c.Request.Context(), middleware.GetUserID(c), from, to, page, pageSize)
	if err != nil {
		h.failure(c, err)
		return
	}
	items := make([]mealResponse, 0, len(result.Items))
	for _, m := range result.Items {
		items = append(items, toResponse(m))
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": result.Total, "page": result.Page, "page_size": result.PageSize})
}
