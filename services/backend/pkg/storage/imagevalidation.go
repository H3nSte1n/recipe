package storage

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// allowedImageTypes maps an accepted raster-image media type to the canonical
// file extension stored on disk. The stored extension is normalized to the
// detected type (never the client-supplied name) so the file-serving handler
// emits a correct Content-Type — required because responses are served with
// X-Content-Type-Options: nosniff.
var allowedImageTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

// DetectImageType sniffs the leading bytes of r, returns the detected image
// media type and its canonical extension, and rewinds r to the start so the
// caller can copy the full content. It rejects anything that is not an allowed
// raster image — notably SVG and HTML, which sniff as text and are the
// stored-XSS vector this guards against.
func DetectImageType(r io.ReadSeeker) (mediaType, ext string, err error) {
	head := make([]byte, 512)
	n, err := io.ReadFull(r, head)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return "", "", fmt.Errorf("failed to read file header: %w", err)
	}
	head = head[:n]

	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return "", "", fmt.Errorf("failed to rewind file: %w", err)
	}

	// Go's http.DetectContentType does not recognise WebP; check its magic
	// directly (RIFF....WEBP).
	if len(head) >= 12 && bytes.Equal(head[0:4], []byte("RIFF")) && bytes.Equal(head[8:12], []byte("WEBP")) {
		return "image/webp", allowedImageTypes["image/webp"], nil
	}

	detected := http.DetectContentType(head)
	media := detected
	if i := strings.IndexByte(detected, ';'); i >= 0 {
		media = strings.TrimSpace(detected[:i])
	}

	if ext, ok := allowedImageTypes[media]; ok {
		return media, ext, nil
	}
	return "", "", fmt.Errorf("unsupported file type %q: only JPEG, PNG, GIF and WebP images are allowed", detected)
}
