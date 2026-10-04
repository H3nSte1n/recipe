package http

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/H3nSte1n/recipe/internal/mealplan/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type writeRequest struct {
	RecipeID *string `json:"recipe_id"`
	CookDate *string `json:"cook_date"`
	MealType *string `json:"meal_type"`
	Servings *int    `json:"servings"`
	Notes    *string `json:"notes"`
}

func decodeWrite(c *gin.Context) (domain.Fields, error) {
	var req writeRequest
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return domain.Fields{}, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return domain.Fields{}, fmt.Errorf("only one JSON object is allowed")
		}
		return domain.Fields{}, err
	}
	if req.RecipeID == nil || req.CookDate == nil {
		return domain.Fields{}, fmt.Errorf("recipe_id and cook_date are required")
	}
	recipeID, err := uuid.Parse(*req.RecipeID)
	if err != nil || recipeID == uuid.Nil {
		return domain.Fields{}, fmt.Errorf("invalid recipe_id")
	}
	date, err := domain.ParseDate(*req.CookDate)
	if err != nil {
		return domain.Fields{}, err
	}
	return domain.Fields{RecipeID: recipeID, CookDate: date, MealType: req.MealType, Servings: req.Servings, Notes: req.Notes}, nil
}

type recipeResponse struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

type mealResponse struct {
	ID        string         `json:"id"`
	RecipeID  string         `json:"recipe_id"`
	CookDate  string         `json:"cook_date"`
	MealType  *string        `json:"meal_type"`
	Servings  *int           `json:"servings"`
	Notes     *string        `json:"notes"`
	Recipe    recipeResponse `json:"recipe"`
	CreatedAt string         `json:"created_at"`
	UpdatedAt string         `json:"updated_at"`
}

func toResponse(m domain.PlannedMeal) mealResponse {
	return mealResponse{ID: m.ID.String(), RecipeID: m.RecipeID.String(), CookDate: m.CookDate.String(),
		MealType: m.MealType, Servings: m.Servings, Notes: m.Notes,
		Recipe:    recipeResponse{ID: m.Recipe.ID.String(), Title: m.Recipe.Title, Status: m.Recipe.Status},
		CreatedAt: m.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
		UpdatedAt: m.UpdatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00")}
}
