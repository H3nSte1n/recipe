package domain

import (
	"mime/multipart"
	"time"
)

type Recipe struct {
	ID           string              `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	UserID       string              `json:"user_id" gorm:"type:uuid;not null"`
	Title        string              `json:"title" gorm:"not null"`
	Description  string              `json:"description"`
	Notes        string              `json:"notes"`
	Rating       float64             `json:"rating" gorm:"default:0"` // 0-5
	ImageURL     string              `json:"image_url,omitempty" gorm:"type:varchar(255)"`
	SourceType   string              `json:"source_type" gorm:"not null"` // URL, MANUAL, PDF, IMAGE
	Source       string              `json:"source,omitempty"`
	IsPrivate    bool                `json:"is_private" gorm:"default:false"`
	Servings     int                 `json:"servings" gorm:"not null"`
	PrepTime     int                 `json:"prep_time"`                   // in minutes
	CookTime     int                 `json:"cook_time"`                   // in minutes
	Status       string              `json:"status" gorm:"default:draft"` // draft, published, archived
	CreatedAt    time.Time           `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time           `json:"updated_at" gorm:"autoUpdateTime"`
	User         *User               `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Ingredients  []RecipeIngredient  `json:"ingredients,omitempty" gorm:"foreignKey:RecipeID"`
	Instructions []RecipeInstruction `json:"instructions,omitempty" gorm:"foreignKey:RecipeID"`
	Nutrition    *RecipeNutrition    `json:"nutrition,omitempty" gorm:"foreignKey:RecipeID"`
	SubRecipes   []SubRecipe         `json:"sub_recipes,omitempty" gorm:"foreignKey:ParentID"`
}

type RecipeIngredient struct {
	ID          string  `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	RecipeID    string  `json:"recipe_id" gorm:"type:uuid;not null"`
	Name        string  `json:"name" gorm:"not null"`
	Description string  `json:"description" gorm:"not null"`
	Amount      float64 `json:"amount"`
	Unit        string  `json:"unit"`
	Notes       string  `json:"notes"`
	Recipe      *Recipe `json:"recipe,omitempty" gorm:"foreignKey:RecipeID"`
}

type RecipeInstruction struct {
	ID          string  `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	RecipeID    string  `json:"recipe_id" gorm:"type:uuid;not null"`
	StepNumber  int     `json:"step_number" gorm:"not null"`
	Instruction string  `json:"instruction" gorm:"not null"`
	Recipe      *Recipe `json:"recipe,omitempty" gorm:"foreignKey:RecipeID"`
}

type RecipeNutrition struct {
	ID             string    `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	RecipeID       string    `json:"recipe_id" gorm:"type:uuid;not null"`
	BaseNutrition            // Embed base nutrition
	MacroNutrition           // Embed macro nutrition
	MicroNutrition           // Embed micro nutrition
	CreatedAt      time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	Recipe         *Recipe   `json:"recipe,omitempty" gorm:"foreignKey:RecipeID"`
}

func (RecipeNutrition) TableName() string {
	return "recipe_nutrition"
}

type BaseNutrition struct {
	Calories   float64 `json:"calories"`
	PerServing bool    `json:"per_serving" gorm:"default:true"`
}

type MacroNutrition struct {
	Protein      float64 `json:"protein"`                                   // in grams
	Carbs        float64 `json:"carbs"`                                     // in grams
	Fat          float64 `json:"fat"`                                       // in grams
	Fiber        float64 `json:"fiber"`                                     // in grams
	Sugar        float64 `json:"sugar"`                                     // in grams
	SaturatedFat float64 `json:"saturated_fat" gorm:"column:saturated_fat"` // in grams
	Cholesterol  float64 `json:"cholesterol"`                               // in mg
	Sodium       float64 `json:"sodium"`                                    // in mg
}

