//go:build integration

package repository

import (
	"context"
	"database/sql"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/H3nSte1n/recipe/internal/domain"
	internalErr "github.com/H3nSte1n/recipe/internal/errors"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func plannedMealTestDatabase(t *testing.T) *gorm.DB {
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
	// The schema is dropped below, so only ever do that on this machine.
	if host := u.Hostname(); host != "localhost" && host != "127.0.0.1" && host != "::1" {
		t.Fatalf("refusing to reset a database on remote host %q", host)
	}
	// Older down migrations are intentionally empty. Reset only the explicitly
	// named disposable database so this test remains repeatable.
	conn, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	_, err = conn.Exec(`DROP SCHEMA public CASCADE; CREATE SCHEMA public`)
	require.NoError(t, err)
	require.NoError(t, conn.Close())
	_, file, _, _ := runtime.Caller(0)
	migrationDir := filepath.Join(filepath.Dir(file), "../../migrations")
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

func plannedMealFixture(t *testing.T, db *gorm.DB) (u, v, r, s string) {
	t.Helper()
	u, v, r, s = uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	for _, x := range []struct{ id, email string }{{u, "one@test.invalid"}, {v, "two@test.invalid"}} {
		require.NoError(t, db.Exec(`INSERT INTO users(id,email,password_hash) VALUES (?,?,?)`, x.id, x.email, "hash").Error)
	}
	for _, x := range []struct{ id, user, title string }{{r, u, "Pasta"}, {s, v, "Soup"}} {
		require.NoError(t, db.Exec(`INSERT INTO recipes(id,user_id,title,source_type,servings,status) VALUES (?,?,?,'MANUAL',2,'published')`, x.id, x.user, x.title).Error)
	}
	return u, v, r, s
}

func newMeal(user, recipe, date string) *domain.PlannedMeal {
	return &domain.PlannedMeal{UserID: user, RecipeID: recipe, CookDate: date}
}

func TestPlannedMealMigrationAndRepository(t *testing.T) {
	db := plannedMealTestDatabase(t)
	// The migration can be rolled back and reapplied without touching the recipe table.
	var exists bool
	require.NoError(t, db.Raw(`SELECT to_regclass('public.planned_meals') IS NOT NULL`).Scan(&exists).Error)
	require.True(t, exists)
	_, file, _, _ := runtime.Caller(0)
	m, err := migrate.New("file://"+filepath.Join(filepath.Dir(file), "../../migrations"), os.Getenv("TEST_DATABASE_URL"))
	require.NoError(t, err)
	require.NoError(t, m.Steps(-1))
	require.NoError(t, db.Raw(`SELECT to_regclass('public.planned_meals') IS NOT NULL`).Scan(&exists).Error)
	require.False(t, exists)
	require.NoError(t, m.Steps(1))
	require.NoError(t, db.Raw(`SELECT to_regclass('public.planned_meals') IS NOT NULL`).Scan(&exists).Error)
	require.True(t, exists)
	_, _ = m.Close()

	u, v, r, s := plannedMealFixture(t, db)
	repo := NewPlannedMealRepository(db)
	ctx := context.Background()
	day := "2026-10-25"

	// Recipe ownership: a user can lock only their own recipe, and sees its status.
	_, err = repo.LockOwnedRecipe(ctx, u, s)
	require.ErrorIs(t, err, internalErr.ErrNotFound)
	recipe, err := repo.LockOwnedRecipe(ctx, u, r)
	require.NoError(t, err)
	require.Equal(t, domain.PlannedMealRecipe{ID: r, Title: "Pasta", Status: "published"}, *recipe)

	// The database itself rejects a meal that points at someone else's recipe.
	require.True(t, internalErr.IsInvalid(repo.Create(ctx, newMeal(u, s, day))))

	first := newMeal(u, r, day)
	require.NoError(t, repo.Create(ctx, first))
	require.NotEmpty(t, first.ID)
	got, err := repo.GetByID(ctx, u, first.ID)
	require.NoError(t, err)
	require.Nil(t, got.MealType)
	require.Nil(t, got.Servings)
	require.Nil(t, got.Notes)
	require.Equal(t, day, got.CookDate)
	require.Equal(t, "Pasta", got.Recipe.Title)

	// Several meals can share a date.
	second := newMeal(u, r, day)
	require.NoError(t, repo.Create(ctx, second))
	require.NotEqual(t, first.ID, second.ID)

	// Update switches recipe and fields.
	alternate := uuid.NewString()
	require.NoError(t, db.Exec(`INSERT INTO recipes(id,user_id,title,source_type,servings,status) VALUES (?,?,?,'MANUAL',2,'published')`, alternate, u, "Rice").Error)
	dinner, servings, notes := "dinner", 4, "extra garlic"
	second.RecipeID, second.MealType, second.Servings, second.Notes = alternate, &dinner, &servings, &notes
	require.NoError(t, repo.Update(ctx, second))
	got, err = repo.GetByID(ctx, u, second.ID)
	require.NoError(t, err)
	require.Equal(t, "Rice", got.Recipe.Title)
	require.Equal(t, "dinner", *got.MealType)
	require.Equal(t, 4, *got.Servings)
	require.Equal(t, "extra garlic", *got.Notes)

	// Other users cannot read, change or delete a meal; failures are indistinguishable from "missing".
	_, err = repo.GetByID(ctx, v, first.ID)
	require.ErrorIs(t, err, internalErr.ErrNotFound)
	foreign := newMeal(v, s, day)
	foreign.ID = first.ID
	require.ErrorIs(t, repo.Update(ctx, foreign), internalErr.ErrNotFound)
	require.ErrorIs(t, repo.Delete(ctx, v, first.ID), internalErr.ErrNotFound)
	// Reassigning to a recipe the user does not own violates the composite foreign key.
	stolen := *first
	stolen.RecipeID = s
	require.True(t, internalErr.IsInvalid(repo.Update(ctx, &stolen)))

	// Listing: inclusive range, stable order, no repeats across page boundaries.
	before := newMeal(u, r, "2026-10-24")
	require.NoError(t, repo.Create(ctx, before))
	items, total, err := repo.List(ctx, u, "2026-10-24", day, 2, 0)
	require.NoError(t, err)
	require.Equal(t, int64(3), total)
	require.Len(t, items, 2)
	require.Equal(t, before.ID, items[0].ID)
	items2, total, err := repo.List(ctx, u, "2026-10-24", day, 2, 2)
	require.NoError(t, err)
	require.Equal(t, int64(3), total)
	require.Len(t, items2, 1)
	seen := map[string]bool{}
	for _, m := range append(items, items2...) {
		require.False(t, seen[m.ID])
		seen[m.ID] = true
	}
	require.True(t, seen[first.ID])
	require.True(t, seen[second.ID])
	// Another user's meals in the same range never appear in the list or the total.
	otherUsers := newMeal(v, s, day)
	require.NoError(t, repo.Create(ctx, otherUsers))
	mine, total, err := repo.List(ctx, u, "2026-10-24", day, 100, 0)
	require.NoError(t, err)
	require.Equal(t, int64(3), total)
	for _, m := range mine {
		require.NotEqual(t, otherUsers.ID, m.ID)
	}
	theirs, total, err := repo.List(ctx, v, "2026-10-24", day, 100, 0)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, theirs, 1)
	require.Equal(t, otherUsers.ID, theirs[0].ID)
	// Counting is per user; the locking read returns the same meal and keeps ownership scoping.
	count, err := repo.CountByUser(ctx, u)
	require.NoError(t, err)
	require.Equal(t, int64(3), count)
	count, err = repo.CountByUser(ctx, v)
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
	locked, err := repo.GetByIDForUpdate(ctx, u, first.ID)
	require.NoError(t, err)
	require.Equal(t, first.ID, locked.ID)
	require.Equal(t, "Pasta", locked.Recipe.Title)
	_, err = repo.GetByIDForUpdate(ctx, v, first.ID)
	require.ErrorIs(t, err, internalErr.ErrNotFound)
	empty, total, err := repo.List(ctx, u, "2030-01-01", "2030-01-02", 100, 0)
	require.NoError(t, err)
	require.Equal(t, int64(0), total)
	require.NotNil(t, empty)
	require.Empty(t, empty)

	// Direct SQL cannot bypass ownership, enum, serving, or note checks.
	for _, q := range []string{
		`INSERT INTO planned_meals(user_id,recipe_id,cook_date) VALUES ('` + u + `','` + s + `','2026-10-25')`,
		`INSERT INTO planned_meals(user_id,recipe_id,cook_date,meal_type) VALUES ('` + u + `','` + r + `','2026-10-25','brunch')`,
		`INSERT INTO planned_meals(user_id,recipe_id,cook_date,servings) VALUES ('` + u + `','` + r + `','2026-10-25',101)`,
	} {
		require.Error(t, db.Exec(q).Error)
	}
	require.Error(t, db.Exec(`INSERT INTO planned_meals(user_id,recipe_id,cook_date,notes) VALUES (?,?,CAST(? AS DATE),?)`,
		u, r, day, strings.Repeat("é", 1001)).Error)
	// Ownership cannot be changed while plans reference the recipe.
	require.Error(t, db.Exec(`UPDATE recipes SET user_id=? WHERE id=?`, v, r).Error)

	// Recipe title and status changes show through; the status is what the service checks.
	require.NoError(t, db.Exec(`UPDATE recipes SET title='New title', status='draft' WHERE id=?`, r).Error)
	updated, err := repo.GetByID(ctx, u, first.ID)
	require.NoError(t, err)
	require.Equal(t, "New title", updated.Recipe.Title)
	require.Equal(t, "draft", updated.Recipe.Status)
	recipe, err = repo.LockOwnedRecipe(ctx, u, r)
	require.NoError(t, err)
	require.Equal(t, "draft", recipe.Status)

	// The date is a calendar day whatever the session timezone.
	for _, tz := range []string{"Pacific/Kiritimati", "America/Los_Angeles"} {
		require.NoError(t, db.Exec(`SET TIME ZONE '`+tz+`'`).Error)
		got, err := repo.GetByID(ctx, u, first.ID)
		require.NoError(t, err)
		require.Equal(t, day, got.CookDate)
	}

	// Deleting a recipe or a user removes the plans that depend on it.
	require.NoError(t, db.Exec(`DELETE FROM recipes WHERE id=?`, r).Error)
	_, err = repo.GetByID(ctx, u, first.ID)
	require.ErrorIs(t, err, internalErr.ErrNotFound)
	_, err = repo.GetByID(ctx, u, second.ID)
	require.NoError(t, err)
	ownedByV := newMeal(v, s, day)
	require.NoError(t, repo.Create(ctx, ownedByV))
	require.NoError(t, db.Exec(`DELETE FROM users WHERE id=?`, v).Error)
	_, err = repo.GetByID(ctx, v, ownedByV.ID)
	require.ErrorIs(t, err, internalErr.ErrNotFound)

	require.NoError(t, repo.Delete(ctx, u, second.ID))
	require.ErrorIs(t, repo.Delete(ctx, u, second.ID), internalErr.ErrNotFound)
}

func TestPlannedMealTransactionRollsBack(t *testing.T) {
	db := plannedMealTestDatabase(t)
	u, _, r, _ := plannedMealFixture(t, db)
	repo := NewPlannedMealRepository(db)
	ctx := context.Background()
	meal := newMeal(u, r, "2026-10-25")

	err := repo.WithTypedTransaction(ctx, func(tx PlannedMealRepository) error {
		if _, err := tx.LockOwnedRecipe(ctx, u, r); err != nil {
			return err
		}
		if err := tx.Create(ctx, meal); err != nil {
			return err
		}
		return internalErr.New("abort", "INVALID")
	})
	require.True(t, internalErr.IsInvalid(err))
	_, err = repo.GetByID(ctx, u, meal.ID)
	require.ErrorIs(t, err, internalErr.ErrNotFound)
}

// A recipe delete from another connection must wait for the planning transaction instead of
// slipping in between the ownership check and the insert.
func TestPlannedMealRecipeDeleteWaitsForPlanningTransaction(t *testing.T) {
	db := plannedMealTestDatabase(t)
	u, _, r, _ := plannedMealFixture(t, db)
	repo := NewPlannedMealRepository(db)
	ctx := context.Background()

	other, err := gorm.Open(postgres.Open(os.Getenv("TEST_DATABASE_URL")), &gorm.Config{})
	require.NoError(t, err)
	otherSQL, err := other.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = otherSQL.Close() })

	meal := newMeal(u, r, "2026-10-25")
	deleted := make(chan error, 1)
	err = repo.WithTypedTransaction(ctx, func(tx PlannedMealRepository) error {
		if _, err := tx.LockOwnedRecipe(ctx, u, r); err != nil {
			return err
		}
		go func() { deleted <- other.Exec(`DELETE FROM recipes WHERE id = ?`, r).Error }()
		select {
		case err := <-deleted:
			t.Fatalf("recipe was deleted while the planning transaction held it (err=%v)", err)
		case <-time.After(500 * time.Millisecond):
		}
		return tx.Create(ctx, meal)
	})
	require.NoError(t, err)

	select {
	case err := <-deleted:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("delete never ran after the planning transaction committed")
	}
	// The delete cascades to the meal that was just planned, so nothing is left dangling.
	_, err = repo.GetByID(ctx, u, meal.ID)
	require.ErrorIs(t, err, internalErr.ErrNotFound)
}

