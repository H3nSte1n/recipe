package storage

import (
	"bytes"
	"context"
	"mime/multipart"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeFileHeader builds a *multipart.FileHeader carrying the given bytes under
// the supplied client filename.
func makeFileHeader(t *testing.T, clientName string, content []byte) *multipart.FileHeader {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("image", clientName)
	require.NoError(t, err)
	_, err = fw.Write(content)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	reader := multipart.NewReader(&buf, w.Boundary())
	form, err := reader.ReadForm(1 << 20)
	require.NoError(t, err)
	return form.File["image"][0]
}

func TestLocalStore_UploadFile_RejectsNonImage(t *testing.T) {
	store, err := NewLocalFileStore(t.TempDir(), "http://localhost:8080/uploads")
	require.NoError(t, err)

	// HTML disguised as a .png by extension must be rejected by content sniffing.
	fh := makeFileHeader(t, "evil.png", []byte("<!DOCTYPE html><script>alert(1)</script>"))
	_, err = store.UploadFile(context.Background(), fh)
	assert.Error(t, err)
}

func TestLocalStore_UploadFile_NormalizesExtension(t *testing.T) {
	store, err := NewLocalFileStore(t.TempDir(), "http://localhost:8080/uploads")
	require.NoError(t, err)

	// A real PNG uploaded with a misleading .jpg name must be stored as .png.
	pngBytes := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	fh := makeFileHeader(t, "photo.jpg", pngBytes)

	url, err := store.UploadFile(context.Background(), fh)
	require.NoError(t, err)
	assert.Equal(t, ".png", filepath.Ext(url), "stored extension must match detected type")
}
