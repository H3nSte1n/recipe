//go:build integration

package postgres

import (
	"context"
	"database/sql"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/H3nSte1n/recipe/internal/mealplan/application"
	"github.com/H3nSte1n/recipe/internal/mealplan/domain"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func testDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Fatal("TEST_DATABASE_URL is required for integration tests")
	}
	u, err := url.Parse(dsn)
	require.NoError(t, err)
	if u.Path != "/recipe_mealplan_test" {
		t.Fatalf("refusing migrations on non-isolated database %q", u.Path)
	}
	// Older down migrations are intentionally empty. Reset only the explicitly
	// named disposable database so this test remains repeatable.
	conn, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	_, err = conn.Exec(`DROP SCHEMA public CASCADE; CREATE SCHEMA public`)
	require.NoError(t, err)
	require.NoError(t, conn.Close())
	_, file, _, _ := runtime.Caller(0)
	migrationDir := filepath.Join(filepath.Dir(file), "../../../../migrations")
	m, err := migrate.New("file://"+migrationDir, dsn)
	require.NoError(t, err)
	defer m.Close()
	require.NoError(t, m.Up())
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

func fixture(t *testing.T, db *gorm.DB) (uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	u, v, r, s := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	for _, x := range []struct {
		id    uuid.UUID
		email string
	}{{u, "one@test.invalid"}, {v, "two@test.invalid"}} {
		require.NoError(t, db.Exec(`INSERT INTO users(id,email,password_hash) VALUES (?,?,?)`, x.id, x.email, "hash").Error)
	}
	for _, x := range []struct {
		id, user uuid.UUID
		title    string
	}{{r, u, "Pasta"}, {s, v, "Soup"}} {
		require.NoError(t, db.Exec(`INSERT INTO recipes(id,user_id,title,source_type,servings,status) VALUES (?,?,?,'MANUAL',2,'published')`, x.id, x.user, x.title).Error)
	}
	return u, v, r, s
}

