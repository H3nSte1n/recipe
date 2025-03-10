package ai

import (
	"context"
	"fmt"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/yourusername/recipe-app/internal/domain"
	"go.uber.org/zap"
)

type ClaudeModel struct {
	client *anthropic.Client
	logger *zap.Logger
}

func NewClaudeModel(apiKey string, logger *zap.Logger) *ClaudeModel {
	return &ClaudeModel{
		client: anthropic.NewClient(option.WithAPIKey(apiKey)),
		logger: logger,
	}
}

func (m *ClaudeModel) Parse(ctx context.Context, content string, contentType string) (*domain.Recipe, error) {
	prompt := createPrompt(content, contentType)

	message, err := m.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.F(anthropic.ModelClaude3_7SonnetLatest),
		MaxTokens: anthropic.F(int64(2000)),
		Messages: anthropic.F([]anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		}),
	})

	if err != nil {
		return nil, fmt.Errorf("Claude API error: %w", err)
	}

	fmt.Print(message)

	if len(message.Content) > 0 {
		return parseAIResponse(message.Content[0].Text)
	}

	return nil, fmt.Errorf("no response content from Claude")
}
