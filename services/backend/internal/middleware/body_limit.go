package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// MaxJSONBodyBytes caps ordinary JSON request bodies globally. It's sized generously above any
// legitimate recipe/shopping-list/AI-config payload (all plain text/JSON, no embedded binary
// data) while still bounding memory/CPU use against oversized-body DoS on routes that have no
// per-handler limit of their own.
const MaxJSONBodyBytes = 2 << 20 // 2 MiB

// BodySizeLimit rejects request bodies over maxBytes before any handler parses them.
//
// Multipart upload routes (recipe image, PDF import) already enforce their own, larger limits
// via http.MaxBytesReader in the handler (see recipe_handler.go) before parsing the form; this
// middleware skips multipart requests so it doesn't double-clamp them to a smaller size.
func BodySizeLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !strings.HasPrefix(c.GetHeader("Content-Type"), "multipart/form-data") {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}
