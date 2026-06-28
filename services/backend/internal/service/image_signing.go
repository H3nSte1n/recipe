package service

import "github.com/H3nSte1n/recipe/internal/domain"

// ImageURLSigner mints a short-lived signed URL for a stored image URL.
// Implemented by pkg/signedurl.Signer. It is nil when local file serving is not
// in use (e.g. S3), in which case stored URLs are returned unchanged.
type ImageURLSigner interface {
	Sign(rawURL string) string
}

// signRecipeImages rewrites every ImageURL reachable from r (the recipe itself
// and any nested sub-recipe children/parents) into a signed, short-lived URL.
// Signing happens only on the response object — the plain URL stays in the DB.
// A visited set keyed by recipe ID makes the walk safe against parent/child
// cycles.
func (s *recipeService) signRecipeImages(r *domain.Recipe) {
	if s.imageSigner == nil || r == nil {
		return
	}
	s.signRecipeImagesRec(r, map[string]bool{})
}

func (s *recipeService) signRecipeImagesRec(r *domain.Recipe, seen map[string]bool) {
	if r == nil {
		return
	}
	if r.ID != "" {
		if seen[r.ID] {
			return
		}
		seen[r.ID] = true
	}

	if r.ImageURL != "" {
		r.ImageURL = s.imageSigner.Sign(r.ImageURL)
	}

	for i := range r.SubRecipes {
		s.signRecipeImagesRec(r.SubRecipes[i].Child, seen)
		s.signRecipeImagesRec(r.SubRecipes[i].Parent, seen)
	}
}

// signRecipeList applies signRecipeImages to each element of a slice.
func (s *recipeService) signRecipeList(recipes []domain.Recipe) {
	for i := range recipes {
		s.signRecipeImages(&recipes[i])
	}
}
