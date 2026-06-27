package urlparser

import (
	"context"
	"github.com/H3nSte1n/recipe/internal/domain"
	"github.com/H3nSte1n/recipe/pkg/ai"
	"go.uber.org/zap"
	"net/http"
	"time"
)

const (
	// requestTimeout bounds the entire URL-import request (connect + redirects + read).
	requestTimeout = 15 * time.Second
)

type Service interface {
	Parse(ctx context.Context, url string, aiModel ai.AIModel) (*domain.Recipe, error)
}

type service struct {
	client  *http.Client
	logger  *zap.Logger
	fetcher ContentFetcher
	parser  ContentParser
}

func NewService(logger *zap.Logger) Service {
	// Transport whose every dial (including redirect hops) is forced through the
	// SSRF guard, with timeouts so a slow or stalling host cannot hang the request.
	transport := &http.Transport{
		DialContext:           safeDialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	client := &http.Client{
		CheckRedirect: defaultRedirectPolicy,
		Transport:     transport,
		Timeout:       requestTimeout,
	}

	return &service{
		client:  client,
		logger:  logger,
		fetcher: NewContentFetcher(client, logger),
		parser:  NewContentParser(logger),
	}
}

func (s *service) Parse(ctx context.Context, urlStr string, aiModel ai.AIModel) (*domain.Recipe, error) {
	// Reject non-public destinations early (scheme + resolved-IP check). The
	// per-dial safeDialContext is the authoritative TOCTOU-safe guard; this gives
	// a fast, clear rejection before any connection is attempted.
	if _, err := validatePublicURL(ctx, urlStr); err != nil {
		s.logger.Warn("rejected URL import destination",
			zap.String("url", urlStr),
			zap.Error(err))
		return nil, err
	}

	// Bound the fetch independently of any caller deadline.
	fetchCtx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	// Fetch raw content
	content, err := s.fetcher.Fetch(fetchCtx, urlStr)
	if err != nil {
		s.logger.Error("failed to fetch content",
			zap.String("url", urlStr),
			zap.Error(err))
		return nil, err
	}

	// Extract relevant content
	cleanContent, err := s.parser.Parse(content)
	if err != nil {
		s.logger.Error("failed to extract recipe content",
			zap.String("url", urlStr),
			zap.Error(err))
		return nil, err
	}

	// Parse with AI model
	object, err := aiModel.Parse(ctx, cleanContent, "webpage")
	if err != nil {
		s.logger.Error("failed to parse content with AI",
			zap.String("url", urlStr),
			zap.Error(err))
		return nil, err
	}

	object.Source = urlStr
	object.SourceType = "URL"

	return object, nil
}
