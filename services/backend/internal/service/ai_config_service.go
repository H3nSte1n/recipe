// internal/service/ai_config_service.go
package service

import (
	"context"
	"errors"
	"github.com/yourusername/recipe-app/internal/domain"
	"github.com/yourusername/recipe-app/internal/repository"
	"gorm.io/gorm"
)

type AIConfigService interface {
	Create(ctx context.Context, userID string, req *domain.CreateUserAIConfigRequest) (*domain.UserAIConfig, error)
	Update(ctx context.Context, userID string, configID string, req *domain.UpdateUserAIConfigRequest) (*domain.UserAIConfig, error)
	GetByID(ctx context.Context, userID string, configID string) (*domain.UserAIConfig, error)
	List(ctx context.Context, userID string) ([]domain.UserAIConfig, error)
	Delete(ctx context.Context, userID string, configID string) error
	ListAIModels(ctx context.Context) ([]domain.AIModel, error)
	SetDefault(ctx context.Context, userID string, configID string) error
	GetDefaultConfig(ctx context.Context, userID string) (*domain.UserAIConfig, error)
}

type aiConfigService struct {
	aiConfigRepo repository.AIConfigRepository
}

func NewAIConfigService(aiConfigRepo repository.AIConfigRepository) AIConfigService {
	return &aiConfigService{
		aiConfigRepo: aiConfigRepo,
	}
}

func (s *aiConfigService) Create(ctx context.Context, userID string, req *domain.CreateUserAIConfigRequest) (*domain.UserAIConfig, error) {
	var config *domain.UserAIConfig

	err := s.aiConfigRepo.WithTypedTransaction(ctx, func(txRepo *repository.AIConfigRepositoryImpl) error {
		if req.IsDefault {
			if err := txRepo.GetDB().Model(&domain.UserAIConfig{}).
				Where("user_id = ?", userID).
				Update("is_default", false).Error; err != nil {
				return err
			}
		}

		config = &domain.UserAIConfig{
			UserID:    userID,
			AIModelID: req.AIModelID,
			APIKey:    req.APIKey,
			IsDefault: req.IsDefault,
			Settings:  req.Settings,
		}

		if err := txRepo.Create(ctx, config); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return config, nil
}

func (s *aiConfigService) Update(ctx context.Context, userID string, configID string, req *domain.UpdateUserAIConfigRequest) (*domain.UserAIConfig, error) {
	config, err := s.GetByID(ctx, userID, configID)
	if err != nil {
		return nil, err
	}

	err = s.aiConfigRepo.WithTypedTransaction(ctx, func(txRepo *repository.AIConfigRepositoryImpl) error {
		if req.IsDefault != nil && *req.IsDefault {
			if err := txRepo.GetDB().Model(&domain.UserAIConfig{}).
				Where("user_id = ? AND id != ?", userID, configID).
				Update("is_default", false).Error; err != nil {
				return err
			}
		}

		if req.APIKey != nil {
			config.APIKey = *req.APIKey
		}
		if req.IsDefault != nil {
			config.IsDefault = *req.IsDefault
		}
		if req.Settings != nil {
			config.Settings = req.Settings
		}

		if err := txRepo.Update(ctx, config); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return config, nil
}

func (s *aiConfigService) GetByID(ctx context.Context, userID string, configID string) (*domain.UserAIConfig, error) {
	config, err := s.aiConfigRepo.GetByID(ctx, configID)
	if err != nil {
		return nil, err
	}

	if config.UserID != userID {
		return nil, errors.New("unauthorized")
	}

	return config, nil
}

func (s *aiConfigService) List(ctx context.Context, userID string) ([]domain.UserAIConfig, error) {
	return s.aiConfigRepo.ListByUserID(ctx, userID)
}

func (s *aiConfigService) Delete(ctx context.Context, userID string, configID string) error {
	config, err := s.GetByID(ctx, userID, configID)
	if err != nil {
		return err
	}

	return s.aiConfigRepo.Delete(ctx, config.ID)
}

func (s *aiConfigService) ListAIModels(ctx context.Context) ([]domain.AIModel, error) {
	return s.aiConfigRepo.GetAIModels(ctx)
}

func (s *aiConfigService) SetDefault(ctx context.Context, userID string, configID string) error {
	config, err := s.aiConfigRepo.GetByID(ctx, configID)
	if err != nil {
		return err
	}

	if config.UserID != userID {
		return errors.New("unauthorized")
	}

	return s.aiConfigRepo.SetDefault(ctx, userID, configID)
}

func (s *aiConfigService) GetDefaultConfig(ctx context.Context, userID string) (*domain.UserAIConfig, error) {
	config, err := s.aiConfigRepo.GetDefaultConfig(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("default AI configuration not found")
		}
		return nil, err
	}

	return config, nil
}
