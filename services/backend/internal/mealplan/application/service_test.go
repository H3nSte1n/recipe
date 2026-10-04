package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/H3nSte1n/recipe/internal/mealplan/domain"
	"github.com/google/uuid"
)

type memoryRepo struct {
	recipes map[uuid.UUID]uuid.UUID
	meals   map[uuid.UUID]domain.PlannedMeal
	err     error
}

func (r *memoryRepo) CreateWithOwnedRecipe(_ context.Context, user uuid.UUID, f domain.Fields) (domain.PlannedMeal, error) {
	if r.err != nil {
		return domain.PlannedMeal{}, r.err
	}
	if r.recipes[f.RecipeID] != user {
		return domain.PlannedMeal{}, ErrNotFound
	}
	m := domain.PlannedMeal{ID: uuid.New(), UserID: user, RecipeID: f.RecipeID, CookDate: f.CookDate, MealType: f.MealType, Servings: f.Servings, Notes: f.Notes, Recipe: domain.RecipeSummary{ID: f.RecipeID, Title: "Pasta", Status: "published"}, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	r.meals[m.ID] = m
	return m, nil
}
func (r *memoryRepo) ReplaceWithOwnedRecipe(ctx context.Context, user, id uuid.UUID, f domain.Fields) (domain.PlannedMeal, error) {
	if r.err != nil {
		return domain.PlannedMeal{}, r.err
	}
	if m, ok := r.meals[id]; !ok || m.UserID != user {
		return domain.PlannedMeal{}, ErrNotFound
	}
	m, err := r.CreateWithOwnedRecipe(ctx, user, f)
	if err != nil {
		return domain.PlannedMeal{}, err
	}
	delete(r.meals, m.ID)
	m.ID = id
	r.meals[id] = m
	return m, nil
}
func (r *memoryRepo) Get(_ context.Context, user, id uuid.UUID) (domain.PlannedMeal, error) {
	if r.err != nil {
		return domain.PlannedMeal{}, r.err
	}
	m, ok := r.meals[id]
	if !ok || m.UserID != user {
		return domain.PlannedMeal{}, ErrNotFound
	}
	return m, nil
}
func (r *memoryRepo) List(_ context.Context, user uuid.UUID, from, to domain.Date, page, size int) ([]domain.PlannedMeal, int64, error) {
	if r.err != nil {
		return nil, 0, r.err
	}
	out := []domain.PlannedMeal{}
	for _, m := range r.meals {
		if m.UserID == user && m.CookDate >= from && m.CookDate <= to {
			out = append(out, m)
		}
	}
	return out, int64(len(out)), nil
}
func (r *memoryRepo) Delete(_ context.Context, user, id uuid.UUID) error {
	if r.err != nil {
		return r.err
	}
	m, ok := r.meals[id]
	if !ok || m.UserID != user {
		return ErrNotFound
	}
	delete(r.meals, id)
	return nil
}

func TestServiceOwnershipAndCRUD(t *testing.T) {
	user, other, recipe, otherRecipe := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	r := &memoryRepo{recipes: map[uuid.UUID]uuid.UUID{recipe: user, otherRecipe: other}, meals: map[uuid.UUID]domain.PlannedMeal{}}
	s := NewService(r)
	ctx := context.Background()
	f := domain.Fields{RecipeID: recipe, CookDate: domain.Date("2026-10-03")}
	if _, err := s.Create(ctx, user.String(), domain.Fields{RecipeID: otherRecipe, CookDate: f.CookDate}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-owner recipe: %v", err)
	}
	a, err := s.Create(ctx, user.String(), f)
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Create(ctx, user.String(), f)
	if err != nil {
		t.Fatal(err)
	}
	if a.ID == b.ID {
		t.Fatal("duplicate dates should create distinct entries")
	}
	p, err := s.List(ctx, user.String(), f.CookDate, f.CookDate, 1, 100)
	if err != nil || p.Total != 2 {
		t.Fatalf("range: %+v %v", p, err)
	}
	if _, err := s.Get(ctx, other.String(), a.ID.String()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-owner read: %v", err)
	}
	if err := s.Delete(ctx, other.String(), a.ID.String()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-owner delete: %v", err)
	}
	f.RecipeID = otherRecipe
	if _, err := s.Replace(ctx, user.String(), a.ID.String(), f); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-owner recipe reassignment: %v", err)
	}
	f.RecipeID = recipe
	f.CookDate = "2026-10-04"
	if _, err := s.Replace(ctx, user.String(), a.ID.String(), f); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, user.String(), a.ID.String()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(ctx, user.String(), a.ID.String()); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
}

func TestServiceValidationAndErrors(t *testing.T) {
	user, recipe := uuid.New(), uuid.New()
	r := &memoryRepo{recipes: map[uuid.UUID]uuid.UUID{recipe: user}, meals: map[uuid.UUID]domain.PlannedMeal{}}
	s := NewService(r)
	ctx := context.Background()
	f := domain.Fields{RecipeID: recipe, CookDate: "2026-10-03"}
	if _, err := s.Create(ctx, "bad", f); !errors.Is(err, ErrUnauthorized) {
		t.Fatal(err)
	}
	if _, err := s.Get(ctx, user.String(), "bad"); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
	if _, err := s.List(ctx, user.String(), "2026-01-01", "2026-05-01", 1, 100); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
	if _, err := s.List(ctx, user.String(), "2026-10-03", "2026-10-03", 0, 100); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
	notes := ""
	f.Notes = &notes
	m, err := s.Create(ctx, user.String(), f)
	if err != nil || m.Notes != nil {
		t.Fatalf("empty note null: %+v %v", m, err)
	}
	r.err = errors.New("storage unavailable")
	if _, err := s.Get(ctx, user.String(), m.ID.String()); err == nil || errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
}
