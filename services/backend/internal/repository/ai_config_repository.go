package repository

import (
	"context"
	"errors"
	"github.com/H3nSte1n/recipe/internal/domain"
	"gorm.io/gorm"
)

type AIConfigRepository interface {
	Repository[domain.UserAIConfig]
	Create(ctx context.Context, config *domain.UserAIConfig) error
	Update(ctx context.Context, config *domain.UserAIConfig) error
	GetByID(ctx context.Context, id string) (*domain.UserAIConfig, error)
	GetByUserAndModel(ctx context.Context, userID, modelID string) (*domain.UserAIConfig, error)
	ListByUserID(ctx context.Context, userID string) ([]domain.UserAIConfig, error)
	Delete(ctx context.Context, id string) error
	GetAIModels(ctx context.Context) ([]domain.AIModel, error)
	GetDefaultConfig(ctx context.Context, userID string) (*domain.UserAIConfig, error)
	SetDefault(ctx context.Context, userID, configID string) error
	WithTypedTransaction(ctx context.Context, fn func(*AIConfigRepositoryImpl) error) error
}

type AIConfigRepositoryImpl struct {
	*BaseRepository[domain.UserAIConfig]
}

func NewAIConfigRepository(db *gorm.DB) AIConfigRepository {
	return &AIConfigRepositoryImpl{
		BaseRepository: NewBaseRepository[domain.UserAIConfig](db),
	}
}

func (r *AIConfigRepositoryImpl) WithTypedTransaction(ctx context.Context, fn func(*AIConfigRepositoryImpl) error) error {
	return r.WithTransaction(ctx, func(txRepo Repository[domain.UserAIConfig]) error {
		typed := &AIConfigRepositoryImpl{
			BaseRepository: txRepo.(*BaseRepository[domain.UserAIConfig]),
		}
		return fn(typed)
	})
}

func (r *AIConfigRepositoryImpl) Create(ctx context.Context, config *domain.UserAIConfig) error {
	return r.db.WithContext(ctx).Create(config).Error
}

func (r *AIConfigRepositoryImpl) Update(ctx context.Context, config *domain.UserAIConfig) error {
	return r.db.WithContext(ctx).Save(config).Error
}

func (r *AIConfigRepositoryImpl) GetByID(ctx context.Context, id string) (*domain.UserAIConfig, error) {
	var config domain.UserAIConfig
	err := r.db.WithContext(ctx).
		Preload("AIModel").
		First(&config, "id = ?", id).Error
	return &config, err
}

func (r *AIConfigRepositoryImpl) GetByUserAndModel(ctx context.Context, userID, modelID string) (*domain.UserAIConfig, error) {
	var config domain.UserAIConfig
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND ai_model_id = ?", userID, modelID).
		First(&config).Error
	return &config, err
}

func (r *AIConfigRepositoryImpl) ListByUserID(ctx context.Context, userID string) ([]domain.UserAIConfig, error) {
	var configs []domain.UserAIConfig
	err := r.db.WithContext(ctx).
		Preload("AIModel").
		Where("user_id = ?", userID).
		Find(&configs).Error
	return configs, err
}

func (r *AIConfigRepositoryImpl) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&domain.UserAIConfig{ID: id}).Error
}

func (r *AIConfigRepositoryImpl) GetAIModels(ctx context.Context) ([]domain.AIModel, error) {
	var models []domain.AIModel
	err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Find(&models).Error
	return models, err
}

func (r *AIConfigRepositoryImpl) GetDefaultConfig(ctx context.Context, userID string) (*domain.UserAIConfig, error) {
	var config domain.UserAIConfig
	err := r.db.WithContext(ctx).
		Preload("AIModel").
		Where("user_id = ? AND is_default = ?", userID, true).
		First(&config).Error
	return &config, err
}

func (r *AIConfigRepositoryImpl) SetDefault(ctx context.Context, userID, configID string) error {
	return r.WithTypedTransaction(ctx, func(txRepo *AIConfigRepositoryImpl) error {
		result := txRepo.GetDB().Model(&domain.UserAIConfig{}).
			Where("id = ? AND user_id = ?", configID, userID).
			Update("is_default", true)

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return errors.New("unable to set default config")
		}

		return txRepo.GetDB().Model(&domain.UserAIConfig{}).
			Where("user_id = ? AND id != ?", userID, configID).
			Update("is_default", false).Error
	})
}
