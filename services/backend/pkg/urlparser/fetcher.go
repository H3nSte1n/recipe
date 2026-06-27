package urlparser

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"io"
	"net/http"
)

// maxResponseBytes caps how much of a fetched page we read into memory, bounding
// memory use against a hostile or accidentally huge response.
const maxResponseBytes = 5 << 20 // 5 MiB

type ContentFetcher interface {
	Fetch(ctx context.Context, url string) (string, error)
}

type contentFetcher struct {
	client *http.Client
	logger *zap.Logger
}

func NewContentFetcher(client *http.Client, logger *zap.Logger) ContentFetcher {
	return &contentFetcher{
		client: client,
		logger: logger,
	}
}

func (f *contentFetcher) Fetch(ctx context.Context, urlStr string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Recipe Parser Bot/1.0")

	resp, err := f.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			f.logger.Error("failed to close response body", zap.Error(err))
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read at most maxResponseBytes+1 so we can distinguish "exactly at the cap"
	// from "over the cap" and reject oversized responses.
	content, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return "", fmt.Errorf("failed to read content: %w", err)
	}
	if len(content) > maxResponseBytes {
		return "", fmt.Errorf("response body exceeds %d byte limit", maxResponseBytes)
	}

	return string(content), nil
}
