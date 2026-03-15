package handler

import (
	"github.com/H3nSte1n/recipe/internal/domain"
	"github.com/H3nSte1n/recipe/internal/middleware"
	"github.com/H3nSte1n/recipe/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
)

type ShoppingListHandler struct {
	service service.ShoppingListService
	logger  *zap.Logger
}

func NewShoppingListHandler(service service.ShoppingListService, logger *zap.Logger) *ShoppingListHandler {
	return &ShoppingListHandler{
		service: service,
		logger:  logger,
	}
}

func (h *ShoppingListHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req domain.CreateShoppingListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	list, err := h.service.Create(c.Request.Context(), userID, &req)
	if err != nil {
		h.logger.Error("failed to create shopping list", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create shopping list"})
		return
	}

	c.JSON(http.StatusCreated, list)
}

func (h *ShoppingListHandler) Get(c *gin.Context) {
	userID := middleware.GetUserID(c)
	listID := c.Param("id")

	sortBy := c.DefaultQuery("sort_by", "")
	sortDirection := c.DefaultQuery("sort_direction", "asc")
	storeName := c.Query("store_name")

	var list *domain.ShoppingList
	var err error

	if sortBy == "store" || storeName != "" {
		if storeName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "store_name is required when sort_by=store"})
			return
		}
		list, err = h.service.GetSortedByStoreName(c.Request.Context(), userID, listID, storeName, sortDirection)
	} else if sortBy != "" {
		list, err = h.service.GetSorted(c.Request.Context(), userID, listID, sortBy, sortDirection)
	} else {
		list, err = h.service.GetByID(c.Request.Context(), userID, listID)
	}

	if err != nil {
		h.logger.Error("failed to get shopping list", zap.Error(err), zap.String("listID", listID))
		c.JSON(http.StatusNotFound, gin.H{"error": "shopping list not found"})
		return
	}

	c.JSON(http.StatusOK, list)
}

func (h *ShoppingListHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)

	lists, err := h.service.ListByUserID(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("failed to list shopping lists", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list shopping lists"})
		return
	}

	c.JSON(http.StatusOK, lists)
}

func (h *ShoppingListHandler) Update(c *gin.Context) {
	userID := middleware.GetUserID(c)
	listID := c.Param("id")

	var req domain.UpdateShoppingListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	list, err := h.service.Update(c.Request.Context(), userID, listID, &req)
	if err != nil {
		h.logger.Error("failed to update shopping list", zap.Error(err), zap.String("listID", listID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update shopping list"})
		return
	}

	c.JSON(http.StatusOK, list)
}

func (h *ShoppingListHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	listID := c.Param("id")

	if err := h.service.Delete(c.Request.Context(), userID, listID); err != nil {
		h.logger.Error("failed to delete shopping list", zap.Error(err), zap.String("listID", listID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete shopping list"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

func (h *ShoppingListHandler) AddItem(c *gin.Context) {
	userID := middleware.GetUserID(c)
	listID := c.Param("id")

	var req domain.ShoppingListItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.AddItem(c.Request.Context(), userID, listID, &req); err != nil {
		h.logger.Error("failed to add item to shopping list", zap.Error(err), zap.String("listID", listID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add item"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "item added successfully"})
}

func (h *ShoppingListHandler) UpdateItem(c *gin.Context) {
	userID := middleware.GetUserID(c)
	itemID := c.Param("itemId")

	var req domain.UpdateShoppingListItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.UpdateItem(c.Request.Context(), userID, itemID, &req); err != nil {
		h.logger.Error("failed to update item", zap.Error(err), zap.String("itemID", itemID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update item"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "item updated successfully"})
}

func (h *ShoppingListHandler) DeleteItem(c *gin.Context) {
	userID := middleware.GetUserID(c)
	itemID := c.Param("itemId")

	if err := h.service.DeleteItem(c.Request.Context(), userID, itemID); err != nil {
		h.logger.Error("failed to delete item", zap.Error(err), zap.String("itemID", itemID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete item"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

func (h *ShoppingListHandler) ToggleItem(c *gin.Context) {
	userID := middleware.GetUserID(c)
	itemID := c.Param("itemId")

	var req struct {
		Checked bool `json:"checked"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.ToggleItem(c.Request.Context(), userID, itemID, req.Checked); err != nil {
		h.logger.Error("failed to toggle item", zap.Error(err), zap.String("itemID", itemID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to toggle item"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "item toggled successfully"})
}

func (h *ShoppingListHandler) AddRecipe(c *gin.Context) {
	userID := middleware.GetUserID(c)
	listID := c.Param("id")

	var req domain.AddRecipeToListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.AddRecipeToList(c.Request.Context(), userID, listID, &req); err != nil {
		h.logger.Error("failed to add recipe to shopping list", zap.Error(err), zap.String("listID", listID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add recipe to list"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "recipe added to list successfully"})
}

func (h *ShoppingListHandler) SortByStore(c *gin.Context) {
	userID := middleware.GetUserID(c)
	listID := c.Param("id")
	chainID := c.Query("chain_id")

	if chainID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chain_id query parameter is required"})
		return
	}

	list, err := h.service.GetSortedForStore(c.Request.Context(), userID, listID, chainID)
	if err != nil {
		h.logger.Error("failed to get sorted shopping list", zap.Error(err), zap.String("listID", listID), zap.String("chainID", chainID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get sorted list"})
		return
	}

	c.JSON(http.StatusOK, list)
}
