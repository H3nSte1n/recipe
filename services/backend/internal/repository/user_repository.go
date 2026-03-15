package repository

import (
	"context"
	"github.com/H3nSte1n/recipe/internal/domain"
	"gorm.io/gorm"
)

type UserRepository interface {
	Repository[domain.User]
	Create(ctx context.Context, user *domain.User) error
	Update(ctx context.Context, user *domain.User) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByID(ctx context.Context, id string) (*domain.User, error)
	Delete(ctx context.Context, userID string) error
	CreateResetToken(ctx context.Context, token *domain.PasswordResetToken) error
	UpdateResetToken(ctx context.Context, token *domain.PasswordResetToken) error
	GetResetTokenByToken(ctx context.Context, token string) (*domain.PasswordResetToken, error)
	WithTypedTransaction(ctx context.Context, fn func(*UserRepositoryImpl) error) error
}

type UserRepositoryImpl struct {
	*BaseRepository[domain.User]
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &UserRepositoryImpl{
		BaseRepository: NewBaseRepository[domain.User](db),
	}
}

func (r *UserRepositoryImpl) WithTypedTransaction(ctx context.Context, fn func(*UserRepositoryImpl) error) error {
	return r.WithTransaction(ctx, func(txRepo Repository[domain.User]) error {
		typed := &UserRepositoryImpl{
			BaseRepository: txRepo.(*BaseRepository[domain.User]),
		}
		return fn(typed)
	})
}

func (r *UserRepositoryImpl) Create(ctx context.Context, user *domain.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *UserRepositoryImpl) Update(ctx context.Context, user *domain.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *UserRepositoryImpl) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepositoryImpl) GetByID(ctx context.Context, id string) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepositoryImpl) CreateResetToken(ctx context.Context, token *domain.PasswordResetToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *UserRepositoryImpl) UpdateResetToken(ctx context.Context, token *domain.PasswordResetToken) error {
	return r.db.WithContext(ctx).Save(token).Error
}

func (r *UserRepositoryImpl) GetResetTokenByToken(ctx context.Context, token string) (*domain.PasswordResetToken, error) {
	var resetToken domain.PasswordResetToken
	err := r.db.WithContext(ctx).
		Where("token = ?", token).
		First(&resetToken).Error
	return &resetToken, err
}

func (r *UserRepositoryImpl) Delete(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).Delete(&domain.User{ID: userID}).Error
}
