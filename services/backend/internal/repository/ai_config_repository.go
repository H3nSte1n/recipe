package repository

import (
	"context"
	"errors"
	"github.com/yourusername/recipe-app/internal/domain"
	"gorm.io/gorm"
)

type AIConfigRepository interface {
	Repository
	Create(ctx context.Context, config *domain.UserAIConfig) error
	Update(ctx context.Context, config *domain.UserAIConfig) error
	GetByID(ctx context.Context, id string) (*domain.UserAIConfig, error)
	GetByUserAndModel(ctx context.Context, userID, modelID string) (*domain.UserAIConfig, error)
	ListByUserID(ctx context.Context, userID string) ([]domain.UserAIConfig, error)
	Delete(ctx context.Context, id string) error
	GetAIModels(ctx context.Context) ([]domain.AIModel, error)
	GetDefaultConfig(ctx context.Context, userID string) (*domain.UserAIConfig, error)
	SetDefault(ctx context.Context, userID, configID string) error
}

type aiConfigRepository struct {
	*baseRepository
}

func NewAIConfigRepository(db *gorm.DB) AIConfigRepository {
	return &aiConfigRepository{
		baseRepository: &baseRepository{db: db},
	}
}

func (r *aiConfigRepository) Create(ctx context.Context, config *domain.UserAIConfig) error {
	return r.db.WithContext(ctx).Create(config).Error
}

func (r *aiConfigRepository) Update(ctx context.Context, config *domain.UserAIConfig) error {
	return r.db.WithContext(ctx).Save(config).Error
}

func (r *aiConfigRepository) GetByID(ctx context.Context, id string) (*domain.UserAIConfig, error) {
	var config domain.UserAIConfig
	err := r.db.WithContext(ctx).
		Preload("AIModel").
		First(&config, "id = ?", id).Error
	return &config, err
}

func (r *aiConfigRepository) GetByUserAndModel(ctx context.Context, userID, modelID string) (*domain.UserAIConfig, error) {
	var config domain.UserAIConfig
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND ai_model_id = ?", userID, modelID).
		First(&config).Error
	return &config, err
}

func (r *aiConfigRepository) ListByUserID(ctx context.Context, userID string) ([]domain.UserAIConfig, error) {
	var configs []domain.UserAIConfig
	err := r.db.WithContext(ctx).
		Preload("AIModel").
		Where("user_id = ?", userID).
		Find(&configs).Error
	return configs, err
}

func (r *aiConfigRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&domain.UserAIConfig{ID: id}).Error
}

func (r *aiConfigRepository) GetAIModels(ctx context.Context) ([]domain.AIModel, error) {
	var models []domain.AIModel
	err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Find(&models).Error
	return models, err
}

func (r *aiConfigRepository) GetDefaultConfig(ctx context.Context, userID string) (*domain.UserAIConfig, error) {
	var config domain.UserAIConfig
	err := r.db.WithContext(ctx).
		Preload("AIModel").
		Where("user_id = ? AND is_default = ?", userID, true).
		First(&config).Error
	return &config, err
}

func (r *aiConfigRepository) SetDefault(ctx context.Context, userID, configID string) error {
	return r.WithTransaction(ctx, func(txRepo Repository) error {
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
