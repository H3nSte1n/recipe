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
	v, _ := args.Get(0).(*domain.Recipe)
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
	v, _ := args.Get(0).(*domain.Recipe)
	return v, args.Error(1)
}

func (m *mockRecipeService) ListUserRecipes(ctx context.Context, userID string) ([]domain.Recipe, error) {
	args := m.Called(ctx, userID)
	v, _ := args.Get(0).([]domain.Recipe)
	return v, args.Error(1)
}

func (m *mockRecipeService) ListPublicRecipes(ctx context.Context, page, pageSize int) ([]domain.Recipe, int64, error) {
	args := m.Called(ctx, page, pageSize)
	v, _ := args.Get(0).([]domain.Recipe)
	return v, args.Get(1).(int64), args.Error(2)
}

func (m *mockRecipeService) ImportFromURL(ctx context.Context, userID string, req *domain.ImportURLRequest) (*domain.Recipe, error) {
	args := m.Called(ctx, userID, req)
	v, _ := args.Get(0).(*domain.Recipe)
	return v, args.Error(1)
}

func (m *mockRecipeService) ImportFromPDF(ctx context.Context, userID string, req *domain.ImportPDFRequest, file []byte) (*domain.Recipe, error) {
	args := m.Called(ctx, userID, req, file)
	v, _ := args.Get(0).(*domain.Recipe)
	return v, args.Error(1)
}

func (m *mockRecipeService) ParsePlainTextInstructions(ctx context.Context, userID string, req *domain.ParsePlainTextInstructionsRequest) (*[]domain.RecipeInstruction, error) {
	args := m.Called(ctx, userID, req)
	v, _ := args.Get(0).(*[]domain.RecipeInstruction)
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

func TestRecipeHandler_Delete(t *testing.T) {
	userID := "550e8400-e29b-41d4-a716-446655440000"
	recipeID := "1_foo"
	tests := []struct {
		name                 string
		setUserID            bool
		expectedStatusCode   int
		expectedBodyContains string
		mockMethod           func(m *mockRecipeService)
	}{
		{
			name:                 "returns 200 status ok when request is successfully",
			setUserID:            true,
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "recipe deleted",
			mockMethod: func(m *mockRecipeService) {
				m.On("Delete", mock.Anything, userID, recipeID).Return(nil).Once()
			},
		},
		{
			name:                 "returns 401 unauthorized when user is not authenticated",
			setUserID:            false,
			expectedStatusCode:   http.StatusUnauthorized,
			expectedBodyContains: "unauthorized",
			mockMethod:           func(m *mockRecipeService) {},
		},
		{
			name:                 "returns 500 internal server error when service returns error",
			setUserID:            true,
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: "failed to delete recipe",
			mockMethod: func(m *mockRecipeService) {
				m.On("Delete", mock.Anything, userID, recipeID).Return(errors.New("service error"))
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
			router.DELETE("/api/v1/recipes/:id", func(ctx *gin.Context) {
				if tt.setUserID {
					ctx.Set("user_id", userID)
				}

				handler.Delete(ctx)
			})

			w := performRequest(router, http.MethodDelete, fmt.Sprintf("/api/v1/recipes/%v", recipeID), nil)

			require.Equal(t, tt.expectedStatusCode, w.Code)
			if tt.expectedBodyContains != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBodyContains)
			}
			m.AssertExpectations(t)
		})
	}
}

