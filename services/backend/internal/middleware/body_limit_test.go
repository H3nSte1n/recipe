package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestBodySizeLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	newEngine := func(maxBytes int64) *gin.Engine {
		engine := gin.New()
		engine.Use(BodySizeLimit(maxBytes))
		engine.POST("/echo", func(c *gin.Context) {
			body, err := io.ReadAll(c.Request.Body)
			if err != nil {
				c.String(http.StatusRequestEntityTooLarge, "too large")
				return
			}
			c.String(http.StatusOK, "%d", len(body))
		})
		return engine
	}

	t.Run("rejects a JSON body over the limit", func(t *testing.T) {
		engine := newEngine(10)
		req := httptest.NewRequest(http.MethodPost, "/echo", bytes.NewReader([]byte("this body is way over the limit")))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	})

	t.Run("allows a JSON body within the limit", func(t *testing.T) {
		engine := newEngine(1024)
		req := httptest.NewRequest(http.MethodPost, "/echo", bytes.NewReader([]byte(`{"ok":true}`)))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("does not clamp multipart requests to the JSON limit", func(t *testing.T) {
		engine := newEngine(10) // far smaller than the multipart body below
		body := strings.Repeat("x", 1000)
		req := httptest.NewRequest(http.MethodPost, "/echo", strings.NewReader(body))
		req.Header.Set("Content-Type", "multipart/form-data; boundary=xyz")

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		// The handler itself (recipe image/PDF upload) is responsible for the multipart
		// size cap via its own http.MaxBytesReader call; this middleware must not
		// double-clamp it to the smaller JSON limit.
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "1000", w.Body.String())
	})
}
