package service

import (
	"context"
	"github.com/H3nSte1n/recipe/internal/domain"
	"github.com/stretchr/testify/mock"
)

type mockUserRepository struct {
	mock.Mock
}

func (m *mockUserRepository) Create(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *mockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	v, _ := args.Get(0).(*domain.User)
	return v, args.Error(1)
}

func (m *mockUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	args := m.Called(ctx, id)
	v, _ := args.Get(0).(*domain.User)
	return v, args.Error(1)
}

func (m *mockUserRepository) Delete(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *mockUserRepository) ListAll(ctx context.Context) ([]domain.User, error) {
	args := m.Called(ctx)
	v, _ := args.Get(0).([]domain.User)
	return v, args.Error(1)
}

func (m *mockUserRepository) UpdatePassword(ctx context.Context, userID string, passwordHash string) error {
	args := m.Called(ctx, userID, passwordHash)
	return args.Error(0)
}

func (m *mockUserRepository) CreateProfile(ctx context.Context, profile *domain.Profile) error {
	args := m.Called(ctx, profile)
	return args.Error(0)
}

func (m *mockUserRepository) CreateResetToken(ctx context.Context, token *domain.PasswordResetToken) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *mockUserRepository) GetResetTokenByToken(ctx context.Context, token string) (*domain.PasswordResetToken, error) {
	args := m.Called(ctx, token)
	v, _ := args.Get(0).(*domain.PasswordResetToken)
	return v, args.Error(1)
}

func (m *mockUserRepository) MarkResetTokenUsed(ctx context.Context, tokenID string) error {
	args := m.Called(ctx, tokenID)
	return args.Error(0)
}

func (m *mockUserRepository) RunTx(ctx context.Context, fn func() error) error {
	args := m.Called(ctx, fn)
	return args.Error(0)
}
