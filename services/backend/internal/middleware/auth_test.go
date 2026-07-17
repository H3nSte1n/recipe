package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

const (
	testSecret   = "test-secret-at-least-32-bytes-long"
	testIssuer   = "recipe-app-test"
	testAudience = "recipe-app-test-api"
)

// stubRevocationChecker returns a fixed revocation timestamp for every user,
// or none if revokedAt is nil.
type stubRevocationChecker struct {
	revokedAt *time.Time
	err       error
}

func (s *stubRevocationChecker) GetTokenRevokedAt(ctx context.Context, userID string) (*time.Time, error) {
	return s.revokedAt, s.err
}

func signToken(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(testSecret))
	require.NoError(t, err)
	return signed
}

func validClaims(userID string, iat time.Time) jwt.MapClaims {
	return jwt.MapClaims{
		"user_id": userID,
		"email":   "user@example.com",
		"iss":     testIssuer,
		"aud":     testAudience,
		"iat":     iat.Unix(),
		"nbf":     iat.Unix(),
		"exp":     iat.Add(time.Hour).Unix(),
	}
}

func performAuthRequest(m *AuthMiddleware, token string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, engine := gin.CreateTestContext(w)
	engine.Use(m.AuthRequired())
	engine.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	c.Request = req
	engine.ServeHTTP(w, req)
	return w
}

func TestAuthRequired_RejectsMissingUserIDClaim(t *testing.T) {
	m := NewAuthMiddleware(testSecret, testIssuer, testAudience, nil)
	claims := validClaims("", time.Now())
	token := signToken(t, claims)

	w := performAuthRequest(m, token)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthRequired_AcceptsValidToken(t *testing.T) {
	m := NewAuthMiddleware(testSecret, testIssuer, testAudience, &stubRevocationChecker{})
	token := signToken(t, validClaims("user-1", time.Now()))

	w := performAuthRequest(m, token)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestAuthRequired_RejectsWrongIssuer(t *testing.T) {
	m := NewAuthMiddleware(testSecret, testIssuer, testAudience, nil)
	claims := validClaims("user-1", time.Now())
	claims["iss"] = "someone-else"
	token := signToken(t, claims)

	w := performAuthRequest(m, token)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthRequired_RejectsWrongAudience(t *testing.T) {
	m := NewAuthMiddleware(testSecret, testIssuer, testAudience, nil)
	claims := validClaims("user-1", time.Now())
	claims["aud"] = "someone-else-api"
	token := signToken(t, claims)

	w := performAuthRequest(m, token)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthRequired_RejectsNotYetValidToken(t *testing.T) {
	m := NewAuthMiddleware(testSecret, testIssuer, testAudience, nil)
	claims := validClaims("user-1", time.Now())
	claims["nbf"] = time.Now().Add(time.Hour).Unix()
	token := signToken(t, claims)

	w := performAuthRequest(m, token)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthRequired_RejectsTokenIssuedBeforeRevocation(t *testing.T) {
	revokedAt := time.Now()
	m := NewAuthMiddleware(testSecret, testIssuer, testAudience, &stubRevocationChecker{revokedAt: &revokedAt})
	// Token issued before the revocation timestamp must be rejected even
	// though it's otherwise well-formed and unexpired.
	token := signToken(t, validClaims("user-1", revokedAt.Add(-time.Minute)))

	w := performAuthRequest(m, token)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthRequired_AcceptsTokenIssuedAfterRevocation(t *testing.T) {
	revokedAt := time.Now().Add(-time.Hour)
	m := NewAuthMiddleware(testSecret, testIssuer, testAudience, &stubRevocationChecker{revokedAt: &revokedAt})
	token := signToken(t, validClaims("user-1", time.Now()))

	w := performAuthRequest(m, token)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestAuthRequired_FailsClosedOnRevocationLookupError(t *testing.T) {
	m := NewAuthMiddleware(testSecret, testIssuer, testAudience, &stubRevocationChecker{err: context.DeadlineExceeded})
	token := signToken(t, validClaims("user-1", time.Now()))

	w := performAuthRequest(m, token)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}
