package repository

import (
	"context"
	"github.com/yourusername/recipe-app/internal/domain"
	"gorm.io/gorm"
)

type ShoppingListRepository interface {
	Repository[domain.User]
	GetByID(ctx context.Context, listID string) (*domain.ShoppingList, error)
	GetItemByID(ctx context.Context, itemID string) (*domain.ShoppingListItem, error)
	Create(ctx context.Context, list *domain.ShoppingList) error
	Update(ctx context.Context, list *domain.ShoppingList) error
	Delete(ctx context.Context, id string) error
	ListByUserID(ctx context.Context, userID string) ([]domain.ShoppingList, error)
	AddItems(ctx context.Context, items []domain.ShoppingListItem) error
	UpdateItem(ctx context.Context, item *domain.ShoppingListItem) error
	DeleteItem(ctx context.Context, id string) error
}

type ShoppingListRepositoryImpl struct {
	*BaseRepository[domain.User]
}

func NewShoppingListRepository(db *gorm.DB) ShoppingListRepository {
	return &ShoppingListRepositoryImpl{
		BaseRepository: NewBaseRepository[domain.User](db),
	}
}

func (r *ShoppingListRepositoryImpl) GetByID(ctx context.Context, listID string) (*domain.ShoppingList, error) {
	var list domain.ShoppingList
	if err := r.db.WithContext(ctx).
		Preload("Items").
		Preload("StoreChain").
		First(&list, "id = ?", listID).Error; err != nil {
		return nil, err
	}
	return &list, nil
}

func (r *ShoppingListRepositoryImpl) GetItemByID(ctx context.Context, itemID string) (*domain.ShoppingListItem, error) {
	var item domain.ShoppingListItem
	if err := r.db.WithContext(ctx).First(&item, "id = ?", itemID).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ShoppingListRepositoryImpl) Create(ctx context.Context, list *domain.ShoppingList) error {
	return r.db.WithContext(ctx).Create(list).Error
}

func (r *ShoppingListRepositoryImpl) Update(ctx context.Context, list *domain.ShoppingList) error {
	return r.db.WithContext(ctx).Save(list).Error
}

func (r *ShoppingListRepositoryImpl) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.ShoppingList{}).Error
}

func (r *ShoppingListRepositoryImpl) ListByUserID(ctx context.Context, userID string) ([]domain.ShoppingList, error) {
	var lists []domain.ShoppingList
	if err := r.db.WithContext(ctx).
		Preload("Items").
		Preload("StoreChain").
		Where("user_id = ?", userID).
		Find(&lists).Error; err != nil {
		return nil, err
	}
	return lists, nil
}

func (r *ShoppingListRepositoryImpl) AddItems(ctx context.Context, items []domain.ShoppingListItem) error {
	return r.db.WithContext(ctx).Create(items).Error
}

func (r *ShoppingListRepositoryImpl) UpdateItem(ctx context.Context, item *domain.ShoppingListItem) error {
	return r.db.WithContext(ctx).Save(item).Error
}

func (r *ShoppingListRepositoryImpl) DeleteItem(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.ShoppingListItem{}).Error
}
