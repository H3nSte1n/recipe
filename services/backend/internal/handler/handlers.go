package handler

import (
	"github.com/yourusername/recipe-app/internal/service"
	"go.uber.org/zap"
)

type Handlers struct {
	UserHandler     *UserHandler
	ProfileHandler  *ProfileHandler
	AIConfigHandler *AIConfigHandler
}

func NewHandlers(services *service.Services, logger *zap.Logger) *Handlers {
	return &Handlers{
		UserHandler:     NewUserHandler(services.UserService),
		ProfileHandler:  NewProfileHandler(services.ProfileService),
		AIConfigHandler: NewAIConfigHandler(services.AIConfigService),
	}
}
