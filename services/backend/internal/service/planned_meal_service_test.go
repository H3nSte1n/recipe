package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/H3nSte1n/recipe/internal/domain"
	internalErr "github.com/H3nSte1n/recipe/internal/errors"
	"github.com/H3nSte1n/recipe/internal/repository"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockPlannedMealRepository struct {
	mock.Mock
}

func (m *mockPlannedMealRepository) LockOwnedRecipe(ctx context.Context, userID, recipeID string) (*domain.PlannedMealRecipe, error) {
	args := m.Called(ctx, userID, recipeID)
	v, _ := args.Get(0).(*domain.PlannedMealRecipe)
	return v, args.Error(1)
}

func (m *mockPlannedMealRepository) GetByID(ctx context.Context, userID, id string) (*domain.PlannedMeal, error) {
	args := m.Called(ctx, userID, id)
	v, _ := args.Get(0).(*domain.PlannedMeal)
	return v, args.Error(1)
}

func (m *mockPlannedMealRepository) Create(ctx context.Context, meal *domain.PlannedMeal) error {
	return m.Called(ctx, meal).Error(0)
}

func (m *mockPlannedMealRepository) Update(ctx context.Context, meal *domain.PlannedMeal) error {
	return m.Called(ctx, meal).Error(0)
}

func (m *mockPlannedMealRepository) Delete(ctx context.Context, userID, id string) error {
	return m.Called(ctx, userID, id).Error(0)
}

func (m *mockPlannedMealRepository) List(ctx context.Context, userID, from, to string, limit, offset int) ([]domain.PlannedMeal, int64, error) {
	args := m.Called(ctx, userID, from, to, limit, offset)
	v, _ := args.Get(0).([]domain.PlannedMeal)
	return v, int64(args.Int(1)), args.Error(2)
}

// WithTypedTransaction runs the closure against the mock itself so the inner expectations fire.
func (m *mockPlannedMealRepository) WithTypedTransaction(ctx context.Context, fn func(repository.PlannedMealRepository) error) error {
	return fn(m)
}

const (
	plannedUserID   = "550e8400-e29b-41d4-a716-446655440000"
	plannedRecipeID = "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
	plannedMealID   = "6ba7b811-9dad-11d1-80b4-00c04fd430c8"
)

func plannedRequest() *domain.PlannedMealRequest {
	return &domain.PlannedMealRequest{RecipeID: plannedRecipeID, CookDate: "2026-10-03"}
}

func publishedRecipe() *domain.PlannedMealRecipe {
	return &domain.PlannedMealRecipe{ID: plannedRecipeID, Title: "Pasta", Status: "published"}
}

