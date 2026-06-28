package storage

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectImageType(t *testing.T) {
	webp := append([]byte("RIFF"), append([]byte{0, 0, 0, 0}, []byte("WEBP")...)...)
	cases := []struct {
		name    string
		data    []byte
		wantExt string
		wantErr bool
	}{
		{"jpeg", []byte{0xFF, 0xD8, 0xFF, 0xE0}, ".jpg", false},
		{"png", []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, ".png", false},
		{"gif", []byte("GIF89a"), ".gif", false},
		{"webp", webp, ".webp", false},
		{"svg rejected", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`), "", true},
		{"html rejected", []byte(`<!DOCTYPE html><html><script>alert(1)</script></html>`), "", true},
		{"plain text rejected", []byte("just some text"), "", true},
		{"empty rejected", []byte{}, "", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := bytes.NewReader(c.data)
			_, ext, err := DetectImageType(r)
			if c.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, c.wantExt, ext)

			// r must be rewound to the start for the subsequent copy.
			pos, err := r.Seek(0, 1)
			require.NoError(t, err)
			assert.Equal(t, int64(0), pos)
		})
	}
}

func TestDetectImageType_RewindsLargeFile(t *testing.T) {
	// A valid PNG header followed by >512 bytes of payload.
	data := append([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, []byte(strings.Repeat("x", 1024))...)
	r := bytes.NewReader(data)
	_, ext, err := DetectImageType(r)
	require.NoError(t, err)
	assert.Equal(t, ".png", ext)
	pos, _ := r.Seek(0, 1)
	assert.Equal(t, int64(0), pos)
}
