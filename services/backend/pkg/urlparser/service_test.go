package urlparser

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestServiceParse_RejectsInternalDestination is the end-to-end wiring test: it
// goes through the real Service.Parse entry point (not the helpers in isolation)
// and asserts an internal destination is rejected. Because validation happens
// before any fetch, this makes no network call and the nil aiModel is never
// reached — a missing call to validatePublicURL would let this through.
func TestServiceParse_RejectsInternalDestination(t *testing.T) {
	svc := NewService(zap.NewNop())

	for _, raw := range []string{
		"http://169.254.169.254/latest/meta-data/", // cloud metadata
		"http://127.0.0.1/",                        // loopback
		"http://10.0.0.1/",                         // RFC1918 (covers Docker app/db range)
		"ftp://example.com/",                       // bad scheme
	} {
		_, err := svc.Parse(context.Background(), raw, nil)
		assert.Error(t, err, "destination should be rejected: %s", raw)
	}
}

func TestServiceParse_RejectsNonHTTPScheme(t *testing.T) {
	svc := NewService(zap.NewNop())
	_, err := svc.Parse(context.Background(), "file:///etc/passwd", nil)
	require.Error(t, err)
}
