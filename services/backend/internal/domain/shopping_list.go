package domain

import "time"

type ShoppingList struct {
	ID           string             `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	UserID       string             `json:"user_id" gorm:"type:uuid;not null"`
	Name         string             `json:"name" gorm:"not null"`
	Description  string             `json:"description"`
	SortType     SortType           `json:"sort_type" gorm:"not null;default:'CATEGORY'"`
	StoreChainID *string            `json:"store_chain_id,omitempty" gorm:"type:uuid"`
	CreatedAt    time.Time          `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time          `json:"updated_at" gorm:"autoUpdateTime"`
	User         *User              `json:"user,omitempty" gorm:"foreignKey:UserID"`
	StoreChain   *StoreChain        `json:"store_chain,omitempty" gorm:"foreignKey:StoreChainID"`
	Items        []ShoppingListItem `json:"items,omitempty" gorm:"foreignKey:ListID"`
}

type ShoppingListItem struct {
	ID        string        `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	ListID    string        `json:"list_id" gorm:"type:uuid;not null"`
	RecipeID  *string       `json:"recipe_id,omitempty" gorm:"type:uuid"`
	Name      string        `json:"name" gorm:"not null"`
	Amount    float64       `json:"amount"`
	Unit      string        `json:"unit"`
	Category  Category      `json:"category" gorm:"not null"`
	IsChecked bool          `json:"is_checked" gorm:"default:false"`
	Notes     string        `json:"notes"`
	CreatedAt time.Time     `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time     `json:"updated_at" gorm:"autoUpdateTime"`
	List      *ShoppingList `json:"list,omitempty" gorm:"foreignKey:ListID"`
	Recipe    *Recipe       `json:"recipe,omitempty" gorm:"foreignKey:RecipeID"`
}

type SortType string

const (
	SortTypeCategory SortType = "CATEGORY"
	SortTypeStore    SortType = "STORE"
)

type Category string

const (
	CategoryProduce   Category = "PRODUCE"
	CategoryMeat      Category = "MEAT"
	CategoryDairy     Category = "DAIRY"
	CategoryBakery    Category = "BAKERY"
	CategoryPantry    Category = "PANTRY"
	CategoryFrozen    Category = "FROZEN"
	CategoryBeverages Category = "BEVERAGES"
	CategoryHousehold Category = "HOUSEHOLD"
	CategoryOther     Category = "OTHER"
)

type CreateShoppingListRequest struct {
	Name         string                    `json:"name" validate:"required"`
	Description  string                    `json:"description"`
	SortType     SortType                  `json:"sort_type" validate:"required,oneof=CATEGORY STORE"`
	StoreChainID string                    `json:"store_chain_id,omitempty"`
	Items        []ShoppingListItemRequest `json:"items,omitempty"`
}

type UpdateShoppingListRequest struct {
	Name        string   `json:"name" validate:"required"`
	Description string   `json:"description"`
	SortType    SortType `json:"sort_type" validate:"required,oneof=CATEGORY STORE"`
}

type UpdateShoppingListItemRequest struct {
	Name     string   `json:"name" validate:"required"`
	Amount   float64  `json:"amount"`
	Unit     string   `json:"unit"`
	Category Category `json:"category" validate:"required"`
	Notes    string   `json:"notes"`
}

type ShoppingListItemRequest struct {
	Name     string   `json:"name" validate:"required"`
	Amount   float64  `json:"amount"`
	Unit     string   `json:"unit"`
	Category Category `json:"category" validate:"required"`
	Notes    string   `json:"notes"`
}

type AddRecipeToListRequest struct {
	RecipeID string  `json:"recipe_id" validate:"required"`
	Servings float64 `json:"servings" validate:"required,min=0.1"`
}
