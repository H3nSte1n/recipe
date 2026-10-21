package service

import (
	"context"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/H3nSte1n/recipe/internal/domain"
	"github.com/H3nSte1n/recipe/internal/errors"
	"github.com/H3nSte1n/recipe/internal/repository"
	"github.com/google/uuid"
)

const (
	dateLayout         = "2006-01-02"
	maxPlannerRange    = 93 // days, inclusive
	maxPlannerPageSize = 100
	maxPlannedServings = 100
	maxPlannedNotesLen = 1000
)

type plannedMealRepository interface {
	LockOwnedRecipe(ctx context.Context, userID, recipeID string) (*domain.PlannedMealRecipe, error)
	GetByID(ctx context.Context, userID, id string) (*domain.PlannedMeal, error)
	Create(ctx context.Context, meal *domain.PlannedMeal) error
	Update(ctx context.Context, meal *domain.PlannedMeal) error
	Delete(ctx context.Context, userID, id string) error
	List(ctx context.Context, userID, from, to string, limit, offset int) ([]domain.PlannedMeal, int64, error)
	WithTypedTransaction(ctx context.Context, fn func(repository.PlannedMealRepository) error) error
}

type PlannedMealService interface {
	Create(ctx context.Context, userID string, req *domain.PlannedMealRequest) (*domain.PlannedMeal, error)
	Update(ctx context.Context, userID string, id string, req *domain.PlannedMealRequest) (*domain.PlannedMeal, error)
	GetByID(ctx context.Context, userID string, id string) (*domain.PlannedMeal, error)
	Delete(ctx context.Context, userID string, id string) error
	List(ctx context.Context, userID string, from, to string, page, pageSize int) (*domain.PlannedMealPage, error)
}

type plannedMealService struct {
	repo plannedMealRepository
}

func NewPlannedMealService(repo plannedMealRepository) PlannedMealService {
	return &plannedMealService{repo: repo}
}

func invalidPlannedMeal(format string, args ...any) error {
	return errors.New(fmt.Sprintf(format, args...), "INVALID")
}

func (s *plannedMealService) Create(ctx context.Context, userID string, req *domain.PlannedMealRequest) (*domain.PlannedMeal, error) {
	meal, err := newPlannedMeal(userID, req)
	if err != nil {
		return nil, err
	}

	// The recipe row stays locked until the meal is written, so it cannot be deleted in between.
	err = s.repo.WithTypedTransaction(ctx, func(tx repository.PlannedMealRepository) error {
		if err := requirePublishedRecipe(ctx, tx, userID, meal.RecipeID); err != nil {
			return err
		}
		return tx.Create(ctx, meal)
	})
	if err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, userID, meal.ID)
}

func (s *plannedMealService) Update(ctx context.Context, userID string, id string, req *domain.PlannedMealRequest) (*domain.PlannedMeal, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, errors.ErrNotFound
	}
	meal, err := newPlannedMeal(userID, req)
	if err != nil {
		return nil, err
	}
	meal.ID = id

	err = s.repo.WithTypedTransaction(ctx, func(tx repository.PlannedMealRepository) error {
		current, err := tx.GetByID(ctx, userID, id)
		if err != nil {
			return err
		}
		// A meal may keep the recipe it already points at even if that recipe was unpublished
		// since; only switching to a different recipe requires a published one.
		if current.RecipeID != meal.RecipeID {
			if err := requirePublishedRecipe(ctx, tx, userID, meal.RecipeID); err != nil {
				return err
			}
		}
		return tx.Update(ctx, meal)
	})
	if err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, userID, id)
}

func (s *plannedMealService) GetByID(ctx context.Context, userID string, id string) (*domain.PlannedMeal, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, errors.ErrNotFound
	}
	return s.repo.GetByID(ctx, userID, id)
}

func (s *plannedMealService) Delete(ctx context.Context, userID string, id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return errors.ErrNotFound
	}
	return s.repo.Delete(ctx, userID, id)
}

func (s *plannedMealService) List(ctx context.Context, userID string, from, to string, page, pageSize int) (*domain.PlannedMealPage, error) {
	if err := validatePlannerRange(from, to); err != nil {
		return nil, err
	}
	if page < 1 || pageSize < 1 || pageSize > maxPlannerPageSize || int64(page-1)*int64(pageSize) > 2147483647 {
		return nil, invalidPlannedMeal("invalid page")
	}

	meals, total, err := s.repo.List(ctx, userID, from, to, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, err
	}
	if meals == nil {
		meals = []domain.PlannedMeal{} // serialise as [] rather than null
	}
	return &domain.PlannedMealPage{Items: meals, Total: total, Page: page, PageSize: pageSize}, nil
}

func requirePublishedRecipe(ctx context.Context, repo plannedMealRepository, userID, recipeID string) error {
	recipe, err := repo.LockOwnedRecipe(ctx, userID, recipeID)
	if err != nil {
		return err
	}
	if recipe.Status != "published" {
		return invalidPlannedMeal("only published recipes can be planned")
	}
	return nil
}

// newPlannedMeal validates the request and turns it into a meal owned by userID.
func newPlannedMeal(userID string, req *domain.PlannedMealRequest) (*domain.PlannedMeal, error) {
	if id, err := uuid.Parse(req.RecipeID); err != nil || id == uuid.Nil {
		return nil, invalidPlannedMeal("invalid recipe_id")
	}
	if _, err := parseCalendarDate(req.CookDate); err != nil {
		return nil, invalidPlannedMeal("invalid cook_date")
	}

	// Empty strings mean "not set"; store them as NULL.
	mealType, notes := req.MealType, req.Notes
	if mealType != nil && *mealType == "" {
		mealType = nil
	}
	if notes != nil && *notes == "" {
		notes = nil
	}

	if mealType != nil {
		switch domain.MealType(*mealType) {
		case domain.MealTypeBreakfast, domain.MealTypeLunch, domain.MealTypeDinner, domain.MealTypeSnack:
		default:
			return nil, invalidPlannedMeal("invalid meal_type")
		}
	}
	if req.Servings != nil && (*req.Servings < 1 || *req.Servings > maxPlannedServings) {
		return nil, invalidPlannedMeal("servings must be 1 to %d", maxPlannedServings)
	}
	if notes != nil && utf8.RuneCountInString(*notes) > maxPlannedNotesLen {
		return nil, invalidPlannedMeal("notes must be at most %d characters", maxPlannedNotesLen)
	}

	return &domain.PlannedMeal{
		UserID:   userID,
		RecipeID: req.RecipeID,
		CookDate: req.CookDate,
		MealType: mealType,
		Servings: req.Servings,
		Notes:    notes,
	}, nil
}

// parseCalendarDate accepts only a real calendar day written exactly as YYYY-MM-DD.
func parseCalendarDate(value string) (time.Time, error) {
	t, err := time.Parse(dateLayout, value)
	if err != nil || t.Year() == 0 || t.Format(dateLayout) != value {
		return time.Time{}, fmt.Errorf("invalid calendar date")
	}
	return t, nil
}

func validatePlannerRange(from, to string) error {
	start, err := parseCalendarDate(from)
	if err != nil {
		return invalidPlannedMeal("invalid from date")
	}
	end, err := parseCalendarDate(to)
	if err != nil {
		return invalidPlannedMeal("invalid to date")
	}
	if end.Before(start) || end.Sub(start) > (maxPlannerRange-1)*24*time.Hour {
		return invalidPlannedMeal("date range must be 1 to %d days", maxPlannerRange)
	}
	return nil
}
