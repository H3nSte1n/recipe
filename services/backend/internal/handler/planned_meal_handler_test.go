package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/H3nSte1n/recipe/internal/domain"
	internalErr "github.com/H3nSte1n/recipe/internal/errors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type mockPlannedMealService struct {
	mock.Mock
}

func (m *mockPlannedMealService) Create(ctx context.Context, userID string, req *domain.PlannedMealRequest) (*domain.PlannedMeal, error) {
	args := m.Called(ctx, userID, req)
	v, _ := args.Get(0).(*domain.PlannedMeal)
	return v, args.Error(1)
}

func (m *mockPlannedMealService) Update(ctx context.Context, userID string, id string, req *domain.PlannedMealRequest) (*domain.PlannedMeal, error) {
	args := m.Called(ctx, userID, id, req)
	v, _ := args.Get(0).(*domain.PlannedMeal)
	return v, args.Error(1)
}

func (m *mockPlannedMealService) GetByID(ctx context.Context, userID string, id string) (*domain.PlannedMeal, error) {
	args := m.Called(ctx, userID, id)
	v, _ := args.Get(0).(*domain.PlannedMeal)
	return v, args.Error(1)
}

func (m *mockPlannedMealService) Delete(ctx context.Context, userID string, id string) error {
	return m.Called(ctx, userID, id).Error(0)
}

func (m *mockPlannedMealService) List(ctx context.Context, userID string, from, to string, page, pageSize int) (*domain.PlannedMealPage, error) {
	args := m.Called(ctx, userID, from, to, page, pageSize)
	v, _ := args.Get(0).(*domain.PlannedMealPage)
	return v, args.Error(1)
}

const (
	handlerUserID = "550e8400-e29b-41d4-a716-446655440000"
	handlerMealID = "6ba7b811-9dad-11d1-80b4-00c04fd430c8"
	handlerRecipe = "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
)

func plannedMealRouter(svc *mockPlannedMealService, setUserID bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	if setUserID {
		r.Use(func(c *gin.Context) { c.Set("user_id", handlerUserID); c.Next() })
	}
	h := NewPlannedMealHandler(svc, zap.NewNop())
	r.POST("/planned-meals", h.Create)
	r.GET("/planned-meals", h.List)
	r.GET("/planned-meals/:id", h.Get)
	r.PUT("/planned-meals/:id", h.Update)
	r.DELETE("/planned-meals/:id", h.Delete)
	return r
}

func sampleHandlerMeal() *domain.PlannedMeal {
	now := time.Now()
	return &domain.PlannedMeal{
		ID: handlerMealID, UserID: handlerUserID, RecipeID: handlerRecipe, CookDate: "2026-10-03",
		Recipe:    domain.PlannedMealRecipe{ID: handlerRecipe, Title: "Pasta", Status: "published"},
		CreatedAt: now, UpdatedAt: now,
	}
}

const validPlannedMealBody = `{"recipe_id":"` + handlerRecipe + `","cook_date":"2026-10-03"}`

