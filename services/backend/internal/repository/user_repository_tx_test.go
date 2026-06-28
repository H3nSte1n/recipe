package repository

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/H3nSte1n/recipe/internal/domain"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// openTestDB opens a temp-file sqlite database. A file (not :memory:) is used so
// every pooled connection sees the same database — :memory: gives each
// connection its own empty DB, which would make a transaction test pass for the
// wrong reason.
func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "test.db")
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	return db
}

// TestUserRepository_WithTypedTransaction_RollsBackOnFailure proves the
// transaction is real: a failure partway through the closure must undo earlier
// writes. We migrate only the users table (not profiles), so CreateProfile
// fails; the user row must NOT survive. This test FAILS if the closure writes on
// the non-transactional connection (the old broken RunTx), and PASSES with
// WithTypedTransaction threading the tx repo.
func TestUserRepository_WithTypedTransaction_RollsBackOnFailure(t *testing.T) {
	db := openTestDB(t)
	// Deliberately migrate ONLY users — no profiles table.
	require.NoError(t, db.AutoMigrate(&domain.User{}))

	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &domain.User{ID: "u1", Email: "a@b.com", PasswordHash: "x"}
	profile := &domain.Profile{ID: "p1", UserID: "u1", Bio: "hi"}

	err := repo.WithTypedTransaction(ctx, func(txRepo UserRepository) error {
		if err := txRepo.Create(ctx, user); err != nil {
			return err
		}
		// Fails: profiles table does not exist -> whole tx must roll back.
		return txRepo.CreateProfile(ctx, profile)
	})
	require.Error(t, err, "closure should surface the CreateProfile failure")

	var count int64
	require.NoError(t, db.Model(&domain.User{}).Where("id = ?", "u1").Count(&count).Error)
	assert.Equal(t, int64(0), count, "user row must be rolled back, not orphaned")
}

// TestUserRepository_WithTypedTransaction_CommitsOnSuccess is the positive case:
// when the closure succeeds, both rows persist.
func TestUserRepository_WithTypedTransaction_CommitsOnSuccess(t *testing.T) {
	db := openTestDB(t)
	require.NoError(t, db.AutoMigrate(&domain.User{}, &domain.Profile{}))

	repo := NewUserRepository(db)
	ctx := context.Background()

	err := repo.WithTypedTransaction(ctx, func(txRepo UserRepository) error {
		if err := txRepo.Create(ctx, &domain.User{ID: "u1", Email: "a@b.com", PasswordHash: "x"}); err != nil {
			return err
		}
		return txRepo.CreateProfile(ctx, &domain.Profile{ID: "p1", UserID: "u1", Bio: "hi"})
	})
	require.NoError(t, err)

	var users, profiles int64
	require.NoError(t, db.Model(&domain.User{}).Count(&users).Error)
	require.NoError(t, db.Model(&domain.Profile{}).Count(&profiles).Error)
	assert.Equal(t, int64(1), users)
	assert.Equal(t, int64(1), profiles)
}
