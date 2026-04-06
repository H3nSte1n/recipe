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

type mockShoppingListService struct {
	mock.Mock
}

func (m *mockShoppingListService) Create(ctx context.Context, userID string, req *domain.CreateShoppingListRequest) (*domain.ShoppingList, error) {
	args := m.Called(ctx, userID, req)
	v, _ := args.Get(0).(*domain.ShoppingList)
	return v, args.Error(1)
}

func (m *mockShoppingListService) Update(ctx context.Context, userID string, listID string, req *domain.UpdateShoppingListRequest) (*domain.ShoppingList, error) {
	args := m.Called(ctx, userID, listID, req)
	v, _ := args.Get(0).(*domain.ShoppingList)
	return v, args.Error(1)
}

func (m *mockShoppingListService) Delete(ctx context.Context, userID string, listID string) error {
	args := m.Called(ctx, userID, listID)
	return args.Error(0)
}

func (m *mockShoppingListService) GetByID(ctx context.Context, userID string, listID string) (*domain.ShoppingList, error) {
	args := m.Called(ctx, userID, listID)
	v, _ := args.Get(0).(*domain.ShoppingList)
	return v, args.Error(1)
}

func (m *mockShoppingListService) GetSorted(ctx context.Context, userID string, listID string, sortBy string, sortDirection string) (*domain.ShoppingList, error) {
	args := m.Called(ctx, userID, listID, sortBy, sortDirection)
	v, _ := args.Get(0).(*domain.ShoppingList)
	return v, args.Error(1)
}

func (m *mockShoppingListService) GetSortedByStoreName(ctx context.Context, userID string, listID string, storeName string, sortDirection string) (*domain.ShoppingList, error) {
	args := m.Called(ctx, userID, listID, storeName, sortDirection)
	v, _ := args.Get(0).(*domain.ShoppingList)
	return v, args.Error(1)
}

func (m *mockShoppingListService) ListByUserID(ctx context.Context, userID string) ([]domain.ShoppingList, error) {
	args := m.Called(ctx, userID)
	v, _ := args.Get(0).([]domain.ShoppingList)
	return v, args.Error(1)
}

func (m *mockShoppingListService) AddItem(ctx context.Context, userID string, listID string, req *domain.ShoppingListItemRequest) error {
	args := m.Called(ctx, userID, listID, req)
	return args.Error(0)
}

func (m *mockShoppingListService) UpdateItem(ctx context.Context, userID string, itemID string, req *domain.UpdateShoppingListItemRequest) error {
	args := m.Called(ctx, userID, itemID, req)
	return args.Error(0)
}

func (m *mockShoppingListService) DeleteItem(ctx context.Context, userID string, itemID string) error {
	args := m.Called(ctx, userID, itemID)
	return args.Error(0)
}

func (m *mockShoppingListService) ToggleItem(ctx context.Context, userID string, itemID string, checked bool) error {
	args := m.Called(ctx, userID, itemID, checked)
	return args.Error(0)
}

func (m *mockShoppingListService) AddRecipeToList(ctx context.Context, userID string, listID string, req *domain.AddRecipeToListRequest) error {
	args := m.Called(ctx, userID, listID, req)
	return args.Error(0)
}

func (m *mockShoppingListService) GetSortedForStore(ctx context.Context, userID string, listID string, chainID string) (*domain.ShoppingList, error) {
	args := m.Called(ctx, userID, listID, chainID)
	v, _ := args.Get(0).(*domain.ShoppingList)
	return v, args.Error(1)
}

