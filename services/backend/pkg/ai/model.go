package ai

import (
	"context"
	"encoding/json"
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

	ModelClaude35Sonnet ModelType = "claude-3-5-sonnet-20241022"
	ModelClaude3Opus    ModelType = "claude-3-opus-20240229"
	ModelClaude3Sonnet  ModelType = "claude-3-sonnet-20240229"
	ModelClaude3Haiku   ModelType = "claude-3-haiku-20240307"

	ModelDefault ModelType = ModelClaude3Haiku
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
	case ModelClaude35Sonnet, ModelClaude3Opus, ModelClaude3Sonnet, ModelClaude3Haiku:
		key := apiKey
		if key == "" {
			key = f.config.AI.AnthropicAPIKey
		}
		return NewClaudeModel(string(modelType), key, f.logger), nil
	default:
		return nil, fmt.Errorf("unsupported model type: %s", modelType)
	}
}

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
            "name": "flour",
            "description": "2 cups flour",
            "amount": 2,
            "unit": "cups",
            "notes": ""
        },
        {
            "name": "salt",
            "description": "1 tsp salt",
            "amount": 1,
            "unit": "tsp",
            "notes": ""
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
- For ingredients: "description" is the full original string (e.g. "2 tablespoons olive oil"), "name" is only the ingredient name (e.g. "olive oil", "garlic"), "amount" is the numeric quantity, "unit" is only the unit of measurement (e.g. "tablespoons", "cups"), "notes" is any extra info (e.g. "chopped", "peeled and diced")
- Do NOT mix the ingredient name into the unit field or vice versa
- If no unit applies (e.g. "2 eggs"), leave "unit" as an empty string
- Include all available information
- If nutrition information is not available, omit the nutrition object
- Ensure proper JSON formatting`, contentType, content)
}

func createParseInstructionsPrompt(content string) string {
	const promptTemplate = `You are a recipe parsing assistant.

Parse the following recipe content into a numbered list of instructions.

Rules:
- Return a JSON array where each element has "step_number" (integer) and "instruction" (string)
- Preserve the original wording of each step
- Do NOT include markdown, code blocks, explanations, or any other text
- Output must start with [ and end with ]

Example output:
[
    {"step_number": 1, "instruction": "Preheat the oven to 180°C."},
    {"step_number": 2, "instruction": "Mix flour and sugar in a bowl."}
]

Content to parse:
%s`

	return fmt.Sprintf(promptTemplate, content)
}

func createPromptToCategorizeShoppingListItems(items []string) string {
	itemsJSON, _ := json.Marshal(items)
	const promptTemplate = `You are a grocery categorization assistant.

Categorize each item in the JSON array below into exactly one of these categories:
PRODUCE, MEAT, DAIRY, BAKERY, PANTRY, FROZEN, BEVERAGES, HOUSEHOLD, OTHER

Rules:
- Return a JSON object where each key is the exact item name from the input and the value is its category
- Every item from the input must appear as a key in the output
- Use only the category values listed above
- Do NOT include markdown, code blocks, explanations, or any other text
- Output must start with { and end with }

Example input:  ["eggs","spinach","olive oil"]
Example output: {"eggs":"DAIRY","spinach":"PRODUCE","olive oil":"PANTRY"}

Input: %s`

	return fmt.Sprintf(promptTemplate, string(itemsJSON))
}
