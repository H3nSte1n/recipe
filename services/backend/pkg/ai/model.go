package ai

import (
	"context"
	"fmt"
	"github.com/H3nSte1n/recipe/internal/domain"
	"github.com/H3nSte1n/recipe/pkg/config"
	"go.uber.org/zap"
)

type ModelType string

const (
	ModelGPT4      ModelType = "gpt-4"
	ModelGPT4Turbo ModelType = "gpt-4-turbo-preview"
	ModelGPT35     ModelType = "gpt-3.5-turbo"

	ModelClaudeSonnet5 ModelType = "claude-sonnet-5"
	ModelClaudeOpus48  ModelType = "claude-opus-4-8"
	ModelClaudeHaiku45 ModelType = "claude-haiku-4-5"

	ModelDefault ModelType = ModelClaudeHaiku45
)

type AIModel interface {
	Parse(ctx context.Context, content string, contentType string) (*domain.Recipe, error)
	ParseInstructions(ctx context.Context, content string) (*[]domain.RecipeInstruction, error)
	CategorizeItems(ctx context.Context, items []string) (map[string]string, error)
}

type ModelFactory struct {
	config *config.Config
	logger *zap.Logger
}

func NewModelFactory(config *config.Config, logger *zap.Logger) *ModelFactory {
	return &ModelFactory{
		config: config,
		logger: logger,
	}
}

func (f *ModelFactory) CreateModel(modelType ModelType, apiKey string) (AIModel, error) {
	switch modelType {
	case ModelGPT4, ModelGPT4Turbo, ModelGPT35:
		key := apiKey
		if key == "" {
			key = f.config.AI.OpenAIAPIKey
		}
		return NewGPTModel(modelType, key, f.logger), nil
	case ModelClaudeSonnet5, ModelClaudeOpus48, ModelClaudeHaiku45:
		key := apiKey
		if key == "" {
			key = f.config.AI.AnthropicAPIKey
		}
		return NewClaudeModel(string(modelType), key, f.logger), nil
	default:
		return nil, fmt.Errorf("unsupported model type: %s", modelType)
	}
}
