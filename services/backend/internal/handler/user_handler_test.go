package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	loginResponse, _ := args.Get(0).(*domain.LoginResponse)
	return loginResponse, args.Error(1)
}

func (m *mockUserService) ValidateToken(token string) (*jwt.Token, error) {
	args := m.Called(token)
	t, _ := args.Get(0).(*jwt.Token)
	return t, args.Error(1)
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

func (m *mockUserService) ListAll(ctx context.Context) ([]domain.User, error) {
	args := m.Called(ctx)
	v, _ := args.Get(0).([]domain.User)
	return v, args.Error(1)
}

func Test_UserHandler_Register(t *testing.T) {
	registerRequest := domain.RegisterRequest{Email: "foo@bar.com", Password: "foo123asdasd", FirstName: "foo", LastName: "bar"}
	jsonRequest, _ := json.Marshal(registerRequest)
	tests := []struct {
		name                 string
		expectedStatusCode   int
		body                 string
		expectedBodyContains string
		mockMethod           func(m *mockUserService)
		shouldCallService    bool
	}{
		{
			name:                 "register successfully",
			expectedStatusCode:   http.StatusCreated,
			body:                 string(jsonRequest),
			shouldCallService:    true,
			expectedBodyContains: registerRequest.Email,
			mockMethod: func(m *mockUserService) {
				m.On("Register", mock.Anything, mock.MatchedBy(func(req *domain.RegisterRequest) bool {
					return req.Email == registerRequest.Email && req.Password == registerRequest.Password && req.FirstName == registerRequest.FirstName && req.LastName == registerRequest.LastName
				})).
					Return(&domain.User{
						ID:    "1",
						Email: registerRequest.Email,
					}, nil).
					Once()
			},
		},
		{
			name:               "invalid json",
			expectedStatusCode: http.StatusBadRequest,
			body:               `{"email":"foo@bar.com","password":"foo123asdasd","first_name":"foo","last_nam`,
			shouldCallService:  false,
			mockMethod:         func(m *mockUserService) {},
		},
		{
			name:                 "service error",
			expectedStatusCode:   http.StatusBadRequest,
			expectedBodyContains: "service error",
			body:                 string(jsonRequest),
			shouldCallService:    true,
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

			assert.Equal(t, tt.expectedStatusCode, w.Code)
			if tt.expectedBodyContains != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBodyContains)
			}

			if !tt.shouldCallService {
				m.AssertNotCalled(t, "Register", mock.Anything, mock.Anything)
			}
			m.AssertExpectations(t)
		})
	}
}

func Test_UserHandler_Login(t *testing.T) {
	loginRequest := domain.LoginRequest{Email: "foo@bar.com", Password: "foo123asdasd"}
	jsonRequest, _ := json.Marshal(loginRequest)

	tests := []struct {
		name                 string
		expectedStatusCode   int
		body                 string
		expectedBodyContains string
		shouldCallService    bool
		mockMethod           func(m *mockUserService)
	}{
		{
			name:                 "login successfully",
			expectedStatusCode:   http.StatusOK,
			body:                 string(jsonRequest),
			shouldCallService:    true,
			expectedBodyContains: "fooBarToken",
			mockMethod: func(m *mockUserService) {
				m.On("Login", mock.Anything, mock.MatchedBy(func(req *domain.LoginRequest) bool {
					return req.Email == loginRequest.Email && req.Password == loginRequest.Password
				})).Return(&domain.LoginResponse{
					Token: "fooBarToken",
					User:  domain.User{},
				}, nil).Once()
			},
		},
		{
			name:               "invalid json",
			expectedStatusCode: http.StatusBadRequest,
			body:               `{"email":"foo@bar.com","password":"foo123as`,
			shouldCallService:  false,
			mockMethod:         func(m *mockUserService) {},
		},
		{
			name:                 "service error",
			expectedStatusCode:   http.StatusUnauthorized,
			body:                 string(jsonRequest),
			expectedBodyContains: "service error",
			shouldCallService:    true,
			mockMethod: func(m *mockUserService) {
				m.On("Login", mock.Anything, mock.Anything).Return(nil, errors.New("service error")).Once()
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
			router.POST("/api/v1/auth/login", handler.Login)

			r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(tt.body))
			w := httptest.NewRecorder()

			router.ServeHTTP(w, r)

			assert.Equal(t, tt.expectedStatusCode, w.Code)
			if tt.expectedBodyContains != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBodyContains)
			}

			if !tt.shouldCallService {
				m.AssertNotCalled(t, "Login", mock.Anything, mock.Anything)
			}
			m.AssertExpectations(t)
		})
	}
}

