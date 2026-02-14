package service

import (
	"context"
	"github.com/H3nSte1n/recipe/internal/domain"
	"github.com/H3nSte1n/recipe/internal/errors"
	"github.com/H3nSte1n/recipe/internal/repository"
	"github.com/H3nSte1n/recipe/pkg/ai"
	"go.uber.org/zap"
	"sort"
)

type ShoppingListService interface {
	Create(ctx context.Context, userID string, req *domain.CreateShoppingListRequest) (*domain.ShoppingList, error)
	Update(ctx context.Context, userID string, listID string, req *domain.UpdateShoppingListRequest) (*domain.ShoppingList, error)
	Delete(ctx context.Context, userID string, listID string) error
	GetByID(ctx context.Context, userID string, listID string) (*domain.ShoppingList, error)
	GetSorted(ctx context.Context, userID string, listID string, sortBy string, sortDirection string) (*domain.ShoppingList, error)
	GetSortedByStoreName(ctx context.Context, userID string, listID string, storeName string, sortDirection string) (*domain.ShoppingList, error)
	ListByUserID(ctx context.Context, userID string) ([]domain.ShoppingList, error)
	AddItem(ctx context.Context, userID string, listID string, req *domain.ShoppingListItemRequest) error
	UpdateItem(ctx context.Context, userID string, itemID string, req *domain.UpdateShoppingListItemRequest) error
	DeleteItem(ctx context.Context, userID string, itemID string) error
	ToggleItem(ctx context.Context, userID string, itemID string, checked bool) error
	AddRecipeToList(ctx context.Context, userID string, listID string, req *domain.AddRecipeToListRequest) error
	GetSortedForStore(ctx context.Context, userID string, listID string, chainID string) (*domain.ShoppingList, error)
}

type shoppingListService struct {
	shoppingListRepo  repository.ShoppingListRepository
	recipeRepo        repository.RecipeRepository
	storeChainService StoreChainService
	aiModel           ai.AIModel
	logger            *zap.Logger
}

func NewShoppingListService(shoppingListRepo repository.ShoppingListRepository, recipeRepo repository.RecipeRepository, storeChainService StoreChainService, aiModel ai.AIModel, logger *zap.Logger) ShoppingListService {
	return &shoppingListService{
		shoppingListRepo:  shoppingListRepo,
		recipeRepo:        recipeRepo,
		storeChainService: storeChainService,
		aiModel:           aiModel,
		logger:            logger,
	}
}

func (s *shoppingListService) Create(ctx context.Context, userID string, req *domain.CreateShoppingListRequest) (*domain.ShoppingList, error) {
	list := &domain.ShoppingList{
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		SortType:    req.SortType,
	}

	if req.StoreChainID != "" {
		list.StoreChainID = &req.StoreChainID
	}

	if err := s.shoppingListRepo.Create(ctx, list); err != nil {
		return nil, err
	}

	// Add initial items if provided
	if len(req.Items) > 0 {
		items := make([]domain.ShoppingListItem, len(req.Items))
		for i, itemReq := range req.Items {
			items[i] = domain.ShoppingListItem{
				ListID:   list.ID,
				Name:     itemReq.Name,
				Amount:   itemReq.Amount,
				Unit:     itemReq.Unit,
				Category: itemReq.Category,
				Notes:    itemReq.Notes,
			}
		}

		if err := s.shoppingListRepo.AddItems(ctx, items); err != nil {
			return nil, err
		}
	}

	// Reload the list with items and store chain
	return s.shoppingListRepo.GetByID(ctx, list.ID)
}

func (s *shoppingListService) Update(ctx context.Context, userID string, listID string, req *domain.UpdateShoppingListRequest) (*domain.ShoppingList, error) {
	list, err := s.shoppingListRepo.GetByID(ctx, listID)
	if err != nil {
		return nil, err
	}
	if list.UserID != userID {
		return nil, errors.ErrUnauthorized
	}

	list.Name = req.Name
	list.Description = req.Description
	list.SortType = req.SortType

	if err := s.shoppingListRepo.Update(ctx, list); err != nil {
		return nil, err
	}

	return list, nil
}

func (s *shoppingListService) Delete(ctx context.Context, userID string, listID string) error {
	list, err := s.shoppingListRepo.GetByID(ctx, listID)
	if err != nil {
		return err
	}
	if list.UserID != userID {
		return errors.ErrUnauthorized
	}

	return s.shoppingListRepo.Delete(ctx, listID)
}

