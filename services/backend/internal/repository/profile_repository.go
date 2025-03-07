package repository

import (
	"context"
	"github.com/yourusername/recipe-app/internal/domain"
	"gorm.io/gorm"
)

type ProfileRepository interface {
	Create(ctx context.Context, profile *domain.Profile) error
	Update(ctx context.Context, profile *domain.Profile) error
	GetByUserID(ctx context.Context, userID string) (*domain.Profile, error)
	Delete(ctx context.Context, userID string) error
}

type profileRepository struct {
	*baseRepository
}

func NewProfileRepository(db *gorm.DB) ProfileRepository {
	return &profileRepository{
		baseRepository: &baseRepository{db: db},
	}
}

func (r *profileRepository) Create(ctx context.Context, profile *domain.Profile) error {
	return r.db.WithContext(ctx).Create(profile).Error
}

func (r *profileRepository) Update(ctx context.Context, profile *domain.Profile) error {
	return r.db.WithContext(ctx).Where("user_id = ?", profile.UserID).Updates(profile).Error
}

func (r *profileRepository) GetByUserID(ctx context.Context, userID string) (*domain.Profile, error) {
	var profile domain.Profile
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Preload("User").
		First(&profile).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func (r *profileRepository) Delete(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&domain.Profile{}).Error
}
