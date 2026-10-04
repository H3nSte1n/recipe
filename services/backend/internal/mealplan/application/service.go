package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/H3nSte1n/recipe/internal/mealplan/domain"
	"github.com/google/uuid"
)

var (
	ErrInvalid      = errors.New("invalid input")
	ErrNotFound     = errors.New("not found")
	ErrUnauthorized = errors.New("unauthorized")
)

type Page struct {
	Items    []domain.PlannedMeal
	Total    int64
	Page     int
	PageSize int
}

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func userID(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, ErrUnauthorized
	}
	return id, nil
}

func planID(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, fmt.Errorf("%w: invalid id", ErrInvalid)
	}
	return id, nil
}

func (s *Service) Create(ctx context.Context, rawUser string, fields domain.Fields) (domain.PlannedMeal, error) {
	u, err := userID(rawUser)
	if err != nil {
		return domain.PlannedMeal{}, err
	}
	fields = normalize(fields)
	if err := fields.Validate(); err != nil {
		return domain.PlannedMeal{}, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	return s.repo.CreateWithOwnedRecipe(ctx, u, fields)
}

func (s *Service) Replace(ctx context.Context, rawUser, rawID string, fields domain.Fields) (domain.PlannedMeal, error) {
	u, err := userID(rawUser)
	if err != nil {
		return domain.PlannedMeal{}, err
	}
	id, err := planID(rawID)
	if err != nil {
		return domain.PlannedMeal{}, err
	}
	fields = normalize(fields)
	if err := fields.Validate(); err != nil {
		return domain.PlannedMeal{}, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	return s.repo.ReplaceWithOwnedRecipe(ctx, u, id, fields)
}

func normalize(f domain.Fields) domain.Fields {
	if f.MealType != nil && *f.MealType == "" {
		f.MealType = nil
	}
	if f.Notes != nil && *f.Notes == "" {
		f.Notes = nil
	}
	return f
}

func (s *Service) Get(ctx context.Context, rawUser, rawID string) (domain.PlannedMeal, error) {
	u, err := userID(rawUser)
	if err != nil {
		return domain.PlannedMeal{}, err
	}
	id, err := planID(rawID)
	if err != nil {
		return domain.PlannedMeal{}, err
	}
	return s.repo.Get(ctx, u, id)
}

func (s *Service) Delete(ctx context.Context, rawUser, rawID string) error {
	u, err := userID(rawUser)
	if err != nil {
		return err
	}
	id, err := planID(rawID)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, u, id)
}

func (s *Service) List(ctx context.Context, rawUser string, from, to domain.Date, page, pageSize int) (Page, error) {
	u, err := userID(rawUser)
	if err != nil {
		return Page{}, err
	}
	if err := domain.ValidateRange(from, to); err != nil {
		return Page{}, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	if page < 1 || pageSize < 1 || pageSize > 100 || int64(page-1)*int64(pageSize) > 2147483647 {
		return Page{}, fmt.Errorf("%w: invalid page", ErrInvalid)
	}
	items, total, err := s.repo.List(ctx, u, from, to, page, pageSize)
	if err != nil {
		return Page{}, err
	}
	if items == nil {
		items = []domain.PlannedMeal{}
	}
	return Page{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}
