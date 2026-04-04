package handler

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/H3nSte1n/recipe/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockStoreChainService struct {
	mock.Mock
}

func (m *mockStoreChainService) GetChain(ctx context.Context, chainID string) (*domain.StoreChain, error) {
	args := m.Called(ctx, chainID)
	v, _ := args.Get(0).(*domain.StoreChain)
	return v, args.Error(1)
}

func (m *mockStoreChainService) GetChainByName(ctx context.Context, name string, country string) (*domain.StoreChain, error) {
	args := m.Called(ctx, name, country)
	v, _ := args.Get(0).(*domain.StoreChain)
	return v, args.Error(1)
}

func (m *mockStoreChainService) ListChains(ctx context.Context, country string) ([]domain.StoreChain, error) {
	args := m.Called(ctx, country)
	v, _ := args.Get(0).([]domain.StoreChain)
	return v, args.Error(1)
}

func (m *mockStoreChainService) OrganizeShoppingList(ctx context.Context, list *domain.ShoppingList, chainID string) error {
	args := m.Called(ctx, list, chainID)
	return args.Error(0)
}

func TestStoreChainHandler_List(t *testing.T) {
	storeChainList := []domain.StoreChain{
		{ID: "1", Name: "Foo"},
		{ID: "2", Name: "Bar"},
	}
	bytesStoreList, _ := json.Marshal(storeChainList)
	tests := []struct {
		name                 string
		expectedStatusCode   int
		expectedResponse     string
		expectedBodyContains string
		mockMethod           func(m *mockStoreChainService)
	}{
		{
			name:               "return status code ok with list of store chains when request is valid",
			expectedResponse:   string(bytesStoreList),
			expectedStatusCode: http.StatusOK,
			mockMethod: func(m *mockStoreChainService) {
				m.On("ListChains", mock.Anything, mock.AnythingOfType("string")).Return(storeChainList, nil).Once()
			},
		},
		{
			name:                 "return status InternalServerError when service throws error",
			expectedBodyContains: "failed to list store chains",
			expectedStatusCode:   http.StatusInternalServerError,
			mockMethod: func(m *mockStoreChainService) {
				m.On("ListChains", mock.Anything, mock.AnythingOfType("string")).Return(nil, errors.New("service error")).Once()
			},
		},
	}

	gin.SetMode(gin.TestMode)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockStoreChainService)
			tt.mockMethod(m)

			handler := NewStoreChainHandler(m, zap.NewNop())
			router := gin.New()
			router.GET("/api/v1/store-chains", handler.List)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/store-chains", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			require.Equal(t, tt.expectedStatusCode, w.Code)
			if tt.expectedResponse != "" {
				assert.JSONEq(t, tt.expectedResponse, w.Body.String())
			}
			if tt.expectedBodyContains != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBodyContains)
			}
			m.AssertExpectations(t)
		})
	}
}

func TestStoreChainHandler_Get(t *testing.T) {
	storeChain := domain.StoreChain{ID: "1", Name: "Foo"}
	bytesStoreChain, _ := json.Marshal(storeChain)
	tests := []struct {
		name                 string
		expectedStatusCode   int
		expectedBodyContains string
		mockMethod           func(m *mockStoreChainService)
	}{
		{
			name:                 "returns status OK with store chain when request is successfully",
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: string(bytesStoreChain),
			mockMethod: func(m *mockStoreChainService) {
				m.On("GetChain", mock.Anything, storeChain.ID).Return(&storeChain, nil).Once()
			},
		},
		{
			name:                 "returns status NotFound when service returns error",
			expectedStatusCode:   http.StatusNotFound,
			expectedBodyContains: "store chain not found",
			mockMethod: func(m *mockStoreChainService) {
				m.On("GetChain", mock.Anything, storeChain.ID).Return(nil, errors.New("service error")).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockStoreChainService)
			tt.mockMethod(m)

			handler := NewStoreChainHandler(m, zap.NewNop())
			router := gin.New()
			router.GET("/api/v1/store-chains/:id", handler.Get)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/store-chains/1", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)
			require.Equal(t, tt.expectedStatusCode, w.Code)
			if tt.expectedBodyContains != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBodyContains)
			}
			m.AssertExpectations(t)
		})
	}
}
