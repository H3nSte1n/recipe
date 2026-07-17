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

// maxMultipartBodyBytes is the backstop for requests declaring a multipart Content-Type. It
// matches the largest existing per-handler multipart limit (PDF import, see recipe_handler.go)
// so legitimate uploads on those routes are unaffected — http.MaxBytesReader nests cleanly, so a
// handler's own smaller limit (e.g. 10 MiB for images) still governs there. Without this
// backstop, a request whose Content-Type merely claims to be multipart — including a spoofed
// header on a JSON route like /auth/login, which binds with ShouldBindJSON regardless of
// Content-Type — would read an unbounded body before any handler-level limit applies.
const maxMultipartBodyBytes = 20 << 20 // 20 MiB

// BodySizeLimit rejects request bodies over maxBytes before any handler parses them. Multipart
// requests get the larger maxMultipartBodyBytes backstop instead of maxBytes, since legitimate
// multipart uploads (recipe image, PDF import) are bigger than a JSON payload; those routes then
// layer their own smaller http.MaxBytesReader on top for their specific limit.
func BodySizeLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := maxBytes
		if strings.HasPrefix(c.GetHeader("Content-Type"), "multipart/form-data") {
			limit = maxMultipartBodyBytes
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		c.Next()
	}
}