func TestRecipeHandler_Get(t *testing.T) {
	userID := "550e8400-e29b-41d4-a716-446655440000"
	recipe := domain.Recipe{ID: "1_foo", Title: "foobar"}
	jsonRecipe := mustJson(t, recipe)
	tests := []struct {
		name                 string
		setUserID            bool
		url                  string
		expectedStatusCode   int
		expectedBodyContains string
		mockMethod           func(m *mockRecipeService)
	}{
		{
			name:                 "returns 200 with recipe when request is successfully",
			setUserID:            true,
			url:                  fmt.Sprintf("/api/v1/recipes/%v", recipe.ID),
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: string(jsonRecipe),
			mockMethod: func(m *mockRecipeService) {
				m.On("GetByID", mock.Anything, userID, recipe.ID, domain.NutritionDetailBase).Return(&recipe, nil).Once()
			},
		},
		{
			name:                 "returns 200 with recipe and specific nutrition level when request is successfully with nutrition_level as query param",
			setUserID:            true,
			url:                  fmt.Sprintf("/api/v1/recipes/%v?nutrition_level=macro", recipe.ID),
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: string(jsonRecipe),
			mockMethod: func(m *mockRecipeService) {
				m.On("GetByID", mock.Anything, userID, recipe.ID, domain.NutritionDetailMacro).Return(&recipe, nil).Once()
			},
		},
		{
			name:                 "returns 401 unauthorized when user is not authenticated",
			setUserID:            false,
			url:                  fmt.Sprintf("/api/v1/recipes/%v", recipe.ID),
			expectedStatusCode:   http.StatusUnauthorized,
			expectedBodyContains: "unauthorized",
			mockMethod:           func(m *mockRecipeService) {},
		},
		{
			name:                 "returns 500 internal server error when service returns error",
			setUserID:            true,
			url:                  fmt.Sprintf("/api/v1/recipes/%v", recipe.ID),
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: "failed to get recipe",
			mockMethod: func(m *mockRecipeService) {
				m.On("GetByID", mock.Anything, userID, recipe.ID, domain.NutritionDetailBase).Return(nil, errors.New("service error")).Once()
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
			router.GET("/api/v1/recipes/:id", func(ctx *gin.Context) {
				if tt.setUserID {
					ctx.Set("user_id", userID)
				}

				handler.Get(ctx)
			})

			w := performRequest(router, http.MethodGet, tt.url, nil)

			require.Equal(t, tt.expectedStatusCode, w.Code)
			if tt.expectedBodyContains != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBodyContains)
			}
			m.AssertExpectations(t)
		})
	}
}

func TestRecipeHandler_ListMine(t *testing.T) {
	userID := "550e8400-e29b-41d4-a716-446655440000"
	recipes := []domain.Recipe{
		{ID: "1_foo", Title: "foobar", UserID: userID},
		{ID: "2_foo", Title: "bar", UserID: userID},
	}
	recipesJson := mustJson(t, recipes)
	tests := []struct {
		name                 string
		setUserID            bool
		expectedStatusCode   int
		expectedBodyContains string
		mockMethod           func(m *mockRecipeService)
	}{
		{
			name:                 "return 200 with all the recipes belonging to a user when request is successfully",
			setUserID:            true,
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: string(recipesJson),
			mockMethod: func(m *mockRecipeService) {
				m.On("ListUserRecipes", mock.Anything, userID).Return(recipes, nil).Once()
			},
		},
		{
			name:                 "returns 401 unauthorized when user is not authenticated",
			setUserID:            false,
			expectedStatusCode:   http.StatusUnauthorized,
			expectedBodyContains: "unauthorized",
			mockMethod:           func(m *mockRecipeService) {},
		},
		{
			name:                 "returns 500 internal server error when service returns error",
			setUserID:            true,
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: "failed to list recipes",
			mockMethod: func(m *mockRecipeService) {
				m.On("ListUserRecipes", mock.Anything, userID).Return(nil, errors.New("service error")).Once()
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
			router.GET("/api/v1/recipes", func(ctx *gin.Context) {
				if tt.setUserID {
					ctx.Set("user_id", userID)
				}

				handler.ListMine(ctx)
			})

			w := performRequest(router, http.MethodGet, "/api/v1/recipes", nil)

			require.Equal(t, tt.expectedStatusCode, w.Code)
			if tt.expectedBodyContains != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBodyContains)
			}
			m.AssertExpectations(t)
		})
	}
}