func (s *shoppingListService) GetByID(ctx context.Context, userID string, listID string) (*domain.ShoppingList, error) {
	list, err := s.shoppingListRepo.GetByID(ctx, listID)
	if err != nil {
		return nil, err
	}
	if list.UserID != userID {
		return nil, errors.ErrUnauthorized
	}

	return list, nil
}

func (s *shoppingListService) GetSorted(ctx context.Context, userID string, listID string, sortBy string, sortDirection string) (*domain.ShoppingList, error) {
	list, err := s.shoppingListRepo.GetByID(ctx, listID)
	if err != nil {
		return nil, err
	}
	if list.UserID != userID {
		return nil, errors.ErrUnauthorized
	}

	// Sort items based on sortBy field
	sortItems(list.Items, sortBy, sortDirection)

	return list, nil
}

func (s *shoppingListService) GetSortedByStoreName(ctx context.Context, userID string, listID string, storeName string, sortDirection string) (*domain.ShoppingList, error) {
	list, err := s.shoppingListRepo.GetByID(ctx, listID)
	if err != nil {
		return nil, err
	}
	if list.UserID != userID {
		return nil, errors.ErrUnauthorized
	}

	// Get store chain by name (try without country first)
	chain, err := s.storeChainService.GetChainByName(ctx, storeName, "")
	if err != nil {
		return nil, err
	}

	// Organize items according to store layout
	if err := s.storeChainService.OrganizeShoppingList(ctx, list, chain.ID); err != nil {
		return nil, err
	}

	// If descending, reverse the order
	if sortDirection == "desc" {
		reverseItems(list.Items)
	}

	return list, nil
}

func (s *shoppingListService) ListByUserID(ctx context.Context, userID string) ([]domain.ShoppingList, error) {
	return s.shoppingListRepo.ListByUserID(ctx, userID)
}

func (s *shoppingListService) AddItem(ctx context.Context, userID string, listID string, req *domain.ShoppingListItemRequest) error {
	// Get the list to verify ownership
	list, err := s.shoppingListRepo.GetByID(ctx, listID)
	if err != nil {
		return err
	}
	if list.UserID != userID {
		return errors.ErrUnauthorized
	}

	// Classify the item
	category := domain.CategoryOther
	categories, err := s.aiModel.CategorizeItems(ctx, []string{req.Name})
	if err != nil {
		s.logger.Warn("failed to classify item", zap.Error(err))
	} else {
		category = domain.Category(categories[0])
	}

	// Create the item
	item := &domain.ShoppingListItem{
		ListID:   listID,
		Name:     req.Name,
		Amount:   req.Amount,
		Unit:     req.Unit,
		Category: category,
		Notes:    req.Notes,
	}

	return s.shoppingListRepo.AddItems(ctx, []domain.ShoppingListItem{*item})
}

func (s *shoppingListService) UpdateItem(ctx context.Context, userID string, itemID string, req *domain.UpdateShoppingListItemRequest) error {
	item, err := s.shoppingListRepo.GetItemByID(ctx, itemID)
	if err != nil {
		return err
	}

	// Verify ownership
	list, err := s.shoppingListRepo.GetByID(ctx, item.ListID)
	if err != nil {
		return err
	}
	if list.UserID != userID {
		return errors.ErrUnauthorized
	}

	item.Name = req.Name
	item.Amount = req.Amount
	item.Unit = req.Unit
	item.Category = req.Category
	item.Notes = req.Notes

	return s.shoppingListRepo.UpdateItem(ctx, item)
}

func (s *shoppingListService) DeleteItem(ctx context.Context, userID string, itemID string) error {
	item, err := s.shoppingListRepo.GetItemByID(ctx, itemID)
	if err != nil {
		return err
	}

	// Verify ownership
	list, err := s.shoppingListRepo.GetByID(ctx, item.ListID)
	if err != nil {
		return err
	}
	if list.UserID != userID {
		return errors.ErrUnauthorized
	}

	return s.shoppingListRepo.DeleteItem(ctx, itemID)
}

