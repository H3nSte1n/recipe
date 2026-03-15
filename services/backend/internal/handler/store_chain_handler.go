package handler

import (
	"github.com/H3nSte1n/recipe/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
)

type StoreChainHandler struct {
	service service.StoreChainService
	logger  *zap.Logger
}

func NewStoreChainHandler(service service.StoreChainService, logger *zap.Logger) *StoreChainHandler {
	return &StoreChainHandler{
		service: service,
		logger:  logger,
	}
}

func (h *StoreChainHandler) List(c *gin.Context) {
	country := c.Query("country")

	chains, err := h.service.ListChains(c.Request.Context(), country)
	if err != nil {
		h.logger.Error("failed to list store chains", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list store chains"})
		return
	}

	c.JSON(http.StatusOK, chains)
}

func (h *StoreChainHandler) Get(c *gin.Context) {
	chainID := c.Param("id")

	chain, err := h.service.GetChain(c.Request.Context(), chainID)
	if err != nil {
		h.logger.Error("failed to get store chain", zap.Error(err), zap.String("chainID", chainID))
		c.JSON(http.StatusNotFound, gin.H{"error": "store chain not found"})
		return
	}

	c.JSON(http.StatusOK, chain)
}