func TestRecipeHandler_ListPublic(t *testing.T) {
	page := 1
	var pageSize int64 = 3
	userID := "550e8400-e29b-41d4-a716-446655440000"
	userID2 := "650e8400-e29b-41d4-a716-446655440000"
	recipes := []domain.Recipe{
		{ID: "1_foo", Title: "foobar", UserID: userID},
		{ID: "2_foo", Title: "bar", UserID: userID},
		{ID: "3_foo", Title: "bar", UserID: userID2},
	}
	jsonRecipes := mustJson(t, recipes)
	tests := []struct {
		name                 string
		setUserID            bool
		expectedStatusCode   int
		expectedBodyContains string
		mockMethod           func(m *mockRecipeService)
	}{
		{
			name:                 "returns 200 with all public recipes when request is successfully",
			setUserID:            true,
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: string(jsonRecipes),
			mockMethod: func(m *mockRecipeService) {
				m.On("ListPublicRecipes", mock.Anything, page, 3).Return(recipes, pageSize, nil).Once()
			},
		},
		{
			name:                 "returns 500 internal server error when service returns error",
			setUserID:            true,
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: "failed to list recipes",
			mockMethod: func(m *mockRecipeService) {
				m.On("ListPublicRecipes", mock.Anything, page, 3).Return(nil, pageSize, errors.New("service error")).Once()
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
			router.GET("/api/v1/recipes/public", func(ctx *gin.Context) {
				if tt.setUserID {
					ctx.Set("user_id", userID)
				}

				handler.ListPublic(ctx)
			})

			w := performRequest(router, http.MethodGet, "/api/v1/recipes/public?page=1&page_size=3", nil)

			require.Equal(t, tt.expectedStatusCode, w.Code)
			if tt.expectedBodyContains != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBodyContains)
			}
			m.AssertExpectations(t)
		})
	}
}

func TestRecipeHandler_ImportFromPDF(t *testing.T) {
	userID := "550e8400-e29b-41d4-a716-446655440000"
	pdfContent := []byte("%PDF-1.4 fake pdf content")
	recipe := domain.Recipe{ID: "1_foo", Description: "foo bar foo", Title: "foobar", IsPrivate: false}
	jsonRecipe := mustJson(t, recipe)

	tests := []struct {
		name                 string
		setUserID            bool
		fileContent          []byte
		expectedStatusCode   int
		expectedBodyContains string
		mockMethod           func(m *mockRecipeService)
	}{
		{
			name:                 "returns 200 with imported recipe when request is successfully",
			setUserID:            true,
			fileContent:          pdfContent,
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: string(jsonRecipe),
			mockMethod: func(m *mockRecipeService) {
				m.On("ImportFromPDF", mock.Anything, userID, mock.MatchedBy(func(req *domain.ImportPDFRequest) bool {
					return req.IsPrivate == false
				}), pdfContent).Return(&recipe, nil).Once()
			},
		},
		{
			name:                 "returns 400 bad request when no file is provided",
			setUserID:            true,
			fileContent:          nil,
			expectedStatusCode:   http.StatusBadRequest,
			expectedBodyContains: "no file provided",
			mockMethod:           func(m *mockRecipeService) {},
		},
		{
			name:                 "returns 500 internal server error when service returns error",
			setUserID:            true,
			fileContent:          pdfContent,
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: "failed to import recipe",
			mockMethod: func(m *mockRecipeService) {
				m.On("ImportFromPDF", mock.Anything, userID, mock.Anything, pdfContent).Return(nil, errors.New("service error")).Once()
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
			router.POST("/api/v1/recipes/import/pdf", func(ctx *gin.Context) {
				if tt.setUserID {
					ctx.Set("user_id", userID)
				}
				handler.ImportFromPDF(ctx)
			})

			w := performMultipartRequest(t, router, http.MethodPost, "/api/v1/recipes/import/pdf", "file", tt.fileContent, nil)

			require.Equal(t, tt.expectedStatusCode, w.Code)
			if tt.expectedBodyContains != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBodyContains)
			}
			m.AssertExpectations(t)
		})
	}
}

