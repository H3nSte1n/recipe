package repository

import "gorm.io/gorm"

type Repositories struct {
	UserRepository         UserRepository
	ProfileRepository      ProfileRepository
	AIConfigRepository     AIConfigRepository
	RecipeRepository       RecipeRepository
	ShoppingListRepository ShoppingListRepository
	StoreChainRepository   StoreChainRepository
	PlannedMealRepository  PlannedMealRepository
}

func NewRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		UserRepository:         NewUserRepository(db),
		ProfileRepository:      NewProfileRepository(db),
		AIConfigRepository:     NewAIConfigRepository(db),
		RecipeRepository:       NewRecipeRepository(db),
		ShoppingListRepository: NewShoppingListRepository(db),
		StoreChainRepository:   NewStoreChainRepository(db),
		PlannedMealRepository:  NewPlannedMealRepository(db),
	}
}
