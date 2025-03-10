package service

import (
	"github.com/yourusername/recipe-app/internal/repository"
	"github.com/yourusername/recipe-app/pkg/ai"
	"github.com/yourusername/recipe-app/pkg/config"
	"github.com/yourusername/recipe-app/pkg/pdfparser"
	"github.com/yourusername/recipe-app/pkg/storage"
	"github.com/yourusername/recipe-app/pkg/urlparser"
	"go.uber.org/zap"
)

type Services struct {
	UserService     UserService
	ProfileService  ProfileService
	AIConfigService AIConfigService
	RecipeService   RecipeService
}

func NewServices(repos *repository.Repositories, config config.Config, fileStorage storage.FileStore, logger *zap.Logger, factory ai.ModelFactory) *Services {
	pdfParserService := pdfparser.NewService(logger)
	urlParserService := urlparser.NewService(logger)

	return &Services{
		UserService:     NewUserService(repos.UserRepository, repos.ProfileRepository, config.JWT.Secret, config),
		ProfileService:  NewProfileService(repos.ProfileRepository, repos.UserRepository),
		AIConfigService: NewAIConfigService(repos.AIConfigRepository),
		RecipeService:   NewRecipeService(repos.RecipeRepository, repos.UserRepository, repos.AIConfigRepository, fileStorage, logger, &factory, urlParserService, pdfParserService),
	}
}
