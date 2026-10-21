package domain

import "time"

type MealType string

const (
	MealTypeBreakfast MealType = "breakfast"
	MealTypeLunch     MealType = "lunch"
	MealTypeDinner    MealType = "dinner"
	MealTypeSnack     MealType = "snack"
)

// PlannedMeal schedules one of the user's recipes on a calendar day. CookDate is a date-only
// "YYYY-MM-DD" string so it never shifts with timezones.
type PlannedMeal struct {
	ID        string            `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	UserID    string            `json:"-" gorm:"type:uuid;not null"`
	RecipeID  string            `json:"recipe_id" gorm:"type:uuid;not null"`
	CookDate  string            `json:"cook_date" gorm:"type:date;not null"`
	MealType  *string           `json:"meal_type"`
	Servings  *int              `json:"servings"`
	Notes     *string           `json:"notes"`
	Recipe    PlannedMealRecipe `json:"recipe" gorm:"-"`
	CreatedAt time.Time         `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time         `json:"updated_at" gorm:"autoUpdateTime"`
}

// PlannedMealRecipe is the slice of a recipe shown next to a planned meal.
type PlannedMealRecipe struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

type PlannedMealRequest struct {
	RecipeID string  `json:"recipe_id" binding:"required"`
	CookDate string  `json:"cook_date" binding:"required"`
	MealType *string `json:"meal_type"`
	Servings *int    `json:"servings"`
	Notes    *string `json:"notes"`
}

type PlannedMealPage struct {
	Items    []PlannedMeal `json:"items"`
	Total    int64         `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
}