// Two updates of the same meal must run one after the other: the second waits for the row lock
// instead of comparing against a recipe the first update is about to change.
func TestPlannedMealUpdateWaitsForRowLock(t *testing.T) {
	db := plannedMealTestDatabase(t)
	u, _, r, _ := plannedMealFixture(t, db)
	repo := NewPlannedMealRepository(db)
	ctx := context.Background()
	meal := newMeal(u, r, "2026-10-25")
	require.NoError(t, repo.Create(ctx, meal))

	other, err := gorm.Open(postgres.Open(os.Getenv("TEST_DATABASE_URL")), &gorm.Config{})
	require.NoError(t, err)
	otherSQL, err := other.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = otherSQL.Close() })
	otherRepo := NewPlannedMealRepository(other)

	locked := make(chan error, 1)
	err = repo.WithTypedTransaction(ctx, func(tx PlannedMealRepository) error {
		if _, err := tx.GetByIDForUpdate(ctx, u, meal.ID); err != nil {
			return err
		}
		go func() {
			locked <- otherRepo.WithTypedTransaction(ctx, func(tx2 PlannedMealRepository) error {
				_, err := tx2.GetByIDForUpdate(ctx, u, meal.ID)
				return err
			})
		}()
		select {
		case err := <-locked:
			t.Fatalf("second update got the meal while the first held its lock (err=%v)", err)
		case <-time.After(500 * time.Millisecond):
		}
		return nil
	})
	require.NoError(t, err)

	select {
	case err := <-locked:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("second update never ran after the first transaction committed")
	}
}
