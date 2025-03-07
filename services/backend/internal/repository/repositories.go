package repository

import "gorm.io/gorm"

type Repositories struct {
	UserRepository     UserRepository
	ProfileRepository  ProfileRepository
	AIConfigRepository AIConfigRepository
}

func NewRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		UserRepository:     NewUserRepository(db),
		ProfileRepository:  NewProfileRepository(db),
		AIConfigRepository: NewAIConfigRepository(db),
	}
}
