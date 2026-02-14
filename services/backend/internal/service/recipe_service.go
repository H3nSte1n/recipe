package service

import (
	"context"
	"fmt"
	"github.com/H3nSte1n/recipe/internal/domain"
	"github.com/H3nSte1n/recipe/internal/errors"
	"github.com/H3nSte1n/recipe/internal/repository"
	"github.com/H3nSte1n/recipe/pkg/ai"
	"github.com/H3nSte1n/recipe/pkg/pdfparser"
	"github.com/H3nSte1n/recipe/pkg/storage"
	"github.com/H3nSte1n/recipe/pkg/urlparser"
	"go.uber.org/zap"
)

type RecipeService interface {
	Create(ctx context.Context, userID string, req *domain.CreateRecipeRequest) (*domain.Recipe, error)
	Update(ctx context.Context, userID string, recipeID string, req *domain.CreateRecipeRequest) (*domain.Recipe, error)
	Delete(ctx context.Context, userID string, recipeID string) error
	GetByID(ctx context.Context, userID string, recipeID string, nutritionLevel domain.NutritionDetailLevel) (*domain.Recipe, error)
	ListUserRecipes(ctx context.Context, userID string) ([]domain.Recipe, error)
	ListPublicRecipes(ctx context.Context, page, pageSize int) ([]domain.Recipe, int64, error)
	ImportFromURL(ctx context.Context, userID string, req *domain.ImportURLRequest) (*domain.Recipe, error)
	ImportFromPDF(ctx context.Context, userID string, req *domain.ImportPDFRequest, file []byte) (*domain.Recipe, error)
	ParsePlainTextInstructions(ctx context.Context, userID string, req *domain.ParsePlainTextInstructionsRequest) (*[]domain.RecipeInstruction, error)
}

type recipeService struct {
	recipeRepo   repository.RecipeRepository
	userRepo     repository.UserRepository
	aiConfigRepo repository.AIConfigRepository
	fileStorage  storage.FileStore
	logger       *zap.Logger
	modelFactory *ai.ModelFactory
	urlParser    urlparser.Service
	pdfParser    pdfparser.Service
}

func NewRecipeService(
	recipeRepo repository.RecipeRepository,
	userRepo repository.UserRepository,
	aiConfigRepo repository.AIConfigRepository,
	fileStorage storage.FileStore,
	logger *zap.Logger,
	modelFactory *ai.ModelFactory,
	urlParser urlparser.Service,
	pdfParser pdfparser.Service,
) RecipeService {
	return &recipeService{
		recipeRepo:   recipeRepo,
		userRepo:     userRepo,
		aiConfigRepo: aiConfigRepo,
		fileStorage:  fileStorage,
		logger:       logger,
		modelFactory: modelFactory,
		urlParser:    urlParser,
		pdfParser:    pdfParser,
	}
}

func (s *recipeService) Create(ctx context.Context, userID string, req *domain.CreateRecipeRequest) (*domain.Recipe, error) {
	_, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	var imageURL string
	if req.Image != nil {
		var err error
		imageURL, err = s.fileStorage.UploadFile(ctx, req.Image)
		if err != nil {
			s.logger.Error("failed to upload image",
				zap.Error(err),
				zap.String("userID", userID))
			return nil, fmt.Errorf("failed to upload image: %w", err)
		}
	}

	if len(req.SubRecipes) > 0 {
		for _, sr := range req.SubRecipes {
			subRecipe, err := s.recipeRepo.GetByID(ctx, sr.RecipeID, domain.NutritionDetailBase)
			if err != nil {
				return nil, errors.New("sub-recipe not found")
			}
			if subRecipe.IsPrivate && subRecipe.UserID != userID {
				return nil, errors.New("unauthorized")
			}
		}
	}

	recipe := &domain.Recipe{
		UserID:       userID,
		Title:        req.Title,
		Description:  req.Description,
		Notes:        req.Notes,
		Rating:       req.Rating,
		ImageURL:     imageURL,
		Status:       req.Status,
		SourceType:   req.SourceType,
		Source:       req.SourceURL,
		IsPrivate:    req.IsPrivate,
		Servings:     req.Servings,
		PrepTime:     req.PrepTime,
		CookTime:     req.CookTime,
		ShelfLife:    req.ShelfLife,
		Ingredients:  req.Ingredients,
		Instructions: req.Instructions,
		Nutrition:    req.Nutrition,
	}

	if err := s.recipeRepo.Create(ctx, recipe); err != nil {
		s.logger.Error("failed to create recipe",
			zap.String("user_id", userID),
			zap.Error(err))
		return nil, err
	}

	if len(req.SubRecipes) > 0 {
		recipe.SubRecipes = make([]domain.SubRecipe, len(req.SubRecipes))
		for i, sr := range req.SubRecipes {
			recipe.SubRecipes[i] = domain.SubRecipe{
				ParentID:      recipe.ID,
				ChildID:       sr.RecipeID,
				ServingFactor: sr.ServingFactor,
			}
		}
		if err := s.recipeRepo.Update(ctx, recipe); err != nil {
			s.logger.Error("failed to add sub-recipes",
				zap.String("recipe_id", recipe.ID),
				zap.Error(err))
			return nil, err
		}
	}

	return s.recipeRepo.GetByID(ctx, recipe.ID, domain.NutritionDetailBase)
}