func TestMigrationAndRepositoryIntegration(t *testing.T) {
	db := testDatabase(t)
	// The migration can be rolled back and reapplied without touching the recipe table.
	var exists bool
	require.NoError(t, db.Raw(`SELECT to_regclass('public.planned_meals') IS NOT NULL`).Scan(&exists).Error)
	require.True(t, exists)
	_, file, _, _ := runtime.Caller(0)
	m, err := migrate.New("file://"+filepath.Join(filepath.Dir(file), "../../../../migrations"), os.Getenv("TEST_DATABASE_URL"))
	require.NoError(t, err)
	require.NoError(t, m.Steps(-1))
	require.NoError(t, db.Raw(`SELECT to_regclass('public.planned_meals') IS NOT NULL`).Scan(&exists).Error)
	require.False(t, exists)
	require.NoError(t, m.Steps(1))
	require.NoError(t, db.Raw(`SELECT to_regclass('public.planned_meals') IS NOT NULL`).Scan(&exists).Error)
	require.True(t, exists)
	_, _ = m.Close()

	u, v, r, s := fixture(t, db)
	repo := NewRepository(db)
	ctx := context.Background()
	day := domain.Date("2026-10-25")
	_, err = repo.CreateWithOwnedRecipe(ctx, u, domain.Fields{RecipeID: s, CookDate: day})
	require.ErrorIs(t, err, application.ErrNotFound)
	first, err := repo.CreateWithOwnedRecipe(ctx, u, domain.Fields{RecipeID: r, CookDate: day})
	require.NoError(t, err)
	require.Nil(t, first.MealType)
	require.Nil(t, first.Servings)
	require.Nil(t, first.Notes)
	second, err := repo.CreateWithOwnedRecipe(ctx, u, domain.Fields{RecipeID: r, CookDate: day})
	require.NoError(t, err)
	alternateRecipe := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO recipes(id,user_id,title,source_type,servings,status) VALUES (?,?,?,'MANUAL',2,'published')`,
		alternateRecipe, u, "Rice").Error)
	second, err = repo.ReplaceWithOwnedRecipe(ctx, u, second.ID, domain.Fields{RecipeID: alternateRecipe, CookDate: day})
	require.NoError(t, err)
	require.Equal(t, alternateRecipe, second.RecipeID)
	require.Equal(t, "Rice", second.Recipe.Title)
	before, err := repo.CreateWithOwnedRecipe(ctx, u, domain.Fields{RecipeID: r, CookDate: "2026-10-24"})
	require.NoError(t, err)
	_, err = repo.Get(ctx, v, first.ID)
	require.ErrorIs(t, err, application.ErrNotFound)
	_, err = repo.ReplaceWithOwnedRecipe(ctx, v, first.ID, domain.Fields{RecipeID: s, CookDate: day})
	require.ErrorIs(t, err, application.ErrNotFound)
	_, err = repo.ReplaceWithOwnedRecipe(ctx, u, first.ID, domain.Fields{RecipeID: s, CookDate: day})
	require.ErrorIs(t, err, application.ErrNotFound)
	require.ErrorIs(t, repo.Delete(ctx, v, first.ID), application.ErrNotFound)
	items, total, err := repo.List(ctx, u, "2026-10-24", day, 1, 2)
	require.NoError(t, err)
	require.Equal(t, int64(3), total)
	require.Len(t, items, 2)
	require.Equal(t, before.ID, items[0].ID)
	items2, total, err := repo.List(ctx, u, "2026-10-24", day, 2, 2)
	require.NoError(t, err)
	require.Equal(t, int64(3), total)
	require.Len(t, items2, 1)
	// Stable ordering and page boundaries must not repeat a meal.
	seen := map[uuid.UUID]bool{}
	for _, m := range append(items, items2...) {
		require.False(t, seen[m.ID])
		seen[m.ID] = true
	}
	require.True(t, seen[first.ID])
	require.True(t, seen[second.ID])
	// Direct SQL cannot bypass ownership, enum, serving, or note checks.
	for _, q := range []string{
		`INSERT INTO planned_meals(user_id,recipe_id,cook_date) VALUES ('` + u.String() + `','` + s.String() + `','2026-10-25')`,
		`INSERT INTO planned_meals(user_id,recipe_id,cook_date,meal_type) VALUES ('` + u.String() + `','` + r.String() + `','2026-10-25','brunch')`,
		`INSERT INTO planned_meals(user_id,recipe_id,cook_date,servings) VALUES ('` + u.String() + `','` + r.String() + `','2026-10-25',101)`,
	} {
		require.Error(t, db.Exec(q).Error)
	}
	require.Error(t, db.Exec(`INSERT INTO planned_meals(user_id,recipe_id,cook_date,notes) VALUES (?,?,CAST(? AS DATE),?)`,
		u, r, day.String(), strings.Repeat("é", 1001)).Error)
	// Ownership cannot be changed while plans reference the recipe.
	require.Error(t, db.Exec(`UPDATE recipes SET user_id=? WHERE id=?`, v, r).Error)
	require.NoError(t, db.Exec(`UPDATE recipes SET title='New title', status='archived' WHERE id=?`, r).Error)
	updated, err := repo.Get(ctx, u, first.ID)
	require.NoError(t, err)
	require.Equal(t, "New title", updated.Recipe.Title)
	require.Equal(t, "archived", updated.Recipe.Status)
	for _, tz := range []string{"Pacific/Kiritimati", "America/Los_Angeles"} {
		require.NoError(t, db.Exec(`SET TIME ZONE '`+tz+`'`).Error)
		got, err := repo.Get(ctx, u, first.ID)
		require.NoError(t, err)
		require.Equal(t, day, got.CookDate)
	}
	require.NoError(t, db.Exec(`DELETE FROM recipes WHERE id=?`, r).Error)
	_, err = repo.Get(ctx, u, first.ID)
	require.ErrorIs(t, err, application.ErrNotFound)
	_, err = repo.Get(ctx, u, second.ID)
	require.NoError(t, err)
	ownedByV, err := repo.CreateWithOwnedRecipe(ctx, v, domain.Fields{RecipeID: s, CookDate: day})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`DELETE FROM users WHERE id=?`, v).Error)
	_, err = repo.Get(ctx, v, ownedByV.ID)
	require.ErrorIs(t, err, application.ErrNotFound)
}
