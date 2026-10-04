package domain

import (
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

type RecipeSummary struct {
	ID     uuid.UUID
	Title  string
	Status string
}

type PlannedMeal struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	RecipeID  uuid.UUID
	CookDate  Date
	MealType  *string
	Servings  *int
	Notes     *string
	Recipe    RecipeSummary
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Fields struct {
	RecipeID uuid.UUID
	CookDate Date
	MealType *string
	Servings *int
	Notes    *string
}

func (f Fields) Validate() error {
	if f.RecipeID == uuid.Nil {
		return fmt.Errorf("recipe_id is required")
	}
	if _, err := ParseDate(string(f.CookDate)); err != nil {
		return fmt.Errorf("invalid cook_date")
	}
	if f.MealType != nil {
		switch *f.MealType {
		case "breakfast", "lunch", "dinner", "snack":
		default:
			return fmt.Errorf("invalid meal_type")
		}
	}
	if f.Servings != nil && (*f.Servings < 1 || *f.Servings > 100) {
		return fmt.Errorf("servings must be 1 to 100")
	}
	if f.Notes != nil && utf8.RuneCountInString(*f.Notes) > 1000 {
		return fmt.Errorf("notes must be at most 1000 characters")
	}
	return nil
}
