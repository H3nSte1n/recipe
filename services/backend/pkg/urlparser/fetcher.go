package urlparser

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"io"
	"net/http"
)

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

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read content: %w", err)
	}

	return string(content), nil
}
