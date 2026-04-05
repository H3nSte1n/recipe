package handler

import (
	"context"
	"github.com/H3nSte1n/recipe/internal/domain"
	"github.com/stretchr/testify/mock"
)

type mockProfileService struct {
	mock.Mock
}

func (m *mockProfileService) UpdateProfile(ctx context.Context, userID string, req *domain.UpdateProfileRequest) (*domain.Profile, error) {
	args := m.Called(ctx, userID, req)
	v := args.Get(0).(*domain.Profile)
	return v, args.Error(1)
}

func (m *mockProfileService) GetProfile(ctx context.Context, userID string) (*domain.Profile, error) {
	args := m.Called(ctx, userID)
	v := args.Get(0).(*domain.Profile)
	return v, args.Error(1)
}
