package repository

import (
	"context"
	"github.com/H3nSte1n/recipe/internal/domain"
	"gorm.io/gorm"
)

type StoreChainRepository interface {
	Repository[domain.User]
	GetChain(ctx context.Context, chainID string) (*domain.StoreChain, error)
	GetChainByName(ctx context.Context, name string, country string) (*domain.StoreChain, error)
	ListChains(ctx context.Context, country string) ([]domain.StoreChain, error)
}

type StoreChainRepositoryImpl struct {
	*BaseRepository[domain.User]
}

func NewStoreChainRepository(db *gorm.DB) StoreChainRepository {
	return &StoreChainRepositoryImpl{
		BaseRepository: NewBaseRepository[domain.User](db),
	}
}

func (r *StoreChainRepositoryImpl) GetChain(ctx context.Context, chainID string) (*domain.StoreChain, error) {
	var chain domain.StoreChain
	if err := r.db.WithContext(ctx).First(&chain, "id = ?", chainID).Error; err != nil {
		return nil, err
	}
	return &chain, nil
}

func (r *StoreChainRepositoryImpl) GetChainByName(ctx context.Context, name string, country string) (*domain.StoreChain, error) {
	var chain domain.StoreChain
	query := r.db.WithContext(ctx).Where("LOWER(name) = LOWER(?)", name)

	if country != "" {
		query = query.Where("country = ?", country)
	}

	if err := query.First(&chain).Error; err != nil {
		return nil, err
	}
	return &chain, nil
}

func (r *StoreChainRepositoryImpl) ListChains(ctx context.Context, country string) ([]domain.StoreChain, error) {
	var chains []domain.StoreChain
	query := r.db.WithContext(ctx)

	if country != "" {
		query = query.Where("country = ?", country)
	}

	if err := query.Find(&chains).Error; err != nil {
		return nil, err
	}
	return chains, nil
}
