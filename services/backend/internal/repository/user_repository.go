package repository

import (
	"context"
	"time"

	"github.com/H3nSte1n/recipe/internal/domain"
	"gorm.io/gorm"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	Update(ctx context.Context, user *domain.User) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByID(ctx context.Context, id string) (*domain.User, error)
	Delete(ctx context.Context, userID string) error
	ListAll(ctx context.Context) ([]domain.User, error)
	UpdatePassword(ctx context.Context, userID string, passwordHash string) error
	CreateProfile(ctx context.Context, profile *domain.Profile) error
	CreateResetToken(ctx context.Context, token *domain.PasswordResetToken) error
	UpdateResetToken(ctx context.Context, token *domain.PasswordResetToken) error
	GetResetTokenByToken(ctx context.Context, token string) (*domain.PasswordResetToken, error)
	MarkResetTokenUsed(ctx context.Context, tokenID string) error
	SetLoginLockoutState(ctx context.Context, userID string, failedAttempts int, lockedUntil *time.Time) error
	ResetLoginLockout(ctx context.Context, userID string) error
	WithTypedTransaction(ctx context.Context, fn func(UserRepository) error) error
}

type UserRepositoryImpl struct {
	*BaseRepository
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &UserRepositoryImpl{
		BaseRepository: NewBaseRepository(db),
	}
}

func (r *UserRepositoryImpl) WithTypedTransaction(ctx context.Context, fn func(UserRepository) error) error {
	return r.RunInTransaction(ctx, func(tx *gorm.DB) error {
		txRepo := &UserRepositoryImpl{BaseRepository: NewBaseRepository(tx)}
		return fn(txRepo)
	})
}

func (r *UserRepositoryImpl) Create(ctx context.Context, user *domain.User) error {
	return r.DB.WithContext(ctx).Create(user).Error
}

func (r *UserRepositoryImpl) Update(ctx context.Context, user *domain.User) error {
	return r.DB.WithContext(ctx).Save(user).Error
}

func (r *UserRepositoryImpl) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	if err := r.DB.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepositoryImpl) GetByID(ctx context.Context, id string) (*domain.User, error) {
	var user domain.User
	if err := r.DB.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepositoryImpl) CreateResetToken(ctx context.Context, token *domain.PasswordResetToken) error {
	return r.DB.WithContext(ctx).Create(token).Error
}

func (r *UserRepositoryImpl) UpdateResetToken(ctx context.Context, token *domain.PasswordResetToken) error {
	return r.DB.WithContext(ctx).Save(token).Error
}

func (r *UserRepositoryImpl) GetResetTokenByToken(ctx context.Context, token string) (*domain.PasswordResetToken, error) {
	var resetToken domain.PasswordResetToken
	err := r.DB.WithContext(ctx).
		Where("token = ?", token).
		First(&resetToken).Error
	return &resetToken, err
}

func (r *UserRepositoryImpl) UpdatePassword(ctx context.Context, userID string, passwordHash string) error {
	return r.DB.WithContext(ctx).Model(&domain.User{}).
		Where("id = ?", userID).
		Update("password_hash", passwordHash).Error
}

func (r *UserRepositoryImpl) CreateProfile(ctx context.Context, profile *domain.Profile) error {
	return r.DB.WithContext(ctx).Create(profile).Error
}

func (r *UserRepositoryImpl) MarkResetTokenUsed(ctx context.Context, tokenID string) error {
	return r.DB.WithContext(ctx).Model(&domain.PasswordResetToken{}).
		Where("id = ?", tokenID).
		Update("used", true).Error
}

func (r *UserRepositoryImpl) Delete(ctx context.Context, userID string) error {
	return r.DB.WithContext(ctx).Delete(&domain.User{ID: userID}).Error
}

func (r *UserRepositoryImpl) ListAll(ctx context.Context) ([]domain.User, error) {
	var users []domain.User
	if err := r.DB.WithContext(ctx).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// SetLoginLockoutState persists the failed-login counter and, once the caller has decided the
// account should be locked, the cooldown expiry. Pass a nil lockedUntil to leave the account
// unlocked while still recording the attempt count.
func (r *UserRepositoryImpl) SetLoginLockoutState(ctx context.Context, userID string, failedAttempts int, lockedUntil *time.Time) error {
	return r.DB.WithContext(ctx).Model(&domain.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"failed_login_attempts": failedAttempts,
			"locked_until":           lockedUntil,
		}).Error
}

// ResetLoginLockout clears the failed-login counter and any lockout, called after a successful
// login so past failures don't carry forward.
func (r *UserRepositoryImpl) ResetLoginLockout(ctx context.Context, userID string) error {
	return r.SetLoginLockoutState(ctx, userID, 0, nil)
}
