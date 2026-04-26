package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/H3nSte1n/recipe/internal/domain"
	apperrors "github.com/H3nSte1n/recipe/internal/errors"
	"github.com/H3nSte1n/recipe/pkg/config"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"testing"
	"time"
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

type mockEmailService struct {
	mock.Mock
}

func (m *mockEmailService) SendPasswordResetEmail(to, resetToken string) error {
	args := m.Called(to, resetToken)
	return args.Error(0)
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
			name:        "returns error when GetByEmail returns a non-NotFound DB error",
			expectedErr: "GetByEmail err",
			mockMethod: func(m *mockUserRepository) {
				m.On("GetByEmail", mock.Anything, req.Email).Return(nil, errors.New("GetByEmail err")).Once()
			},
		},
		{
			name: "continues registration when GetByEmail returns ErrNotFound",
			mockMethod: func(m *mockUserRepository) {
				m.On("GetByEmail", mock.Anything, req.Email).Return(nil, apperrors.ErrNotFound).Once()
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

			srv := NewUserService(m, "foobarJWT", config.Config{}, nil, zap.NewNop())
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

			srv := NewUserService(m, "foobarJWT", config.Config{}, nil, zap.NewNop())
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

func TestUserService_ForgotPassword(t *testing.T) {
	validPassword, _ := bcrypt.GenerateFromPassword([]byte("foobar"), bcrypt.MinCost)
	user := domain.User{ID: "1_foo", Email: "foo@bar.com", PasswordHash: string(validPassword)}
	req := domain.ForgotPasswordRequest{Email: "foo@bar.com"}
	tests := []struct {
		name            string
		expectedErr     string
		shouldSendEmail bool
		mockRepo        func(m *mockUserRepository)
		mockEmail       func(m *mockEmailService)
	}{
		{
			name:            "returns nil when no user is found for the email",
			shouldSendEmail: false,
			mockRepo: func(m *mockUserRepository) {
				m.On("GetByEmail", mock.Anything, req.Email).Return(nil, errors.New("email doesnt exist")).Once()
			},
		},
		{
			name:            "returns error when CreateResetToken fails",
			expectedErr:     "failed to save resetToken",
			shouldSendEmail: false,
			mockRepo: func(m *mockUserRepository) {
				m.On("GetByEmail", mock.Anything, req.Email).Return(&user, nil).Once()
				m.On("CreateResetToken", mock.Anything, mock.Anything).Return(errors.New("failed to save resetToken")).Once()
			},
		},
		{
			name:            "creates password reset token with correct fields",
			shouldSendEmail: true,
			mockRepo: func(m *mockUserRepository) {
				m.On("GetByEmail", mock.Anything, req.Email).Return(&user, nil).Once()
				m.On("CreateResetToken", mock.Anything, mock.MatchedBy(func(resetToken *domain.PasswordResetToken) bool {
					return resetToken.UserID == user.ID && !resetToken.ExpiresAt.IsZero() && resetToken.Token != ""
				})).Return(nil).Once()
			},
			mockEmail: func(m *mockEmailService) {
				m.On("SendPasswordResetEmail", user.Email, mock.AnythingOfType("string")).Return(nil).Once()
			},
		},
		{
			name:            "calls SendPasswordResetEmail with correct email and token",
			shouldSendEmail: true,
			mockRepo: func(m *mockUserRepository) {
				m.On("GetByEmail", mock.Anything, req.Email).Return(&user, nil).Once()
				m.On("CreateResetToken", mock.Anything, mock.Anything).Return(nil).Once()
			},
			mockEmail: func(m *mockEmailService) {
				m.On("SendPasswordResetEmail", user.Email, mock.MatchedBy(func(token string) bool {
					return len(token) == 64
				})).Return(nil).Once()
			},
		},
		{
			name:            "returns error when SendPasswordResetEmail fails",
			expectedErr:     "failed to send password reset email",
			shouldSendEmail: true,
			mockRepo: func(m *mockUserRepository) {
				m.On("GetByEmail", mock.Anything, req.Email).Return(&user, nil).Once()
				m.On("CreateResetToken", mock.Anything, mock.Anything).Return(nil).Once()
			},
			mockEmail: func(m *mockEmailService) {
				m.On("SendPasswordResetEmail", user.Email, mock.MatchedBy(func(token string) bool {
					return len(token) == 64
				})).Return(errors.New("failed to send password reset email")).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mRepo := new(mockUserRepository)
			mEmail := new(mockEmailService)
			if tt.mockRepo != nil {
				tt.mockRepo(mRepo)
			}
			if tt.mockEmail != nil {
				tt.mockEmail(mEmail)
			}

			srv := NewUserService(mRepo, "foobar", config.Config{}, mEmail, zap.NewNop())
			err := srv.ForgotPassword(context.Background(), &req)

			if tt.expectedErr != "" {
				require.ErrorContains(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
			mRepo.AssertExpectations(t)
			mEmail.AssertExpectations(t)

			if !tt.shouldSendEmail {
				mEmail.AssertNotCalled(t, "SendPasswordResetEmail", mock.Anything, mock.Anything)
			}
		})
	}
}

func TestUserService_ResetPassword(t *testing.T) {
	user := domain.User{ID: "1_foo", PasswordHash: "asdasd"}
	req := domain.ResetPasswordRequest{Token: "foobarasd", Password: "foobar"}
	resetTokenUsed := domain.PasswordResetToken{ID: "1_token", UserID: user.ID, Token: req.Token, ExpiresAt: time.Now().Add(1 * time.Hour), Used: true}
	resetTokenExpired := domain.PasswordResetToken{ID: "1_token", UserID: user.ID, Token: req.Token, ExpiresAt: time.Now().Add(-1 * time.Hour), Used: false}
	resetTokenValid := domain.PasswordResetToken{ID: "1_token", UserID: user.ID, Token: req.Token, ExpiresAt: time.Now().Add(1 * time.Hour), Used: false}
	tests := []struct {
		name        string
		expectedErr string
		mockMethod  func(m *mockUserRepository)
	}{
		{
			name:        "returns error 'invalid token' when GetResetTokenByToken fails",
			expectedErr: "invalid token",
			mockMethod: func(m *mockUserRepository) {
				m.On("GetResetTokenByToken", mock.Anything, req.Token).Return(nil, errors.New("reset token issue")).Once()
			},
		},
		{
			name:        "returns error  'token expired or already used' when token is already used",
			expectedErr: "token expired or already used",
			mockMethod: func(m *mockUserRepository) {
				m.On("GetResetTokenByToken", mock.Anything, req.Token).Return(&resetTokenUsed, nil).Once()
			},
		},
		{
			name:        "returns error 'token expired or already used' when token is already expired",
			expectedErr: "token expired or already used",
			mockMethod: func(m *mockUserRepository) {
				m.On("GetResetTokenByToken", mock.Anything, req.Token).Return(&resetTokenExpired, nil).Once()
			},
		},
		{
			name:        "returns error from repo when GetByID fails",
			expectedErr: "user doesnt exist with this id",
			mockMethod: func(m *mockUserRepository) {
				m.On("GetResetTokenByToken", mock.Anything, req.Token).Return(&resetTokenValid, nil).Once()
				m.On("GetByID", mock.Anything, resetTokenValid.UserID).Return(nil, errors.New("user doesnt exist with this id")).Once()
			},
		},
		{
			name:        "returns error when UpdatePassword fails",
			expectedErr: "user password update failed",
			mockMethod: func(m *mockUserRepository) {
				m.On("GetResetTokenByToken", mock.Anything, req.Token).Return(&resetTokenValid, nil).Once()
				m.On("GetByID", mock.Anything, resetTokenValid.UserID).Return(&user, nil).Once()
				m.On("RunTx", mock.Anything, mock.Anything).Return(nil).Once()
				m.On("UpdatePassword", mock.Anything, user.ID, mock.AnythingOfType("string")).Return(errors.New("user password update failed")).Once()
			},
		},
		{
			name:        "returns error when MarkResetTokenUsed fails",
			expectedErr: "mark reset token as used failed",
			mockMethod: func(m *mockUserRepository) {
				m.On("GetResetTokenByToken", mock.Anything, req.Token).Return(&resetTokenValid, nil).Once()
				m.On("GetByID", mock.Anything, resetTokenValid.UserID).Return(&user, nil).Once()
				m.On("RunTx", mock.Anything, mock.Anything).Return(nil).Once()
				m.On("UpdatePassword", mock.Anything, user.ID, mock.AnythingOfType("string")).Return(nil).Once()
				m.On("MarkResetTokenUsed", mock.Anything, resetTokenValid.ID).Return(errors.New("mark reset token as used failed")).Once()
			},
		},
		{
			name:        "returns error when tx returns returns error",
			expectedErr: "tx error occurred",
			mockMethod: func(m *mockUserRepository) {
				m.On("GetResetTokenByToken", mock.Anything, req.Token).Return(&resetTokenValid, nil).Once()
				m.On("GetByID", mock.Anything, resetTokenValid.UserID).Return(&user, nil).Once()
				m.On("RunTx", mock.Anything, mock.Anything).Return(errors.New("tx error occurred")).Once()
			},
		},
		{
			name: "returns nil when password reset succeeds",
			mockMethod: func(m *mockUserRepository) {
				m.On("GetResetTokenByToken", mock.Anything, req.Token).Return(&resetTokenValid, nil).Once()
				m.On("GetByID", mock.Anything, resetTokenValid.UserID).Return(&user, nil).Once()
				m.On("RunTx", mock.Anything, mock.Anything).Return(nil).Once()
				m.On("UpdatePassword", mock.Anything, user.ID, mock.AnythingOfType("string")).Return(nil).Once()
				m.On("MarkResetTokenUsed", mock.Anything, resetTokenValid.ID).Return(nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockUserRepository)
			tt.mockMethod(m)

			srv := NewUserService(m, "foobar", config.Config{}, new(mockEmailService), zap.NewNop())
			err := srv.ResetPassword(context.Background(), &req)

			if tt.expectedErr != "" {
				require.ErrorContains(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
			m.AssertExpectations(t)
		})
	}
}

func TestUserService_Delete(t *testing.T) {
	user := domain.User{ID: "1_foo"}
	tests := []struct {
		name        string
		expectedErr string
		mockMethod  func(m *mockUserRepository)
	}{
		{
			name:        "returns error 'user not found' when GetByID fails",
			expectedErr: "user not found",
			mockMethod: func(m *mockUserRepository) {
				m.On("GetByID", mock.Anything, user.ID).Return(nil, apperrors.ErrNotFound).Once()
			},
		},
		{
			name:        "returns error 'delete failed' when Delete fails",
			expectedErr: "delete failed",
			mockMethod: func(m *mockUserRepository) {
				m.On("GetByID", mock.Anything, user.ID).Return(&user, nil).Once()
				m.On("RunTx", mock.Anything, mock.Anything).Return(nil).Once()
				m.On("Delete", mock.Anything, user.ID).Return(errors.New("delete failed")).Once()
			},
		},
		{
			name:        "returns error when RunTx fails",
			expectedErr: "tx error",
			mockMethod: func(m *mockUserRepository) {
				m.On("GetByID", mock.Anything, user.ID).Return(&user, nil).Once()
				m.On("RunTx", mock.Anything, mock.Anything).Return(errors.New("tx error")).Once()
			},
		},
		{
			name: "returns nil when user is deleted successfully",
			mockMethod: func(m *mockUserRepository) {
				m.On("GetByID", mock.Anything, user.ID).Return(&user, nil).Once()
				m.On("RunTx", mock.Anything, mock.Anything).Return(nil).Once()
				m.On("Delete", mock.Anything, user.ID).Return(nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockUserRepository)
			tt.mockMethod(m)

			srv := NewUserService(m, "foobar", config.Config{}, new(mockEmailService), zap.NewNop())
			err := srv.Delete(context.Background(), user.ID)

			if tt.expectedErr != "" {
				require.ErrorContains(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
			m.AssertExpectations(t)
		})
	}
}

func TestUserService_ListAll(t *testing.T) {
	t.Run("returns users when users are fetched successfully", func(t *testing.T) {
		users := []domain.User{{ID: "1_foo"}, {ID: "2_foo"}}
		m := new(mockUserRepository)
		m.On("ListAll", mock.Anything).Return(users, nil).Once()

		srv := NewUserService(m, "foobar", config.Config{}, new(mockEmailService), zap.NewNop())
		userList, err := srv.ListAll(context.Background())

		require.NoError(t, err)
		require.Equal(t, users, userList)
		m.AssertExpectations(t)
	})

	t.Run("returns error when ListAll fails", func(t *testing.T) {
		expectedErr := errors.New("user fetch failed")
		m := new(mockUserRepository)
		m.On("ListAll", mock.Anything).Return(nil, expectedErr).Once()

		srv := NewUserService(m, "foobar", config.Config{}, new(mockEmailService), zap.NewNop())
		_, err := srv.ListAll(context.Background())

		require.ErrorIs(t, err, expectedErr)
		m.AssertExpectations(t)
	})
}
