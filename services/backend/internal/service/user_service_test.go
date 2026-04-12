package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/H3nSte1n/recipe/internal/domain"
	"github.com/H3nSte1n/recipe/pkg/config"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"testing"
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
	if args.Error(0) != nil {
		return args.Error(0)
	}
	return fn()
}

func TestUserService_Register(t *testing.T) {
	user := domain.User{ID: "1_foo", FirstName: "Foo", LastName: "Bar", Email: "foo@bar.com"}
	req := domain.RegisterRequest{Email: user.Email, Password: "foobar", FirstName: user.FirstName, LastName: user.LastName}
	var createdUser *domain.User
	tests := []struct {
		name        string
		expectedErr string
		mockMethod  func(m *mockUserRepository)
	}{
		{
			name:        "returns error when email is already registered",
			expectedErr: "email already registered",
			mockMethod: func(m *mockUserRepository) {
				m.On("GetByEmail", mock.Anything, req.Email).Return(&user, nil).Once()
			},
		},
		{
			name: "continues registration when email is not found",
			mockMethod: func(m *mockUserRepository) {
				m.On("GetByEmail", mock.Anything, req.Email).Return(nil, nil).Once()
				m.On("RunTx", mock.Anything, mock.Anything).Return(nil).Once()
				m.On("Create", mock.Anything, mock.Anything).Return(nil).Once()
				m.On("CreateProfile", mock.Anything, mock.Anything).Return(nil).Once()
			},
		},
		{
			name: "continues registration when email is not found and GetByEmail returns err",
			mockMethod: func(m *mockUserRepository) {
				m.On("GetByEmail", mock.Anything, req.Email).Return(nil, errors.New("GetByEmail err")).Once()
				m.On("RunTx", mock.Anything, mock.Anything).Return(nil).Once()
				m.On("Create", mock.Anything, mock.Anything).Return(nil).Once()
				m.On("CreateProfile", mock.Anything, mock.Anything).Return(nil).Once()
			},
		},
		{
			name:        "returns error when repos Create method returns error",
			expectedErr: "repo create error",
			mockMethod: func(m *mockUserRepository) {
				m.On("GetByEmail", mock.Anything, req.Email).Return(nil, nil).Once()
				m.On("RunTx", mock.Anything, mock.Anything).Return(nil).Once()
				m.On("Create", mock.Anything, mock.Anything).Return(errors.New("repo create error")).Once()
			},
		},
		{
			name:        "returns error when repos CreateProfile method returns error",
			expectedErr: "repo createProfile error",
			mockMethod: func(m *mockUserRepository) {
				m.On("GetByEmail", mock.Anything, req.Email).Return(nil, nil).Once()
				m.On("RunTx", mock.Anything, mock.Anything).Return(nil).Once()
				m.On("Create", mock.Anything, mock.Anything).Return(nil).Once()
				m.On("CreateProfile", mock.Anything, mock.Anything).Return(errors.New("repo createProfile error")).Once()
			},
		},
		{
			name:        "returns error when RunTx returns err",
			expectedErr: "tx error",
			mockMethod: func(m *mockUserRepository) {
				m.On("GetByEmail", mock.Anything, req.Email).Return(nil, nil).Once()
				m.On("RunTx", mock.Anything, mock.Anything).Return(errors.New("tx error")).Once()
			},
		},
		{
			name: "creates user and profile and returns user when request is successfully",
			mockMethod: func(m *mockUserRepository) {
				m.On("GetByEmail", mock.Anything, req.Email).Return(nil, nil).Once()
				m.On("RunTx", mock.Anything, mock.Anything).Return(nil).Once()
				m.On("Create", mock.Anything, mock.MatchedBy(func(userReq *domain.User) bool {
					createdUser = userReq
					return userReq.Email == user.Email && userReq.FirstName == user.FirstName && userReq.LastName == user.LastName && userReq.ID != ""
				})).Return(nil).Once()
				m.On("CreateProfile", mock.Anything, mock.MatchedBy(func(profileReq *domain.Profile) bool {
					return profileReq.UserID == createdUser.ID && profileReq.Location == "" && !profileReq.CreatedAt.IsZero() && !profileReq.UpdatedAt.IsZero() && profileReq.ID != "" && profileReq.Bio == fmt.Sprintf("Hello, I'm %s %s", user.FirstName, user.LastName)
				})).Return(nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockUserRepository)
			if tt.mockMethod != nil {
				tt.mockMethod(m)
			}

			srv := NewUserService(m, "foobarJWT", config.Config{})
			u, err := srv.Register(context.Background(), &req)

			if tt.expectedErr != "" {
				require.ErrorContains(t, err, tt.expectedErr)
				require.Nil(t, u)
			} else {
				require.NoError(t, err)
				require.NotNil(t, u)
			}
			m.AssertExpectations(t)
		})
	}
}

func TestUserService_Login(t *testing.T) {
	validPassword, _ := bcrypt.GenerateFromPassword([]byte("foobar"), bcrypt.MinCost)
	invalidPassword, _ := bcrypt.GenerateFromPassword([]byte("barfoo"), bcrypt.MinCost)
	user := domain.User{ID: "1_foo", Email: "foo@bar.com", PasswordHash: string(validPassword)}
	invalidUser := domain.User{ID: "1_foo", Email: "foo@bar.com", PasswordHash: string(invalidPassword)}
	req := domain.LoginRequest{Password: "foobar", Email: user.Email}
	tests := []struct {
		name        string
		expectedErr string
		mockMethod  func(m *mockUserRepository)
	}{
		{
			name:        "returns invalid Credentials when GetByEmail returns error",
			expectedErr: "invalid credentials",
			mockMethod: func(m *mockUserRepository) {
				m.On("GetByEmail", mock.Anything, req.Email).Return(nil, errors.New("login error")).Once()
			},
		},
		{
			name:        "returns invalid Credentials when password does not match",
			expectedErr: "invalid credentials",
			mockMethod: func(m *mockUserRepository) {
				m.On("GetByEmail", mock.Anything, req.Email).Return(&invalidUser, nil).Once()
			},
		},
		{
			name: "returns LoginResponse when credentials are valid",
			mockMethod: func(m *mockUserRepository) {
				m.On("GetByEmail", mock.Anything, req.Email).Return(&user, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockUserRepository)
			tt.mockMethod(m)

			srv := NewUserService(m, "foobar", config.Config{})
			resp, err := srv.Login(context.Background(), &req)

			if tt.expectedErr != "" {
				require.ErrorContains(t, err, tt.expectedErr)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.NotEmpty(t, resp.Token)
				require.Equal(t, user.ID, resp.User.ID)
				require.Equal(t, user.Email, resp.User.Email)
			}
			m.AssertExpectations(t)
		})
	}
}