func (s *shoppingListService) ToggleItem(ctx context.Context, userID string, itemID string, checked bool) error {
	item, err := s.shoppingListRepo.GetItemByID(ctx, itemID)
	if err != nil {
		return err
	}

	// Verify ownership
	list, err := s.shoppingListRepo.GetByID(ctx, item.ListID)
	if err != nil {
		return err
	}
	if list.UserID != userID {
		return errors.ErrUnauthorized
	}

	item.IsChecked = checked
	return s.shoppingListRepo.UpdateItem(ctx, item)
}

func (s *shoppingListService) AddRecipeToList(ctx context.Context, userID string, listID string, req *domain.AddRecipeToListRequest) error {
	// Get the list to verify ownership
	list, err := s.shoppingListRepo.GetByID(ctx, listID)
	if err != nil {
		return err
	}
	if list.UserID != userID {
		return errors.ErrUnauthorized
	}

	// Get the recipe with base nutrition level (we only need ingredients)
	recipe, err := s.recipeRepo.GetByID(ctx, req.RecipeID, domain.NutritionDetailBase)
	if err != nil {
		return err
	}

	// Calculate scaling factor
	scalingFactor := req.Servings / float64(recipe.Servings)

	// Prepare item names for categorization
	itemNames := make([]string, len(recipe.Ingredients))
	for i, ingredient := range recipe.Ingredients {
		itemNames[i] = ingredient.Description
	}

	// Categorize all items at once
	categories, err := s.aiModel.CategorizeItems(ctx, itemNames)
	if err != nil {
		s.logger.Warn("failed to classify items", zap.Error(err))
		// Fill with default category if categorization fails
		categories = make([]string, len(itemNames))
		for i := range categories {
			categories[i] = string(domain.CategoryOther)
		}
	}

	// Create shopping list items from recipe ingredients
	items := make([]domain.ShoppingListItem, len(recipe.Ingredients))
	for i, ingredient := range recipe.Ingredients {
		category := domain.CategoryOther
		if i < len(categories) {
			category = domain.Category(categories[i])
		}

		items[i] = domain.ShoppingListItem{
			ListID:   listID,
			RecipeID: &recipe.ID,
			Name:     ingredient.Description,
			Amount:   ingredient.Amount * scalingFactor,
			Unit:     ingredient.Unit,
			Category: category,
			Notes:    "",
		}
	}

	return s.shoppingListRepo.AddItems(ctx, items)
}

func (s *shoppingListService) GetSortedForStore(ctx context.Context, userID string, listID string, chainID string) (*domain.ShoppingList, error) {
	// Get the shopping list and verify ownership
	list, err := s.shoppingListRepo.GetByID(ctx, listID)
	if err != nil {
		return nil, err
	}
	if list.UserID != userID {
		return nil, errors.ErrUnauthorized
	}

	// Organize items according to store layout (sorts in-memory, doesn't persist)
	if err := s.storeChainService.OrganizeShoppingList(ctx, list, chainID); err != nil {
		return nil, err
	}

	return list, nil
}

func (s *shoppingListService) SortForStore(ctx context.Context, listID string, chainID string) error {
	// Get the shopping list
	list, err := s.shoppingListRepo.GetByID(ctx, listID)
	if err != nil {
		return err
	}

	// Organize items according to store layout
	if err := s.storeChainService.OrganizeShoppingList(ctx, list, chainID); err != nil {
		return err
	}

	// Save the organized list
	return s.shoppingListRepo.Update(ctx, list)
}

// sortItems sorts shopping list items based on the specified field and direction
func sortItems(items []domain.ShoppingListItem, sortBy string, sortDirection string) {
	if len(items) == 0 {
		return
	}

	sort.SliceStable(items, func(i, j int) bool {
		var less bool

		switch sortBy {
		case "name":
			less = items[i].Name < items[j].Name
		case "category":
			less = items[i].Category < items[j].Category
		case "amount":
			less = items[i].Amount < items[j].Amount
		case "checked":
			less = !items[i].IsChecked && items[j].IsChecked
		case "created_at":
			less = items[i].CreatedAt.Before(items[j].CreatedAt)
		default:
			// Default to name sorting
			less = items[i].Name < items[j].Name
		}

		// Reverse if direction is desc
		if sortDirection == "desc" {
			return !less
		}
		return less
	})
}

// reverseItems reverses the order of items (for descending store order)
func reverseItems(items []domain.ShoppingListItem) {
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
}
