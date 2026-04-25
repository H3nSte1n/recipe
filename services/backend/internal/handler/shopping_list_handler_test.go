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

func (m *mockShoppingListService) GetSortedByStoreName(ctx context.Context, userID string, listID string, storeName string, country string, sortDirection string) (*domain.ShoppingList, error) {
	args := m.Called(ctx, userID, listID, storeName, country, sortDirection)
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
				m.On("GetSortedByStoreName", mock.Anything, userID, shoppingList.ID, "foo", "", "asc").Return(&shoppingList, nil).Once()
			},
		},
		{
			name:                 "returns status 200 with shopping list sorted by store with country when country query param is attached",
			url:                  fmt.Sprintf("/api/v1/shopping-lists/%v?sort_by=store&sort_direction=asc&store_name=foo&country=DE", shoppingList.ID),
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: string(jsonShoppingList),
			setUserID:            true,
			mockMethod: func(m *mockShoppingListService) {
				m.On("GetSortedByStoreName", mock.Anything, userID, shoppingList.ID, "foo", "DE", "asc").Return(&shoppingList, nil).Once()
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
		{
			name:                 "returns 400 when sort_by=store but store_name is missing",
			url:                  fmt.Sprintf("/api/v1/shopping-lists/%v?sort_by=store", shoppingList.ID),
			expectedStatusCode:   http.StatusBadRequest,
			expectedBodyContains: "store_name is required",
			setUserID:            true,
			mockMethod:           func(m *mockShoppingListService) {},
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

func TestShoppingListHandler_Update(t *testing.T) {
	userID := "550e8400-e29b-41d4-a716-446655440000"
	listID := "list-uuid-1234"
	updateReq := domain.UpdateShoppingListRequest{Name: "updated name", Description: "updated desc", SortType: domain.SortTypeCategory}
	updatedList := domain.ShoppingList{ID: listID, UserID: userID, Name: updateReq.Name}
	jsonUpdateReq := mustJson(t, updateReq)
	jsonUpdatedList := mustJson(t, updatedList)

	tests := []struct {
		name                 string
		body                 []byte
		setUserID            bool
		expectedStatusCode   int
		expectedBodyContains string
		mockMethod           func(m *mockShoppingListService)
	}{
		{
			name:                 "returns 200 with updated shopping list when request is successfully",
			body:                 jsonUpdateReq,
			setUserID:            true,
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: string(jsonUpdatedList),
			mockMethod: func(m *mockShoppingListService) {
				m.On("Update", mock.Anything, userID, listID, mock.MatchedBy(func(req *domain.UpdateShoppingListRequest) bool {
					return req.Name == updateReq.Name && req.Description == updateReq.Description && req.SortType == updateReq.SortType
				})).Return(&updatedList, nil).Once()
			},
		},
		{
			name:                 "returns 401 unauthorized when user is not authenticated",
			body:                 jsonUpdateReq,
			setUserID:            false,
			expectedStatusCode:   http.StatusUnauthorized,
			expectedBodyContains: "unauthorized",
			mockMethod:           func(m *mockShoppingListService) {},
		},
		{
			name:                 "returns 400 when json body is invalid",
			body:                 []byte(`{invalid`),
			setUserID:            true,
			expectedStatusCode:   http.StatusBadRequest,
			expectedBodyContains: "error",
			mockMethod:           func(m *mockShoppingListService) {},
		},
		{
			name:                 "returns 500 when service returns error",
			body:                 jsonUpdateReq,
			setUserID:            true,
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: "failed to update shopping list",
			mockMethod: func(m *mockShoppingListService) {
				m.On("Update", mock.Anything, userID, listID, mock.Anything).Return(nil, errors.New("service error")).Once()
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
			router.PUT("/api/v1/shopping-lists/:id", func(ctx *gin.Context) {
				if tt.setUserID {
					ctx.Set("user_id", userID)
				}
				handler.Update(ctx)
			})

			w := performRequest(router, http.MethodPut, fmt.Sprintf("/api/v1/shopping-lists/%v", listID), tt.body)

			require.Equal(t, tt.expectedStatusCode, w.Code)
			if tt.expectedBodyContains != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBodyContains)
			}
			m.AssertExpectations(t)
		})
	}
}

func TestShoppingListHandler_Delete(t *testing.T) {
	userID := "550e8400-e29b-41d4-a716-446655440000"
	listID := "list-uuid-1234"

	tests := []struct {
		name               string
		setUserID          bool
		expectedStatusCode int
		mockMethod         func(m *mockShoppingListService)
	}{
		{
			name:               "returns 204 when shopping list is deleted successfully",
			setUserID:          true,
			expectedStatusCode: http.StatusNoContent,
			mockMethod: func(m *mockShoppingListService) {
				m.On("Delete", mock.Anything, userID, listID).Return(nil).Once()
			},
		},
		{
			name:               "returns 401 unauthorized when user is not authenticated",
			setUserID:          false,
			expectedStatusCode: http.StatusUnauthorized,
			mockMethod:         func(m *mockShoppingListService) {},
		},
		{
			name:               "returns 500 when service returns error",
			setUserID:          true,
			expectedStatusCode: http.StatusInternalServerError,
			mockMethod: func(m *mockShoppingListService) {
				m.On("Delete", mock.Anything, userID, listID).Return(errors.New("service error")).Once()
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
			router.DELETE("/api/v1/shopping-lists/:id", func(ctx *gin.Context) {
				if tt.setUserID {
					ctx.Set("user_id", userID)
				}
				handler.Delete(ctx)
			})

			w := performRequest(router, http.MethodDelete, fmt.Sprintf("/api/v1/shopping-lists/%v", listID), nil)

			require.Equal(t, tt.expectedStatusCode, w.Code)
			m.AssertExpectations(t)
		})
	}
}

func TestShoppingListHandler_AddItem(t *testing.T) {
	userID := "550e8400-e29b-41d4-a716-446655440000"
	listID := "list-uuid-1234"
	itemReq := domain.ShoppingListItemRequest{Name: "Milk", Amount: 2, Unit: "L", Category: domain.CategoryDairy}
	jsonItemReq := mustJson(t, itemReq)

	tests := []struct {
		name                 string
		body                 []byte
		setUserID            bool
		expectedStatusCode   int
		expectedBodyContains string
		mockMethod           func(m *mockShoppingListService)
	}{
		{
			name:                 "returns 201 when item is added successfully",
			body:                 jsonItemReq,
			setUserID:            true,
			expectedStatusCode:   http.StatusCreated,
			expectedBodyContains: "item added successfully",
			mockMethod: func(m *mockShoppingListService) {
				m.On("AddItem", mock.Anything, userID, listID, mock.MatchedBy(func(req *domain.ShoppingListItemRequest) bool {
					return req.Name == itemReq.Name && req.Category == itemReq.Category
				})).Return(nil).Once()
			},
		},
		{
			name:                 "returns 400 when json body is invalid",
			body:                 []byte(`{invalid`),
			setUserID:            true,
			expectedStatusCode:   http.StatusBadRequest,
			expectedBodyContains: "error",
			mockMethod:           func(m *mockShoppingListService) {},
		},
		{
			name:                 "returns 401 unauthorized when user is not authenticated",
			body:                 jsonItemReq,
			setUserID:            false,
			expectedStatusCode:   http.StatusUnauthorized,
			expectedBodyContains: "unauthorized",
			mockMethod:           func(m *mockShoppingListService) {},
		},
		{
			name:                 "returns 500 when service returns error",
			body:                 jsonItemReq,
			setUserID:            true,
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: "failed to add item",
			mockMethod: func(m *mockShoppingListService) {
				m.On("AddItem", mock.Anything, userID, listID, mock.Anything).Return(errors.New("service error")).Once()
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
			router.POST("/api/v1/shopping-lists/:id/items", func(ctx *gin.Context) {
				if tt.setUserID {
					ctx.Set("user_id", userID)
				}
				handler.AddItem(ctx)
			})

			w := performRequest(router, http.MethodPost, fmt.Sprintf("/api/v1/shopping-lists/%v/items", listID), tt.body)

			require.Equal(t, tt.expectedStatusCode, w.Code)
			if tt.expectedBodyContains != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBodyContains)
			}
			m.AssertExpectations(t)
		})
	}
}

