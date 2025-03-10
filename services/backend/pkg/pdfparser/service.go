package pdfparser

import (
	"bytes"
	"context"
	"fmt"
	"github.com/ledongthuc/pdf"
	"github.com/yourusername/recipe-app/internal/domain"
	"github.com/yourusername/recipe-app/pkg/ai"
	"go.uber.org/zap"
)

type Service interface {
	Parse(ctx context.Context, pdfData []byte, aiModel ai.AIModel) (*domain.Recipe, error)
}

type service struct {
	logger *zap.Logger
}

func NewService(logger *zap.Logger) Service {
	return &service{
		logger: logger,
	}
}

func (s *service) Parse(ctx context.Context, pdfData []byte, aiModel ai.AIModel) (*domain.Recipe, error) {
	text, err := s.extractText(pdfData)
	if err != nil {
		s.logger.Error("failed to extract text from PDF", zap.Error(err))
		return nil, fmt.Errorf("failed to extract text from PDF: %w", err)
	}

	recipe, err := aiModel.Parse(ctx, text, "pdf")
	if err != nil {
		s.logger.Error("failed to parse PDF content with AI", zap.Error(err))
		return nil, err
	}

	recipe.Source = "PDF"

	return recipe, nil
}

func (s *service) extractText(pdfData []byte) (string, error) {
	// Create a temporary reader from the PDF data
	reader, err := pdf.NewReader(bytes.NewReader(pdfData), int64(len(pdfData)))
	if err != nil {
		return "", fmt.Errorf("failed to create PDF reader: %w", err)
	}

	var text string
	numPages := reader.NumPage()

	// Extract text from each page
	for i := 1; i <= numPages; i++ {
		page := reader.Page(i)
		content, err := page.GetPlainText(nil)
		if err == nil {
			text += content
		}
	}

	if text == "" {
		return "", fmt.Errorf("no text content found in PDF")
	}

	return text, nil
}