func TestRecipeHandler_ImportFromURL(t *testing.T) {
	userID := "550e8400-e29b-41d4-a716-446655440000"
	importUrlRequest := domain.ImportURLRequest{URL: "https://steinhauer.dev/", IsPrivate: false}
	recipe := domain.Recipe{ID: "1_foo", Description: "foo bar foo", Title: "foobar", IsPrivate: true}
	jsonImportUrlRequest := mustJson(t, importUrlRequest)
	jsonRecipe := mustJson(t, recipe)
	tests := []struct {
		name                 string
		setUserID            bool
		body                 []byte
		expectedStatusCode   int
		expectedBodyContains string
		mockMethod           func(m *mockRecipeService)
	}{
		{
			name:                 "returns 200 with imported recipe when request is successfully",
			setUserID:            true,
			body:                 jsonImportUrlRequest,
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: string(jsonRecipe),
			mockMethod: func(m *mockRecipeService) {
				m.On("ImportFromURL", mock.Anything, userID, mock.MatchedBy(func(req *domain.ImportURLRequest) bool {
					return req.URL == importUrlRequest.URL && req.IsPrivate == importUrlRequest.IsPrivate
				})).Return(&recipe, nil).Once()
			},
		},
		{
			name:                 "returns 401 unauthorized when user is not authenticated",
			setUserID:            false,
			expectedStatusCode:   http.StatusUnauthorized,
			expectedBodyContains: "unauthorized",
			mockMethod:           func(m *mockRecipeService) {},
		},
		{
			name:                 "returns 400 bad request when json is invalid",
			setUserID:            true,
			body:                 []byte(`{invalid`),
			expectedStatusCode:   http.StatusBadRequest,
			expectedBodyContains: "error",
			mockMethod:           func(m *mockRecipeService) {},
		},
		{
			name:                 "returns 500 internal server error when service returns error",
			setUserID:            true,
			body:                 jsonImportUrlRequest,
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: "failed to import recipe",
			mockMethod: func(m *mockRecipeService) {
				m.On("ImportFromURL", mock.Anything, userID, mock.Anything).Return(nil, errors.New("service error")).Once()
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
			router.POST("/api/v1/recipes/import/url", func(ctx *gin.Context) {
				if tt.setUserID {
					ctx.Set("user_id", userID)
				}

				handler.ImportFromURL(ctx)
			})

			w := performRequest(router, http.MethodPost, "/api/v1/recipes/import/url", tt.body)

			require.Equal(t, tt.expectedStatusCode, w.Code)
			if tt.expectedBodyContains != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBodyContains)
			}
			m.AssertExpectations(t)
		})
	}
}

func TestRecipeHandler_ParsePlainTextInstructions(t *testing.T) {
	userID := "550e8400-e29b-41d4-a716-446655440000"
	plainTextInstructionsRequest := domain.ParsePlainTextInstructionsRequest{PlainText: "foobar foo bar"}
	jsonPlainTextInstructionsRequest := mustJson(t, plainTextInstructionsRequest)
	instructions := []domain.RecipeInstruction{{ID: "1_foo", RecipeID: "1", StepNumber: 1, Instruction: "foobar"}}
	jsonInstructions := mustJson(t, instructions)
	tests := []struct {
		name                 string
		setUserID            bool
		body                 []byte
		expectedStatusCode   int
		expectedBodyContains string
		mockMethod           func(m *mockRecipeService)
	}{
		{
			name:                 "returns 200 with instructions when request is successfully",
			setUserID:            true,
			body:                 jsonPlainTextInstructionsRequest,
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: string(jsonInstructions),
			mockMethod: func(m *mockRecipeService) {
				m.On("ParsePlainTextInstructions", mock.Anything, userID, mock.MatchedBy(func(req *domain.ParsePlainTextInstructionsRequest) bool {
					return req.PlainText == plainTextInstructionsRequest.PlainText
				})).Return(&instructions, nil).Once()
			},
		},
		{
			name:                 "returns 401 unauthorized when user is not authenticated",
			setUserID:            false,
			expectedStatusCode:   http.StatusUnauthorized,
			expectedBodyContains: "unauthorized",
			mockMethod:           func(m *mockRecipeService) {},
		},
		{
			name:                 "returns 400 bad request when json is invalid",
			setUserID:            true,
			body:                 []byte(`{invalid`),
			expectedStatusCode:   http.StatusBadRequest,
			expectedBodyContains: "error",
			mockMethod:           func(m *mockRecipeService) {},
		},
		{
			name:                 "returns 500 internal server error when service returns error",
			setUserID:            true,
			body:                 jsonPlainTextInstructionsRequest,
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: "failed to parse instructions",
			mockMethod: func(m *mockRecipeService) {
				m.On("ParsePlainTextInstructions", mock.Anything, userID, mock.Anything).Return(nil, errors.New("service error")).Once()
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
			router.POST("/api/v1/recipes/parser/instructions", func(ctx *gin.Context) {
				if tt.setUserID {
					ctx.Set("user_id", userID)
				}

				handler.ParsePlainTextInstructions(ctx)
			})

			w := performRequest(router, http.MethodPost, "/api/v1/recipes/parser/instructions", tt.body)

			require.Equal(t, tt.expectedStatusCode, w.Code)
			if tt.expectedBodyContains != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBodyContains)
			}
			m.AssertExpectations(t)
		})
	}
}