func TestShoppingListHandler_UpdateItem(t *testing.T) {
	userID := "550e8400-e29b-41d4-a716-446655440000"
	listID := "list-uuid-1234"
	itemID := "item-uuid-5678"
	updateItemReq := domain.UpdateShoppingListItemRequest{Name: "Butter", Amount: 1, Unit: "kg", Category: domain.CategoryDairy}
	jsonUpdateItemReq := mustJson(t, updateItemReq)

	tests := []struct {
		name                 string
		body                 []byte
		setUserID            bool
		expectedStatusCode   int
		expectedBodyContains string
		mockMethod           func(m *mockShoppingListService)
	}{
		{
			name:                 "returns 200 when item is updated successfully",
			body:                 jsonUpdateItemReq,
			setUserID:            true,
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "item updated successfully",
			mockMethod: func(m *mockShoppingListService) {
				m.On("UpdateItem", mock.Anything, userID, itemID, mock.MatchedBy(func(req *domain.UpdateShoppingListItemRequest) bool {
					return req.Name == updateItemReq.Name && req.Category == updateItemReq.Category
				})).Return(nil).Once()
			},
		},
		{
			name:                 "returns 400 when json body is invalid",
			body:                 []byte(`{invalid`),
			setUserID:            true,
			expectedStatusCode:   http.StatusBadRequest,
			expectedBodyContains: "error",
			mockMethod:           func(m *mockShoppingListService) {},
		},
		{
			name:                 "returns 401 unauthorized when user is not authenticated",
			body:                 jsonUpdateItemReq,
			setUserID:            false,
			expectedStatusCode:   http.StatusUnauthorized,
			expectedBodyContains: "unauthorized",
			mockMethod:           func(m *mockShoppingListService) {},
		},
		{
			name:                 "returns 500 when service returns error",
			body:                 jsonUpdateItemReq,
			setUserID:            true,
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: "failed to update item",
			mockMethod: func(m *mockShoppingListService) {
				m.On("UpdateItem", mock.Anything, userID, itemID, mock.Anything).Return(errors.New("service error")).Once()
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
			router.PUT("/api/v1/shopping-lists/:id/items/:itemId", func(ctx *gin.Context) {
				if tt.setUserID {
					ctx.Set("user_id", userID)
				}
				handler.UpdateItem(ctx)
			})

			w := performRequest(router, http.MethodPut, fmt.Sprintf("/api/v1/shopping-lists/%v/items/%v", listID, itemID), tt.body)

			require.Equal(t, tt.expectedStatusCode, w.Code)
			if tt.expectedBodyContains != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBodyContains)
			}
			m.AssertExpectations(t)
		})
	}
}

