package ai

import (
	"encoding/json"
	"fmt"
	"github.com/H3nSte1n/recipe/internal/domain"
	"strings"
)

// validCategories is the allowlist the LLM's categorization output is checked
// against. Anything outside it (including an injected value) is normalized to
// OTHER rather than trusted verbatim.
var validCategories = map[domain.Category]bool{
	domain.CategoryProduce:   true,
	domain.CategoryMeat:      true,
	domain.CategoryDairy:     true,
	domain.CategoryBakery:    true,
	domain.CategoryPantry:    true,
	domain.CategoryFrozen:    true,
	domain.CategoryBeverages: true,
	domain.CategoryHousehold: true,
	domain.CategoryOther:     true,
}

// normalizeCategory upper-cases and validates a category string against the
// allowlist, falling back to OTHER (matching the CategoryOther fallback used
// when adding recipe items to a shopping list).
func normalizeCategory(raw string) string {
	c := domain.Category(strings.ToUpper(strings.TrimSpace(raw)))
	if validCategories[c] {
		return string(c)
	}
	return string(domain.CategoryOther)
}

// clampNonNegative bounds an LLM-supplied integer so an injected negative or
// absurd value cannot flow into the domain model. The upper bound is generous so
// legitimately large recipes are unaffected.
func clampNonNegative(v int) int {
	const maxReasonable = 1_000_000
	if v < 0 {
		return 0
	}
	if v > maxReasonable {
		return maxReasonable
	}
	return v
}

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
		Servings:    clampNonNegative(aiResponse.Servings),
		PrepTime:    clampNonNegative(aiResponse.PrepTime),
		CookTime:    clampNonNegative(aiResponse.CookTime),
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

	// Validate every category against the allowlist so an injected/unknown value
	// cannot reach the domain model.
	for item, category := range result {
		result[item] = normalizeCategory(category)
	}

	return result, nil
}