type MicroNutrition struct {
	VitaminA   float64 `json:"vitamin_a"`   // in IU
	VitaminC   float64 `json:"vitamin_c"`   // in mg
	VitaminD   float64 `json:"vitamin_d"`   // in IU
	VitaminE   float64 `json:"vitamin_e"`   // in mg
	VitaminK   float64 `json:"vitamin_k"`   // in mcg
	Thiamin    float64 `json:"thiamin"`     // in mg
	Riboflavin float64 `json:"riboflavin"`  // in mg
	Niacin     float64 `json:"niacin"`      // in mg
	VitaminB6  float64 `json:"vitamin_b6"`  // in mg
	VitaminB12 float64 `json:"vitamin_b12"` // in mcg
	Folate     float64 `json:"folate"`      // in mcg
	Calcium    float64 `json:"calcium"`     // in mg
	Iron       float64 `json:"iron"`        // in mg
	Magnesium  float64 `json:"magnesium"`   // in mg
	Phosphorus float64 `json:"phosphorus"`  // in mg
	Potassium  float64 `json:"potassium"`   // in mg
	Zinc       float64 `json:"zinc"`        // in mg
	Selenium   float64 `json:"selenium"`    // in mcg
	Copper     float64 `json:"copper"`      // in mg
	Manganese  float64 `json:"manganese"`   // in mg
}

type SubRecipe struct {
	ID            string  `json:"id" gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	ParentID      string  `json:"parent_id" gorm:"type:uuid;not null"`
	ChildID       string  `json:"child_id" gorm:"type:uuid;not null"`
	ServingFactor float64 `json:"serving_factor" gorm:"default:1"` // to adjust quantities
	Parent        *Recipe `json:"parent,omitempty" gorm:"foreignKey:ParentID"`
	Child         *Recipe `json:"child,omitempty" gorm:"foreignKey:ChildID"`
}

type NutritionDetailLevel string

const (
	NutritionDetailBase  NutritionDetailLevel = "base"
	NutritionDetailMacro NutritionDetailLevel = "macro"
	NutritionDetailMicro NutritionDetailLevel = "micro"
)

type CreateRecipeRequest struct {
	Title                string                `json:"title" validate:"required"`
	Description          string                `json:"description"`
	SourceType           string                `json:"source_type" validate:"required,oneof=URL MANUAL PDF IMAGE"`
	SourceURL            string                `json:"source_url,omitempty"`
	IsPrivate            bool                  `json:"is_private"`
	Servings             int                   `json:"servings" validate:"required,min=1"`
	PrepTime             int                   `json:"prep_time"`
	CookTime             int                   `json:"cook_time"`
	Ingredients          []RecipeIngredient    `json:"ingredients"`
	Instructions         []RecipeInstruction   `json:"instructions"`
	Notes                string                `json:"notes"`
	Rating               float64               `json:"ratings" validate:"omitempty,min=0,max=5"`
	Image                *multipart.FileHeader `json:"-" form:"image"`
	Status               string                `json:"status" validate:"omitempty,oneof=draft published archived"`
	Nutrition            *RecipeNutrition      `json:"nutrition,omitempty"`
	NutritionDetailLevel NutritionDetailLevel  `json:"nutrition_detail_level" validate:"omitempty,oneof=base macro micro"`
	SubRecipes           []SubRecipeRequest    `json:"sub_recipes,omitempty"`
}

type SubRecipeRequest struct {
	RecipeID      string  `json:"recipe_id" validate:"required"`
	ServingFactor float64 `json:"serving_factor" validate:"required,min=0.1"`
}

type ImportURLRequest struct {
	URL       string `json:"url" validate:"required,url"`
	IsPrivate bool   `json:"is_private"`
}

type ParsePlainTextInstructionsRequest struct {
	PlainText string `json:"plain_text" validate:"required"`
}

type ImportPDFRequest struct {
	IsPrivate bool `json:"is_private"`
	// PDF file will be handled by multipart form data
}

type ImportImageRequest struct {
	IsPrivate bool `json:"is_private"`
	// Image file will be handled by multipart form data
}
