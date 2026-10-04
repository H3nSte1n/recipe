package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/H3nSte1n/recipe/internal/handler"
	mealhttp "github.com/H3nSte1n/recipe/internal/mealplan/adapter/http"
	"github.com/H3nSte1n/recipe/internal/mealplan/application"
	"github.com/H3nSte1n/recipe/internal/mealplan/domain"
	"github.com/H3nSte1n/recipe/internal/service"
	"github.com/H3nSte1n/recipe/pkg/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type verificationService struct {
	service.UserService
	verified bool
}

func (s *verificationService) IsEmailVerified(context.Context, string) (bool, error) {
	return s.verified, nil
}

type plannerRouteStub struct {
	mealhttp.UseCases
	user string
}

func (s *plannerRouteStub) List(_ context.Context, user string, _, _ domain.Date, page, size int) (application.Page, error) {
	s.user = user
	return application.Page{Items: []domain.PlannedMeal{}, Page: page, PageSize: size}, nil
}
func (s *plannerRouteStub) Create(_ context.Context, user string, f domain.Fields) (domain.PlannedMeal, error) {
	s.user = user
	now := time.Now()
	return domain.PlannedMeal{ID: uuid.New(), RecipeID: f.RecipeID, CookDate: f.CookDate, Recipe: domain.RecipeSummary{ID: f.RecipeID, Title: "Pasta", Status: "published"}, CreatedAt: now, UpdatedAt: now}, nil
}

func TestPlannedMealRoutesRequireJWTAndVerifiedWrites(t *testing.T) {
	user := uuid.New().String()
	recipe := uuid.New().String()
	cfg := config.Config{JWT: config.JWTConfig{Secret: "test-secret", Issuer: "test-issuer", Audience: "test-audience"}, CORS: config.CORSConfig{AllowedOrigins: []string{"http://localhost"}}}
	verify := &verificationService{}
	services := &handler.Handlers{UserHandler: handler.NewUserHandler(verify)}
	planner := &plannerRouteStub{}
	r := NewRouter(services, mealhttp.NewHandler(planner, zap.NewNop()), cfg, zap.NewNop(), nil)
	engine := r.SetupRoutes()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"user_id": user, "iss": "test-issuer", "aud": "test-audience", "exp": time.Now().Add(time.Hour).Unix()}).SignedString([]byte("test-secret"))
	require.NoError(t, err)

	request := func(method, path, body, auth string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if auth != "" {
			req.Header.Set("Authorization", "Bearer "+auth)
		}
		engine.ServeHTTP(w, req)
		return w
	}
	base := "/api/v1/planned-meals"
	list := base + "?from=2026-10-03&to=2026-10-03"
	body := `{"recipe_id":"` + recipe + `","cook_date":"2026-10-03"}`
	require.Equal(t, 401, request(http.MethodGet, list, "", "").Code)
	require.Equal(t, 401, request(http.MethodPost, base, body, "").Code)
	require.Equal(t, 401, request(http.MethodGet, list, "", "invalid").Code)
	require.Equal(t, 200, request(http.MethodGet, list, "", token).Code)
	require.Equal(t, user, planner.user)
	require.Equal(t, 403, request(http.MethodPost, base, body, token).Code)
	require.Equal(t, 403, request(http.MethodPut, base+"/"+uuid.New().String(), body, token).Code)
	require.Equal(t, 403, request(http.MethodDelete, base+"/"+uuid.New().String(), "", token).Code)
	verify.verified = true
	require.Equal(t, 201, request(http.MethodPost, base, body, token).Code)
	require.Equal(t, user, planner.user)
}
