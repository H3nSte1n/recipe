package service

import (
	"context"
	"errors"
	"github.com/H3nSte1n/recipe/internal/domain"
	apperrors "github.com/H3nSte1n/recipe/internal/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"strings"
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

func (m *mockProfileRepository) UpdateWithUser(ctx context.Context, profile *domain.Profile, firstName, lastName *string) error {
	args := m.Called(ctx, profile, firstName, lastName)
	return args.Error(0)
}

func TestProfileService_GetProfile(t *testing.T) {
	var errGetByUserID = errors.New("getByUserID error")

	userID := "550e8400-e29b-41d4-a716-446655440000"
	profile := domain.Profile{ID: "profile-1", UserID: userID, Bio: "foobar"}

	tests := []struct {
		name           string
		expectedReturn *domain.Profile
		expectedErr    error
		mockFunc       func(m *mockProfileRepository)
	}{
		{
			name:        "returns error when GetByUserID fails",
			expectedErr: errGetByUserID,
			mockFunc: func(m *mockProfileRepository) {
				m.On("GetByUserID", mock.Anything, userID).Return(nil, errGetByUserID).Once()
			},
		},
		{
			name:           "returns profile when request is successfully",
			expectedReturn: &profile,
			mockFunc: func(m *mockProfileRepository) {
				m.On("GetByUserID", mock.Anything, userID).Return(&profile, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockProfileRepository)
			tt.mockFunc(m)

			svc := NewProfileService(m)
			got, err := svc.GetProfile(context.Background(), userID)

			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
				require.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedReturn, got)
			}
			m.AssertExpectations(t)
		})
	}
}

func TestProfileService_UpdateProfile(t *testing.T) {
	var (
		errGetByUserID = errors.New("getByUserID error")
		errUpdate      = errors.New("update error")
	)

	strPtr := func(s string) *string { return &s }
	userID := "550e8400-e29b-41d4-a716-446655440000"

	newProfile := func() *domain.Profile {
		return &domain.Profile{
			ID:         "profile-1",
			UserID:     userID,
			Bio:        "old bio",
			Location:   "old location",
			WebsiteURL: "https://old.example.com",
		}
	}

	tests := []struct {
		name            string
		req             *domain.UpdateProfileRequest
		expectedReturn  *domain.Profile
		expectedErr     error
		expectedErrCode string
		mockFunc        func(m *mockProfileRepository)
	}{
		{
			name:            "returns ErrNotFound when profile does not exist",
			req:             &domain.UpdateProfileRequest{Bio: strPtr("new bio")},
			expectedErrCode: "NOT_FOUND",
			mockFunc: func(m *mockProfileRepository) {
				m.On("GetByUserID", mock.Anything, userID).Return(nil, apperrors.ErrNotFound).Once()
			},
		},
		{
			name:        "passes through non-NotFound repo errors from GetByUserID",
			req:         &domain.UpdateProfileRequest{Bio: strPtr("new bio")},
			expectedErr: errGetByUserID,
			mockFunc: func(m *mockProfileRepository) {
				m.On("GetByUserID", mock.Anything, userID).Return(nil, errGetByUserID).Once()
			},
		},
		{
			name:        "returns error when Update fails",
			req:         &domain.UpdateProfileRequest{Bio: strPtr("new bio")},
			expectedErr: errUpdate,
			mockFunc: func(m *mockProfileRepository) {
				m.On("GetByUserID", mock.Anything, userID).Return(newProfile(), nil).Once()
				m.On("Update", mock.Anything, mock.Anything).Return(errUpdate).Once()
			},
		},
		{
			name:        "returns error when final GetByUserID reload fails",
			req:         &domain.UpdateProfileRequest{Bio: strPtr("new bio")},
			expectedErr: errGetByUserID,
			mockFunc: func(m *mockProfileRepository) {
				m.On("GetByUserID", mock.Anything, userID).Return(newProfile(), nil).Once()
				m.On("Update", mock.Anything, mock.Anything).Return(nil).Once()
				m.On("GetByUserID", mock.Anything, userID).Return(nil, errGetByUserID).Once()
			},
		},
		{
			name: "updates only bio when only bio is set",
			req:  &domain.UpdateProfileRequest{Bio: strPtr("new bio")},
			expectedReturn: &domain.Profile{
				ID: "profile-1", UserID: userID,
				Bio: "new bio", Location: "old location", WebsiteURL: "https://old.example.com",
			},
			mockFunc: func(m *mockProfileRepository) {
				m.On("GetByUserID", mock.Anything, userID).Return(newProfile(), nil).Once()
				m.On("Update", mock.Anything, mock.MatchedBy(func(p *domain.Profile) bool {
					return p.Bio == "new bio" && p.Location == "old location" && p.WebsiteURL == "https://old.example.com"
				})).Return(nil).Once()
				m.On("GetByUserID", mock.Anything, userID).Return(&domain.Profile{
					ID: "profile-1", UserID: userID,
					Bio: "new bio", Location: "old location", WebsiteURL: "https://old.example.com",
				}, nil).Once()
			},
		},
		{
			name: "updates all fields when all are set",
			req: &domain.UpdateProfileRequest{
				Bio:        strPtr("new bio"),
				Location:   strPtr("new location"),
				WebsiteURL: strPtr("https://new.example.com"),
			},
			expectedReturn: &domain.Profile{
				ID: "profile-1", UserID: userID,
				Bio: "new bio", Location: "new location", WebsiteURL: "https://new.example.com",
			},
			mockFunc: func(m *mockProfileRepository) {
				m.On("GetByUserID", mock.Anything, userID).Return(newProfile(), nil).Once()
				m.On("Update", mock.Anything, mock.MatchedBy(func(p *domain.Profile) bool {
					return p.Bio == "new bio" && p.Location == "new location" && p.WebsiteURL == "https://new.example.com"
				})).Return(nil).Once()
				m.On("GetByUserID", mock.Anything, userID).Return(&domain.Profile{
					ID: "profile-1", UserID: userID,
					Bio: "new bio", Location: "new location", WebsiteURL: "https://new.example.com",
				}, nil).Once()
			},
		},
		{
			name: "preserves unchanged fields when request is empty",
			req:  &domain.UpdateProfileRequest{},
			expectedReturn: &domain.Profile{
				ID: "profile-1", UserID: userID,
				Bio: "old bio", Location: "old location", WebsiteURL: "https://old.example.com",
			},
			mockFunc: func(m *mockProfileRepository) {
				m.On("GetByUserID", mock.Anything, userID).Return(newProfile(), nil).Once()
				m.On("Update", mock.Anything, mock.MatchedBy(func(p *domain.Profile) bool {
					return p.Bio == "old bio" && p.Location == "old location" && p.WebsiteURL == "https://old.example.com"
				})).Return(nil).Once()
				m.On("GetByUserID", mock.Anything, userID).Return(&domain.Profile{
					ID: "profile-1", UserID: userID,
					Bio: "old bio", Location: "old location", WebsiteURL: "https://old.example.com",
				}, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockProfileRepository)
			tt.mockFunc(m)

			svc := NewProfileService(m)
			got, err := svc.UpdateProfile(context.Background(), userID, tt.req)

			if tt.expectedErrCode != "" {
				require.Error(t, err)
				var appErr *apperrors.AppError
				require.ErrorAs(t, err, &appErr)
				require.Equal(t, tt.expectedErrCode, appErr.Code)
				require.Nil(t, got)
			} else if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
				require.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedReturn, got)
			}
			m.AssertExpectations(t)
		})
	}
}