func TestShoppingListHandler_Create(t *testing.T) {
	userID := "550e8400-e29b-41d4-a716-446655440000"
	shoppingListRequest := domain.CreateShoppingListRequest{Name: "foobar", Description: "foo description", SortType: domain.SortType("CATEGORY")}
	shoppingList := domain.ShoppingList{ID: "1_foobar", UserID: userID, Name: shoppingListRequest.Name}
	jsonShoppingListRequest := mustJson(t, shoppingListRequest)
	jsonShoppingList := mustJson(t, shoppingList)
	tests := []struct {
		name                 string
		body                 []byte
		expectedContainsBody string
		expectedStatusCode   int
		setUserID            bool
		mockMethod           func(m *mockShoppingListService)
	}{
		{
			name:                 "returns 200 with creation of shopping list when request is successfully",
			body:                 jsonShoppingListRequest,
			expectedContainsBody: string(jsonShoppingList),
			expectedStatusCode:   http.StatusCreated,
			setUserID:            true,
			mockMethod: func(m *mockShoppingListService) {
				m.On("Create", mock.Anything, userID, mock.MatchedBy(func(req *domain.CreateShoppingListRequest) bool {
					return req.Name == shoppingListRequest.Name && req.Description == shoppingListRequest.Description && req.SortType == shoppingListRequest.SortType
				})).Return(&shoppingList, nil).Once()
			},
		},
		{
			name:                 "retuns 400 error when json is invalid",
			body:                 []byte(`{invalid`),
			expectedContainsBody: "error",
			expectedStatusCode:   http.StatusBadRequest,
			setUserID:            true,
			mockMethod:           func(m *mockShoppingListService) {},
		},
		{
			name:                 "returns 401 unauthorized when user is not authenticated",
			body:                 jsonShoppingListRequest,
			expectedContainsBody: "unauthorized",
			expectedStatusCode:   http.StatusUnauthorized,
			setUserID:            false,
			mockMethod:           func(m *mockShoppingListService) {},
		},
		{
			name:                 "returns 500 error when service returns error",
			body:                 jsonShoppingListRequest,
			expectedContainsBody: "failed to create shopping list",
			expectedStatusCode:   http.StatusInternalServerError,
			setUserID:            true,
			mockMethod: func(m *mockShoppingListService) {
				m.On("Create", mock.Anything, userID, mock.Anything).Return(nil, errors.New("service error")).Once()
			},
		},
	}

	gin.SetMode(gin.TestMode)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockShoppingListService)
			tt.mockMethod(m)

			handler := NewShoppingListHandler(m, zap.NewNop())
			router := gin.New()
			router.POST("/api/v1/shopping-lists", func(ctx *gin.Context) {
				if tt.setUserID {
					ctx.Set("user_id", userID)
				}

				handler.Create(ctx)
			})

			w := performRequest(router, http.MethodPost, "/api/v1/shopping-lists", tt.body)

			require.Equal(t, tt.expectedStatusCode, w.Code)
			if tt.expectedContainsBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedContainsBody)
			}
			m.AssertExpectations(t)
		})
	}
}

