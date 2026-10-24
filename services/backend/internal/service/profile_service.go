package service

import (
	"context"
	"github.com/H3nSte1n/recipe/internal/domain"
	"github.com/H3nSte1n/recipe/internal/errors"
	"strings"
	"unicode/utf8"
)

type profileRepository interface {
	GetByUserID(ctx context.Context, userID string) (*domain.Profile, error)
	Update(ctx context.Context, profile *domain.Profile) error
	UpdateWithUser(ctx context.Context, profile *domain.Profile, firstName, lastName *string) error
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
	for _, name := range []*string{req.FirstName, req.LastName} {
		if name != nil {
			*name = strings.TrimSpace(*name)
			if *name == "" || utf8.RuneCountInString(*name) > 100 {
				return nil, errors.New("name must be between 1 and 100 characters", "INVALID_ARGUMENT")
			}
		}
	}
	profile, err := s.profileRepo.GetByUserID(ctx, userID)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, errors.ErrNotFound.Wrap("profile not found")
		}
		return nil, err
	}

	if req.Bio != nil {
		profile.Bio = *req.Bio
	}
	if req.Location != nil {
		profile.Location = *req.Location
	}
	if req.WebsiteURL != nil {
		profile.WebsiteURL = *req.WebsiteURL
	}

	if req.FirstName != nil || req.LastName != nil {
		err = s.profileRepo.UpdateWithUser(ctx, profile, req.FirstName, req.LastName)
	} else {
		err = s.profileRepo.Update(ctx, profile)
	}
	if err != nil {
		return nil, err
	}

	return s.profileRepo.GetByUserID(ctx, userID)
}

func (s *profileService) GetProfile(ctx context.Context, userID string) (*domain.Profile, error) {
	return s.profileRepo.GetByUserID(ctx, userID)
}
