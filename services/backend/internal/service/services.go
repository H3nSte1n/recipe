package service

import (
	"github.com/yourusername/recipe-app/internal/repository"
	"github.com/yourusername/recipe-app/pkg/config"
)

type Services struct {
	AuthService AuthService
}

func NewServices(repos *repository.Repositories, config config.Config) *Services {
	return &Services{
		AuthService: NewAuthService(repos.UserRepository, config.JWTSecret, config),
	}
}
