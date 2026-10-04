package domain

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestParseDate(t *testing.T) {
	for _, value := range []string{"2024-02-29", "2026-10-03", "2026-03-29", "2026-10-25"} {
		got, err := ParseDate(value)
		if err != nil || got.String() != value {
			t.Fatalf("%q: %v, %q", value, err, got)
		}
	}
	for _, value := range []string{"", "0000-01-01", "2025-02-29", "2024-02-30", "2026-13-01", "2026-1-01", "2026-01-1", "2026-01-01T00:00:00Z", "2026-0a-01"} {
		if _, err := ParseDate(value); err == nil {
			t.Errorf("accepted %q", value)
		}
	}
}

func TestValidateRange(t *testing.T) {
	for _, tc := range []struct {
		from, to string
		good     bool
	}{
		{"2026-01-01", "2026-01-01", true},
		{"2026-01-01", "2026-04-03", true}, // 93 inclusive
		{"2026-01-01", "2026-04-04", false},
		{"2026-04-01", "2026-03-31", false},
	} {
		err := ValidateRange(Date(tc.from), Date(tc.to))
		if (err == nil) != tc.good {
			t.Errorf("%s..%s: %v", tc.from, tc.to, err)
		}
	}
}

func TestFieldsValidateAndNulls(t *testing.T) {
	id := uuid.New()
	f := Fields{RecipeID: id, CookDate: Date("2026-10-03")}
	if err := f.Validate(); err != nil {
		t.Fatal(err)
	}
	if f.MealType != nil || f.Servings != nil || f.Notes != nil {
		t.Fatal("optional values should remain nil")
	}
	meal, servings, notes := "dinner", 100, strings.Repeat("é", 1000)
	f.MealType, f.Servings, f.Notes = &meal, &servings, &notes
	if err := f.Validate(); err != nil {
		t.Fatal(err)
	}
	servings = 101
	if f.Validate() == nil {
		t.Fatal("accepted servings over bound")
	}
	servings = 1
	meal = "brunch"
	if f.Validate() == nil {
		t.Fatal("accepted invalid meal type")
	}
	meal = "dinner"
	notes += "é"
	if f.Validate() == nil {
		t.Fatal("accepted long notes")
	}
	f.RecipeID = uuid.Nil
	if f.Validate() == nil {
		t.Fatal("accepted missing recipe")
	}
}
