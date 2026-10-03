package repository

import (
	"context"
	"github.com/H3nSte1n/recipe/internal/domain"
	"gorm.io/gorm"
)

type ProfileRepository interface {
	Create(ctx context.Context, profile *domain.Profile) error
	Update(ctx context.Context, profile *domain.Profile) error
	UpdateWithUser(ctx context.Context, profile *domain.Profile, firstName, lastName *string) error
	GetByUserID(ctx context.Context, userID string) (*domain.Profile, error)
	Delete(ctx context.Context, userID string) error
}

type profileRepository struct {
	*BaseRepository
}

func NewProfileRepository(db *gorm.DB) ProfileRepository {
	return &profileRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

func (r *profileRepository) Create(ctx context.Context, profile *domain.Profile) error {
	return r.DB.WithContext(ctx).Create(profile).Error
}

func (r *profileRepository) Update(ctx context.Context, profile *domain.Profile) error {
	return r.DB.WithContext(ctx).Model(&domain.Profile{}).Where("user_id = ?", profile.UserID).Updates(map[string]interface{}{
		"bio": profile.Bio, "location": profile.Location, "website_url": profile.WebsiteURL,
	}).Error
}

func (r *profileRepository) UpdateWithUser(ctx context.Context, profile *domain.Profile, firstName, lastName *string) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		profileRepo := &profileRepository{BaseRepository: NewBaseRepository(tx)}
		if err := profileRepo.Update(ctx, profile); err != nil {
			return err
		}
		fields := map[string]interface{}{}
		if firstName != nil {
			fields["first_name"] = *firstName
		}
		if lastName != nil {
			fields["last_name"] = *lastName
		}
		return tx.WithContext(ctx).Model(&domain.User{}).Where("id = ?", profile.UserID).Updates(fields).Error
	})
}

func (r *profileRepository) GetByUserID(ctx context.Context, userID string) (*domain.Profile, error) {
	var profile domain.Profile
	err := r.DB.WithContext(ctx).
		Where("user_id = ?", userID).
		Preload("User").
		First(&profile).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func (r *profileRepository) Delete(ctx context.Context, userID string) error {
	return r.DB.WithContext(ctx).Where("user_id = ?", userID).Delete(&domain.Profile{}).Error
}
