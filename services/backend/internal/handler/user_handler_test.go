package handler

import (
	"context"
	"errors"
	"github.com/H3nSte1n/recipe/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type mockUserService struct {
	mock.Mock
}

func (m *mockUserService) Register(ctx context.Context, req *domain.RegisterRequest) (*domain.User, error) {
	args := m.Called(ctx, req)
	user, _ := args.Get(0).(*domain.User)
	return user, args.Error(1)
}

func (m *mockUserService) Login(ctx context.Context, req *domain.LoginRequest) (*domain.LoginResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*domain.LoginResponse), args.Error(1)
}

func (m *mockUserService) ValidateToken(token string) (*jwt.Token, error) {
	args := m.Called(token)
	return args.Get(0).(*jwt.Token), args.Error(0)
}

func (m *mockUserService) ForgotPassword(ctx context.Context, req *domain.ForgotPasswordRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *mockUserService) ResetPassword(ctx context.Context, req *domain.ResetPasswordRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *mockUserService) Delete(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

// WIP
func Test_UserHandler_Register(t *testing.T) {
	tests := []struct {
		name               string
		expectedStatusCode int
		body               string
		expectedErr        error
		mockMethod         func(m *mockUserService)
	}{
		{
			name:               "register successfully",
			expectedStatusCode: http.StatusCreated,
			body:               `{"email":"foo@bar.com","password":"foo123asdasd","first_name":"foo","last_name":"bar"}`,
			mockMethod: func(m *mockUserService) {
				m.On("Register", mock.Anything, mock.Anything).
					Return(&domain.User{
						ID:    "1",
						Email: "foo@bar.com",
					}, nil).
					Once()
			},
		},
		{
			name:               "invalid json",
			expectedStatusCode: http.StatusBadRequest,
			body:               `{"email":"foo@bar.com","password":"foo123asdasd","first_name":"foo","last_nam`,
			mockMethod:         func(m *mockUserService) {},
		},
		{
			name:               "service error",
			expectedStatusCode: http.StatusBadRequest,
			expectedErr:        errors.New("service error"),
			body:               `{"email":"foo@bar.com","password":"foo123asdasd","first_name":"foo","last_name":"bar"}`,
			mockMethod: func(m *mockUserService) {
				m.On("Register", mock.Anything, mock.Anything).Return(nil, errors.New("service error")).Once()
			},
		},
	}

	gin.SetMode(gin.TestMode)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockUserService)
			tt.mockMethod(m)

			handler := NewUserHandler(m)
			router := gin.New()
			router.POST("/api/v1/auth/register", handler.Register)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(tt.body))
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)
			t.Log("response body:", w.Body.String())

			assert.Equal(t, tt.expectedStatusCode, w.Code)
			if tt.expectedErr != nil {
				assert.Contains(t, w.Body.String(), tt.expectedErr.Error())
			}
			m.AssertExpectations(t)
		})
	}
}