func TestShoppingListHandler_DeleteItem(t *testing.T) {
	userID := "550e8400-e29b-41d4-a716-446655440000"
	listID := "list-uuid-1234"
	itemID := "item-uuid-5678"

	tests := []struct {
		name               string
		setUserID          bool
		expectedStatusCode int
		mockMethod         func(m *mockShoppingListService)
	}{
		{
			name:               "returns 204 when item is deleted successfully",
			setUserID:          true,
			expectedStatusCode: http.StatusNoContent,
			mockMethod: func(m *mockShoppingListService) {
				m.On("DeleteItem", mock.Anything, userID, itemID).Return(nil).Once()
			},
		},
		{
			name:               "returns 401 unauthorized when user is not authenticated",
			setUserID:          false,
			expectedStatusCode: http.StatusUnauthorized,
			mockMethod:         func(m *mockShoppingListService) {},
		},
		{
			name:               "returns 500 when service returns error",
			setUserID:          true,
			expectedStatusCode: http.StatusInternalServerError,
			mockMethod: func(m *mockShoppingListService) {
				m.On("DeleteItem", mock.Anything, userID, itemID).Return(errors.New("service error")).Once()
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
			router.DELETE("/api/v1/shopping-lists/:id/items/:itemId", func(ctx *gin.Context) {
				if tt.setUserID {
					ctx.Set("user_id", userID)
				}
				handler.DeleteItem(ctx)
			})

			w := performRequest(router, http.MethodDelete, fmt.Sprintf("/api/v1/shopping-lists/%v/items/%v", listID, itemID), nil)

			require.Equal(t, tt.expectedStatusCode, w.Code)
			m.AssertExpectations(t)
		})
	}
}

