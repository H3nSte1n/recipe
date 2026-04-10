package service

import (
	"context"
	"errors"
	"github.com/H3nSte1n/recipe/internal/domain"
	apperrors "github.com/H3nSte1n/recipe/internal/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
)

type mockProfileRepository struct {
	mock.Mock
}

func (m *mockProfileRepository) GetByUserID(ctx context.Context, userID string) (*domain.Profile, error) {
	args := m.Called(ctx, userID)
	v, _ := args.Get(0).(*domain.Profile)
	return v, args.Error(1)
}

func (m *mockProfileRepository) Update(ctx context.Context, profile *domain.Profile) error {
	args := m.Called(ctx, profile)
	return args.Error(0)
}

func TestProfileService_GetProfile_Success(t *testing.T) {
	userID := "550e8400-e29b-41d4-a716-446655440000"
	profile := domain.Profile{ID: "1_foo", UserID: userID, Bio: "foobar"}

	m := new(mockProfileRepository)
	m.On("GetByUserID", mock.Anything, userID).Return(&profile, nil)

	service := NewProfileService(m)
	p, err := service.GetProfile(context.Background(), userID)

	require.NoError(t, err)
	require.Equal(t, profile, *p)
	m.AssertExpectations(t)
}

func TestProfileService_GetProfile_Err(t *testing.T) {
	userID := "550e8400-e29b-41d4-a716-446655440000"
	expectedErr := errors.New("repo error")

	m := new(mockProfileRepository)
	m.On("GetByUserID", mock.Anything, userID).Return(nil, expectedErr)

	service := NewProfileService(m)
	p, err := service.GetProfile(context.Background(), userID)

	require.ErrorIs(t, err, expectedErr)
	require.Nil(t, p)
	m.AssertExpectations(t)
}

func TestProfileService_UpdateProfile(t *testing.T) {
	userID := "550e8400-e29b-41d4-a716-446655440000"

	strPtr := func(s string) *string { return &s }

	existingProfile := &domain.Profile{
		ID:         "profile-1",
		UserID:     userID,
		Bio:        "old bio",
		Location:   "old location",
		WebsiteURL: "https://old.example.com",
	}

	tests := []struct {
		name        string
		req         *domain.UpdateProfileRequest
		setupMock   func(m *mockProfileRepository)
		wantProfile *domain.Profile
		wantErrCode string
	}{
		{
			name: "profile not found",
			req:  &domain.UpdateProfileRequest{Bio: strPtr("new bio")},
			setupMock: func(m *mockProfileRepository) {
				m.On("GetByUserID", mock.Anything, userID).Return(nil, errors.New("not found"))
			},
			wantErrCode: "NOT_FOUND",
		},
		{
			name: "updates only bio when only bio is set",
			req:  &domain.UpdateProfileRequest{Bio: strPtr("new bio")},
			setupMock: func(m *mockProfileRepository) {
				m.On("GetByUserID", mock.Anything, userID).Return(existingProfile, nil)
				m.On("Update", mock.Anything, mock.MatchedBy(func(p *domain.Profile) bool {
					return p.Bio == "new bio" &&
						p.Location == "old location" &&
						p.WebsiteURL == "https://old.example.com"
				})).Return(nil)
			},
			wantProfile: &domain.Profile{
				ID:         "profile-1",
				UserID:     userID,
				Bio:        "new bio",
				Location:   "old location",
				WebsiteURL: "https://old.example.com",
			},
		},
		{
			name: "updates all fields when all are set",
			req: &domain.UpdateProfileRequest{
				Bio:        strPtr("new bio"),
				Location:   strPtr("new location"),
				WebsiteURL: strPtr("https://new.example.com"),
			},
			setupMock: func(m *mockProfileRepository) {
				m.On("GetByUserID", mock.Anything, userID).Return(existingProfile, nil)
				m.On("Update", mock.Anything, mock.MatchedBy(func(p *domain.Profile) bool {
					return p.Bio == "new bio" &&
						p.Location == "new location" &&
						p.WebsiteURL == "https://new.example.com"
				})).Return(nil)
			},
			wantProfile: &domain.Profile{
				ID:         "profile-1",
				UserID:     userID,
				Bio:        "new bio",
				Location:   "new location",
				WebsiteURL: "https://new.example.com",
			},
		},
		{
			name: "no fields updated when request is empty",
			req:  &domain.UpdateProfileRequest{},
			setupMock: func(m *mockProfileRepository) {
				m.On("GetByUserID", mock.Anything, userID).Return(existingProfile, nil)
				m.On("Update", mock.Anything, existingProfile).Return(nil)
			},
			wantProfile: existingProfile,
		},
		{
			name: "returns error when repo Update fails",
			req:  &domain.UpdateProfileRequest{Bio: strPtr("new bio")},
			setupMock: func(m *mockProfileRepository) {
				m.On("GetByUserID", mock.Anything, userID).Return(existingProfile, nil)
				m.On("Update", mock.Anything, mock.Anything).Return(errors.New("db error"))
			},
			wantErrCode: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			existingProfile.Bio = "old bio"
			existingProfile.Location = "old location"
			existingProfile.WebsiteURL = "https://old.example.com"

			m := new(mockProfileRepository)
			tt.setupMock(m)

			svc := NewProfileService(m)
			got, err := svc.UpdateProfile(context.Background(), userID, tt.req)

			if tt.wantErrCode != "" {
				require.Error(t, err)
				var appErr *apperrors.AppError
				require.ErrorAs(t, err, &appErr)
				require.Equal(t, tt.wantErrCode, appErr.Code)
				require.Nil(t, got)
			} else if tt.wantProfile == nil {
				require.Error(t, err)
				require.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantProfile, got)
			}

			m.AssertExpectations(t)
		})
	}
}
