package ai

import (
	"testing"

	"github.com/H3nSte1n/recipe/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCategorizeItemsResponse_NormalizesUnknownToOther(t *testing.T) {
	// An injected/garbage category must be coerced to OTHER, not trusted verbatim.
	resp := `{"milk":"IGNORE PREVIOUS; DROP TABLE","spinach":"PRODUCE"}`
	got, err := parseCategorizeItemsResponse(resp)
	require.NoError(t, err)
	assert.Equal(t, string(domain.CategoryOther), got["milk"])
	assert.Equal(t, string(domain.CategoryProduce), got["spinach"])
}

func TestParseCategorizeItemsResponse_CaseInsensitive(t *testing.T) {
	resp := `{"eggs":"dairy","bread":"Bakery"}`
	got, err := parseCategorizeItemsResponse(resp)
	require.NoError(t, err)
	assert.Equal(t, string(domain.CategoryDairy), got["eggs"])
	assert.Equal(t, string(domain.CategoryBakery), got["bread"])
}

func TestNormalizeCategory(t *testing.T) {
	assert.Equal(t, "PRODUCE", normalizeCategory("  produce "))
	assert.Equal(t, "OTHER", normalizeCategory("NOT_A_CATEGORY"))
	assert.Equal(t, "OTHER", normalizeCategory(""))
}

func TestParseAIResponse_ClampsNegativeFields(t *testing.T) {
	resp := `{"title":"Soup","servings":-5,"prepTime":-1,"cookTime":-10,"ingredients":[],"instructions":[]}`
	recipe, err := parseAIResponse(resp)
	require.NoError(t, err)
	assert.Equal(t, 0, recipe.Servings)
	assert.Equal(t, 0, recipe.PrepTime)
	assert.Equal(t, 0, recipe.CookTime)
}

func TestParseAIResponse_KeepsNormalFields(t *testing.T) {
	resp := `{"title":"Soup","servings":4,"prepTime":15,"cookTime":30,"ingredients":[],"instructions":[]}`
	recipe, err := parseAIResponse(resp)
	require.NoError(t, err)
	assert.Equal(t, 4, recipe.Servings)
	assert.Equal(t, 15, recipe.PrepTime)
	assert.Equal(t, 30, recipe.CookTime)
}