func TestShoppingListHandler_ToggleItem(t *testing.T) {
	userID := "550e8400-e29b-41d4-a716-446655440000"
	listID := "list-uuid-1234"
	itemID := "item-uuid-5678"

	tests := []struct {
		name                 string
		body                 []byte
		setUserID            bool
		expectedStatusCode   int
		expectedBodyContains string
		mockMethod           func(m *mockShoppingListService)
	}{
		{
			name:                 "returns 200 when item is toggled to checked",
			body:                 mustJson(t, map[string]bool{"checked": true}),
			setUserID:            true,
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "item toggled successfully",
			mockMethod: func(m *mockShoppingListService) {
				m.On("ToggleItem", mock.Anything, userID, itemID, true).Return(nil).Once()
			},
		},
		{
			name:                 "returns 200 when item is toggled to unchecked",
			body:                 mustJson(t, map[string]bool{"checked": false}),
			setUserID:            true,
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: "item toggled successfully",
			mockMethod: func(m *mockShoppingListService) {
				m.On("ToggleItem", mock.Anything, userID, itemID, false).Return(nil).Once()
			},
		},
		{
			name:                 "returns 400 when json body is invalid",
			body:                 []byte(`{invalid`),
			setUserID:            true,
			expectedStatusCode:   http.StatusBadRequest,
			expectedBodyContains: "error",
			mockMethod:           func(m *mockShoppingListService) {},
		},
		{
			name:                 "returns 401 unauthorized when user is not authenticated",
			body:                 mustJson(t, map[string]bool{"checked": true}),
			setUserID:            false,
			expectedStatusCode:   http.StatusUnauthorized,
			expectedBodyContains: "unauthorized",
			mockMethod:           func(m *mockShoppingListService) {},
		},
		{
			name:                 "returns 500 when service returns error",
			body:                 mustJson(t, map[string]bool{"checked": true}),
			setUserID:            true,
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: "failed to toggle item",
			mockMethod: func(m *mockShoppingListService) {
				m.On("ToggleItem", mock.Anything, userID, itemID, true).Return(errors.New("service error")).Once()
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
			router.PATCH("/api/v1/shopping-lists/:id/items/:itemId/toggle", func(ctx *gin.Context) {
				if tt.setUserID {
					ctx.Set("user_id", userID)
				}
				handler.ToggleItem(ctx)
			})

			w := performRequest(router, http.MethodPatch, fmt.Sprintf("/api/v1/shopping-lists/%v/items/%v/toggle", listID, itemID), tt.body)

			require.Equal(t, tt.expectedStatusCode, w.Code)
			if tt.expectedBodyContains != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBodyContains)
			}
			m.AssertExpectations(t)
		})
	}
}