func TestPlannedMealService_Create(t *testing.T) {
	ctx := context.Background()
	created := &domain.PlannedMeal{ID: plannedMealID, RecipeID: plannedRecipeID, CookDate: "2026-10-03"}

	t.Run("stores the meal for the user after locking a published recipe", func(t *testing.T) {
		repo := new(mockPlannedMealRepository)
		repo.On("LockOwnedRecipe", ctx, plannedUserID, plannedRecipeID).Return(publishedRecipe(), nil).Once()
		repo.On("Create", ctx, mock.MatchedBy(func(m *domain.PlannedMeal) bool {
			return m.UserID == plannedUserID && m.RecipeID == plannedRecipeID && m.CookDate == "2026-10-03"
		})).Return(nil).Once()
		repo.On("GetByID", ctx, plannedUserID, mock.Anything).Return(created, nil).Once()

		meal, err := NewPlannedMealService(repo).Create(ctx, plannedUserID, plannedRequest())

		require.NoError(t, err)
		require.Equal(t, created, meal)
		repo.AssertExpectations(t)
	})

	t.Run("treats empty meal type and notes as unset", func(t *testing.T) {
		repo := new(mockPlannedMealRepository)
		empty := ""
		req := plannedRequest()
		req.MealType, req.Notes = &empty, &empty
		repo.On("LockOwnedRecipe", ctx, plannedUserID, plannedRecipeID).Return(publishedRecipe(), nil).Once()
		repo.On("Create", ctx, mock.MatchedBy(func(m *domain.PlannedMeal) bool { return m.MealType == nil && m.Notes == nil })).Return(nil).Once()
		repo.On("GetByID", ctx, plannedUserID, mock.Anything).Return(created, nil).Once()

		_, err := NewPlannedMealService(repo).Create(ctx, plannedUserID, req)

		require.NoError(t, err)
		repo.AssertExpectations(t)
	})

	t.Run("rejects a recipe the user does not own", func(t *testing.T) {
		repo := new(mockPlannedMealRepository)
		repo.On("LockOwnedRecipe", ctx, plannedUserID, plannedRecipeID).Return(nil, internalErr.ErrNotFound).Once()

		_, err := NewPlannedMealService(repo).Create(ctx, plannedUserID, plannedRequest())

		require.True(t, internalErr.IsNotFound(err))
		repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})

	for _, status := range []string{"draft", "archived"} {
		t.Run("rejects a "+status+" recipe", func(t *testing.T) {
			repo := new(mockPlannedMealRepository)
			repo.On("LockOwnedRecipe", ctx, plannedUserID, plannedRecipeID).Return(&domain.PlannedMealRecipe{ID: plannedRecipeID, Status: status}, nil).Once()

			_, err := NewPlannedMealService(repo).Create(ctx, plannedUserID, plannedRequest())

			require.True(t, internalErr.IsInvalid(err))
			repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
		})
	}

	t.Run("returns repository failures unchanged", func(t *testing.T) {
		repo := new(mockPlannedMealRepository)
		boom := errors.New("storage unavailable")
		repo.On("LockOwnedRecipe", ctx, plannedUserID, plannedRecipeID).Return(publishedRecipe(), nil).Once()
		repo.On("Create", ctx, mock.Anything).Return(boom).Once()

		_, err := NewPlannedMealService(repo).Create(ctx, plannedUserID, plannedRequest())

		require.ErrorIs(t, err, boom)
	})
}

func TestPlannedMealService_ValidatesRequests(t *testing.T) {
	servings := func(n int) *int { return &n }
	text := func(s string) *string { return &s }
	tests := []struct {
		name   string
		mutate func(*domain.PlannedMealRequest)
	}{
		{"invalid recipe id", func(r *domain.PlannedMealRequest) { r.RecipeID = "bad" }},
		{"nil recipe id", func(r *domain.PlannedMealRequest) { r.RecipeID = "00000000-0000-0000-0000-000000000000" }},
		{"impossible date", func(r *domain.PlannedMealRequest) { r.CookDate = "2025-02-29" }},
		{"year zero", func(r *domain.PlannedMealRequest) { r.CookDate = "0000-01-01" }},
		{"unpadded date", func(r *domain.PlannedMealRequest) { r.CookDate = "2026-1-01" }},
		{"timestamp instead of date", func(r *domain.PlannedMealRequest) { r.CookDate = "2026-01-01T00:00:00Z" }},
		{"unknown meal type", func(r *domain.PlannedMealRequest) { r.MealType = text("brunch") }},
		{"servings too low", func(r *domain.PlannedMealRequest) { r.Servings = servings(0) }},
		{"servings too high", func(r *domain.PlannedMealRequest) { r.Servings = servings(101) }},
		{"notes too long", func(r *domain.PlannedMealRequest) { r.Notes = text(strings.Repeat("é", 1001)) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := new(mockPlannedMealRepository)
			req := plannedRequest()
			tc.mutate(req)

			_, err := NewPlannedMealService(repo).Create(context.Background(), plannedUserID, req)

			require.True(t, internalErr.IsInvalid(err), "got %v", err)
			repo.AssertNotCalled(t, "LockOwnedRecipe", mock.Anything, mock.Anything, mock.Anything)
		})
	}

	t.Run("accepts the upper bounds", func(t *testing.T) {
		repo := new(mockPlannedMealRepository)
		req := plannedRequest()
		req.Servings, req.Notes, req.MealType = servings(100), text(strings.Repeat("é", 1000)), text("snack")
		repo.On("LockOwnedRecipe", mock.Anything, plannedUserID, plannedRecipeID).Return(publishedRecipe(), nil)
		repo.On("Create", mock.Anything, mock.Anything).Return(nil)
		repo.On("GetByID", mock.Anything, plannedUserID, mock.Anything).Return(&domain.PlannedMeal{}, nil)

		_, err := NewPlannedMealService(repo).Create(context.Background(), plannedUserID, req)

		require.NoError(t, err)
	})
}

