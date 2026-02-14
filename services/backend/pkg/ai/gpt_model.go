package ai

import (
	"context"
	"fmt"
	"github.com/H3nSte1n/recipe/internal/domain"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

type GPTModel struct {
	modelType ModelType
	client    *openai.Client
	logger    *zap.Logger
}

func NewGPTModel(modelType ModelType, apiKey string, logger *zap.Logger) *GPTModel {
	return &GPTModel{
		modelType: modelType,
		client:    openai.NewClient(apiKey),
		logger:    logger,
	}
}

func (m *GPTModel) Parse(ctx context.Context, content string, contentType string) (*domain.Recipe, error) {
	prompt := createPrompt(content, contentType)

	resp, err := m.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:     string(m.modelType),
		Messages:  []openai.ChatCompletionMessage{{Role: "user", Content: prompt}},
		MaxTokens: 2000,
	})
	if err != nil {
		return nil, fmt.Errorf("GPT API error: %w", err)
	}

	return parseAIResponse(resp.Choices[0].Message.Content)
}

func (m *GPTModel) ParseInstructions(ctx context.Context, content string) (*[]domain.RecipeInstruction, error) {
	prompt := createParseInstructionsPrompt(content)

	resp, err := m.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:     string(m.modelType),
		Messages:  []openai.ChatCompletionMessage{{Role: "user", Content: prompt}},
		MaxTokens: 2000,
	})

	if err != nil {
		return nil, fmt.Errorf("GPT API error: %w", err)
	}

	return parseInstructions(resp.Choices[0].Message.Content)
}

func (m *GPTModel) CategorizeItems(ctx context.Context, content []string) ([]string, error) {
	prompt := createPromptToCategorizeShoppingListItems(content)

	resp, err := m.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:     string(m.modelType),
		Messages:  []openai.ChatCompletionMessage{{Role: "user", Content: prompt}},
		MaxTokens: 2000,
	})

	if err != nil {
		return nil, fmt.Errorf("GPT API error: %w", err)
	}

	return parseCategorizeItemsResponse(resp.Choices[0].Message.Content)
}
