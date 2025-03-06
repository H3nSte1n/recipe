package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/yourusername/recipe-app/internal/domain"
)

// GetCurrentUser retrieves the authenticated user from the context
func GetCurrentUser(c *gin.Context) *domain.User {
	user, exists := c.Get("user")
	if !exists {
		return nil
	}
	return user.(*domain.User)
}
