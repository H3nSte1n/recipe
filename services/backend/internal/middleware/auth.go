package middleware

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"strings"
	"time"
)

// TokenRevocationChecker looks up the newest revocation timestamp for a
// user. Any JWT whose "iat" claim predates this timestamp must be rejected,
// even if the token has not yet expired naturally. Satisfied by
// repository.UserRepository; declared locally so this package doesn't need
// to import the repository package.
type TokenRevocationChecker interface {
	GetTokenRevokedAt(ctx context.Context, userID string) (*time.Time, error)
}

type AuthMiddleware struct {
	secretKey   string
	issuer      string
	audience    string
	revocations TokenRevocationChecker
}

func NewAuthMiddleware(secretKey, issuer, audience string, revocations TokenRevocationChecker) *AuthMiddleware {
	return &AuthMiddleware{
		secretKey:   secretKey,
		issuer:      issuer,
		audience:    audience,
		revocations: revocations,
	}
}

func (m *AuthMiddleware) AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no authorization header"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			c.Abort()
			return
		}

		token, err := m.validateToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			c.Abort()
			return
		}

		// Fail closed: a missing/empty user_id must reject the request
		// outright rather than letting an empty-string user_id flow
		// downstream.
		userID, _ := claims["user_id"].(string)
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			c.Abort()
			return
		}

		if m.revocations != nil {
			revokedAt, err := m.revocations.GetTokenRevokedAt(c.Request.Context(), userID)
			if err != nil {
				// Fail closed: if we can't verify the token wasn't revoked,
				// don't let it through.
				c.JSON(http.StatusUnauthorized, gin.H{"error": "unable to verify token"})
				c.Abort()
				return
			}
			if revokedAt != nil {
				issuedAt, ok := issuedAtTime(claims)
				if !ok || issuedAt.Before(*revokedAt) {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "token has been revoked"})
					c.Abort()
					return
				}
			}
		}

		email, _ := claims["email"].(string)
		c.Set("user_id", userID)
		c.Set("email", email)

		c.Next()
	}
}

func issuedAtTime(claims jwt.MapClaims) (time.Time, bool) {
	iat, err := claims.GetIssuedAt()
	if err != nil || iat == nil {
		return time.Time{}, false
	}
	return iat.Time, true
}

func (m *AuthMiddleware) validateToken(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(m.secretKey), nil
	}, jwt.WithIssuer(m.issuer), jwt.WithAudience(m.audience))
}
