package repository

import (
	"context"
	"github.com/H3nSte1n/recipe/internal/domain"
	"gorm.io/gorm"
)

type StoreChainRepository interface {
	GetChain(ctx context.Context, chainID string) (*domain.StoreChain, error)
	GetChainByName(ctx context.Context, name string, country string) (*domain.StoreChain, error)
	ListChains(ctx context.Context, country string) ([]domain.StoreChain, error)
}

type StoreChainRepositoryImpl struct {
	*BaseRepository
}

func NewStoreChainRepository(db *gorm.DB) StoreChainRepository {
	return &StoreChainRepositoryImpl{
		BaseRepository: NewBaseRepository(db),
	}
}

func (r *StoreChainRepositoryImpl) GetChain(ctx context.Context, chainID string) (*domain.StoreChain, error) {
	var chain domain.StoreChain
	if err := r.DB.WithContext(ctx).First(&chain, "id = ?", chainID).Error; err != nil {
		return nil, err
	}
	return &chain, nil
}

func (r *StoreChainRepositoryImpl) GetChainByName(ctx context.Context, name string, country string) (*domain.StoreChain, error) {
	var chain domain.StoreChain
	query := r.DB.WithContext(ctx).Where("LOWER(name) = LOWER(?)", name)

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
	query := r.DB.WithContext(ctx)

	if country != "" {
		query = query.Where("country = ?", country)
	}

	if err := query.Find(&chains).Error; err != nil {
		return nil, err
	}
	return chains, nil
}
