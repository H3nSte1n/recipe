package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/H3nSte1n/recipe/pkg/signedurl"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupUploadsTest(t *testing.T) (*gin.Engine, *signedurl.Signer, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dir := t.TempDir()
	// A 1x1 PNG header is enough; the handler serves bytes verbatim.
	pngBytes := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "abc.png"), pngBytes, 0o600))

	signer := signedurl.NewSigner("test-secret", time.Hour)
	h := NewUploadsHandler(dir, signer)

	r := gin.New()
	r.GET("/uploads/:filename", h.Serve)
	return r, signer, dir
}

// signedQuery returns the exp/sig query string for a filename.
func signedQuery(signer *signedurl.Signer, filename string) string {
	signed := signer.Sign("http://x/uploads/" + filename)
	u, _ := url.Parse(signed)
	return u.RawQuery
}

func TestUploads_ServesWithValidSignature(t *testing.T) {
	r, signer, _ := setupUploadsTest(t)

	req := httptest.NewRequest(http.MethodGet, "/uploads/abc.png?"+signedQuery(signer, "abc.png"), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "attachment", w.Header().Get("Content-Disposition"))
	// nosniff requires a correct Content-Type for <img> to render.
	assert.Contains(t, w.Header().Get("Content-Type"), "image/png")
}

func TestUploads_RejectsMissingSignature(t *testing.T) {
	r, _, _ := setupUploadsTest(t)

	req := httptest.NewRequest(http.MethodGet, "/uploads/abc.png", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestUploads_RejectsTamperedSignature(t *testing.T) {
	r, signer, _ := setupUploadsTest(t)
	q := signedQuery(signer, "abc.png")

	// Request a different file with abc.png's signature.
	require.NoError(t, os.WriteFile(filepath.Join(t.TempDir(), "ignore"), nil, 0o600))
	req := httptest.NewRequest(http.MethodGet, "/uploads/other.png?"+q, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestUploads_RejectsPathTraversal(t *testing.T) {
	r, signer, _ := setupUploadsTest(t)

	req := httptest.NewRequest(http.MethodGet, "/uploads/..?"+signedQuery(signer, ".."), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
