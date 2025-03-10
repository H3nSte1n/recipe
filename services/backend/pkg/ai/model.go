package ai

import (
	"context"
	"fmt"
	"github.com/yourusername/recipe-app/internal/domain"
	"github.com/yourusername/recipe-app/pkg/config"
	"go.uber.org/zap"
)

type ModelType string

const (
	ModelGPT4    ModelType = "gpt-4"
	ModelGPT35   ModelType = "gpt-3.5-turbo"
	ModelClaude2 ModelType = "claude-3-7-sonnet-latest"
	ModelDefault ModelType = ModelClaude2
)

type AIModel interface {
	Parse(ctx context.Context, content string, contentType string) (*domain.Recipe, error)
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
	case ModelGPT4, ModelGPT35:
		key := apiKey
		if key == "" {
			key = f.config.AI.OpenAIAPIKey
		}
		return NewGPTModel(modelType, key, f.logger), nil
	case ModelClaude2:
		key := apiKey
		if key == "" {
			key = f.config.AI.AnthropicAPIKey
		}
		return NewClaudeModel(key, f.logger), nil
	default:
		return nil, fmt.Errorf("unsupported model type: %s", modelType)
	}
}

// Common helper functions
func createPrompt(content string, contentType string) string {
	return fmt.Sprintf(`Parse the following %s content into a recipe and return it as JSON with this exact structure:

{
    "title": "Recipe Title",
    "description": "Recipe description",
    "servings": 4,
    "prepTime": 30,
    "cookTime": 45,
    "ingredients": [
        {
            "description": "2 cups flour"
        },
        {
            "description": "1 tsp salt"
        }
    ],
    "instructions": [
        {
            "stepNumber": 1,
            "description": "First step description"
        },
        {
            "stepNumber": 2,
            "description": "Second step description"
        }
    ],
    "nutrition": {
        "calories": 350,
        "protein": 12,
        "carbs": 45,
        "fat": 15,
        "fiber": 3,
        "sugar": 8
    }
}

Content to parse:
%s

Important:
- Return valid JSON only
- Follow the exact structure shown above
- Use numbers for numeric values (not strings)
- Include all available information
- If nutrition information is not available, omit the nutrition object
- Ensure proper JSON formatting`, contentType, content)
}
