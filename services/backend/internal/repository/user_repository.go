package repository

import (
	"context"
	"github.com/yourusername/recipe-app/internal/domain"
	"gorm.io/gorm"
)

type UserRepository interface {
	Repository
	// User operations
	Create(ctx context.Context, user *domain.User) error
	Update(ctx context.Context, user *domain.User) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByID(ctx context.Context, id string) (*domain.User, error)
	Delete(ctx context.Context, userID string) error

	// Password reset token operations
	CreateResetToken(ctx context.Context, token *domain.PasswordResetToken) error
	UpdateResetToken(ctx context.Context, token *domain.PasswordResetToken) error
	GetResetTokenByToken(ctx context.Context, token string) (*domain.PasswordResetToken, error)
}

type userRepository struct {
	*baseRepository
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{
		baseRepository: &baseRepository{db: db},
	}
}

// User operations
func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *userRepository) Update(ctx context.Context, user *domain.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// Password reset token operations
func (r *userRepository) CreateResetToken(ctx context.Context, token *domain.PasswordResetToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *userRepository) UpdateResetToken(ctx context.Context, token *domain.PasswordResetToken) error {
	return r.db.WithContext(ctx).Save(token).Error
}

func (r *userRepository) GetResetTokenByToken(ctx context.Context, token string) (*domain.PasswordResetToken, error) {
	var resetToken domain.PasswordResetToken
	err := r.db.WithContext(ctx).
		Where("token = ?", token).
		First(&resetToken).Error
	return &resetToken, err
}

func (r *userRepository) Delete(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).Delete(&domain.User{ID: userID}).Error
}
