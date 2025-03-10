package ai

import (
	"encoding/json"
	"fmt"
	"github.com/yourusername/recipe-app/internal/domain"
	"regexp"
	"strconv"
	"strings"
)

func parseAIResponse(response string) (*domain.Recipe, error) {
	startIndex := strings.Index(response, "{")
	endIndex := strings.LastIndex(response, "}")

	if startIndex == -1 || endIndex == -1 {
		return nil, fmt.Errorf("no JSON found in response")
	}

	jsonContent := response[startIndex : endIndex+1]

	// Parse into intermediate struct
	var aiResponse AIRecipeResponse
	if err := json.Unmarshal([]byte(jsonContent), &aiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// Convert to domain model
	recipe := &domain.Recipe{
		Title:       aiResponse.Title,
		Description: aiResponse.Description,
		Servings:    aiResponse.Servings,
		PrepTime:    aiResponse.PrepTime,
		CookTime:    aiResponse.CookTime,
		Notes:       aiResponse.Notes,
		Status:      "draft", // Set default status
	}

	// Convert ingredients
	recipe.Ingredients = make([]domain.RecipeIngredient, len(aiResponse.Ingredients))
	for i, ing := range aiResponse.Ingredients {
		// Parse ingredient description into components
		name, amount, unit, notes := parseIngredient(ing.Description)
		recipe.Ingredients[i] = domain.RecipeIngredient{
			Name:   name,
			Amount: amount,
			Unit:   unit,
			Notes:  notes,
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

func parseIngredient(description string) (name string, amount float64, unit string, notes string) {
	re := regexp.MustCompile(`^(?:(\d+(?:/\d+)?(?:\.\d+)?)\s*([a-zA-Z]+(?:\s+[a-zA-Z]+)?))?\s*([^(]*)(?:\((.*)\))?$`)
	matches := re.FindStringSubmatch(description)

	if len(matches) > 1 && matches[1] != "" {
		// Handle fractions like "1/2"
		if strings.Contains(matches[1], "/") {
			parts := strings.Split(matches[1], "/")
			if len(parts) == 2 {
				num, _ := strconv.ParseFloat(parts[0], 64)
				den, _ := strconv.ParseFloat(parts[1], 64)
				if den != 0 {
					amount = num / den
				}
			}
		} else {
			amount, _ = strconv.ParseFloat(matches[1], 64)
		}
	}
	if len(matches) > 2 {
		unit = strings.TrimSpace(matches[2])
	}
	if len(matches) > 3 {
		name = strings.TrimSpace(matches[3])
	}
	if len(matches) > 4 {
		notes = strings.TrimSpace(matches[4])
	}

	return name, amount, unit, notes
}