func TestPlannedMealHandler_Create(t *testing.T) {
	t.Run("returns 201 with the meal and uses the authenticated user", func(t *testing.T) {
		svc := new(mockPlannedMealService)
		svc.On("Create", mock.Anything, handlerUserID, mock.MatchedBy(func(r *domain.PlannedMealRequest) bool {
			return r.RecipeID == handlerRecipe && r.CookDate == "2026-10-03"
		})).Return(sampleHandlerMeal(), nil).Once()

		w := performRequest(plannedMealRouter(svc, true), http.MethodPost, "/planned-meals", []byte(validPlannedMealBody))

		require.Equal(t, http.StatusCreated, w.Code)
		var out map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
		require.Equal(t, "2026-10-03", out["cook_date"])
		require.Equal(t, handlerRecipe, out["recipe_id"])
		require.Nil(t, out["notes"])
		require.NotContains(t, out, "user_id")
		require.Equal(t, "Pasta", out["recipe"].(map[string]any)["title"])
	})

	t.Run("ignores a user_id supplied in the body", func(t *testing.T) {
		svc := new(mockPlannedMealService)
		svc.On("Create", mock.Anything, handlerUserID, mock.Anything).Return(sampleHandlerMeal(), nil).Once()
		body := `{"user_id":"someone-else","recipe_id":"` + handlerRecipe + `","cook_date":"2026-10-03"}`

		w := performRequest(plannedMealRouter(svc, true), http.MethodPost, "/planned-meals", []byte(body))

		require.Equal(t, http.StatusCreated, w.Code)
		svc.AssertExpectations(t)
	})

	t.Run("returns 401 without a user", func(t *testing.T) {
		w := performRequest(plannedMealRouter(new(mockPlannedMealService), false), http.MethodPost, "/planned-meals", []byte(validPlannedMealBody))
		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	for name, body := range map[string]string{
		"malformed json":    `{`,
		"empty object":      `{}`,
		"wrong field type":  `{"recipe_id":"` + handlerRecipe + `","cook_date":"2026-10-03","servings":2.5}`,
		"missing cook_date": `{"recipe_id":"` + handlerRecipe + `"}`,
		"null recipe_id":    `{"recipe_id":null,"cook_date":"2026-10-03"}`,
	} {
		t.Run("returns 400 for "+name+" without leaking decoder details", func(t *testing.T) {
			svc := new(mockPlannedMealService)

			w := performRequest(plannedMealRouter(svc, true), http.MethodPost, "/planned-meals", []byte(body))

			require.Equal(t, http.StatusBadRequest, w.Code)
			require.Contains(t, w.Body.String(), "invalid request body")
			require.NotContains(t, w.Body.String(), "Go struct")
			svc.AssertNotCalled(t, "Create", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestPlannedMealHandler_ErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		status     int
		notContain string
	}{
		{"not found", internalErr.ErrNotFound, http.StatusNotFound, ""},
		{"invalid input", internalErr.New("only published recipes can be planned", "INVALID"), http.StatusBadRequest, ""},
		{"unexpected failure", errors.New("secret connection detail"), http.StatusInternalServerError, "secret connection detail"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := new(mockPlannedMealService)
			svc.On("Create", mock.Anything, handlerUserID, mock.Anything).Return(nil, tc.err).Once()

			w := performRequest(plannedMealRouter(svc, true), http.MethodPost, "/planned-meals", []byte(validPlannedMealBody))

			require.Equal(t, tc.status, w.Code)
			if tc.notContain != "" {
				require.NotContains(t, w.Body.String(), tc.notContain)
			}
		})
	}

	t.Run("invalid input returns the message to the client", func(t *testing.T) {
		svc := new(mockPlannedMealService)
		svc.On("Create", mock.Anything, handlerUserID, mock.Anything).Return(nil, internalErr.New("only published recipes can be planned", "INVALID")).Once()
		w := performRequest(plannedMealRouter(svc, true), http.MethodPost, "/planned-meals", []byte(validPlannedMealBody))
		require.Contains(t, w.Body.String(), "only published recipes can be planned")
	})
}

func TestPlannedMealHandler_Get(t *testing.T) {
	t.Run("returns the meal", func(t *testing.T) {
		svc := new(mockPlannedMealService)
		svc.On("GetByID", mock.Anything, handlerUserID, handlerMealID).Return(sampleHandlerMeal(), nil).Once()
		w := performRequest(plannedMealRouter(svc, true), http.MethodGet, "/planned-meals/"+handlerMealID, nil)
		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns 404 for a missing or foreign meal", func(t *testing.T) {
		svc := new(mockPlannedMealService)
		svc.On("GetByID", mock.Anything, handlerUserID, handlerMealID).Return(nil, internalErr.ErrNotFound).Once()
		w := performRequest(plannedMealRouter(svc, true), http.MethodGet, "/planned-meals/"+handlerMealID, nil)
		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("returns 401 without a user", func(t *testing.T) {
		w := performRequest(plannedMealRouter(new(mockPlannedMealService), false), http.MethodGet, "/planned-meals/"+handlerMealID, nil)
		require.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestPlannedMealHandler_Update(t *testing.T) {
	t.Run("returns 200 with the updated meal", func(t *testing.T) {
		svc := new(mockPlannedMealService)
		svc.On("Update", mock.Anything, handlerUserID, handlerMealID, mock.Anything).Return(sampleHandlerMeal(), nil).Once()
		w := performRequest(plannedMealRouter(svc, true), http.MethodPut, "/planned-meals/"+handlerMealID, []byte(validPlannedMealBody))
		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns 400 for a bad body", func(t *testing.T) {
		w := performRequest(plannedMealRouter(new(mockPlannedMealService), true), http.MethodPut, "/planned-meals/"+handlerMealID, []byte(`{`))
		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 404 for a missing meal", func(t *testing.T) {
		svc := new(mockPlannedMealService)
		svc.On("Update", mock.Anything, handlerUserID, handlerMealID, mock.Anything).Return(nil, internalErr.ErrNotFound).Once()
		w := performRequest(plannedMealRouter(svc, true), http.MethodPut, "/planned-meals/"+handlerMealID, []byte(validPlannedMealBody))
		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("returns 401 without a user", func(t *testing.T) {
		w := performRequest(plannedMealRouter(new(mockPlannedMealService), false), http.MethodPut, "/planned-meals/"+handlerMealID, []byte(validPlannedMealBody))
		require.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestPlannedMealHandler_Delete(t *testing.T) {
	t.Run("returns 204 with an empty body", func(t *testing.T) {
		svc := new(mockPlannedMealService)
		svc.On("Delete", mock.Anything, handlerUserID, handlerMealID).Return(nil).Once()
		w := performRequest(plannedMealRouter(svc, true), http.MethodDelete, "/planned-meals/"+handlerMealID, nil)
		require.Equal(t, http.StatusNoContent, w.Code)
		require.Empty(t, w.Body.String())
	})

	t.Run("returns 404 for a missing meal", func(t *testing.T) {
		svc := new(mockPlannedMealService)
		svc.On("Delete", mock.Anything, handlerUserID, handlerMealID).Return(internalErr.ErrNotFound).Once()
		w := performRequest(plannedMealRouter(svc, true), http.MethodDelete, "/planned-meals/"+handlerMealID, nil)
		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("returns 401 without a user", func(t *testing.T) {
		w := performRequest(plannedMealRouter(new(mockPlannedMealService), false), http.MethodDelete, "/planned-meals/"+handlerMealID, nil)
		require.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestPlannedMealHandler_List(t *testing.T) {
	t.Run("defaults to page 1 and 100 items", func(t *testing.T) {
		svc := new(mockPlannedMealService)
		page := &domain.PlannedMealPage{Items: []domain.PlannedMeal{*sampleHandlerMeal()}, Total: 1, Page: 1, PageSize: 100}
		svc.On("List", mock.Anything, handlerUserID, "2026-10-03", "2026-10-04", 1, 100).Return(page, nil).Once()

		w := performRequest(plannedMealRouter(svc, true), http.MethodGet, "/planned-meals?from=2026-10-03&to=2026-10-04", nil)

		require.Equal(t, http.StatusOK, w.Code)
		var out map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
		require.Len(t, out["items"], 1)
		require.Equal(t, float64(100), out["page_size"])
		require.Equal(t, float64(1), out["total"])
	})

	t.Run("passes explicit paging through", func(t *testing.T) {
		svc := new(mockPlannedMealService)
		svc.On("List", mock.Anything, handlerUserID, "2026-10-03", "2026-10-04", 2, 10).Return(&domain.PlannedMealPage{Items: []domain.PlannedMeal{}}, nil).Once()
		w := performRequest(plannedMealRouter(svc, true), http.MethodGet, "/planned-meals?from=2026-10-03&to=2026-10-04&page=2&page_size=10", nil)
		require.Equal(t, http.StatusOK, w.Code)
		svc.AssertExpectations(t)
	})

	for _, path := range []string{
		"/planned-meals?from=2026-10-03&to=2026-10-04&page=abc",
		"/planned-meals?from=2026-10-03&to=2026-10-04&page_size=x",
	} {
		t.Run("returns 400 for "+path, func(t *testing.T) {
			svc := new(mockPlannedMealService)
			w := performRequest(plannedMealRouter(svc, true), http.MethodGet, path, nil)
			require.Equal(t, http.StatusBadRequest, w.Code)
			svc.AssertNotCalled(t, "List", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		})
	}

	t.Run("returns 400 when the service rejects the range", func(t *testing.T) {
		svc := new(mockPlannedMealService)
		svc.On("List", mock.Anything, handlerUserID, "", "", 1, 100).Return(nil, internalErr.New("invalid from date", "INVALID")).Once()
		w := performRequest(plannedMealRouter(svc, true), http.MethodGet, "/planned-meals", nil)
		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 401 without a user", func(t *testing.T) {
		w := performRequest(plannedMealRouter(new(mockPlannedMealService), false), http.MethodGet, "/planned-meals?from=2026-10-03&to=2026-10-04", nil)
		require.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
