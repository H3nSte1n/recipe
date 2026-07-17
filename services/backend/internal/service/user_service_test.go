package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/H3nSte1n/recipe/internal/domain"
	apperrors "github.com/H3nSte1n/recipe/internal/errors"
	"github.com/H3nSte1n/recipe/internal/repository"
	"github.com/H3nSte1n/recipe/pkg/config"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
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

func (m *mockUserRepository) SetTokenRevocation(ctx context.Context, userID string, revokedAt time.Time) error {
	args := m.Called(ctx, userID, revokedAt)
	return args.Error(0)
}

func (m *mockUserRepository) GetTokenRevokedAt(ctx context.Context, userID string) (*time.Time, error) {
	args := m.Called(ctx, userID)
	v, _ := args.Get(0).(*time.Time)
	return v, args.Error(1)
}

func (m *mockUserRepository) CreateVerificationToken(ctx context.Context, token *domain.EmailVerificationToken) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *mockUserRepository) GetVerificationTokenByToken(ctx context.Context, token string) (*domain.EmailVerificationToken, error) {
	args := m.Called(ctx, token)
	v, _ := args.Get(0).(*domain.EmailVerificationToken)
	return v, args.Error(1)
}

func (m *mockUserRepository) MarkVerificationTokenUsed(ctx context.Context, tokenID string) error {
	args := m.Called(ctx, tokenID)
	return args.Error(0)
}

func (m *mockUserRepository) GetLatestVerificationToken(ctx context.Context, userID string) (*domain.EmailVerificationToken, error) {
	args := m.Called(ctx, userID)
	v, _ := args.Get(0).(*domain.EmailVerificationToken)
	return v, args.Error(1)
}

func (m *mockUserRepository) MarkEmailVerified(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *mockUserRepository) IsEmailVerified(ctx context.Context, userID string) (bool, error) {
	args := m.Called(ctx, userID)
	return args.Bool(0), args.Error(1)
}

func (m *mockUserRepository) Update(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *mockUserRepository) UpdateResetToken(ctx context.Context, token *domain.PasswordResetToken) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *mockUserRepository) RecordFailedLogin(ctx context.Context, userID string, maxAttempts int, lockUntil time.Time) (int, *time.Time, error) {
	args := m.Called(ctx, userID, maxAttempts, lockUntil)
	attempts, _ := args.Get(0).(int)
	lockedUntil, _ := args.Get(1).(*time.Time)
	return attempts, lockedUntil, args.Error(2)
}

func (m *mockUserRepository) ResetLoginLockout(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *mockUserRepository) WithTypedTransaction(ctx context.Context, fn func(repository.UserRepository) error) error {
	args := m.Called(ctx, fn)
	if args.Error(0) != nil {
		return args.Error(0)
	}
	return fn(m)
}

type mockEmailService struct {
	mock.Mock
}

func (m *mockEmailService) SendPasswordResetEmail(to, resetToken string) error {
	args := m.Called(to, resetToken)
	return args.Error(0)
}

