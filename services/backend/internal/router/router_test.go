package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrustedProxiesFromEnv(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want []string
	}{
		{name: "unset defaults to loopback", env: "", want: []string{"127.0.0.1"}},
		{name: "single value", env: "10.0.0.5", want: []string{"10.0.0.5"}},
		{name: "comma separated with spaces", env: " 10.0.0.5 , 10.0.0.6", want: []string{"10.0.0.5", "10.0.0.6"}},
		{name: "blank entries ignored, all blank falls back to default", env: " , ,", want: []string{"127.0.0.1"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TRUSTED_PROXIES", tt.env)
			assert.Equal(t, tt.want, trustedProxiesFromEnv())
		})
	}
}

// TestClientIP_ForgedHeaderFromUntrustedSource verifies the actual effect of restricting Gin's
// trusted proxies: a forged X-Forwarded-For from a source that isn't in the trusted list must not
// be treated as the real client IP, while a request that genuinely arrives via a trusted proxy
// (e.g. the local nginx reverse proxy at 127.0.0.1) should still have its forwarded IP honored.
func TestClientIP_ForgedHeaderFromUntrustedSource(t *testing.T) {
	gin.SetMode(gin.TestMode)

	newEngine := func(trustedProxies []string) *gin.Engine {
		engine := gin.New()
		require.NoError(t, engine.SetTrustedProxies(trustedProxies))
		engine.GET("/ip", func(c *gin.Context) {
			c.String(http.StatusOK, c.ClientIP())
		})
		return engine
	}

	t.Run("untrusted source: forged X-Forwarded-For is ignored", func(t *testing.T) {
		engine := newEngine([]string{"127.0.0.1"})

		req := httptest.NewRequest(http.MethodGet, "/ip", nil)
		req.RemoteAddr = "203.0.113.5:54321" // not in the trusted proxy list
		req.Header.Set("X-Forwarded-For", "8.8.8.8")

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		// Must fall back to the actual TCP peer, not the attacker-supplied header.
		assert.Equal(t, "203.0.113.5", w.Body.String())
	})

	t.Run("trusted source: X-Forwarded-For is honored", func(t *testing.T) {
		engine := newEngine([]string{"127.0.0.1"})

		req := httptest.NewRequest(http.MethodGet, "/ip", nil)
		req.RemoteAddr = "127.0.0.1:54321" // the trusted local reverse proxy
		req.Header.Set("X-Forwarded-For", "8.8.8.8")

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		assert.Equal(t, "8.8.8.8", w.Body.String())
	})
}
