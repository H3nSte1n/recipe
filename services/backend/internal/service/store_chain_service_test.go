package service

import (
	"context"
	"errors"
	"github.com/H3nSte1n/recipe/internal/domain"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"testing"
)

type mockStoreChainRepo struct {
	mock.Mock
}

func (m *mockStoreChainRepo) GetChain(ctx context.Context, chainID string) (*domain.StoreChain, error) {
	args := m.Called(ctx, chainID)
	v, _ := args.Get(0).(*domain.StoreChain)
	return v, args.Error(1)
}

func (m *mockStoreChainRepo) GetChainByName(ctx context.Context, name string, country string) (*domain.StoreChain, error) {
	args := m.Called(ctx, name, country)
	v, _ := args.Get(0).(*domain.StoreChain)
	return v, args.Error(1)
}

func (m *mockStoreChainRepo) ListChains(ctx context.Context, country string) ([]domain.StoreChain, error) {
	args := m.Called(ctx, country)
	v, _ := args.Get(0).([]domain.StoreChain)
	return v, args.Error(1)
}

func TestStoreChainService_GetChain_Success(t *testing.T) {
	storeChain := domain.StoreChain{ID: "1_foo"}
	m := new(mockStoreChainRepo)
	m.On("GetChain", mock.Anything, storeChain.ID).Return(&storeChain, nil).Once()

	srv := NewStoreChainService(m, zap.NewNop())
	v, err := srv.GetChain(context.Background(), storeChain.ID)

	require.NoError(t, err)
	require.Equal(t, storeChain, *v)
	m.AssertExpectations(t)
}

func TestStoreChainService_GetChain_Err(t *testing.T) {
	storeChainID := "1_foo"
	expectedErr := errors.New("service error")
	m := new(mockStoreChainRepo)
	m.On("GetChain", mock.Anything, storeChainID).Return(nil, expectedErr).Once()

	srv := NewStoreChainService(m, zap.NewNop())
	v, err := srv.GetChain(context.Background(), storeChainID)

	require.ErrorIs(t, err, expectedErr)
	require.Nil(t, v)
	m.AssertExpectations(t)
}

func TestStoreChainService_GetChainByName_Success(t *testing.T) {
	storeChain := domain.StoreChain{ID: "1_foo"}
	name := "foo"
	country := "germany"
	m := new(mockStoreChainRepo)
	m.On("GetChainByName", mock.Anything, name, country).Return(&storeChain, nil).Once()

	srv := NewStoreChainService(m, zap.NewNop())
	v, err := srv.GetChainByName(context.Background(), name, country)

	require.NoError(t, err)
	require.Equal(t, storeChain, *v)
	m.AssertExpectations(t)
}

func TestStoreChainService_GetChainByName_Err(t *testing.T) {
	expectedErr := errors.New("service error")
	name := "foo"
	country := "germany"
	m := new(mockStoreChainRepo)
	m.On("GetChainByName", mock.Anything, name, country).Return(nil, expectedErr).Once()

	srv := NewStoreChainService(m, zap.NewNop())
	v, err := srv.GetChainByName(context.Background(), name, country)

	require.ErrorIs(t, err, expectedErr)
	require.Nil(t, v)
	m.AssertExpectations(t)
}

func TestStoreChainService_ListChains_Success(t *testing.T) {
	storeChains := []domain.StoreChain{{ID: "1_foo"}, {ID: "2_foo"}}
	country := "netherlands"
	m := new(mockStoreChainRepo)
	m.On("ListChains", mock.Anything, country).Return(storeChains, nil).Once()

	srv := NewStoreChainService(m, zap.NewNop())
	v, err := srv.ListChains(context.Background(), country)

	require.NoError(t, err)
	require.Equal(t, storeChains, v)
	m.AssertExpectations(t)
}

func TestStoreChainService_ListChains_Err(t *testing.T) {
	expectedErr := errors.New("service error")
	country := "netherlands"
	m := new(mockStoreChainRepo)
	m.On("ListChains", mock.Anything, country).Return(nil, expectedErr).Once()

	srv := NewStoreChainService(m, zap.NewNop())
	v, err := srv.ListChains(context.Background(), country)

	require.ErrorIs(t, err, expectedErr)
	require.Nil(t, v)
	m.AssertExpectations(t)
}

func TestStoreChainService_OrganizeShoppingList(t *testing.T) {
	chainID := "1"
	layout := []domain.StoreSection{
		{Order: 0, Name: "foo", Categories: []domain.Category{domain.CategoryBeverages}},
		{Order: 1, Name: "foo", Categories: []domain.Category{domain.CategoryProduce}},
		{Order: 2, Name: "bar", Categories: []domain.Category{domain.CategoryBakery}},
	}
	storeChain := domain.StoreChain{ID: chainID, Layout: layout}
	sortedShoppingList := domain.ShoppingList{ID: "1_foo", Items: []domain.ShoppingListItem{{Category: domain.CategoryBeverages}, {Category: domain.CategoryProduce}, {Category: domain.CategoryBakery}, {Category: domain.CategoryBakery}}}

	errStoreChainNotFound := errors.New("store chain not found")

	tests := []struct {
		name        string
		expectedErr error
		mockMethod  func(m *mockStoreChainRepo)
	}{
		{
			name:        "returns err when repo GetChain method returns error",
			expectedErr: errStoreChainNotFound,
			mockMethod: func(m *mockStoreChainRepo) {
				m.On("GetChain", mock.Anything, chainID).Return(nil, errStoreChainNotFound).Once()
			},
		},
		{
			name: "sorts shopping list and returns nil when sorting was successfully",
			mockMethod: func(m *mockStoreChainRepo) {
				m.On("GetChain", mock.Anything, chainID).Return(&storeChain, nil).Once()
			},
		},
		{
			name: "edge case, handles unknown category",
			mockMethod: func(m *mockStoreChainRepo) {
				m.On("GetChain", mock.Anything, chainID).Return(&storeChain, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockStoreChainRepo)
			tt.mockMethod(m)
			shoppingList := domain.ShoppingList{ID: "1_foo", Items: []domain.ShoppingListItem{{Category: domain.CategoryBakery}, {Category: domain.CategoryProduce}, {Category: domain.CategoryBakery}, {Category: domain.CategoryBeverages}}}

			srv := NewStoreChainService(m, zap.NewNop())
			err := srv.OrganizeShoppingList(context.Background(), &shoppingList, chainID)

			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, sortedShoppingList, shoppingList)
			}
			m.AssertExpectations(t)
		})
	}
}