func Test_UserHandler_ForgotPassword(t *testing.T) {
	forgotPasswordRequest := domain.ForgotPasswordRequest{Email: "foo@bar.com"}
	jsonRequest, _ := json.Marshal(forgotPasswordRequest)
	tests := []struct {
		name                 string
		expectedStatusCode   int
		body                 string
		expectedBodyContains string
		shouldCallService    bool
		mockMethod           func(m *mockUserService)
	}{
		{
			name:                 "forgot password successfully",
			expectedStatusCode:   http.StatusOK,
			body:                 string(jsonRequest),
			shouldCallService:    true,
			expectedBodyContains: "if the email exists, a password reset link will be sent",
			mockMethod: func(m *mockUserService) {
				m.On("ForgotPassword", mock.Anything, mock.MatchedBy(func(req *domain.ForgotPasswordRequest) bool {
					return req.Email == forgotPasswordRequest.Email
				})).Return(nil).Once()
			},
		},
		{
			name:               "invalid json",
			expectedStatusCode: http.StatusBadRequest,
			body:               `{"email":"foo@bar.c`,
			shouldCallService:  false,
			mockMethod:         func(m *mockUserService) {},
		},
		{
			name:                 "service error",
			expectedStatusCode:   http.StatusInternalServerError,
			body:                 string(jsonRequest),
			expectedBodyContains: "service error",
			shouldCallService:    true,
			mockMethod: func(m *mockUserService) {
				m.On("ForgotPassword", mock.Anything, mock.Anything).Return(errors.New("service error")).Once()
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
			router.POST("/api/v1/auth/forgot-password", handler.ForgotPassword)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", strings.NewReader(tt.body))
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatusCode, w.Code)
			if tt.expectedBodyContains != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBodyContains)
			}
			if !tt.shouldCallService {
				m.AssertNotCalled(t, "ForgotPassword", mock.Anything, mock.Anything)
			}
			m.AssertExpectations(t)
		})
	}
}

func Test_UserHandler_ResetPassword(t *testing.T) {
	resetPasswordRequest := domain.ResetPasswordRequest{Token: "FoobarToken", Password: "Foobar12315123"}
	jsonRequest, _ := json.Marshal(resetPasswordRequest)
	tests := []struct {
		name                 string
		body                 string
		expectedStatusCode   int
		expectedBodyContains string
		shouldCallService    bool
		mockMethod           func(m *mockUserService)
	}{
		{
			name:                 "reset password successfully",
			body:                 string(jsonRequest),
			expectedStatusCode:   http.StatusOK,
			shouldCallService:    true,
			expectedBodyContains: "password successfully reset",
			mockMethod: func(m *mockUserService) {
				m.On("ResetPassword", mock.Anything, mock.MatchedBy(func(req *domain.ResetPasswordRequest) bool {
					return resetPasswordRequest.Password == req.Password && resetPasswordRequest.Token == req.Token
				})).Return(nil).Once()
			},
		},
		{
			name:               "invalid json",
			body:               `{"password":"foobar","token":"foobar12`,
			expectedStatusCode: http.StatusBadRequest,
			shouldCallService:  false,
			mockMethod:         func(m *mockUserService) {},
		},
		{
			name:                 "service error",
			body:                 string(jsonRequest),
			expectedStatusCode:   http.StatusBadRequest,
			expectedBodyContains: "service error",
			shouldCallService:    true,
			mockMethod: func(m *mockUserService) {
				m.On("ResetPassword", mock.Anything, mock.Anything).Return(errors.New("service error")).Once()
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
			router.POST("/api/v1/auth/reset-password", handler.ResetPassword)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", strings.NewReader(tt.body))
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatusCode, w.Code)
			if tt.expectedBodyContains != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBodyContains)
			}
			if !tt.shouldCallService {
				m.AssertNotCalled(t, "ResetPassword", mock.Anything, mock.Anything)
			}
			m.AssertExpectations(t)
		})
	}
}

