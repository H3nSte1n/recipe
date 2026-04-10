package repository

import (
	"context"
	"errors"
	"github.com/H3nSte1n/recipe/internal/domain"
	"gorm.io/gorm"
)

type AIConfigRepository interface {
	Create(ctx context.Context, config *domain.UserAIConfig) error
	Update(ctx context.Context, config *domain.UserAIConfig) error
	GetByID(ctx context.Context, id string) (*domain.UserAIConfig, error)
	GetByUserAndModel(ctx context.Context, userID, modelID string) (*domain.UserAIConfig, error)
	ListByUserID(ctx context.Context, userID string) ([]domain.UserAIConfig, error)
	Delete(ctx context.Context, id string) error
	GetAIModels(ctx context.Context) ([]domain.AIModel, error)
	GetDefaultConfig(ctx context.Context, userID string) (*domain.UserAIConfig, error)
	SetDefault(ctx context.Context, userID, configID string) error
	ClearDefaultByUserID(ctx context.Context, userID string, excludeIDs ...string) error
	WithTypedTransaction(ctx context.Context, fn func(AIConfigRepository) error) error
	RunTx(ctx context.Context, fn func() error) error
}

type AIConfigRepositoryImpl struct {
	*BaseRepository
}

func NewAIConfigRepository(db *gorm.DB) AIConfigRepository {
	return &AIConfigRepositoryImpl{
		BaseRepository: NewBaseRepository(db),
	}
}

func (r *AIConfigRepositoryImpl) WithTypedTransaction(ctx context.Context, fn func(AIConfigRepository) error) error {
	return r.RunInTransaction(ctx, func(tx *gorm.DB) error {
		txRepo := &AIConfigRepositoryImpl{BaseRepository: NewBaseRepository(tx)}
		return fn(txRepo)
	})
}

func (r *AIConfigRepositoryImpl) RunTx(ctx context.Context, fn func() error) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn()
	})
}

func (r *AIConfigRepositoryImpl) Create(ctx context.Context, config *domain.UserAIConfig) error {
	return r.DB.WithContext(ctx).Create(config).Error
}

func (r *AIConfigRepositoryImpl) Update(ctx context.Context, config *domain.UserAIConfig) error {
	return r.DB.WithContext(ctx).Save(config).Error
}

func (r *AIConfigRepositoryImpl) GetByID(ctx context.Context, id string) (*domain.UserAIConfig, error) {
	var config domain.UserAIConfig
	err := r.DB.WithContext(ctx).
		Preload("AIModel").
		First(&config, "id = ?", id).Error
	return &config, err
}

func (r *AIConfigRepositoryImpl) GetByUserAndModel(ctx context.Context, userID, modelID string) (*domain.UserAIConfig, error) {
	var config domain.UserAIConfig
	err := r.DB.WithContext(ctx).
		Where("user_id = ? AND ai_model_id = ?", userID, modelID).
		First(&config).Error
	return &config, err
}

func (r *AIConfigRepositoryImpl) ListByUserID(ctx context.Context, userID string) ([]domain.UserAIConfig, error) {
	var configs []domain.UserAIConfig
	err := r.DB.WithContext(ctx).
		Preload("AIModel").
		Where("user_id = ?", userID).
		Find(&configs).Error
	return configs, err
}

func (r *AIConfigRepositoryImpl) Delete(ctx context.Context, id string) error {
	return r.DB.WithContext(ctx).Delete(&domain.UserAIConfig{ID: id}).Error
}

func (r *AIConfigRepositoryImpl) GetAIModels(ctx context.Context) ([]domain.AIModel, error) {
	var models []domain.AIModel
	err := r.DB.WithContext(ctx).
		Where("is_active = ?", true).
		Find(&models).Error
	return models, err
}

func (r *AIConfigRepositoryImpl) GetDefaultConfig(ctx context.Context, userID string) (*domain.UserAIConfig, error) {
	var config domain.UserAIConfig
	err := r.DB.WithContext(ctx).
		Preload("AIModel").
		Where("user_id = ? AND is_default = ?", userID, true).
		First(&config).Error
	return &config, err
}

func (r *AIConfigRepositoryImpl) SetDefault(ctx context.Context, userID, configID string) error {
	return r.RunInTransaction(ctx, func(tx *gorm.DB) error {
		result := tx.Model(&domain.UserAIConfig{}).
			Where("id = ? AND user_id = ?", configID, userID).
			Update("is_default", true)

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return errors.New("unable to set default config")
		}

		return tx.Model(&domain.UserAIConfig{}).
			Where("user_id = ? AND id != ?", userID, configID).
			Update("is_default", false).Error
	})
}

func (r *AIConfigRepositoryImpl) ClearDefaultByUserID(ctx context.Context, userID string, excludeIDs ...string) error {
	query := r.DB.WithContext(ctx).Model(&domain.UserAIConfig{}).Where("user_id = ?", userID)
	if len(excludeIDs) > 0 {
		query = query.Where("id NOT IN ?", excludeIDs)
	}
	return query.Update("is_default", false).Error
}
