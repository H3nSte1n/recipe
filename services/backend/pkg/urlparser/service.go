package urlparser

import (
	"context"
	"github.com/H3nSte1n/recipe/internal/domain"
	"github.com/H3nSte1n/recipe/pkg/ai"
	"go.uber.org/zap"
	"net/http"
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
	client := &http.Client{
		CheckRedirect: defaultRedirectPolicy,
	}

	return &service{
		client:  client,
		logger:  logger,
		fetcher: NewContentFetcher(client, logger),
		parser:  NewContentParser(logger),
	}
}

func (s *service) Parse(ctx context.Context, urlStr string, aiModel ai.AIModel) (*domain.Recipe, error) {
	// Fetch raw content
	content, err := s.fetcher.Fetch(ctx, urlStr)
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
