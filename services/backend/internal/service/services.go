package service

import (
	"github.com/H3nSte1n/recipe/internal/repository"
	"github.com/H3nSte1n/recipe/pkg/ai"
	"github.com/H3nSte1n/recipe/pkg/config"
	"github.com/H3nSte1n/recipe/pkg/email"
	"github.com/H3nSte1n/recipe/pkg/pdfparser"
	"github.com/H3nSte1n/recipe/pkg/storage"
	"github.com/H3nSte1n/recipe/pkg/urlparser"
	"go.uber.org/zap"
)

type Services struct {
	UserService         UserService
	ProfileService      ProfileService
	AIConfigService     AIConfigService
	RecipeService       RecipeService
	ShoppingListService ShoppingListService
	StoreChainService   StoreChainService
}

func NewServices(repos *repository.Repositories, config config.Config, fileStorage storage.FileStore, logger *zap.Logger, factory ai.ModelFactory) *Services {
	pdfParserService := pdfparser.NewService(logger)
	urlParserService := urlparser.NewService(logger)

	// Create AI model for shopping list service
	aiModel, err := factory.CreateModel(ai.ModelDefault, "")
	if err != nil {
		logger.Warn("failed to create AI model for shopping list service", zap.Error(err))
	}

	// Initialize store chain service first since shopping list service depends on it
	storeChainService := NewStoreChainService(repos.StoreChainRepository, logger)
	emailSvc := email.NewEmailService(config.SMTP.From, config.SMTP.Password, config.SMTP.Host, config.SMTP.Port, config.Frontend.Url)

	return &Services{
		UserService:         NewUserService(repos.UserRepository, config.JWT.Secret, config, emailSvc, logger),
		ProfileService:      NewProfileService(repos.ProfileRepository),
		AIConfigService:     NewAIConfigService(repos.AIConfigRepository),
		RecipeService:       NewRecipeService(repos.RecipeRepository, repos.UserRepository, repos.AIConfigRepository, fileStorage, logger, &factory, urlParserService, pdfParserService),
		ShoppingListService: NewShoppingListService(repos.ShoppingListRepository, repos.RecipeRepository, storeChainService, aiModel, logger),
		StoreChainService:   storeChainService,
	}
}
