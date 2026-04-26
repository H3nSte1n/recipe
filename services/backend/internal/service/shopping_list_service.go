package service

import (
	"context"
	"github.com/H3nSte1n/recipe/internal/domain"
	"github.com/H3nSte1n/recipe/internal/errors"
	"github.com/H3nSte1n/recipe/pkg/ai"
	"go.uber.org/zap"
	"sort"
)

type shoppingListRepository interface {
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

type shoppingListRecipeRepository interface {
	GetByID(ctx context.Context, id string, nutritionLevel domain.NutritionDetailLevel) (*domain.Recipe, error)
}

type ShoppingListService interface {
	Create(ctx context.Context, userID string, req *domain.CreateShoppingListRequest) (*domain.ShoppingList, error)
	Update(ctx context.Context, userID string, listID string, req *domain.UpdateShoppingListRequest) (*domain.ShoppingList, error)
	Delete(ctx context.Context, userID string, listID string) error
	GetByID(ctx context.Context, userID string, listID string) (*domain.ShoppingList, error)
	GetSorted(ctx context.Context, userID string, listID string, sortBy string, sortDirection string) (*domain.ShoppingList, error)
	GetSortedByStoreName(ctx context.Context, userID string, listID string, storeName string, country string, sortDirection string) (*domain.ShoppingList, error)
	ListByUserID(ctx context.Context, userID string) ([]domain.ShoppingList, error)
	AddItem(ctx context.Context, userID string, listID string, req *domain.ShoppingListItemRequest) error
	UpdateItem(ctx context.Context, userID string, itemID string, req *domain.UpdateShoppingListItemRequest) error
	DeleteItem(ctx context.Context, userID string, itemID string) error
	ToggleItem(ctx context.Context, userID string, itemID string, checked bool) error
	AddRecipeToList(ctx context.Context, userID string, listID string, req *domain.AddRecipeToListRequest) error
	GetSortedForStore(ctx context.Context, userID string, listID string, chainID string) (*domain.ShoppingList, error)
}

type shoppingListService struct {
	shoppingListRepo  shoppingListRepository
	recipeRepo        shoppingListRecipeRepository
	storeChainService StoreChainService
	aiModel           ai.AIModel
	logger            *zap.Logger
}

func NewShoppingListService(shoppingListRepo shoppingListRepository, recipeRepo shoppingListRecipeRepository, storeChainService StoreChainService, aiModel ai.AIModel, logger *zap.Logger) ShoppingListService {
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

	if list.ID == "" {
		return nil, errors.New("failed to retrieve generated list ID after creation", "INTERNAL")
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

	return s.shoppingListRepo.GetByID(ctx, list.ID)
}

func (s *shoppingListService) verifyListOwnership(ctx context.Context, userID string, listID string) (*domain.ShoppingList, error) {
	list, err := s.shoppingListRepo.GetByID(ctx, listID)
	if err != nil {
		return nil, err
	}
	if list.UserID != userID {
		return nil, errors.ErrUnauthorized
	}
	return list, nil
}

func (s *shoppingListService) Update(ctx context.Context, userID string, listID string, req *domain.UpdateShoppingListRequest) (*domain.ShoppingList, error) {
	list, err := s.verifyListOwnership(ctx, userID, listID)
	if err != nil {
		return nil, err
	}

	list.Name = req.Name
	list.Description = req.Description
	list.SortType = req.SortType

	if err := s.shoppingListRepo.Update(ctx, list); err != nil {
		return nil, err
	}

	// Reload from DB to reflect any fields set on save (e.g. UpdatedAt)
	return s.shoppingListRepo.GetByID(ctx, listID)
}

func (s *shoppingListService) Delete(ctx context.Context, userID string, listID string) error {
	if _, err := s.verifyListOwnership(ctx, userID, listID); err != nil {
		return err
	}
	return s.shoppingListRepo.Delete(ctx, listID)
}

func (s *shoppingListService) GetByID(ctx context.Context, userID string, listID string) (*domain.ShoppingList, error) {
	return s.verifyListOwnership(ctx, userID, listID)
}

func (s *shoppingListService) GetSorted(ctx context.Context, userID string, listID string, sortBy string, sortDirection string) (*domain.ShoppingList, error) {
	list, err := s.verifyListOwnership(ctx, userID, listID)
	if err != nil {
		return nil, err
	}

	// Warn on unknown sortBy so callers can detect misconfiguration
	validSortFields := map[string]bool{"name": true, "category": true, "amount": true, "checked": true, "created_at": true}
	if !validSortFields[sortBy] {
		s.logger.Warn("unknown sortBy field, defaulting to name", zap.String("sortBy", sortBy))
	}

	sortItems(list.Items, sortBy, sortDirection)

	return list, nil
}

func (s *shoppingListService) GetSortedByStoreName(ctx context.Context, userID string, listID string, storeName string, country string, sortDirection string) (*domain.ShoppingList, error) {
	list, err := s.verifyListOwnership(ctx, userID, listID)
	if err != nil {
		return nil, err
	}

	chain, err := s.storeChainService.GetChainByName(ctx, storeName, country)
	if err != nil {
		return nil, err
	}

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
	if _, err := s.verifyListOwnership(ctx, userID, listID); err != nil {
		return err
	}

	// Classify the item
	category := domain.CategoryOther
	if s.aiModel != nil {
		categories, err := s.aiModel.CategorizeItems(ctx, []string{req.Name})
		if err != nil {
			s.logger.Warn("failed to classify item", zap.Error(err))
		} else if cat, ok := categories[req.Name]; ok {
			category = domain.Category(cat)
		}
	}

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

func (s *shoppingListService) verifyItemOwnership(ctx context.Context, userID string, itemID string) (*domain.ShoppingListItem, error) {
	item, err := s.shoppingListRepo.GetItemByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	list, err := s.shoppingListRepo.GetByID(ctx, item.ListID)
	if err != nil {
		return nil, err
	}
	if list.UserID != userID {
		return nil, errors.ErrUnauthorized
	}
	return item, nil
}

func (s *shoppingListService) UpdateItem(ctx context.Context, userID string, itemID string, req *domain.UpdateShoppingListItemRequest) error {
	item, err := s.verifyItemOwnership(ctx, userID, itemID)
	if err != nil {
		return err
	}

	item.Name = req.Name
	item.Amount = req.Amount
	item.Unit = req.Unit
	item.Category = req.Category
	item.Notes = req.Notes

	return s.shoppingListRepo.UpdateItem(ctx, item)
}

func (s *shoppingListService) DeleteItem(ctx context.Context, userID string, itemID string) error {
	item, err := s.verifyItemOwnership(ctx, userID, itemID)
	if err != nil {
		return err
	}
	return s.shoppingListRepo.DeleteItem(ctx, item.ID)
}

func (s *shoppingListService) ToggleItem(ctx context.Context, userID string, itemID string, checked bool) error {
	item, err := s.verifyItemOwnership(ctx, userID, itemID)
	if err != nil {
		return err
	}
	item.IsChecked = checked
	return s.shoppingListRepo.UpdateItem(ctx, item)
}

func (s *shoppingListService) AddRecipeToList(ctx context.Context, userID string, listID string, req *domain.AddRecipeToListRequest) error {
	if _, err := s.verifyListOwnership(ctx, userID, listID); err != nil {
		return err
	}

	// Get the recipe with base nutrition level (we only need ingredients)
	recipe, err := s.recipeRepo.GetByID(ctx, req.RecipeID, domain.NutritionDetailBase)
	if err != nil {
		return err
	}

	// Guard against divide by zero
	if recipe.Servings == 0 {
		return errors.New("recipe has no servings defined", "INVALID_INPUT")
	}

	// Short-circuit if recipe has no ingredients — avoids a needless AI call
	if len(recipe.Ingredients) == 0 {
		return nil
	}

	// Calculate scaling factor
	scalingFactor := req.Servings / float64(recipe.Servings)

	// Prepare item names for categorization
	itemNames := make([]string, len(recipe.Ingredients))
	for i, ingredient := range recipe.Ingredients {
		itemNames[i] = ingredient.Name
	}

	// Categorize all items at once
	categories := make(map[string]string)
	if s.aiModel != nil {
		if cats, err := s.aiModel.CategorizeItems(ctx, itemNames); err != nil {
			s.logger.Warn("failed to classify items", zap.Error(err))
		} else {
			categories = cats
		}
	}

	// Create shopping list items from recipe ingredients
	items := make([]domain.ShoppingListItem, len(recipe.Ingredients))
	for i, ingredient := range recipe.Ingredients {
		category := domain.CategoryOther
		if cat, ok := categories[ingredient.Name]; ok {
			category = domain.Category(cat)
		}

		items[i] = domain.ShoppingListItem{
			ListID:   listID,
			RecipeID: &recipe.ID,
			Name:     ingredient.Name,
			Amount:   ingredient.Amount * scalingFactor,
			Unit:     ingredient.Unit,
			Category: category,
			// Recipe ingredients don't carry free-text notes — set manually by the user after adding
			Notes: "",
		}
	}

	return s.shoppingListRepo.AddItems(ctx, items)
}

func (s *shoppingListService) GetSortedForStore(ctx context.Context, userID string, listID string, chainID string) (*domain.ShoppingList, error) {
	list, err := s.verifyListOwnership(ctx, userID, listID)
	if err != nil {
		return nil, err
	}

	// Organize items according to store layout (sorts in-memory, doesn't persist)
	if err := s.storeChainService.OrganizeShoppingList(ctx, list, chainID); err != nil {
		return nil, err
	}

	return list, nil
}

// sortItems sorts shopping list items based on the specified field and direction.
// Unknown sortBy values fall back to name sort — callers should validate before invoking.
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
			less = items[i].Name < items[j].Name
		}

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