func TestProfileService_UpdateNames(t *testing.T) {
	ctx := context.Background()
	repo := new(mockProfileRepository)
	profile := &domain.Profile{ID: "profile-1", UserID: "user-1", Bio: "bio"}
	first := " Ada "
	last := " Lovelace "
	repo.On("GetByUserID", mock.Anything, "user-1").Return(profile, nil).Once()
	repo.On("UpdateWithUser", mock.Anything, profile, mock.MatchedBy(func(v *string) bool { return v != nil && *v == "Ada" }), mock.MatchedBy(func(v *string) bool { return v != nil && *v == "Lovelace" })).Return(nil).Once()
	repo.On("GetByUserID", mock.Anything, "user-1").Return(profile, nil).Once()
	_, err := NewProfileService(repo).UpdateProfile(ctx, "user-1", &domain.UpdateProfileRequest{FirstName: &first, LastName: &last})
	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestProfileService_RejectsBlankName(t *testing.T) {
	repo := new(mockProfileRepository)
	blank := "  "
	_, err := NewProfileService(repo).UpdateProfile(context.Background(), "user-1", &domain.UpdateProfileRequest{FirstName: &blank})
	require.Error(t, err)
	require.Equal(t, "INVALID_ARGUMENT", err.(*apperrors.AppError).Code)
	repo.AssertNotCalled(t, "UpdateWithUser")
}

func TestProfileService_NameLengthMatchesDatabaseCharacterLimit(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name    string
		value   string
		allowed bool
	}{
		{name: "100 ASCII characters", value: strings.Repeat("a", 100), allowed: true},
		{name: "101 ASCII characters", value: strings.Repeat("a", 101)},
		{name: "100 multibyte characters", value: strings.Repeat("é", 100), allowed: true},
		{name: "101 multibyte characters", value: strings.Repeat("é", 101)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := new(mockProfileRepository)
			profile := &domain.Profile{ID: "profile-1", UserID: "user-1"}
			if tc.allowed {
				repo.On("GetByUserID", mock.Anything, "user-1").Return(profile, nil).Once()
				repo.On("UpdateWithUser", mock.Anything, profile, mock.MatchedBy(func(v *string) bool { return v != nil && *v == tc.value }), (*string)(nil)).Return(nil).Once()
				repo.On("GetByUserID", mock.Anything, "user-1").Return(profile, nil).Once()
			}
			_, err := NewProfileService(repo).UpdateProfile(ctx, "user-1", &domain.UpdateProfileRequest{FirstName: &tc.value})
			if tc.allowed {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Equal(t, "INVALID_ARGUMENT", err.(*apperrors.AppError).Code)
				require.ErrorContains(t, err, "100 characters")
			}
			repo.AssertExpectations(t)
		})
	}
}
