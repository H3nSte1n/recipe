package ai

import (
	"encoding/json"
	"fmt"
	"github.com/H3nSte1n/recipe/internal/domain"
	"strings"
)

func parseAIResponse(response string) (*domain.Recipe, error) {
	startIndex := strings.Index(response, "{")
	endIndex := strings.LastIndex(response, "}")

	if startIndex == -1 || endIndex == -1 {
		return nil, fmt.Errorf("no JSON found in response")
	}

	jsonContent := response[startIndex : endIndex+1]

	var aiResponse AIRecipeResponse
	if err := json.Unmarshal([]byte(jsonContent), &aiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	recipe := &domain.Recipe{
		Title:       aiResponse.Title,
		Description: aiResponse.Description,
		Servings:    aiResponse.Servings,
		PrepTime:    aiResponse.PrepTime,
		CookTime:    aiResponse.CookTime,
		Notes:       aiResponse.Notes,
		Status:      "draft",
	}

	recipe.Ingredients = make([]domain.RecipeIngredient, len(aiResponse.Ingredients))
	for i, ing := range aiResponse.Ingredients {
		recipe.Ingredients[i] = domain.RecipeIngredient{
			Name:        ing.Name,
			Description: ing.Description,
			Amount:      ing.Amount,
			Unit:        ing.Unit,
			Notes:       ing.Notes,
		}
	}

	// Convert instructions
	recipe.Instructions = make([]domain.RecipeInstruction, len(aiResponse.Instructions))
	for i, inst := range aiResponse.Instructions {
		recipe.Instructions[i] = domain.RecipeInstruction{
			StepNumber:  inst.StepNumber,
			Instruction: inst.Description,
		}
	}

	// Convert nutrition if available
	if aiResponse.Nutrition != nil {
		recipe.Nutrition = &domain.RecipeNutrition{
			BaseNutrition: domain.BaseNutrition{
				Calories:   aiResponse.Nutrition.Calories,
				PerServing: true,
			},
			MacroNutrition: domain.MacroNutrition{
				Protein:      aiResponse.Nutrition.Protein,
				Carbs:        aiResponse.Nutrition.Carbs,
				Fat:          aiResponse.Nutrition.Fat,
				Fiber:        aiResponse.Nutrition.Fiber,
				Sugar:        aiResponse.Nutrition.Sugar,
				SaturatedFat: aiResponse.Nutrition.SaturatedFat,
				Cholesterol:  aiResponse.Nutrition.Cholesterol,
				Sodium:       aiResponse.Nutrition.Sodium,
			},
		}
	}

	if recipe.Title == "" {
		return nil, fmt.Errorf("parsed recipe missing title")
	}

	return recipe, nil
}

func stripMarkdownFences(content string) string {
	if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	}
	return content
}

func parseInstructions(content string) (*[]domain.RecipeInstruction, error) {
	content = strings.TrimSpace(content)
	content = stripMarkdownFences(content)

	// Validate that content is a JSON array
	if !strings.HasPrefix(content, "[") || !strings.HasSuffix(content, "]") {
		return nil, fmt.Errorf("invalid JSON array format")
	}

	// Parse into intermediate struct
	var aiResponse []AIRecipeInstructions
	if err := json.Unmarshal([]byte(content), &aiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// Convert to domain model
	instructions := make([]domain.RecipeInstruction, len(aiResponse))
	for i, inst := range aiResponse {
		instructions[i] = domain.RecipeInstruction{
			StepNumber:  inst.StepNumber,
			Instruction: inst.Instruction,
		}
	}

	return &instructions, nil
}

func parseCategorizeItemsResponse(content string) (map[string]string, error) {
	content = strings.TrimSpace(content)
	content = stripMarkdownFences(content)

	// Validate that content is a JSON object
	if !strings.HasPrefix(content, "{") || !strings.HasSuffix(content, "}") {
		return nil, fmt.Errorf("invalid JSON object format")
	}

	var result map[string]string
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return result, nil
}
