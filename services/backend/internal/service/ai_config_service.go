package service

import (
	"context"
	"github.com/H3nSte1n/recipe/internal/domain"
	apperrors "github.com/H3nSte1n/recipe/internal/errors"
)

type aiConfigRepository interface {
	Create(ctx context.Context, config *domain.UserAIConfig) error
	Update(ctx context.Context, config *domain.UserAIConfig) error
	GetByID(ctx context.Context, id string) (*domain.UserAIConfig, error)
	ListByUserID(ctx context.Context, userID string) ([]domain.UserAIConfig, error)
	Delete(ctx context.Context, id string) error
	GetAIModels(ctx context.Context) ([]domain.AIModel, error)
	GetDefaultConfig(ctx context.Context, userID string) (*domain.UserAIConfig, error)
	SetDefault(ctx context.Context, userID, configID string) error
	ClearDefaultByUserID(ctx context.Context, userID string, excludeIDs ...string) error
	RunTx(ctx context.Context, fn func() error) error
}

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
	aiConfigRepo aiConfigRepository
}

func NewAIConfigService(aiConfigRepo aiConfigRepository) AIConfigService {
	return &aiConfigService{
		aiConfigRepo: aiConfigRepo,
	}
}

func (s *aiConfigService) Create(ctx context.Context, userID string, req *domain.CreateUserAIConfigRequest) (*domain.UserAIConfig, error) {
	var config *domain.UserAIConfig

	err := s.aiConfigRepo.RunTx(ctx, func() error {
		if req.IsDefault {
			if err := s.aiConfigRepo.ClearDefaultByUserID(ctx, userID); err != nil {
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

		return s.aiConfigRepo.Create(ctx, config)
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

	err = s.aiConfigRepo.RunTx(ctx, func() error {
		if req.IsDefault != nil && *req.IsDefault {
			if err := s.aiConfigRepo.ClearDefaultByUserID(ctx, userID, configID); err != nil {
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

		return s.aiConfigRepo.Update(ctx, config)
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
		return nil, apperrors.ErrUnauthorized
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
		return apperrors.ErrUnauthorized
	}

	return s.aiConfigRepo.SetDefault(ctx, userID, configID)
}

func (s *aiConfigService) GetDefaultConfig(ctx context.Context, userID string) (*domain.UserAIConfig, error) {
	config, err := s.aiConfigRepo.GetDefaultConfig(ctx, userID)
	if err != nil {
		return nil, apperrors.ErrNotFound.Wrap("default AI configuration not found")
	}

	return config, nil
}
