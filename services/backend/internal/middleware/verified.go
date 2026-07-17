package middleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

// EmailVerificationChecker looks up whether a user has completed email
// verification. It's satisfied by handler.UserHandler (which delegates to
// UserService), so this middleware needs no DB/service wiring of its own.
type EmailVerificationChecker func(ctx context.Context, userID string) (bool, error)

// RequireVerified blocks state-changing requests from users who haven't
// confirmed their email address yet. It must run after AuthRequired, since it
// relies on user_id already being set in the Gin context.
//
// Judgment call: unverified users can still log in and read data; only
// mutating routes are gated behind this middleware. See registration/email
// verification remediation notes for the reasoning — flagged for the owner
// to confirm this is the desired policy.
func RequireVerified(checker EmailVerificationChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := GetUserID(c)
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		verified, err := checker(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check verification status"})
			c.Abort()
			return
		}
		if !verified {
			c.JSON(http.StatusForbidden, gin.H{"error": "email verification required before performing this action"})
			c.Abort()
			return
		}

		c.Next()
	}
}
