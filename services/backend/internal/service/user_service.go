package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/yourusername/recipe-app/internal/domain"
	"github.com/yourusername/recipe-app/internal/repository"
	"github.com/yourusername/recipe-app/pkg/config"
	"github.com/yourusername/recipe-app/pkg/email"
	"time"
)

type UserService interface {
	Register(ctx context.Context, req *domain.RegisterRequest) (*domain.User, error)
	Login(ctx context.Context, req *domain.LoginRequest) (*domain.LoginResponse, error)
	ValidateToken(token string) (*jwt.Token, error)
	ForgotPassword(ctx context.Context, req *domain.ForgotPasswordRequest) error
	ResetPassword(ctx context.Context, req *domain.ResetPasswordRequest) error
	Delete(ctx context.Context, userID string) error
}

type userService struct {
	userRepo     repository.UserRepository
	profileRepo  repository.ProfileRepository
	jwtSecret    []byte
	jwtDuration  time.Duration
	emailService email.EmailService
}

func NewUserService(userRepo repository.UserRepository, profileRepo repository.ProfileRepository, jwtSecret string, config config.Config) UserService {
	return &userService{
		userRepo:     userRepo,
		profileRepo:  profileRepo,
		jwtSecret:    []byte(jwtSecret),
		jwtDuration:  config.JWT.Duration,
		emailService: email.NewEmailService(config.SMTP.From, config.SMTP.Password, config.SMTP.Host, config.SMTP.Port),
	}
}

func (s *userService) Register(ctx context.Context, req *domain.RegisterRequest) (*domain.User, error) {
	existingUser, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return nil, errors.New("email already registered")
	}

	var user *domain.User
	err = s.userRepo.WithTypedTransaction(ctx, func(txRepo *repository.UserRepositoryImpl) error {
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

		if err := txRepo.Create(ctx, user); err != nil {
			return err
		}

		profile := &domain.Profile{
			ID:        uuid.New().String(),
			UserID:    user.ID,
			Bio:       fmt.Sprintf("Hello, I'm %s %s", user.FirstName, user.LastName),
			Location:  "", // Default empty or can come from request if you add it
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := txRepo.GetDB().Create(profile).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *userService) Login(ctx context.Context, req *domain.LoginRequest) (*domain.LoginResponse, error) {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if !domain.CheckPasswordHash(req.Password, user.PasswordHash) {
		return nil, errors.New("invalid credentials")
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
		return errors.New("invalid token")
	}

	if resetToken.Used || time.Now().After(resetToken.ExpiresAt) {
		return errors.New("token expired or already used")
	}

	user, err := s.userRepo.GetByID(ctx, resetToken.UserID)
	if err != nil {
		return errors.New("user not found")
	}

	hashedPassword, err := domain.HashPassword(req.Password)
	if err != nil {
		return err
	}

	return s.userRepo.WithTypedTransaction(ctx, func(txRepo *repository.UserRepositoryImpl) error {
		if err := txRepo.GetDB().Model(&domain.User{}).
			Where("id = ?", user.ID).
			Update("password_hash", hashedPassword).Error; err != nil {
			return err
		}

		if err := txRepo.GetDB().Model(&domain.PasswordResetToken{}).
			Where("id = ?", resetToken.ID).
			Update("used", true).Error; err != nil {
			return err
		}

		return nil
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
			return nil, errors.New("unexpected signing method")
		}
		return s.jwtSecret, nil
	})
}

func (s *userService) Delete(ctx context.Context, userID string) error {
	return s.userRepo.WithTypedTransaction(ctx, func(userRepo *repository.UserRepositoryImpl) error {
		var user domain.User
		if err, _ := userRepo.GetByID(ctx, userID); err != nil {
			return errors.New("user not found")
		}

		if err := userRepo.Delete(ctx, user.ID); err != nil {
			return err
		}

		return nil
	})
}