func (s *recipeService) Update(ctx context.Context, userID string, recipeID string, req *domain.CreateRecipeRequest) (*domain.Recipe, error) {
	existingRecipe, err := s.recipeRepo.GetByID(ctx, recipeID, domain.NutritionDetailBase)
	if err != nil {
		return nil, errors.ErrNotFound
	}
	if existingRecipe.UserID != userID {
		return nil, errors.ErrUnauthorized
	}

	imageURL := existingRecipe.ImageURL
	if req.Image != nil {
		newImageURL, err := s.fileStorage.UploadFile(ctx, req.Image)
		if err != nil {
			s.logger.Error("failed to upload new image",
				zap.Error(err),
				zap.String("userID", userID),
				zap.String("recipeID", recipeID))
			return nil, errors.ErrInternal.Wrap("failed to upload image")
		}

		if existingRecipe.ImageURL != "" {
			if err := s.fileStorage.DeleteFile(ctx, existingRecipe.ImageURL); err != nil {
				s.logger.Warn("failed to delete old image",
					zap.Error(err),
					zap.String("imageURL", existingRecipe.ImageURL))
			}
		}

		imageURL = newImageURL
	}

	if len(req.SubRecipes) > 0 {
		for _, sr := range req.SubRecipes {
			if sr.RecipeID == recipeID {
				return nil, errors.New("recipe cannot include itself as a sub-recipe")
			}

			subRecipe, err := s.recipeRepo.GetByID(ctx, sr.RecipeID, domain.NutritionDetailBase)
			if err != nil {
				return nil, errors.ErrNotFound.Wrap("sub-recipe not found")
			}
			if subRecipe.IsPrivate && subRecipe.UserID != userID {
				return nil, errors.ErrUnauthorized
			}
		}
	}

	err = s.recipeRepo.WithTypedTransaction(ctx, func(recipeRepo *repository.RecipeRepositoryImpl) error {
		recipe := &domain.Recipe{
			ID:           recipeID,
			UserID:       userID,
			Title:        req.Title,
			Description:  req.Description,
			Notes:        req.Notes,  // Update notes
			Rating:       req.Rating, // Update rating
			ImageURL:     imageURL,   // Updated image URL
			Status:       req.Status, // Update status
			SourceType:   req.SourceType,
			Source:       req.SourceURL,
			IsPrivate:    req.IsPrivate,
			Servings:     req.Servings,
			PrepTime:     req.PrepTime,
			CookTime:     req.CookTime,
			Ingredients:  req.Ingredients,
			Instructions: req.Instructions,
			Nutrition:    req.Nutrition,
		}

		// Update base recipe
		if err := recipeRepo.Update(ctx, recipe); err != nil {
			return err
		}

		// Update sub-recipes if provided
		if len(req.SubRecipes) > 0 {
			recipe.SubRecipes = make([]domain.SubRecipe, len(req.SubRecipes))
			for i, sr := range req.SubRecipes {
				recipe.SubRecipes[i] = domain.SubRecipe{
					ParentID:      recipe.ID,
					ChildID:       sr.RecipeID,
					ServingFactor: sr.ServingFactor,
				}
			}
			if err := recipeRepo.Update(ctx, recipe); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		// If update failed and we uploaded a new image, clean it up
		if req.Image != nil && imageURL != existingRecipe.ImageURL {
			if cleanupErr := s.fileStorage.DeleteFile(ctx, imageURL); cleanupErr != nil {
				s.logger.Error("failed to cleanup new image after update failure",
					zap.Error(cleanupErr),
					zap.String("imageURL", imageURL))
			}
		}
		s.logger.Error("failed to update recipe",
			zap.String("user_id", userID),
			zap.String("recipe_id", recipeID),
			zap.Error(err))
		return nil, err
	}

	return s.recipeRepo.GetByID(ctx, recipeID, domain.NutritionDetailBase)
}

func (s *recipeService) Delete(ctx context.Context, userID string, recipeID string) error {
	recipe, err := s.recipeRepo.GetByID(ctx, recipeID, domain.NutritionDetailBase)
	if err != nil {
		return errors.ErrNotFound
	}
	if recipe.UserID != userID {
		return errors.ErrUnauthorized
	}

	return s.recipeRepo.WithTypedTransaction(ctx, func(recipeRepo *repository.RecipeRepositoryImpl) error {
		if err := recipeRepo.Delete(ctx, recipeID); err != nil {
			return err
		}

		if recipe.ImageURL != "" {
			if err := s.fileStorage.DeleteFile(ctx, recipe.ImageURL); err != nil {
				s.logger.Warn("failed to delete recipe image",
					zap.Error(err),
					zap.String("imageURL", recipe.ImageURL))
			}
		}

		return nil
	})
}

func (s *recipeService) GetByID(ctx context.Context, userID string, recipeID string, nutritionLevel domain.NutritionDetailLevel) (*domain.Recipe, error) {
	recipe, err := s.recipeRepo.GetByID(ctx, recipeID, nutritionLevel)
	if err != nil {
		return nil, errors.New("recipe not found")
	}

	// Check access rights
	if recipe.IsPrivate && recipe.UserID != userID {
		return nil, errors.New("unauthorized")
	}

	return recipe, nil
}

func (s *recipeService) ListUserRecipes(ctx context.Context, userID string) ([]domain.Recipe, error) {
	return s.recipeRepo.ListByUserID(ctx, userID, true)
}

func (s *recipeService) ListPublicRecipes(ctx context.Context, page, pageSize int) ([]domain.Recipe, int64, error) {
	return s.recipeRepo.ListPublic(ctx, page, pageSize)
}

func (s *recipeService) ImportFromURL(ctx context.Context, userID string, req *domain.ImportURLRequest) (*domain.Recipe, error) {
	userPrefs, err := s.getUserAIPreferences(ctx, userID)
	if err != nil || userPrefs == nil {
		userPrefs = &ai.UserAIPreferences{
			ModelType: ai.ModelDefault,
		}
	}

	aiModel, err := s.modelFactory.CreateModel(userPrefs.ModelType, userPrefs.APIKey)
	if err != nil {
		s.logger.Error("failed to create AI model", zap.Error(err))
		return nil, err
	}

	parsedRecipe, err := s.urlParser.Parse(ctx, req.URL, aiModel)
	if err != nil {
		s.logger.Error("failed to parse recipe from URL",
			zap.String("url", req.URL),
			zap.Error(err))
		return nil, err
	}

	return parsedRecipe, nil
}

func (s *recipeService) ImportFromPDF(ctx context.Context, userID string, req *domain.ImportPDFRequest, file []byte) (*domain.Recipe, error) {
	userPrefs, err := s.getUserAIPreferences(ctx, userID)
	if err != nil || userPrefs == nil {
		userPrefs = &ai.UserAIPreferences{
			ModelType: ai.ModelDefault,
		}
	}

	// Create AI model instance
	aiModel, err := s.modelFactory.CreateModel(userPrefs.ModelType, userPrefs.APIKey)
	if err != nil {
		s.logger.Error("failed to create AI model", zap.Error(err))
		return nil, err
	}

	// Parse recipe from PDF
	parsedRecipe, err := s.pdfParser.Parse(ctx, file, aiModel)
	if err != nil {
		s.logger.Error("failed to parse recipe from PDF", zap.Error(err))
		return nil, err
	}

	return parsedRecipe, nil
}

func (s *recipeService) ParsePlainTextInstructions(ctx context.Context, userID string, req *domain.ParsePlainTextInstructionsRequest) (*[]domain.RecipeInstruction, error) {
	userPrefs, err := s.getUserAIPreferences(ctx, userID)
	if err != nil || userPrefs == nil {
		userPrefs = &ai.UserAIPreferences{
			ModelType: ai.ModelDefault,
		}
	}

	aiModel, err := s.modelFactory.CreateModel(userPrefs.ModelType, userPrefs.APIKey)
	if err != nil {
		s.logger.Error("failed to create AI model", zap.Error(err))
		return nil, err
	}

	return aiModel.ParseInstructions(ctx, req.PlainText)
}

func (s *recipeService) getUserAIPreferences(ctx context.Context, userID string) (*ai.UserAIPreferences, error) {
	userAIConfig, err := s.aiConfigRepo.GetDefaultConfig(ctx, userID)
	if err != nil {
		s.logger.Warn("failed to get user AI config",
			zap.String("userID", userID),
			zap.Error(err))
		return nil, err
	}

	if userAIConfig == nil {
		return nil, nil
	}

	// Get the AI model details
	aiModel := userAIConfig.AIModel
	if aiModel == nil {
		s.logger.Error("AI model not found for config",
			zap.String("configID", userAIConfig.ID),
			zap.String("modelID", userAIConfig.AIModelID))
		return nil, fmt.Errorf("AI model not found")
	}

	modelType, err := s.mapAIModelToModelType(aiModel)
	if err != nil {
		s.logger.Error("unsupported AI model",
			zap.String("provider", aiModel.Provider),
			zap.String("modelVersion", aiModel.ModelVersion))
		return nil, err
	}

	return &ai.UserAIPreferences{
		ModelType: modelType,
		APIKey:    userAIConfig.APIKey,
	}, nil
}

func (s *recipeService) mapAIModelToModelType(model *domain.AIModel) (ai.ModelType, error) {
	key := fmt.Sprintf("%s-%s", model.Provider, model.ModelVersion)

	switch key {
	case "openai-gpt-4":
		return ai.ModelGPT4, nil
	case "openai-gpt-3.5-turbo":
		return ai.ModelGPT35, nil
	case "anthropic-claude-2":
		return ai.ModelClaude2, nil
	default:
		return "", fmt.Errorf("unsupported model: %s", key)
	}
}
