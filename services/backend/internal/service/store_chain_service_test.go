package service

import (
	"context"
	"github.com/H3nSte1n/recipe/internal/domain"
	"github.com/stretchr/testify/mock"
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
