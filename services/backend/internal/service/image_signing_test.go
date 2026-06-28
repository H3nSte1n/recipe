package service

import (
	"testing"
	"time"

	"github.com/H3nSte1n/recipe/internal/domain"
	"github.com/H3nSte1n/recipe/pkg/signedurl"
	"github.com/stretchr/testify/assert"
)

func TestSignRecipeImages_SignsRecipeAndNested(t *testing.T) {
	signer := signedurl.NewSigner("test-secret", time.Hour)
	s := &recipeService{imageSigner: signer}

	child := &domain.Recipe{ID: "child", ImageURL: "http://localhost:8080/uploads/child.png"}
	r := &domain.Recipe{
		ID:       "parent",
		ImageURL: "http://localhost:8080/uploads/parent.png",
		SubRecipes: []domain.SubRecipe{
			{ChildID: "child", Child: child},
		},
	}

	s.signRecipeImages(r)

	assert.Contains(t, r.ImageURL, "sig=", "top-level image must be signed")
	assert.Contains(t, r.ImageURL, "exp=")
	assert.Contains(t, r.SubRecipes[0].Child.ImageURL, "sig=", "nested sub-recipe image must be signed")
}

func TestSignRecipeImages_LeavesExternalURLs(t *testing.T) {
	signer := signedurl.NewSigner("test-secret", time.Hour)
	s := &recipeService{imageSigner: signer}

	external := "https://example.com/images/photo.jpg"
	r := &domain.Recipe{ID: "r", ImageURL: external}
	s.signRecipeImages(r)

	assert.Equal(t, external, r.ImageURL, "external image URLs must not be signed")
}

func TestSignRecipeImages_NilSignerIsNoOp(t *testing.T) {
	s := &recipeService{imageSigner: nil}
	r := &domain.Recipe{ID: "r", ImageURL: "http://localhost:8080/uploads/a.png"}
	s.signRecipeImages(r)
	assert.Equal(t, "http://localhost:8080/uploads/a.png", r.ImageURL)
}

func TestSignRecipeImages_HandlesCycle(t *testing.T) {
	signer := signedurl.NewSigner("test-secret", time.Hour)
	s := &recipeService{imageSigner: signer}

	// parent <-> child cycle must not loop forever.
	parent := &domain.Recipe{ID: "p", ImageURL: "http://localhost:8080/uploads/p.png"}
	child := &domain.Recipe{ID: "c", ImageURL: "http://localhost:8080/uploads/c.png"}
	parent.SubRecipes = []domain.SubRecipe{{Child: child}}
	child.SubRecipes = []domain.SubRecipe{{Parent: parent}}

	s.signRecipeImages(parent)
	assert.Contains(t, parent.ImageURL, "sig=")
	assert.Contains(t, child.ImageURL, "sig=")
}
