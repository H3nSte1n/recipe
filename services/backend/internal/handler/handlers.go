package handler

import (
	"github.com/yourusername/recipe-app/internal/service"
	"go.uber.org/zap"
)

type Handlers struct {
	UserHandler         *UserHandler
	ProfileHandler      *ProfileHandler
	AIConfigHandler     *AIConfigHandler
	RecipeHandler       *RecipeHandler
	ShoppingListHandler *ShoppingListHandler
	StoreChainHandler   *StoreChainHandler
}

func NewHandlers(services *service.Services, logger *zap.Logger) *Handlers {
	return &Handlers{
		UserHandler:         NewUserHandler(services.UserService),
		ProfileHandler:      NewProfileHandler(services.ProfileService),
		AIConfigHandler:     NewAIConfigHandler(services.AIConfigService),
		RecipeHandler:       NewRecipeHandler(services.RecipeService, logger),
		ShoppingListHandler: NewShoppingListHandler(services.ShoppingListService, logger),
		StoreChainHandler:   NewStoreChainHandler(services.StoreChainService, logger),
	}
}