func TestPlannedMealService_Update(t *testing.T) {
	ctx := context.Background()
	otherRecipe := "6ba7b812-9dad-11d1-80b4-00c04fd430c8"
	current := &domain.PlannedMeal{ID: plannedMealID, RecipeID: plannedRecipeID, CookDate: "2026-10-03"}

	t.Run("treats a malformed id as not found", func(t *testing.T) {
		_, err := NewPlannedMealService(new(mockPlannedMealRepository)).Update(ctx, plannedUserID, "bad", plannedRequest())
		require.True(t, internalErr.IsNotFound(err))
	})

	t.Run("keeps the current recipe without requiring it to be published", func(t *testing.T) {
		repo := new(mockPlannedMealRepository)
		repo.On("GetByID", ctx, plannedUserID, plannedMealID).Return(current, nil).Twice()
		repo.On("Update", ctx, mock.MatchedBy(func(m *domain.PlannedMeal) bool { return m.ID == plannedMealID && m.UserID == plannedUserID })).Return(nil).Once()

		_, err := NewPlannedMealService(repo).Update(ctx, plannedUserID, plannedMealID, plannedRequest())

		require.NoError(t, err)
		repo.AssertNotCalled(t, "LockOwnedRecipe", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("recognises the current recipe when the client sends it in uppercase", func(t *testing.T) {
		repo := new(mockPlannedMealRepository)
		req := plannedRequest()
		req.RecipeID = strings.ToUpper(plannedRecipeID)
		repo.On("GetByID", ctx, plannedUserID, plannedMealID).Return(current, nil).Twice()
		repo.On("Update", ctx, mock.MatchedBy(func(m *domain.PlannedMeal) bool { return m.RecipeID == plannedRecipeID })).Return(nil).Once()

		_, err := NewPlannedMealService(repo).Update(ctx, plannedUserID, plannedMealID, req)

		require.NoError(t, err)
		repo.AssertNotCalled(t, "LockOwnedRecipe", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("requires a different recipe to be owned and published", func(t *testing.T) {
		repo := new(mockPlannedMealRepository)
		req := plannedRequest()
		req.RecipeID = otherRecipe
		repo.On("GetByID", ctx, plannedUserID, plannedMealID).Return(current, nil).Once()
		repo.On("LockOwnedRecipe", ctx, plannedUserID, otherRecipe).Return(&domain.PlannedMealRecipe{ID: otherRecipe, Status: "draft"}, nil).Once()

		_, err := NewPlannedMealService(repo).Update(ctx, plannedUserID, plannedMealID, req)

		require.True(t, internalErr.IsInvalid(err))
		repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	})

	t.Run("returns not found for another user's meal", func(t *testing.T) {
		repo := new(mockPlannedMealRepository)
		repo.On("GetByID", ctx, plannedUserID, plannedMealID).Return(nil, internalErr.ErrNotFound).Once()

		_, err := NewPlannedMealService(repo).Update(ctx, plannedUserID, plannedMealID, plannedRequest())

		require.True(t, internalErr.IsNotFound(err))
		repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	})
}

func TestPlannedMealService_GetAndDelete(t *testing.T) {
	ctx := context.Background()

	t.Run("get treats a malformed id as not found without querying", func(t *testing.T) {
		repo := new(mockPlannedMealRepository)
		_, err := NewPlannedMealService(repo).GetByID(ctx, plannedUserID, "bad")
		require.True(t, internalErr.IsNotFound(err))
		repo.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("get passes the user id so other users' meals are not found", func(t *testing.T) {
		repo := new(mockPlannedMealRepository)
		repo.On("GetByID", ctx, plannedUserID, plannedMealID).Return(nil, internalErr.ErrNotFound).Once()
		_, err := NewPlannedMealService(repo).GetByID(ctx, plannedUserID, plannedMealID)
		require.True(t, internalErr.IsNotFound(err))
	})

	t.Run("delete treats a malformed id as not found without querying", func(t *testing.T) {
		repo := new(mockPlannedMealRepository)
		err := NewPlannedMealService(repo).Delete(ctx, plannedUserID, "bad")
		require.True(t, internalErr.IsNotFound(err))
		repo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("delete is scoped to the user", func(t *testing.T) {
		repo := new(mockPlannedMealRepository)
		repo.On("Delete", ctx, plannedUserID, plannedMealID).Return(nil).Once()
		require.NoError(t, NewPlannedMealService(repo).Delete(ctx, plannedUserID, plannedMealID))
		repo.AssertExpectations(t)
	})
}

func TestPlannedMealService_List(t *testing.T) {
	ctx := context.Background()

	t.Run("pages the range and reports the total", func(t *testing.T) {
		repo := new(mockPlannedMealRepository)
		meals := []domain.PlannedMeal{{ID: plannedMealID}}
		repo.On("List", ctx, plannedUserID, "2026-10-01", "2026-10-31", 50, 100).Return(meals, 120, nil).Once()

		page, err := NewPlannedMealService(repo).List(ctx, plannedUserID, "2026-10-01", "2026-10-31", 3, 50)

		require.NoError(t, err)
		require.Equal(t, &domain.PlannedMealPage{Items: meals, Total: 120, Page: 3, PageSize: 50}, page)
	})

	t.Run("accepts a range of exactly 93 days", func(t *testing.T) {
		repo := new(mockPlannedMealRepository)
		repo.On("List", ctx, plannedUserID, "2026-01-01", "2026-04-03", 100, 0).Return([]domain.PlannedMeal{}, 0, nil).Once()
		_, err := NewPlannedMealService(repo).List(ctx, plannedUserID, "2026-01-01", "2026-04-03", 1, 100)
		require.NoError(t, err)
	})

	invalid := []struct {
		name           string
		from, to       string
		page, pageSize int
	}{
		{"range of 94 days", "2026-01-01", "2026-04-04", 1, 100},
		{"end before start", "2026-04-01", "2026-03-31", 1, 100},
		{"missing from", "", "2026-04-01", 1, 100},
		{"bad to", "2026-04-01", "2026-13-01", 1, 100},
		{"page zero", "2026-10-03", "2026-10-03", 0, 100},
		{"page size zero", "2026-10-03", "2026-10-03", 1, 0},
		{"page size over limit", "2026-10-03", "2026-10-03", 1, 101},
		{"offset overflow", "2026-10-03", "2026-10-03", 2147483647, 100},
	}
	for _, tc := range invalid {
		t.Run("rejects "+tc.name, func(t *testing.T) {
			repo := new(mockPlannedMealRepository)
			_, err := NewPlannedMealService(repo).List(ctx, plannedUserID, tc.from, tc.to, tc.page, tc.pageSize)
			require.True(t, internalErr.IsInvalid(err), "got %v", err)
			repo.AssertNotCalled(t, "List", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}
}
