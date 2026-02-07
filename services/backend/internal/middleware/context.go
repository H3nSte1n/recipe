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

// GetUserID retrieves the authenticated user's ID from the context
func GetUserID(c *gin.Context) string {
	userID, exists := c.Get("user_id")
	if !exists {
		return ""
	}

	// Handle both string and interface{} types
	switch v := userID.(type) {
	case string:
		return v
	default:
		// Try to convert to string
		if str, ok := userID.(string); ok {
			return str
		}
		return ""
	}
}