func Test_UserHandler_DeleteAccount(t *testing.T) {
	tests := []struct {
		name                 string
		confirm              string
		userID               string
		expectedStatusCode   int
		expectedBodyContains string
		shouldCallService    bool
		mockMethod           func(m *mockUserService)
	}{
		{
			name:                 "delete account successfully",
			confirm:              "true",
			userID:               "123",
			expectedStatusCode:   http.StatusOK,
			shouldCallService:    true,
			expectedBodyContains: "account successfully deleted",
			mockMethod: func(m *mockUserService) {
				m.On("Delete", mock.Anything, "123").Return(nil).Once()
			},
		},
		{
			name:                 "delete account without confirmation",
			confirm:              "false",
			userID:               "123",
			expectedStatusCode:   http.StatusBadRequest,
			shouldCallService:    false,
			expectedBodyContains: "please confirm account deletion by adding ?confirm=true to the request",
			mockMethod:           func(m *mockUserService) {},
		},
		{
			name:                 "service error",
			confirm:              "true",
			userID:               "123",
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: "Internal Server Error",
			shouldCallService:    true,
			mockMethod: func(m *mockUserService) {
				m.On("Delete", mock.Anything, "123").Return(errors.New("service error")).Once()
			},
		},
		{
			name:                 "unauthorized",
			expectedStatusCode:   http.StatusUnauthorized,
			shouldCallService:    false,
			expectedBodyContains: "unauthorized",
			mockMethod:           func(m *mockUserService) {},
		},
	}

	gin.SetMode(gin.TestMode)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockUserService)
			tt.mockMethod(m)

			handler := NewUserHandler(m)
			router := gin.New()
			router.DELETE("/api/v1/users/me", func(c *gin.Context) {
				if tt.userID != "" {
					c.Set("user_id", tt.userID)
				}
				handler.DeleteAccount(c)
			})

			req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/users/me?confirm=%s", tt.confirm), nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatusCode, w.Code)
			if tt.expectedBodyContains != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBodyContains)
			}
			if !tt.shouldCallService {
				m.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
			}
			m.AssertExpectations(t)
		})
	}
}

func Test_UserHandler_ListAll(t *testing.T) {
	users := []domain.User{
		{ID: "1", Email: "foo@bar.com", FirstName: "Foo", LastName: "Bar"},
		{ID: "2", Email: "baz@bar.com", FirstName: "Baz", LastName: "Qux"},
	}
	jsonUsers := mustJson(t, users)

	tests := []struct {
		name                 string
		expectedStatusCode   int
		expectedBodyContains string
		mockMethod           func(m *mockUserService)
	}{
		{
			name:                 "returns 200 with all users when request is successful",
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: string(jsonUsers),
			mockMethod: func(m *mockUserService) {
				m.On("ListAll", mock.Anything).Return(users, nil).Once()
			},
		},
		{
			name:                 "returns 500 internal server error when service returns error",
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: "failed to list users",
			mockMethod: func(m *mockUserService) {
				m.On("ListAll", mock.Anything).Return(nil, errors.New("service error")).Once()
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
			router.GET("/api/v1/users/list", handler.ListAll)

			w := performRequest(router, http.MethodGet, "/api/v1/users/list", nil)

			assert.Equal(t, tt.expectedStatusCode, w.Code)
			if tt.expectedBodyContains != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBodyContains)
			}
			m.AssertExpectations(t)
		})
	}
}
