package service

import (
	"github.com/yourusername/recipe-app/internal/repository"
	"github.com/yourusername/recipe-app/pkg/config"
)

type Services struct {
	UserService     UserService
	ProfileService  ProfileService
	AIConfigService AIConfigService
}

func NewServices(repos *repository.Repositories, config config.Config) *Services {
	return &Services{
		UserService:     NewUserService(repos.UserRepository, repos.ProfileRepository, config.JWTSecret, config),
		ProfileService:  NewProfileService(repos.ProfileRepository, repos.UserRepository),
		AIConfigService: NewAIConfigService(repos.AIConfigRepository),
	}
}