func (m *mockEmailService) SendVerificationEmail(to, verificationToken string) error {
	args := m.Called(to, verificationToken)
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
				m.On("WithTypedTransaction", mock.Anything, mock.Anything).Return(nil).Once()
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
				m.On("WithTypedTransaction", mock.Anything, mock.Anything).Return(nil).Once()
				m.On("Create", mock.Anything, mock.Anything).Return(nil).Once()
				m.On("CreateProfile", mock.Anything, mock.Anything).Return(nil).Once()
			},
		},
		{
			name: "continues registration when GetByEmail returns gorm.ErrRecordNotFound",
			mockMethod: func(m *mockUserRepository) {
				m.On("GetByEmail", mock.Anything, req.Email).Return(nil, gorm.ErrRecordNotFound).Once()
				m.On("WithTypedTransaction", mock.Anything, mock.Anything).Return(nil).Once()
				m.On("Create", mock.Anything, mock.Anything).Return(nil).Once()
				m.On("CreateProfile", mock.Anything, mock.Anything).Return(nil).Once()
			},
		},
		{
			name:        "returns error when repos Create method returns error",
			expectedErr: "repo create error",
			mockMethod: func(m *mockUserRepository) {
				m.On("GetByEmail", mock.Anything, req.Email).Return(nil, nil).Once()
				m.On("WithTypedTransaction", mock.Anything, mock.Anything).Return(nil).Once()
				m.On("Create", mock.Anything, mock.Anything).Return(errors.New("repo create error")).Once()
			},
		},
		{
			name:        "returns error when repos CreateProfile method returns error",
			expectedErr: "repo createProfile error",
			mockMethod: func(m *mockUserRepository) {
				m.On("GetByEmail", mock.Anything, req.Email).Return(nil, nil).Once()
				m.On("WithTypedTransaction", mock.Anything, mock.Anything).Return(nil).Once()
				m.On("Create", mock.Anything, mock.Anything).Return(nil).Once()
				m.On("CreateProfile", mock.Anything, mock.Anything).Return(errors.New("repo createProfile error")).Once()
			},
		},
		{
			name:        "returns error when RunTx returns err",
			expectedErr: "tx error",
			mockMethod: func(m *mockUserRepository) {
				m.On("GetByEmail", mock.Anything, req.Email).Return(nil, nil).Once()
				m.On("WithTypedTransaction", mock.Anything, mock.Anything).Return(errors.New("tx error")).Once()
			},
		},
		{
			name: "creates user and profile and returns user when request is successfully",
			mockMethod: func(m *mockUserRepository) {
				m.On("GetByEmail", mock.Anything, req.Email).Return(nil, nil).Once()
				m.On("WithTypedTransaction", mock.Anything, mock.Anything).Return(nil).Once()
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
			// Successful registrations best-effort issue + email a
			// verification token; these calls are Maybe() since paths that
			// return early (before a user is created) never reach them, and
			// a token/send failure here must not fail Register itself.
			m.On("CreateVerificationToken", mock.Anything, mock.Anything).Return(nil).Maybe()

			mEmail := new(mockEmailService)
			mEmail.On("SendVerificationEmail", mock.Anything, mock.Anything).Return(nil).Maybe()

			srv := NewUserService(m, "foobarJWT", config.Config{}, mEmail, zap.NewNop())
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

// TestUserService_Register_SucceedsDespiteVerificationEmailFailure pins the
// "mail outage must not block registration" behavior: unlike ForgotPassword
// (which propagates a send error to a waiting user), Register must return
// the created user with no error even when both the verification-token
// write and the send itself fail, since the account is already committed.
func TestUserService_Register_SucceedsDespiteVerificationEmailFailure(t *testing.T) {
	user := domain.User{ID: "1_foo", FirstName: "Foo", LastName: "Bar", Email: "foo@bar.com"}
	req := domain.RegisterRequest{Email: user.Email, Password: "foobar", FirstName: user.FirstName, LastName: user.LastName}

	t.Run("send failure does not fail Register", func(t *testing.T) {
		m := new(mockUserRepository)
		m.On("GetByEmail", mock.Anything, req.Email).Return(nil, nil).Once()
		m.On("WithTypedTransaction", mock.Anything, mock.Anything).Return(nil).Once()
		m.On("Create", mock.Anything, mock.Anything).Return(nil).Once()
		m.On("CreateProfile", mock.Anything, mock.Anything).Return(nil).Once()
		m.On("CreateVerificationToken", mock.Anything, mock.Anything).Return(nil).Once()

		mEmail := new(mockEmailService)
		mEmail.On("SendVerificationEmail", user.Email, mock.AnythingOfType("string")).Return(errors.New("smtp unreachable")).Once()

		srv := NewUserService(m, "foobarJWT", config.Config{}, mEmail, zap.NewNop())
		u, err := srv.Register(context.Background(), &req)

		require.NoError(t, err)
		require.NotNil(t, u)
		m.AssertExpectations(t)
		mEmail.AssertExpectations(t)
	})

	t.Run("token persistence failure does not fail Register", func(t *testing.T) {
		m := new(mockUserRepository)
		m.On("GetByEmail", mock.Anything, req.Email).Return(nil, nil).Once()
		m.On("WithTypedTransaction", mock.Anything, mock.Anything).Return(nil).Once()
		m.On("Create", mock.Anything, mock.Anything).Return(nil).Once()
		m.On("CreateProfile", mock.Anything, mock.Anything).Return(nil).Once()
		m.On("CreateVerificationToken", mock.Anything, mock.Anything).Return(errors.New("db write failed")).Once()

		mEmail := new(mockEmailService)
		// Never called: token creation failed, so there's nothing to send.

		srv := NewUserService(m, "foobarJWT", config.Config{}, mEmail, zap.NewNop())
		u, err := srv.Register(context.Background(), &req)

		require.NoError(t, err)
		require.NotNil(t, u)
		m.AssertExpectations(t)
		mEmail.AssertExpectations(t)
		mEmail.AssertNotCalled(t, "SendVerificationEmail", mock.Anything, mock.Anything)
	})
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
				m.On("RecordFailedLogin", mock.Anything, invalidUser.ID, maxFailedLoginAttempts, mock.AnythingOfType("time.Time")).
					Return(1, (*time.Time)(nil), nil).Once()
			},
		},
		{
			name: "returns LoginResponse when credentials are valid",
			mockMethod: func(m *mockUserRepository) {
				m.On("GetByEmail", mock.Anything, req.Email).Return(&user, nil).Once()
			},
		},
		{
			name:        "returns account locked error when locked_until is in the future",
			expectedErr: "account temporarily locked",
			mockMethod: func(m *mockUserRepository) {
				lockedUser := user
				until := time.Now().Add(10 * time.Minute)
				lockedUser.LockedUntil = &until
				m.On("GetByEmail", mock.Anything, req.Email).Return(&lockedUser, nil).Once()
			},
		},
		{
			name:        "locks account once the failed-attempt threshold is reached",
			expectedErr: "invalid credentials",
			mockMethod: func(m *mockUserRepository) {
				almostLockedUser := invalidUser
				almostLockedUser.FailedLoginAttempts = maxFailedLoginAttempts - 1
				lockedUntil := time.Now().Add(accountLockDuration)
				m.On("GetByEmail", mock.Anything, req.Email).Return(&almostLockedUser, nil).Once()
				m.On("RecordFailedLogin", mock.Anything, almostLockedUser.ID, maxFailedLoginAttempts, mock.AnythingOfType("time.Time")).
					Return(maxFailedLoginAttempts, &lockedUntil, nil).Once()
			},
		},
		{
			name: "clears an expired lockout and lets a correct password through",
			mockMethod: func(m *mockUserRepository) {
				expiredLockUser := user
				expiredLockUser.FailedLoginAttempts = maxFailedLoginAttempts
				expired := time.Now().Add(-1 * time.Minute)
				expiredLockUser.LockedUntil = &expired
				m.On("GetByEmail", mock.Anything, req.Email).Return(&expiredLockUser, nil).Once()
				m.On("ResetLoginLockout", mock.Anything, expiredLockUser.ID).Return(nil).Once()
			},
		},
		{
			name: "resets the failed-attempt counter on a successful login",
			mockMethod: func(m *mockUserRepository) {
				priorFailuresUser := user
				priorFailuresUser.FailedLoginAttempts = 2
				m.On("GetByEmail", mock.Anything, req.Email).Return(&priorFailuresUser, nil).Once()
				m.On("ResetLoginLockout", mock.Anything, priorFailuresUser.ID).Return(nil).Once()
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
				m.On("WithTypedTransaction", mock.Anything, mock.Anything).Return(nil).Once()
				m.On("UpdatePassword", mock.Anything, user.ID, mock.AnythingOfType("string")).Return(errors.New("user password update failed")).Once()
			},
		},
		{
			name:        "returns error when MarkResetTokenUsed fails",
			expectedErr: "mark reset token as used failed",
			mockMethod: func(m *mockUserRepository) {
				m.On("GetResetTokenByToken", mock.Anything, req.Token).Return(&resetTokenValid, nil).Once()
				m.On("GetByID", mock.Anything, resetTokenValid.UserID).Return(&user, nil).Once()
				m.On("WithTypedTransaction", mock.Anything, mock.Anything).Return(nil).Once()
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
				m.On("WithTypedTransaction", mock.Anything, mock.Anything).Return(errors.New("tx error occurred")).Once()
			},
		},
		{
			name: "returns nil when password reset succeeds",
			mockMethod: func(m *mockUserRepository) {
				m.On("GetResetTokenByToken", mock.Anything, req.Token).Return(&resetTokenValid, nil).Once()
				m.On("GetByID", mock.Anything, resetTokenValid.UserID).Return(&user, nil).Once()
				m.On("WithTypedTransaction", mock.Anything, mock.Anything).Return(nil).Once()
				m.On("UpdatePassword", mock.Anything, user.ID, mock.AnythingOfType("string")).Return(nil).Once()
				m.On("MarkResetTokenUsed", mock.Anything, resetTokenValid.ID).Return(nil).Once()
				m.On("SetTokenRevocation", mock.Anything, user.ID, mock.AnythingOfType("time.Time")).Return(nil).Once()
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
				m.On("WithTypedTransaction", mock.Anything, mock.Anything).Return(nil).Once()
				m.On("SetTokenRevocation", mock.Anything, user.ID, mock.AnythingOfType("time.Time")).Return(nil).Once()
				m.On("Delete", mock.Anything, user.ID).Return(errors.New("delete failed")).Once()
			},
		},
		{
			name:        "returns error when RunTx fails",
			expectedErr: "tx error",
			mockMethod: func(m *mockUserRepository) {
				m.On("GetByID", mock.Anything, user.ID).Return(&user, nil).Once()
				m.On("WithTypedTransaction", mock.Anything, mock.Anything).Return(errors.New("tx error")).Once()
			},
		},
		{
			name: "returns nil when user is deleted successfully",
			mockMethod: func(m *mockUserRepository) {
				m.On("GetByID", mock.Anything, user.ID).Return(&user, nil).Once()
				m.On("WithTypedTransaction", mock.Anything, mock.Anything).Return(nil).Once()
				m.On("SetTokenRevocation", mock.Anything, user.ID, mock.AnythingOfType("time.Time")).Return(nil).Once()
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
	t.Run("returns non-PII summaries when users are fetched successfully", func(t *testing.T) {
		users := []domain.User{
			{ID: "1_foo", Email: "a@b.com", FirstName: "A", LastName: "B"},
			{ID: "2_foo", Email: "c@d.com", FirstName: "C", LastName: "D"},
		}
		m := new(mockUserRepository)
		m.On("ListAll", mock.Anything).Return(users, nil).Once()

		srv := NewUserService(m, "foobar", config.Config{}, new(mockEmailService), zap.NewNop())
		userList, err := srv.ListAll(context.Background())

		require.NoError(t, err)
		// Service projects to summaries that carry no email.
		require.Equal(t, []domain.UserSummary{
			{ID: "1_foo", FirstName: "A", LastName: "B"},
			{ID: "2_foo", FirstName: "C", LastName: "D"},
		}, userList)
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

func TestUserService_VerifyEmail(t *testing.T) {
	req := domain.VerifyEmailRequest{Token: "foobarasd"}
	tokenUsed := domain.EmailVerificationToken{ID: "1_token", UserID: "1_foo", Token: req.Token, ExpiresAt: time.Now().Add(1 * time.Hour), Used: true}
	tokenExpired := domain.EmailVerificationToken{ID: "1_token", UserID: "1_foo", Token: req.Token, ExpiresAt: time.Now().Add(-1 * time.Hour), Used: false}
	tokenValid := domain.EmailVerificationToken{ID: "1_token", UserID: "1_foo", Token: req.Token, ExpiresAt: time.Now().Add(1 * time.Hour), Used: false}
	tests := []struct {
		name        string
		expectedErr string
		mockMethod  func(m *mockUserRepository)
	}{
		{
			name:        "returns error 'invalid token' when GetVerificationTokenByToken fails",
			expectedErr: "invalid token",
			mockMethod: func(m *mockUserRepository) {
				m.On("GetVerificationTokenByToken", mock.Anything, req.Token).Return(nil, errors.New("lookup error")).Once()
			},
		},
		{
			name:        "returns error 'token expired or already used' when token already used",
			expectedErr: "token expired or already used",
			mockMethod: func(m *mockUserRepository) {
				m.On("GetVerificationTokenByToken", mock.Anything, req.Token).Return(&tokenUsed, nil).Once()
			},
		},
		{
			name:        "returns error 'token expired or already used' when token expired",
			expectedErr: "token expired or already used",
			mockMethod: func(m *mockUserRepository) {
				m.On("GetVerificationTokenByToken", mock.Anything, req.Token).Return(&tokenExpired, nil).Once()
			},
		},
		{
			name:        "returns error when MarkEmailVerified fails",
			expectedErr: "mark verified failed",
			mockMethod: func(m *mockUserRepository) {
				m.On("GetVerificationTokenByToken", mock.Anything, req.Token).Return(&tokenValid, nil).Once()
				m.On("MarkEmailVerified", mock.Anything, tokenValid.UserID).Return(errors.New("mark verified failed")).Once()
			},
		},
		{
			name:        "returns error when MarkVerificationTokenUsed fails",
			expectedErr: "mark token used failed",
			mockMethod: func(m *mockUserRepository) {
				m.On("GetVerificationTokenByToken", mock.Anything, req.Token).Return(&tokenValid, nil).Once()
				m.On("MarkEmailVerified", mock.Anything, tokenValid.UserID).Return(nil).Once()
				m.On("MarkVerificationTokenUsed", mock.Anything, tokenValid.ID).Return(errors.New("mark token used failed")).Once()
			},
		},
		{
			name: "returns nil when verification succeeds",
			mockMethod: func(m *mockUserRepository) {
				m.On("GetVerificationTokenByToken", mock.Anything, req.Token).Return(&tokenValid, nil).Once()
				m.On("MarkEmailVerified", mock.Anything, tokenValid.UserID).Return(nil).Once()
				m.On("MarkVerificationTokenUsed", mock.Anything, tokenValid.ID).Return(nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockUserRepository)
			tt.mockMethod(m)

			srv := NewUserService(m, "foobar", config.Config{}, new(mockEmailService), zap.NewNop())
			err := srv.VerifyEmail(context.Background(), &req)

			if tt.expectedErr != "" {
				require.ErrorContains(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
			m.AssertExpectations(t)
		})
	}
}

func TestUserService_ResendVerification(t *testing.T) {
	req := domain.ResendVerificationRequest{Email: "foo@bar.com"}
	unverifiedUser := domain.User{ID: "1_foo", Email: req.Email}
	verifiedAt := time.Now()
	verifiedUser := domain.User{ID: "1_foo", Email: req.Email, EmailVerifiedAt: &verifiedAt}
	recentToken := domain.EmailVerificationToken{ID: "1_token", UserID: unverifiedUser.ID, CreatedAt: time.Now()}
	staleToken := domain.EmailVerificationToken{ID: "1_token", UserID: unverifiedUser.ID, CreatedAt: time.Now().Add(-1 * time.Hour)}

	tests := []struct {
		name            string
		expectedErr     string
		shouldSendEmail bool
		sendEmailErr    error
		mockRepo        func(m *mockUserRepository)
	}{
		{
			name:            "returns nil when no user is found for the email",
			shouldSendEmail: false,
			mockRepo: func(m *mockUserRepository) {
				m.On("GetByEmail", mock.Anything, req.Email).Return(nil, errors.New("not found")).Once()
			},
		},
		{
			name:            "returns nil without sending when user is already verified",
			shouldSendEmail: false,
			mockRepo: func(m *mockUserRepository) {
				m.On("GetByEmail", mock.Anything, req.Email).Return(&verifiedUser, nil).Once()
			},
		},
		{
			name:            "returns nil without sending when a recent token exists (cooldown)",
			shouldSendEmail: false,
			mockRepo: func(m *mockUserRepository) {
				m.On("GetByEmail", mock.Anything, req.Email).Return(&unverifiedUser, nil).Once()
				m.On("GetLatestVerificationToken", mock.Anything, unverifiedUser.ID).Return(&recentToken, nil).Once()
			},
		},
		{
			name:            "issues a new token and sends email when cooldown has passed",
			shouldSendEmail: true,
			mockRepo: func(m *mockUserRepository) {
				m.On("GetByEmail", mock.Anything, req.Email).Return(&unverifiedUser, nil).Once()
				m.On("GetLatestVerificationToken", mock.Anything, unverifiedUser.ID).Return(&staleToken, nil).Once()
				m.On("CreateVerificationToken", mock.Anything, mock.Anything).Return(nil).Once()
			},
		},
		{
			// Regression test: a token-creation failure must not surface as an error —
			// doing so would let the handler's response distinguish this case (registered,
			// unverified, eligible) from the nil-returning "unknown email"/"already
			// verified"/"in cooldown" cases above, defeating the non-enumeration design.
			name: "swallows a token-creation failure instead of returning it",
			mockRepo: func(m *mockUserRepository) {
				m.On("GetByEmail", mock.Anything, req.Email).Return(&unverifiedUser, nil).Once()
				m.On("GetLatestVerificationToken", mock.Anything, unverifiedUser.ID).Return(&staleToken, nil).Once()
				m.On("CreateVerificationToken", mock.Anything, mock.Anything).Return(errors.New("db write failed")).Once()
			},
		},
		{
			// Same non-enumeration requirement for an SMTP failure: the client must see the
			// same generic outcome whether the send succeeded or a mail-provider outage hit.
			name:            "swallows a send failure instead of returning it",
			shouldSendEmail: true,
			sendEmailErr:    errors.New("smtp: connection refused"),
			mockRepo: func(m *mockUserRepository) {
				m.On("GetByEmail", mock.Anything, req.Email).Return(&unverifiedUser, nil).Once()
				m.On("GetLatestVerificationToken", mock.Anything, unverifiedUser.ID).Return(&staleToken, nil).Once()
				m.On("CreateVerificationToken", mock.Anything, mock.Anything).Return(nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mRepo := new(mockUserRepository)
			mEmail := new(mockEmailService)
			tt.mockRepo(mRepo)
			if tt.shouldSendEmail {
				mEmail.On("SendVerificationEmail", req.Email, mock.AnythingOfType("string")).Return(tt.sendEmailErr).Once()
			}

			srv := NewUserService(mRepo, "foobar", config.Config{}, mEmail, zap.NewNop())
			err := srv.ResendVerification(context.Background(), &req)

			if tt.expectedErr != "" {
				require.ErrorContains(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
			mRepo.AssertExpectations(t)
			mEmail.AssertExpectations(t)
		})
	}
}
