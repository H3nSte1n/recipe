package domain

// RecipeSummary is the few recipe fields other features need to check a recipe without loading it whole.
type RecipeSummary struct {
	ID     string
	Title  string
	Status string
}
