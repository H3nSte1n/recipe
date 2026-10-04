package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/H3nSte1n/recipe/internal/mealplan/application"
	"github.com/H3nSte1n/recipe/internal/mealplan/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubService struct {
	user   string
	fields domain.Fields
	err    error
	meal   domain.PlannedMeal
}

func (s *stubService) Create(_ context.Context, u string, f domain.Fields) (domain.PlannedMeal, error) {
	s.user = u
	s.fields = f
	return s.meal, s.err
}
func (s *stubService) Replace(_ context.Context, u, id string, f domain.Fields) (domain.PlannedMeal, error) {
	s.user = u
	s.fields = f
	return s.meal, s.err
}
func (s *stubService) Get(_ context.Context, u, id string) (domain.PlannedMeal, error) {
	s.user = u
	return s.meal, s.err
}
func (s *stubService) List(_ context.Context, u string, from, to domain.Date, page, size int) (application.Page, error) {
	s.user = u
	return application.Page{Items: []domain.PlannedMeal{s.meal}, Total: 1, Page: page, PageSize: size}, s.err
}
func (s *stubService) Delete(_ context.Context, u, id string) error { s.user = u; return s.err }

func sampleMeal() domain.PlannedMeal {
	r := uuid.New()
	return domain.PlannedMeal{ID: uuid.New(), RecipeID: r, CookDate: "2026-10-03", Recipe: domain.RecipeSummary{ID: r, Title: "Pasta", Status: "published"}, CreatedAt: time.Now(), UpdatedAt: time.Now()}
}
func testEngine(s *stubService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	g := gin.New()
	g.Use(func(c *gin.Context) { c.Set("user_id", "trusted-user"); c.Next() })
	h := NewHandler(s, zap.NewNop())
	g.POST("/planned-meals", h.Create)
	g.GET("/planned-meals", h.List)
	g.GET("/planned-meals/:id", h.Get)
	g.PUT("/planned-meals/:id", h.Replace)
	g.DELETE("/planned-meals/:id", h.Delete)
	return g
}
func perform(g *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	g.ServeHTTP(w, req)
	return w
}

func TestHandlerCreateResponseAndTrustedIdentity(t *testing.T) {
	s := &stubService{meal: sampleMeal()}
	g := testEngine(s)
	w := perform(g, http.MethodPost, "/planned-meals", `{"recipe_id":"`+s.meal.RecipeID.String()+`","cook_date":"2026-10-03"}`)
	require.Equal(t, 201, w.Code)
	require.Equal(t, "trusted-user", s.user)
	var out map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	require.Equal(t, "2026-10-03", out["cook_date"])
	require.Nil(t, out["notes"])
	require.Nil(t, out["servings"])
	require.Nil(t, out["meal_type"])
	require.NotContains(t, out, "user_id")
	require.Equal(t, "Pasta", out["recipe"].(map[string]any)["title"])
}

func TestHandlerRejectsMalformedInput(t *testing.T) {
	s := &stubService{meal: sampleMeal()}
	g := testEngine(s)
	valid := `{"recipe_id":"` + s.meal.RecipeID.String() + `","cook_date":"2026-10-03"}`
	for _, body := range []string{`{`, `{}`, `{"recipe_id":null,"cook_date":"2026-10-03"}`, `{"recipe_id":"bad","cook_date":"2026-10-03"}`, `{"recipe_id":"` + s.meal.RecipeID.String() + `","cook_date":"2026-02-30"}`, strings.TrimSuffix(valid, "}") + `,"unknown":1}`, valid + valid} {
		w := perform(g, http.MethodPost, "/planned-meals", body)
		require.Equal(t, 400, w.Code, body)
	}
	for _, path := range []string{"/planned-meals", "/planned-meals?from=2026-10-03", "/planned-meals?from=2026-10-03&to=2026-10-04&page=abc", "/planned-meals?from=2026-10-03&to=2026-10-04&page_size=x"} {
		w := perform(g, http.MethodGet, path, "")
		require.Equal(t, 400, w.Code, path)
	}
}

func TestHandlerStatusAndSafeErrors(t *testing.T) {
	s := &stubService{meal: sampleMeal()}
	g := testEngine(s)
	id := s.meal.ID.String()
	w := perform(g, http.MethodDelete, "/planned-meals/"+id, "")
	require.Equal(t, 204, w.Code)
	require.Empty(t, w.Body.String())
	w = perform(g, http.MethodGet, "/planned-meals?from=2026-10-03&to=2026-10-03", "")
	require.Equal(t, 200, w.Code)
	var result map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	require.Len(t, result["items"], 1)
	require.Equal(t, float64(100), result["page_size"])
	for _, tc := range []struct {
		err    error
		status int
	}{{application.ErrNotFound, 404}, {application.ErrInvalid, 400}, {application.ErrUnauthorized, 401}, {errors.New("secret connection detail"), 500}} {
		s.err = tc.err
		w = perform(g, http.MethodGet, "/planned-meals/"+id, "")
		require.Equal(t, tc.status, w.Code)
		require.NotContains(t, w.Body.String(), "secret connection detail")
	}
}
