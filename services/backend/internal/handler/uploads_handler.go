package handler

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/H3nSte1n/recipe/pkg/signedurl"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// UploadsHandler serves locally-stored upload files. Access is gated by a
// short-lived HMAC signature in the query string (minted when a recipe is
// returned) rather than the JWT, because <img> tags cannot send an
// Authorization header. Files are served with nosniff + attachment so a
// non-image that slips past upload validation cannot execute in a browser.
type UploadsHandler struct {
	uploadDir string
	signer    *signedurl.Signer
	logger    *zap.Logger
}

func NewUploadsHandler(uploadDir string, signer *signedurl.Signer, logger *zap.Logger) *UploadsHandler {
	return &UploadsHandler{uploadDir: uploadDir, signer: signer, logger: logger}
}

func (h *UploadsHandler) Serve(c *gin.Context) {
	filename := c.Param("filename")

	// Defense in depth against path traversal (the route param already excludes
	// slashes, and the signature would not match a traversal payload).
	if filename == "" || filename != filepath.Base(filename) || strings.Contains(filename, "..") {
		h.logger.Warn("rejected upload request with invalid filename", zap.String("filename", filename))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid filename"})
		return
	}

	if err := h.signer.Verify(filename, c.Query("exp"), c.Query("sig")); err != nil {
		h.logger.Warn("rejected upload request with invalid signature",
			zap.String("filename", filename), zap.Error(err))
		c.JSON(http.StatusForbidden, gin.H{"error": "invalid or expired link"})
		return
	}

	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Disposition", "attachment")
	// c.File -> http.ServeFile sets Content-Type from the (normalized) extension.
	c.File(filepath.Join(h.uploadDir, filename))
}