func TestShoppingListHandler_AddRecipe(t *testing.T) {
	userID := "550e8400-e29b-41d4-a716-446655440000"
	listID := "list-uuid-1234"
	addRecipeReq := domain.AddRecipeToListRequest{RecipeID: "recipe-uuid-9999", Servings: 2}
	jsonAddRecipeReq := mustJson(t, addRecipeReq)

	tests := []struct {
		name                 string
		body                 []byte
		setUserID            bool
		expectedStatusCode   int
		expectedBodyContains string
		mockMethod           func(m *mockShoppingListService)
	}{
		{
			name:                 "returns 201 when recipe is added to list successfully",
			body:                 jsonAddRecipeReq,
			setUserID:            true,
			expectedStatusCode:   http.StatusCreated,
			expectedBodyContains: "recipe added to list successfully",
			mockMethod: func(m *mockShoppingListService) {
				m.On("AddRecipeToList", mock.Anything, userID, listID, mock.MatchedBy(func(req *domain.AddRecipeToListRequest) bool {
					return req.RecipeID == addRecipeReq.RecipeID && req.Servings == addRecipeReq.Servings
				})).Return(nil).Once()
			},
		},
		{
			name:                 "returns 400 when json body is invalid",
			body:                 []byte(`{invalid`),
			setUserID:            true,
			expectedStatusCode:   http.StatusBadRequest,
			expectedBodyContains: "error",
			mockMethod:           func(m *mockShoppingListService) {},
		},
		{
			name:                 "returns 401 unauthorized when user is not authenticated",
			body:                 jsonAddRecipeReq,
			setUserID:            false,
			expectedStatusCode:   http.StatusUnauthorized,
			expectedBodyContains: "unauthorized",
			mockMethod:           func(m *mockShoppingListService) {},
		},
		{
			name:                 "returns 500 when service returns error",
			body:                 jsonAddRecipeReq,
			setUserID:            true,
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: "failed to add recipe to list",
			mockMethod: func(m *mockShoppingListService) {
				m.On("AddRecipeToList", mock.Anything, userID, listID, mock.Anything).Return(errors.New("service error")).Once()
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
			router.POST("/api/v1/shopping-lists/:id/recipes", func(ctx *gin.Context) {
				if tt.setUserID {
					ctx.Set("user_id", userID)
				}
				handler.AddRecipe(ctx)
			})

			w := performRequest(router, http.MethodPost, fmt.Sprintf("/api/v1/shopping-lists/%v/recipes", listID), tt.body)

			require.Equal(t, tt.expectedStatusCode, w.Code)
			if tt.expectedBodyContains != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBodyContains)
			}
			m.AssertExpectations(t)
		})
	}
}

func TestShoppingListHandler_SortByStore(t *testing.T) {
	userID := "550e8400-e29b-41d4-a716-446655440000"
	listID := "list-uuid-1234"
	chainID := "chain-uuid-0001"
	sortedList := domain.ShoppingList{ID: listID, UserID: userID, Name: "My List"}
	jsonSortedList := mustJson(t, sortedList)

	tests := []struct {
		name                 string
		url                  string
		setUserID            bool
		expectedStatusCode   int
		expectedBodyContains string
		mockMethod           func(m *mockShoppingListService)
	}{
		{
			name:                 "returns 200 with sorted list when chain_id is provided",
			url:                  fmt.Sprintf("/api/v1/shopping-lists/%v/sort-by-store?chain_id=%v", listID, chainID),
			setUserID:            true,
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: string(jsonSortedList),
			mockMethod: func(m *mockShoppingListService) {
				m.On("GetSortedForStore", mock.Anything, userID, listID, chainID).Return(&sortedList, nil).Once()
			},
		},
		{
			name:                 "returns 400 when chain_id query param is missing",
			url:                  fmt.Sprintf("/api/v1/shopping-lists/%v/sort-by-store", listID),
			setUserID:            true,
			expectedStatusCode:   http.StatusBadRequest,
			expectedBodyContains: "chain_id query parameter is required",
			mockMethod:           func(m *mockShoppingListService) {},
		},
		{
			name:                 "returns 401 unauthorized when user is not authenticated",
			url:                  fmt.Sprintf("/api/v1/shopping-lists/%v/sort-by-store?chain_id=%v", listID, chainID),
			setUserID:            false,
			expectedStatusCode:   http.StatusUnauthorized,
			expectedBodyContains: "unauthorized",
			mockMethod:           func(m *mockShoppingListService) {},
		},
		{
			name:                 "returns 500 when service returns error",
			url:                  fmt.Sprintf("/api/v1/shopping-lists/%v/sort-by-store?chain_id=%v", listID, chainID),
			setUserID:            true,
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: "failed to get sorted list",
			mockMethod: func(m *mockShoppingListService) {
				m.On("GetSortedForStore", mock.Anything, userID, listID, chainID).Return(nil, errors.New("service error")).Once()
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
			router.GET("/api/v1/shopping-lists/:id/sort-by-store", func(ctx *gin.Context) {
				if tt.setUserID {
					ctx.Set("user_id", userID)
				}
				handler.SortByStore(ctx)
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
