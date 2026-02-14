package repository

import (
	"context"
	"errors"
	"github.com/H3nSte1n/recipe/internal/domain"
	"gorm.io/gorm"
)

type RecipeRepository interface {
	Repository[domain.Recipe]
	Create(ctx context.Context, recipe *domain.Recipe) error
	Update(ctx context.Context, recipe *domain.Recipe) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string, nutritionLevel domain.NutritionDetailLevel) (*domain.Recipe, error)
	ListByUserID(ctx context.Context, userID string, includePrivate bool) ([]domain.Recipe, error)
	ListPublic(ctx context.Context, page, pageSize int) ([]domain.Recipe, int64, error)
	Exists(ctx context.Context, id string) (bool, error)
	WithTypedTransaction(ctx context.Context, fn func(*RecipeRepositoryImpl) error) error
}

type RecipeRepositoryImpl struct {
	*BaseRepository[domain.Recipe]
}

func NewRecipeRepository(db *gorm.DB) RecipeRepository {
	return &RecipeRepositoryImpl{
		BaseRepository: NewBaseRepository[domain.Recipe](db),
	}
}

func (r *RecipeRepositoryImpl) WithTypedTransaction(ctx context.Context, fn func(*RecipeRepositoryImpl) error) error {
	return r.WithTransaction(ctx, func(txRepo Repository[domain.Recipe]) error {
		typed := &RecipeRepositoryImpl{
			BaseRepository: txRepo.(*BaseRepository[domain.Recipe]),
		}
		return fn(typed)
	})
}

func (r *RecipeRepositoryImpl) Create(ctx context.Context, recipe *domain.Recipe) error {
	return r.WithTypedTransaction(ctx, func(txRepo *RecipeRepositoryImpl) error {
		// Create recipe with all its associations
		return txRepo.GetDB().Create(recipe).Error
	})
}

func (r *RecipeRepositoryImpl) Update(ctx context.Context, recipe *domain.Recipe) error {
	return r.WithTypedTransaction(ctx, func(txRepo *RecipeRepositoryImpl) error {
		// Delete existing related data
		if err := txRepo.GetDB().Where("recipe_id = ?", recipe.ID).Delete(&domain.RecipeIngredient{}).Error; err != nil {
			return err
		}
		if err := txRepo.GetDB().Where("recipe_id = ?", recipe.ID).Delete(&domain.RecipeInstruction{}).Error; err != nil {
			return err
		}
		if err := txRepo.GetDB().Where("recipe_id = ?", recipe.ID).Delete(&domain.RecipeNutrition{}).Error; err != nil {
			return err
		}
		if err := txRepo.GetDB().Where("parent_id = ?", recipe.ID).Delete(&domain.SubRecipe{}).Error; err != nil {
			return err
		}

		// Update recipe base data
		if err := txRepo.GetDB().Model(recipe).
			Select("title", "description", "source_type", "source_url", "is_private",
				"servings", "prep_time", "cook_time", "updated_at").
			Updates(recipe).Error; err != nil {
			return err
		}

		// Create new ingredients
		if len(recipe.Ingredients) > 0 {
			if err := txRepo.GetDB().Create(&recipe.Ingredients).Error; err != nil {
				return err
			}
		}

		// Create new instructions
		if len(recipe.Instructions) > 0 {
			if err := txRepo.GetDB().Create(&recipe.Instructions).Error; err != nil {
				return err
			}
		}

		// Create new nutrition
		if recipe.Nutrition != nil {
			if err := txRepo.GetDB().Create(recipe.Nutrition).Error; err != nil {
				return err
			}
		}

		// Create new sub-recipes
		if len(recipe.SubRecipes) > 0 {
			if err := txRepo.GetDB().Create(&recipe.SubRecipes).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *RecipeRepositoryImpl) Delete(ctx context.Context, id string) error {
	return r.WithTypedTransaction(ctx, func(txRepo *RecipeRepositoryImpl) error {
		// Due to ON DELETE CASCADE, we only need to delete the recipe
		return txRepo.GetDB().Delete(&domain.Recipe{ID: id}).Error
	})
}

func (r *RecipeRepositoryImpl) GetByID(ctx context.Context, id string, nutritionLevel domain.NutritionDetailLevel) (*domain.Recipe, error) {
	var recipe domain.Recipe
	query := r.db.WithContext(ctx).
		Preload("Ingredients").
		Preload("Instructions").
		Preload("SubRecipes").
		Preload("SubRecipes.Child")

	// Select nutrition fields based on detail level
	switch nutritionLevel {
	case domain.NutritionDetailBase:
		query = query.Preload("Nutrition", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, recipe_id, calories, per_serving")
		})
	case domain.NutritionDetailMacro:
		query = query.Preload("Nutrition", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, recipe_id, calories, per_serving, protein, carbs, fat, fiber, sugar, saturated_fat, cholesterol, sodium")
		})
	case domain.NutritionDetailMicro:
		query = query.Preload("Nutrition")
	default:
		query = query.Preload("Nutrition", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, recipe_id, calories, per_serving")
		})
	}

	err := query.First(&recipe, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("recipe not found")
		}
		return nil, err
	}

	return &recipe, nil
}

func (r *RecipeRepositoryImpl) ListByUserID(ctx context.Context, userID string, includePrivate bool) ([]domain.Recipe, error) {
	var recipes []domain.Recipe
	query := r.db.WithContext(ctx).
		Preload("Ingredients", func(db *gorm.DB) *gorm.DB {
			return db.Order("recipe_ingredients.id")
		}).
		Preload("Instructions", func(db *gorm.DB) *gorm.DB {
			return db.Order("recipe_instructions.step_number")
		}).
		Preload("Nutrition").
		Preload("SubRecipes", func(db *gorm.DB) *gorm.DB {
			return db.Order("sub_recipes.id")
		}).
		Preload("SubRecipes.Child", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, title, servings")
		}).
		Where("user_id = ?", userID)

	if !includePrivate {
		query = query.Where("is_private = ?", false)
	}

	err := query.Order("created_at DESC").Find(&recipes).Error
	if err != nil {
		return nil, err
	}

	return recipes, nil
}

func (r *RecipeRepositoryImpl) ListPublic(ctx context.Context, page, pageSize int) ([]domain.Recipe, int64, error) {
	var recipes []domain.Recipe
	var total int64

	// Get total count
	if err := r.db.WithContext(ctx).
		Model(&domain.Recipe{}).
		Where("is_private = ?", false).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated recipes
	err := r.db.WithContext(ctx).
		Preload("Ingredients", func(db *gorm.DB) *gorm.DB {
			return db.Order("recipe_ingredients.id")
		}).
		Preload("Instructions", func(db *gorm.DB) *gorm.DB {
			return db.Order("recipe_instructions.step_number")
		}).
		Preload("Nutrition").
		Preload("SubRecipes", func(db *gorm.DB) *gorm.DB {
			return db.Order("sub_recipes.id")
		}).
		Preload("SubRecipes.Child", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, title, servings")
		}).
		Where("is_private = ?", false).
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&recipes).Error

	if err != nil {
		return nil, 0, err
	}

	return recipes, total, nil
}

func (r *RecipeRepositoryImpl) Exists(ctx context.Context, id string) (bool, error) {
	var exists bool
	err := r.db.WithContext(ctx).
		Model(&domain.Recipe{}).
		Select("1").
		Where("id = ?", id).
		First(&exists).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
