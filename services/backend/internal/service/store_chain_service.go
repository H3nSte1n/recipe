package service

import (
	"context"
	"github.com/H3nSte1n/recipe/internal/domain"
	"go.uber.org/zap"
	"math"
	"sort"
)

type storeChainRepository interface {
	GetChain(ctx context.Context, chainID string) (*domain.StoreChain, error)
	GetChainByName(ctx context.Context, name string, country string) (*domain.StoreChain, error)
	ListChains(ctx context.Context, country string) ([]domain.StoreChain, error)
}

type StoreChainService interface {
	GetChain(ctx context.Context, chainID string) (*domain.StoreChain, error)
	GetChainByName(ctx context.Context, name string, country string) (*domain.StoreChain, error)
	ListChains(ctx context.Context, country string) ([]domain.StoreChain, error)
	OrganizeShoppingList(ctx context.Context, list *domain.ShoppingList, chainID string) error
}

type storeChainService struct {
	storeChainRepo storeChainRepository
	logger         *zap.Logger
}

func NewStoreChainService(storeChainRepo storeChainRepository, logger *zap.Logger) StoreChainService {
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
	chain, err := s.storeChainRepo.GetChain(ctx, chainID)
	if err != nil {
		return err
	}

	organizeByChain(list, chain)
	return nil
}

// organizeByChain sorts list.Items in-place according to the store chain's section layout.
// Items whose category is not found in the layout are moved to the end.
func organizeByChain(list *domain.ShoppingList, chain *domain.StoreChain) {
	sectionOrder := make(map[domain.Category]int)
	for _, section := range chain.Layout {
		for _, category := range section.Categories {
			sectionOrder[category] = section.Order
		}
	}

	sort.SliceStable(list.Items, func(i, j int) bool {
		orderI, okI := sectionOrder[list.Items[i].Category]
		orderJ, okJ := sectionOrder[list.Items[j].Category]

		if !okI {
			orderI = math.MaxInt
		}
		if !okJ {
			orderJ = math.MaxInt
		}

		if orderI != orderJ {
			return orderI < orderJ
		}

		return i < j
	})
}
