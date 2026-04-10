package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/H3nSte1n/recipe/internal/domain"
	apperrors "github.com/H3nSte1n/recipe/internal/errors"
	"github.com/H3nSte1n/recipe/pkg/config"
	"github.com/H3nSte1n/recipe/pkg/email"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"time"
)

type userRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByID(ctx context.Context, id string) (*domain.User, error)
	Delete(ctx context.Context, userID string) error
	ListAll(ctx context.Context) ([]domain.User, error)
	UpdatePassword(ctx context.Context, userID string, passwordHash string) error
	CreateProfile(ctx context.Context, profile *domain.Profile) error
	CreateResetToken(ctx context.Context, token *domain.PasswordResetToken) error
	GetResetTokenByToken(ctx context.Context, token string) (*domain.PasswordResetToken, error)
	MarkResetTokenUsed(ctx context.Context, tokenID string) error
	RunTx(ctx context.Context, fn func() error) error
}

type UserService interface {
	Register(ctx context.Context, req *domain.RegisterRequest) (*domain.User, error)
	Login(ctx context.Context, req *domain.LoginRequest) (*domain.LoginResponse, error)
	ValidateToken(token string) (*jwt.Token, error)
	ForgotPassword(ctx context.Context, req *domain.ForgotPasswordRequest) error
	ResetPassword(ctx context.Context, req *domain.ResetPasswordRequest) error
	Delete(ctx context.Context, userID string) error
	ListAll(ctx context.Context) ([]domain.User, error)
}

type userService struct {
	userRepo     userRepository
	jwtSecret    []byte
	jwtDuration  time.Duration
	emailService email.EmailService
}

func NewUserService(userRepo userRepository, jwtSecret string, config config.Config) UserService {
	return &userService{
		userRepo:     userRepo,
		jwtSecret:    []byte(jwtSecret),
		jwtDuration:  config.JWT.Duration,
		emailService: email.NewEmailService(config.SMTP.From, config.SMTP.Password, config.SMTP.Host, config.SMTP.Port, config.Frontend.Url),
	}
}

func (s *userService) Register(ctx context.Context, req *domain.RegisterRequest) (*domain.User, error) {
	existingUser, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return nil, apperrors.New("email already registered")
	}

	var user *domain.User
	err = s.userRepo.RunTx(ctx, func() error {
		hashedPassword, err := domain.HashPassword(req.Password)
		if err != nil {
			return err
		}

		user = &domain.User{
			ID:           uuid.New().String(),
			Email:        req.Email,
			PasswordHash: hashedPassword,
			FirstName:    req.FirstName,
			LastName:     req.LastName,
		}

		if err := s.userRepo.Create(ctx, user); err != nil {
			return err
		}

		profile := &domain.Profile{
			ID:        uuid.New().String(),
			UserID:    user.ID,
			Bio:       fmt.Sprintf("Hello, I'm %s %s", user.FirstName, user.LastName),
			Location:  "",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		return s.userRepo.CreateProfile(ctx, profile)
	})

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *userService) Login(ctx context.Context, req *domain.LoginRequest) (*domain.LoginResponse, error) {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, apperrors.New("invalid credentials")
	}

	if !domain.CheckPasswordHash(req.Password, user.PasswordHash) {
		return nil, apperrors.New("invalid credentials")
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &domain.LoginResponse{
		Token: token,
		User:  *user,
	}, nil
}

func (s *userService) ForgotPassword(ctx context.Context, req *domain.ForgotPasswordRequest) error {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return err
	}
	tokenString := hex.EncodeToString(tokenBytes)

	resetToken := &domain.PasswordResetToken{
		UserID:    user.ID,
		Token:     tokenString,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	if err := s.userRepo.CreateResetToken(ctx, resetToken); err != nil {
		return err
	}

	return s.emailService.SendPasswordResetEmail(user.Email, tokenString)
}

func (s *userService) ResetPassword(ctx context.Context, req *domain.ResetPasswordRequest) error {
	resetToken, err := s.userRepo.GetResetTokenByToken(ctx, req.Token)
	if err != nil {
		return apperrors.New("invalid token")
	}

	if resetToken.Used || time.Now().After(resetToken.ExpiresAt) {
		return apperrors.New("token expired or already used")
	}

	user, err := s.userRepo.GetByID(ctx, resetToken.UserID)
	if err != nil {
		return apperrors.New("user not found")
	}

	hashedPassword, err := domain.HashPassword(req.Password)
	if err != nil {
		return err
	}

	return s.userRepo.RunTx(ctx, func() error {
		if err := s.userRepo.UpdatePassword(ctx, user.ID, hashedPassword); err != nil {
			return err
		}
		return s.userRepo.MarkResetTokenUsed(ctx, resetToken.ID)
	})
}

func (s *userService) generateToken(user *domain.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"exp":     time.Now().Add(s.jwtDuration).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *userService) ValidateToken(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, apperrors.New("unexpected signing method")
		}
		return s.jwtSecret, nil
	})
}

func (s *userService) Delete(ctx context.Context, userID string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return apperrors.ErrNotFound.Wrap("user not found")
	}
	return s.userRepo.RunTx(ctx, func() error {
		return s.userRepo.Delete(ctx, user.ID)
	})
}

func (s *userService) ListAll(ctx context.Context) ([]domain.User, error) {
	return s.userRepo.ListAll(ctx)
}
