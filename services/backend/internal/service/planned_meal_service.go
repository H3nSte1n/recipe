package service

import (
	"context"
	"fmt"
	"strings"
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
	// maxPlannedMealsPerUser bounds storage per account. It is checked before the insert without a
	// lock, so a burst of concurrent creates can overshoot it slightly; that is acceptable here.
	maxPlannedMealsPerUser = 5000
)

type plannedMealRepository interface {
	LockOwnedRecipe(ctx context.Context, userID, recipeID string) (*domain.PlannedMealRecipe, error)
	GetByID(ctx context.Context, userID, id string) (*domain.PlannedMeal, error)
	GetByIDForUpdate(ctx context.Context, userID, id string) (*domain.PlannedMeal, error)
	CountByUser(ctx context.Context, userID string) (int64, error)
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
	// The meal is read back in the same transaction so a delete right after commit cannot make a
	// successful create look like a failure.
	var created *domain.PlannedMeal
	err = s.repo.WithTypedTransaction(ctx, func(tx repository.PlannedMealRepository) error {
		if err := requirePublishedRecipe(ctx, tx, userID, meal.RecipeID); err != nil {
			return err
		}
		total, err := tx.CountByUser(ctx, userID)
		if err != nil {
			return err
		}
		if total >= maxPlannedMealsPerUser {
			return invalidPlannedMeal("you can plan at most %d meals", maxPlannedMealsPerUser)
		}
		if err := tx.Create(ctx, meal); err != nil {
			return err
		}
		stored, err := tx.GetByID(ctx, userID, meal.ID)
		created = stored
		return err
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (s *plannedMealService) Update(ctx context.Context, userID string, id string, req *domain.PlannedMealRequest) (*domain.PlannedMeal, error) {
	id, err := canonicalMealID(id)
	if err != nil {
		return nil, err
	}
	meal, err := newPlannedMeal(userID, req)
	if err != nil {
		return nil, err
	}
	meal.ID = id

	var updated *domain.PlannedMeal
	err = s.repo.WithTypedTransaction(ctx, func(tx repository.PlannedMealRepository) error {
		// Locked so a concurrent update or delete cannot change the recipe this check compares against.
		current, err := tx.GetByIDForUpdate(ctx, userID, id)
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
		if err := tx.Update(ctx, meal); err != nil {
			return err
		}
		updated, err = tx.GetByID(ctx, userID, id)
		return err
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *plannedMealService) GetByID(ctx context.Context, userID string, id string) (*domain.PlannedMeal, error) {
	id, err := canonicalMealID(id)
	if err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, userID, id)
}

func (s *plannedMealService) Delete(ctx context.Context, userID string, id string) error {
	id, err := canonicalMealID(id)
	if err != nil {
		return err
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

// canonicalMealID returns the id in the lowercase hyphenated form Postgres accepts. uuid.Parse also
// takes forms such as "urn:uuid:..." that the database rejects, which would surface as a 500.
func canonicalMealID(id string) (string, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return "", errors.ErrNotFound
	}
	return parsed.String(), nil
}

func requirePublishedRecipe(ctx context.Context, repo plannedMealRepository, userID, recipeID string) error {
	recipe, err := repo.LockOwnedRecipe(ctx, userID, recipeID)
	if errors.IsNotFound(err) {
		// Same answer for a missing and a foreign recipe, so ids of other users' recipes stay private.
		return invalidPlannedMeal("recipe not found")
	}
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
	recipeID, err := uuid.Parse(req.RecipeID)
	if err != nil || recipeID == uuid.Nil {
		return nil, invalidPlannedMeal("invalid recipe_id")
	}
	if _, err := parseCalendarDate(req.CookDate); err != nil {
		return nil, invalidPlannedMeal("invalid cook_date")
	}

	// Empty strings (and notes with only whitespace) mean "not set"; store them as NULL.
	mealType, notes := req.MealType, req.Notes
	if mealType != nil && *mealType == "" {
		mealType = nil
	}
	if notes != nil {
		trimmed := strings.TrimSpace(*notes)
		if trimmed == "" {
			notes = nil
		} else {
			notes = &trimmed
		}
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
		RecipeID: recipeID.String(), // canonical lowercase, so it compares equal to stored IDs
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
