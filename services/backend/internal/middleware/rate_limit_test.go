package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func newRateLimitedEngine(r rate.Limit, burst int) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(RateLimit(r, burst))
	engine.POST("/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return engine
}

func doRequest(engine *gin.Engine, remoteAddr string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	req.RemoteAddr = remoteAddr
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

func TestRateLimit_AllowsBurstThenRejects(t *testing.T) {
	// rate.Limit(0) means "no steady-state refill", so only the initial burst tokens are
	// available; this isolates the test from timing flakiness.
	engine := newRateLimitedEngine(0, 3)

	for i := 0; i < 3; i++ {
		rec := doRequest(engine, "1.2.3.4:1234")
		require.Equal(t, http.StatusOK, rec.Code, "request %d within burst should be allowed", i+1)
	}

	rec := doRequest(engine, "1.2.3.4:1234")
	require.Equal(t, http.StatusTooManyRequests, rec.Code, "request beyond burst should be rate-limited")
}

func TestRateLimit_TracksLimitsPerIP(t *testing.T) {
	engine := newRateLimitedEngine(0, 1)

	rec := doRequest(engine, "1.1.1.1:1")
	require.Equal(t, http.StatusOK, rec.Code)

	// Same IP again: burst of 1 already consumed.
	rec = doRequest(engine, "1.1.1.1:2")
	require.Equal(t, http.StatusTooManyRequests, rec.Code)

	// Different IP: gets its own independent burst.
	rec = doRequest(engine, "2.2.2.2:1")
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimit_IndependentPerMiddlewareInstance(t *testing.T) {
	// Two separate routes/handlers each get their own RateLimit() call in router.go, so
	// exhausting one endpoint's limiter must not affect another's.
	loginEngine := newRateLimitedEngine(0, 1)
	registerEngine := newRateLimitedEngine(0, 1)

	rec := doRequest(loginEngine, "9.9.9.9:1")
	require.Equal(t, http.StatusOK, rec.Code)
	rec = doRequest(loginEngine, "9.9.9.9:2")
	require.Equal(t, http.StatusTooManyRequests, rec.Code)

	// registerEngine has its own limiter instance, unaffected by loginEngine's state.
	rec = doRequest(registerEngine, "9.9.9.9:1")
	require.Equal(t, http.StatusOK, rec.Code)
}