func TestShoppingListHandler_Get(t *testing.T) {
	userID := "550e8400-e29b-41d4-a716-446655440000"
	shoppingList := domain.ShoppingList{ID: "1_foo", Name: "Foo", UserID: userID}
	jsonShoppingList := mustJson(t, shoppingList)
	tests := []struct {
		name                 string
		body                 []byte
		url                  string
		expectedStatusCode   int
		expectedBodyContains string
		setUserID            bool
		mockMethod           func(m *mockShoppingListService)
	}{
		{
			name:                 "returns status 200 with shopping list sorted by store when query params sortBy and storeName are attached and request is successfully",
			url:                  fmt.Sprintf("/api/v1/shopping-lists/%v?sort_by=store&sort_direction=asc&store_name=foo", shoppingList.ID),
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: string(jsonShoppingList),
			setUserID:            true,
			mockMethod: func(m *mockShoppingListService) {
				m.On("GetSortedByStoreName", mock.Anything, userID, shoppingList.ID, "foo", "asc").Return(&shoppingList, nil).Once()
			},
		},
		{
			name:                 "returns status 200 with shopping list sorted by name when query param sortBy is attached and request is successfully",
			url:                  fmt.Sprintf("/api/v1/shopping-lists/%v?sort_by=name&sort_direction=asc", shoppingList.ID),
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: string(jsonShoppingList),
			setUserID:            true,
			mockMethod: func(m *mockShoppingListService) {
				m.On("GetSorted", mock.Anything, userID, shoppingList.ID, "name", "asc").Return(&shoppingList, nil).Once()
			},
		},
		{
			name:                 "returns status 200 with shopping list sorted by default when no sort_by is attached and request is successfully",
			url:                  fmt.Sprintf("/api/v1/shopping-lists/%v", shoppingList.ID),
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: string(jsonShoppingList),
			setUserID:            true,
			mockMethod: func(m *mockShoppingListService) {
				m.On("GetByID", mock.Anything, userID, shoppingList.ID).Return(&shoppingList, nil).Once()
			},
		},
		{
			name:                 "returns 401 unauthorized when user is not authenticated",
			url:                  fmt.Sprintf("/api/v1/shopping-lists/%v", shoppingList.ID),
			expectedStatusCode:   http.StatusUnauthorized,
			expectedBodyContains: "unauthorized",
			setUserID:            false,
			mockMethod:           func(m *mockShoppingListService) {},
		},
		{
			name:                 "returns 404 not found error when user service returns error",
			url:                  fmt.Sprintf("/api/v1/shopping-lists/%v", shoppingList.ID),
			expectedStatusCode:   http.StatusNotFound,
			expectedBodyContains: "shopping list not found",
			setUserID:            true,
			mockMethod: func(m *mockShoppingListService) {
				m.On("GetByID", mock.Anything, userID, shoppingList.ID).Return(nil, errors.New("service error")).Once()
			},
		},
	}

	gin.SetMode(gin.TestMode)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockShoppingListService)
			tt.mockMethod(m)

			handler := NewShoppingListHandler(m, zap.NewNop())
			router := gin.New()
			router.GET("/api/v1/shopping-lists/:id", func(ctx *gin.Context) {
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

func TestShoppingListHandler_List(t *testing.T) {
	userID := "550e8400-e29b-41d4-a716-446655440000"
	shoppingLists := []domain.ShoppingList{{ID: "1_foo", UserID: userID, Name: "foo"}, {ID: "2_foo", UserID: userID, Name: "bar"}}
	jsonShoppingLists := mustJson(t, shoppingLists)
	tests := []struct {
		name                 string
		setUserID            bool
		expectedStatusCode   int
		expectedBodyContains string
		mockMethod           func(m *mockShoppingListService)
	}{
		{
			name:                 "returns 200 with shopping lists when request is successfully",
			setUserID:            true,
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: string(jsonShoppingLists),
			mockMethod: func(m *mockShoppingListService) {
				m.On("ListByUserID", mock.Anything, userID).Return(shoppingLists, nil).Once()
			},
		},
		{
			name:                 "returns 401 unauthorized when user is not authenticated",
			setUserID:            false,
			expectedStatusCode:   http.StatusUnauthorized,
			expectedBodyContains: "unauthorized",
			mockMethod:           func(m *mockShoppingListService) {},
		},
		{
			name:                 "returns 500 internal server error when service returns error",
			setUserID:            true,
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: "failed to list shopping lists",
			mockMethod: func(m *mockShoppingListService) {
				m.On("ListByUserID", mock.Anything, userID).Return(nil, errors.New("service error")).Once()
			},
		},
	}

	gin.SetMode(gin.TestMode)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockShoppingListService)
			tt.mockMethod(m)

			handler := NewShoppingListHandler(m, zap.NewNop())
			router := gin.New()
			router.GET("/api/v1/shopping-lists", func(ctx *gin.Context) {
				if tt.setUserID {
					ctx.Set("user_id", userID)
				}

				handler.List(ctx)
			})

			w := performRequest(router, http.MethodGet, "/api/v1/shopping-lists", nil)

			require.Equal(t, tt.expectedStatusCode, w.Code)
			if tt.expectedBodyContains != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBodyContains)
			}
			m.AssertExpectations(t)
		})
	}
}
