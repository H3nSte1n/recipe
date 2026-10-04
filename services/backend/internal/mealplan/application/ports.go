package application

import (
	"context"

	"github.com/H3nSte1n/recipe/internal/mealplan/domain"
	"github.com/google/uuid"
)

// Repository owns the transaction around checking recipe ownership and writing.
// A recipe deletion cannot interleave between these operations.
type Repository interface {
	CreateWithOwnedRecipe(context.Context, uuid.UUID, domain.Fields) (domain.PlannedMeal, error)
	ReplaceWithOwnedRecipe(context.Context, uuid.UUID, uuid.UUID, domain.Fields) (domain.PlannedMeal, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (domain.PlannedMeal, error)
	List(context.Context, uuid.UUID, domain.Date, domain.Date, int, int) ([]domain.PlannedMeal, int64, error)
	Delete(context.Context, uuid.UUID, uuid.UUID) error
}
