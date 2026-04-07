package handler

import (
	"context"
	"errors"
	"fmt"
	"github.com/H3nSte1n/recipe/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"net/http"
	"testing"
)

type mockRecipeService struct {
	mock.Mock
}

func (m *mockRecipeService) Create(ctx context.Context, userID string, req *domain.CreateRecipeRequest) (*domain.Recipe, error) {
	args := m.Called(ctx, userID, req)
	v := args.Get(0).(*domain.Recipe)
	return v, args.Error(1)
}

func (m *mockRecipeService) Update(ctx context.Context, userID string, recipeID string, req *domain.CreateRecipeRequest) (*domain.Recipe, error) {
	args := m.Called(ctx, userID, recipeID, req)
	v, _ := args.Get(0).(*domain.Recipe)
	return v, args.Error(1)
}

func (m *mockRecipeService) Delete(ctx context.Context, userID string, recipeID string) error {
	args := m.Called(ctx, userID, recipeID)
	return args.Error(0)
}

func (m *mockRecipeService) GetByID(ctx context.Context, userID string, recipeID string, nutritionLevel domain.NutritionDetailLevel) (*domain.Recipe, error) {
	args := m.Called(ctx, userID, recipeID, nutritionLevel)
	v := args.Get(0).(*domain.Recipe)
	return v, args.Error(1)
}

func (m *mockRecipeService) ListUserRecipes(ctx context.Context, userID string) ([]domain.Recipe, error) {
	args := m.Called(ctx, userID)
	v := args.Get(0).([]domain.Recipe)
	return v, args.Error(1)
}

func (m *mockRecipeService) ListPublicRecipes(ctx context.Context, page, pageSize int) ([]domain.Recipe, int64, error) {
	args := m.Called(ctx, page, pageSize)
	v := args.Get(0).([]domain.Recipe)
	return v, args.Get(1).(int64), args.Error(2)
}

func (m *mockRecipeService) ImportFromURL(ctx context.Context, userID string, req *domain.ImportURLRequest) (*domain.Recipe, error) {
	args := m.Called(ctx, userID, req)
	v := args.Get(0).(*domain.Recipe)
	return v, args.Error(1)
}

func (m *mockRecipeService) ImportFromPDF(ctx context.Context, userID string, req *domain.ImportPDFRequest, file []byte) (*domain.Recipe, error) {
	args := m.Called(ctx, userID, req, file)
	v := args.Get(0).(*domain.Recipe)
	return v, args.Error(1)
}

func (m *mockRecipeService) ParsePlainTextInstructions(ctx context.Context, userID string, req *domain.ParsePlainTextInstructionsRequest) (*[]domain.RecipeInstruction, error) {
	args := m.Called(ctx, userID, req)
	v := args.Get(0).(*[]domain.RecipeInstruction)
	return v, args.Error(1)
}

func TestRecipeHandler_Update(t *testing.T) {
	userID := "550e8400-e29b-41d4-a716-446655440000"
	createRecipeRequest := domain.CreateRecipeRequest{Description: "Foo", Title: "Foobar", IsPrivate: false, SourceType: "MANUAL", Servings: 1}
	recipe := domain.Recipe{ID: "1_foo", Description: createRecipeRequest.Description, Title: createRecipeRequest.Title, IsPrivate: createRecipeRequest.IsPrivate}
	jsonCreateRecipeRequest := mustJson(t, createRecipeRequest)
	jsonRecipe := mustJson(t, recipe)
	tests := []struct {
		name                 string
		body                 []byte
		setUserID            bool
		expectedStatusCode   int
		expectedBodyContains string
		mockMethod           func(m *mockRecipeService)
	}{
		{
			name:                 "returns 200 with updated Recipe when request is successfully",
			body:                 jsonCreateRecipeRequest,
			setUserID:            true,
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: string(jsonRecipe),
			mockMethod: func(m *mockRecipeService) {
				m.On("Update", mock.Anything, userID, recipe.ID, mock.MatchedBy(func(req *domain.CreateRecipeRequest) bool {
					return req.Title == createRecipeRequest.Title && req.Description == createRecipeRequest.Description && req.IsPrivate == createRecipeRequest.IsPrivate && req.SourceType == createRecipeRequest.SourceType && req.Servings == createRecipeRequest.Servings
				})).Return(&recipe, nil).Once()
			},
		},
		{
			name:                 "returns 401 unauthorized when user is not authenticated",
			setUserID:            false,
			body:                 jsonCreateRecipeRequest,
			expectedStatusCode:   http.StatusUnauthorized,
			expectedBodyContains: "unauthorized",
			mockMethod:           func(m *mockRecipeService) {},
		},
		{
			name:                 "returns 400 bad request error when json is invalid",
			body:                 []byte(`{invalid`),
			expectedStatusCode:   http.StatusBadRequest,
			expectedBodyContains: "error",
			mockMethod:           func(m *mockRecipeService) {},
		},
		{
			name:                 "returns 500 internal server error when service returns error",
			body:                 jsonCreateRecipeRequest,
			setUserID:            true,
			expectedBodyContains: "failed to update recipe",
			expectedStatusCode:   http.StatusInternalServerError,
			mockMethod: func(m *mockRecipeService) {
				m.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("service error")).Once()
			},
		},
	}

	gin.SetMode(gin.TestMode)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRecipeService)
			tt.mockMethod(m)

			handler := NewRecipeHandler(m, zap.NewNop())
			router := gin.New()
			router.PUT("/api/v1/recipes/:id", func(ctx *gin.Context) {
				if tt.setUserID {
					ctx.Set("user_id", userID)
				}

				handler.Update(ctx)
			})

			w := performRequest(router, http.MethodPut, fmt.Sprintf("/api/v1/recipes/%v", recipe.ID), tt.body)

			require.Equal(t, tt.expectedStatusCode, w.Code)
			if tt.expectedBodyContains != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBodyContains)
			}
			m.AssertExpectations(t)
		})
	}
}
