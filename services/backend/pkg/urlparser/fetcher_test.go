package urlparser

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests inject a plain *http.Client pointed at a loopback httptest server.
// The fetcher itself contains no SSRF logic (that lives in safeDialContext), so
// it can be exercised against 127.0.0.1 without tripping the guard.

func TestFetch_RejectsOversizedBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(strings.Repeat("a", maxResponseBytes+10)))
	}))
	defer srv.Close()

	fetcher := NewContentFetcher(srv.Client(), zap.NewNop())
	_, err := fetcher.Fetch(context.Background(), srv.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "limit")
}

func TestFetch_AcceptsBodyAtLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("small body"))
	}))
	defer srv.Close()

	fetcher := NewContentFetcher(srv.Client(), zap.NewNop())
	content, err := fetcher.Fetch(context.Background(), srv.URL)
	require.NoError(t, err)
	assert.Equal(t, "small body", content)
}

func TestFetch_HonorsClientTimeout(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		<-release // block until the test releases it
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	defer close(release)

	client := srv.Client()
	client.Timeout = 50 * time.Millisecond

	fetcher := NewContentFetcher(client, zap.NewNop())
	start := time.Now()
	_, err := fetcher.Fetch(context.Background(), srv.URL)
	require.Error(t, err)
	assert.Less(t, time.Since(start), 5*time.Second, "should time out quickly, not hang")
}
