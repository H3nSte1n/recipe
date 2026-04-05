package handler

import (
	"context"
	"errors"
	"github.com/H3nSte1n/recipe/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

type mockProfileService struct {
	mock.Mock
}

func (m *mockProfileService) UpdateProfile(ctx context.Context, userID string, req *domain.UpdateProfileRequest) (*domain.Profile, error) {
	args := m.Called(ctx, userID, req)
	v, _ := args.Get(0).(*domain.Profile)
	return v, args.Error(1)
}

func (m *mockProfileService) GetProfile(ctx context.Context, userID string) (*domain.Profile, error) {
	args := m.Called(ctx, userID)
	v, _ := args.Get(0).(*domain.Profile)
	return v, args.Error(1)
}

func strPtr(s string) *string { return &s }

func TestProfileHandler_Update(t *testing.T) {
	profileID := "550e8400-e29b-41d4-a716-446655440000"
	updateProfile := domain.UpdateProfileRequest{
		Bio:        strPtr("foobar"),
		Location:   strPtr("Munich"),
		WebsiteURL: strPtr("https://steinhauer.dev"),
	}
	profile := domain.Profile{
		ID:         profileID,
		Bio:        *updateProfile.Bio,
		Location:   *updateProfile.Location,
		WebsiteURL: *updateProfile.WebsiteURL,
	}
	jsonRequest := mustJson(t, updateProfile)
	jsonProfileResponse := mustJson(t, profile)

	tests := []struct {
		name                 string
		body                 []byte
		setUserID            bool
		expectedStatusCode   int
		expectedContainsBody string
		mockMethod           func(m *mockProfileService)
	}{
		{
			name:                 "returns 200 with updated profile on success",
			body:                 jsonRequest,
			setUserID:            true,
			expectedStatusCode:   http.StatusOK,
			expectedContainsBody: string(jsonProfileResponse),
			mockMethod: func(m *mockProfileService) {
				m.On("UpdateProfile", mock.Anything, profileID, mock.MatchedBy(func(req *domain.UpdateProfileRequest) bool {
					return *req.Bio == *updateProfile.Bio &&
						*req.WebsiteURL == *updateProfile.WebsiteURL &&
						*req.Location == *updateProfile.Location
				})).Return(&profile, nil).Once()
			},
		},
		{
			name:                 "returns 400 when request body is invalid JSON",
			body:                 []byte(`{invalid`),
			setUserID:            true,
			expectedStatusCode:   http.StatusBadRequest,
			expectedContainsBody: "error",
			mockMethod:           func(m *mockProfileService) {},
		},
		{
			name:                 "returns 401 when user is not authenticated",
			body:                 jsonRequest,
			setUserID:            false,
			expectedStatusCode:   http.StatusUnauthorized,
			expectedContainsBody: "unauthorized",
			mockMethod:           func(m *mockProfileService) {},
		},
		{
			name:                 "returns 500 when service returns an error",
			body:                 jsonRequest,
			setUserID:            true,
			expectedStatusCode:   http.StatusInternalServerError,
			expectedContainsBody: "service error",
			mockMethod: func(m *mockProfileService) {
				m.On("UpdateProfile", mock.Anything, profileID, mock.Anything).
					Return(nil, errors.New("service error")).Once()
			},
		},
	}

	gin.SetMode(gin.TestMode)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockProfileService)
			tt.mockMethod(m)

			handler := NewProfileHandler(m)
			router := gin.New()
			router.PUT("/api/v1/users", func(ctx *gin.Context) {
				if tt.setUserID {
					ctx.Set("user_id", profileID)
				}
				handler.Update(ctx)
			})

			w := performRequest(router, http.MethodPut, "/api/v1/users", tt.body)

			require.Equal(t, tt.expectedStatusCode, w.Code)
			if tt.expectedContainsBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedContainsBody)
			}
			m.AssertExpectations(t)
		})
	}
}

func TestProfileHandler_Get(t *testing.T) {
	profileID := "550e8400-e29b-41d4-a716-446655440000"
	profile := domain.Profile{
		ID:         profileID,
		Bio:        "foobar",
		Location:   "Munich",
		WebsiteURL: "https://steinhauer.dev",
	}
	jsonProfileResponse := mustJson(t, profile)

	tests := []struct {
		name                 string
		setUserID            bool
		expectedStatusCode   int
		expectedContainsBody string
		mockMethod           func(m *mockProfileService)
	}{
		{
			name:                 "returns 200 with profile on success",
			setUserID:            true,
			expectedStatusCode:   http.StatusOK,
			expectedContainsBody: string(jsonProfileResponse),
			mockMethod: func(m *mockProfileService) {
				m.On("GetProfile", mock.Anything, profileID).Return(&profile, nil).Once()
			},
		},
		{
			name:                 "returns 401 when user is not authenticated",
			setUserID:            false,
			expectedStatusCode:   http.StatusUnauthorized,
			expectedContainsBody: "unauthorized",
			mockMethod:           func(m *mockProfileService) {},
		},
		{
			name:                 "returns 404 when profile is not found",
			setUserID:            true,
			expectedStatusCode:   http.StatusNotFound,
			expectedContainsBody: "profile not found",
			mockMethod: func(m *mockProfileService) {
				m.On("GetProfile", mock.Anything, profileID).
					Return(nil, errors.New("service error")).Once()
			},
		},
	}

	gin.SetMode(gin.TestMode)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockProfileService)
			tt.mockMethod(m)

			handler := NewProfileHandler(m)
			router := gin.New()
			router.GET("", func(ctx *gin.Context) {
				if tt.setUserID {
					ctx.Set("user_id", profileID)
				}
				handler.Get(ctx)
			})

			w := performRequest(router, http.MethodGet, "/", nil)

			require.Equal(t, tt.expectedStatusCode, w.Code)
			if tt.expectedContainsBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedContainsBody)
			}
			m.AssertExpectations(t)
		})
	}
}
