package service

import (
	"context"
	"github.com/H3nSte1n/recipe/internal/domain"
	apperrors "github.com/H3nSte1n/recipe/internal/errors"
	"github.com/H3nSte1n/recipe/internal/repository"
	"go.uber.org/zap"
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
	WithTypedTransaction(ctx context.Context, fn func(repository.AIConfigRepository) error) error
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
	cipher       APIKeyCipher
	logger       *zap.Logger
}

func NewAIConfigService(aiConfigRepo aiConfigRepository, cipher APIKeyCipher, logger *zap.Logger) AIConfigService {
	return &aiConfigService{
		aiConfigRepo: aiConfigRepo,
		cipher:       cipher,
		logger:       logger,
	}
}

func (s *aiConfigService) Create(ctx context.Context, userID string, req *domain.CreateUserAIConfigRequest) (*domain.UserAIConfig, error) {
	var configID string

	encryptedKey, err := s.cipher.Encrypt(req.APIKey)
	if err != nil {
		return nil, err
	}

	err = s.aiConfigRepo.WithTypedTransaction(ctx, func(txRepo repository.AIConfigRepository) error {
		if req.IsDefault {
			if err := txRepo.ClearDefaultByUserID(ctx, userID); err != nil {
				return err
			}
		}

		config := &domain.UserAIConfig{
			UserID:    userID,
			AIModelID: req.AIModelID,
			APIKey:    encryptedKey,
			IsDefault: req.IsDefault,
			Settings:  req.Settings,
		}

		if err := txRepo.Create(ctx, config); err != nil {
			return err
		}
		configID = config.ID
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Return via the service read path so the API key is decrypted for the caller.
	return s.GetByID(ctx, userID, configID)
}

func (s *aiConfigService) Update(ctx context.Context, userID string, configID string, req *domain.UpdateUserAIConfigRequest) (*domain.UserAIConfig, error) {
	config, err := s.GetByID(ctx, userID, configID)
	if err != nil {
		return nil, err
	}

	err = s.aiConfigRepo.WithTypedTransaction(ctx, func(txRepo repository.AIConfigRepository) error {
		if req.IsDefault != nil && *req.IsDefault {
			if err := txRepo.ClearDefaultByUserID(ctx, userID, configID); err != nil {
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

		// config.APIKey is plaintext here (GetByID decrypts on read); encrypt before
		// persisting. This also re-encrypts any legacy plaintext row on next update.
		encryptedKey, err := s.cipher.Encrypt(config.APIKey)
		if err != nil {
			return err
		}
		config.APIKey = encryptedKey

		return txRepo.Update(ctx, config)
	})

	if err != nil {
		return nil, err
	}

	// Return via the service read path so the API key is decrypted for the caller.
	return s.GetByID(ctx, userID, configID)
}

func (s *aiConfigService) GetByID(ctx context.Context, userID string, configID string) (*domain.UserAIConfig, error) {
	config, err := s.aiConfigRepo.GetByID(ctx, configID)
	if err != nil {
		return nil, err
	}

	if config.UserID != userID {
		// Deliberately ErrNotFound, not ErrUnauthorized: a 403 here would tell an
		// unauthorized caller "this config ID exists, just not yours," letting them
		// enumerate valid IDs by observing 403 vs 404. Reads must be indistinguishable
		// from a genuinely missing config.
		return nil, apperrors.ErrNotFound
	}

	config.APIKey = decryptAPIKey(s.cipher, s.logger, config.APIKey)
	return config, nil
}

func (s *aiConfigService) List(ctx context.Context, userID string) ([]domain.UserAIConfig, error) {
	configs, err := s.aiConfigRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	for i := range configs {
		configs[i].APIKey = decryptAPIKey(s.cipher, s.logger, configs[i].APIKey)
	}
	return configs, nil
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
		if apperrors.IsNotFound(err) {
			return nil, apperrors.ErrNotFound.Wrap("default AI configuration not found")
		}
		return nil, err
	}

	config.APIKey = decryptAPIKey(s.cipher, s.logger, config.APIKey)
	return config, nil
}
