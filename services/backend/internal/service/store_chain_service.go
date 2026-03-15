package service

import (
	"context"
	"github.com/H3nSte1n/recipe/internal/domain"
	"github.com/H3nSte1n/recipe/internal/repository"
	"go.uber.org/zap"
	"sort"
)

type StoreChainService interface {
	GetChain(ctx context.Context, chainID string) (*domain.StoreChain, error)
	GetChainByName(ctx context.Context, name string, country string) (*domain.StoreChain, error)
	ListChains(ctx context.Context, country string) ([]domain.StoreChain, error)
	OrganizeShoppingList(ctx context.Context, list *domain.ShoppingList, chainID string) error
}

type storeChainService struct {
	storeChainRepo repository.StoreChainRepository
	logger         *zap.Logger
}

func NewStoreChainService(storeChainRepo repository.StoreChainRepository, logger *zap.Logger) StoreChainService {
	return &storeChainService{
		storeChainRepo: storeChainRepo,
		logger:         logger,
	}
}

func (s *storeChainService) GetChain(ctx context.Context, chainID string) (*domain.StoreChain, error) {
	return s.storeChainRepo.GetChain(ctx, chainID)
}

func (s *storeChainService) GetChainByName(ctx context.Context, name string, country string) (*domain.StoreChain, error) {
	return s.storeChainRepo.GetChainByName(ctx, name, country)
}

func (s *storeChainService) ListChains(ctx context.Context, country string) ([]domain.StoreChain, error) {
	return s.storeChainRepo.ListChains(ctx, country)
}

func (s *storeChainService) OrganizeShoppingList(ctx context.Context, list *domain.ShoppingList, chainID string) error {
	// Get store layout
	chain, err := s.storeChainRepo.GetChain(ctx, chainID)
	if err != nil {
		return err
	}

	// Create a map for quick category lookup
	sectionOrder := make(map[domain.Category]int)
	for _, section := range chain.Layout {
		for _, category := range section.Categories {
			sectionOrder[category] = section.Order
		}
	}

	// Sort items based on section order
	sort.SliceStable(list.Items, func(i, j int) bool {
		orderI := sectionOrder[list.Items[i].Category]
		orderJ := sectionOrder[list.Items[j].Category]

		// If items are in different sections, sort by section order
		if orderI != orderJ {
			return orderI < orderJ
		}

		// If in same section, keep original order
		return i < j
	})

	return nil
}
