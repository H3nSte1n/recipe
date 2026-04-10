package service

import (
	"context"
	"github.com/H3nSte1n/recipe/internal/domain"
	"github.com/H3nSte1n/recipe/internal/errors"
)

type profileRepository interface {
	GetByUserID(ctx context.Context, userID string) (*domain.Profile, error)
	Update(ctx context.Context, profile *domain.Profile) error
}

type ProfileService interface {
	UpdateProfile(ctx context.Context, userID string, req *domain.UpdateProfileRequest) (*domain.Profile, error)
	GetProfile(ctx context.Context, userID string) (*domain.Profile, error)
}

type profileService struct {
	profileRepo profileRepository
}

func NewProfileService(profileRepo profileRepository) ProfileService {
	return &profileService{
		profileRepo: profileRepo,
	}
}

func (s *profileService) UpdateProfile(ctx context.Context, userID string, req *domain.UpdateProfileRequest) (*domain.Profile, error) {
	profile, err := s.profileRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, errors.ErrNotFound.Wrap("profile not found")
	}

	// Update only provided fields
	if req.Bio != nil {
		profile.Bio = *req.Bio
	}
	if req.Location != nil {
		profile.Location = *req.Location
	}
	if req.WebsiteURL != nil {
		profile.WebsiteURL = *req.WebsiteURL
	}

	if err := s.profileRepo.Update(ctx, profile); err != nil {
		return nil, err
	}

	return profile, nil
}

func (s *profileService) GetProfile(ctx context.Context, userID string) (*domain.Profile, error) {
	return s.profileRepo.GetByUserID(ctx, userID)
}
