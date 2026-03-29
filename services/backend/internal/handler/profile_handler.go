package handler

import (
	"github.com/H3nSte1n/recipe/internal/domain"
	"github.com/H3nSte1n/recipe/internal/middleware"
	"github.com/H3nSte1n/recipe/internal/service"
	"github.com/gin-gonic/gin"
	"net/http"
)

type ProfileHandler struct {
	profileService service.ProfileService
}

func NewProfileHandler(profileService service.ProfileService) *ProfileHandler {
	return &ProfileHandler{
		profileService: profileService,
	}
}

func (h *ProfileHandler) Update(c *gin.Context) {
	var req domain.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	profile, err := h.profileService.UpdateProfile(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, profile)
}

func (h *ProfileHandler) Get(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	profile, err := h.profileService.GetProfile(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
		return
	}

	c.JSON(http.StatusOK, profile)
}
