package ai

type AIRecipeResponse struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Servings    int    `json:"servings"`
	PrepTime    int    `json:"prepTime"`
	CookTime    int    `json:"cookTime"`
	Ingredients []struct {
		Description string `json:"description"`
	} `json:"ingredients"`
	Instructions []struct {
		StepNumber  int    `json:"stepNumber"`
		Description string `json:"description"`
	} `json:"instructions"`
	// Optional fields
	Notes     string             `json:"notes,omitempty"`
	Nutrition *AIRecipeNutrition `json:"nutrition,omitempty"`
}

type AIRecipeNutrition struct {
	Calories     float64 `json:"calories,omitempty"`
	Protein      float64 `json:"protein,omitempty"`
	Carbs        float64 `json:"carbs,omitempty"`
	Fat          float64 `json:"fat,omitempty"`
	Fiber        float64 `json:"fiber,omitempty"`
	Sugar        float64 `json:"sugar,omitempty"`
	SaturatedFat float64 `json:"saturatedFat,omitempty"`
	Cholesterol  float64 `json:"cholesterol,omitempty"`
	Sodium       float64 `json:"sodium,omitempty"`
}
