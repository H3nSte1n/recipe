package handler

import (
	"github.com/yourusername/recipe-app/internal/service"
	"go.uber.org/zap"
)

type Handlers struct {
	AuthHandler *AuthHandler
}

func NewHandlers(services *service.Services, logger *zap.Logger) *Handlers {
	return &Handlers{
		AuthHandler: NewAuthHandler(services.AuthService),
	}
}
