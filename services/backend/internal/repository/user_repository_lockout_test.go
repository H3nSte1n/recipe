package repository

import (
	"context"
	"testing"
	"time"

	"github.com/H3nSte1n/recipe/internal/domain"
	"github.com/stretchr/testify/require"
)

// TestUserRepository_SetLoginLockoutState_PersistsAndClears proves the map-based Updates call
// in SetLoginLockoutState correctly writes both a non-zero locked_until and, on reset, a NULL
// locked_until — GORM's map-based Updates (unlike struct-based Updates) must not skip the nil
// value or the account would stay locked forever after ResetLoginLockout.
func TestUserRepository_SetLoginLockoutState_PersistsAndClears(t *testing.T) {
	db := openTestDB(t)
	require.NoError(t, db.AutoMigrate(&domain.User{}))

	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &domain.User{ID: "u1", Email: "a@b.com", PasswordHash: "x"}
	require.NoError(t, repo.Create(ctx, user))

	lockUntil := time.Now().Add(15 * time.Minute).Truncate(time.Second)
	require.NoError(t, repo.SetLoginLockoutState(ctx, user.ID, 5, &lockUntil))

	got, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, 5, got.FailedLoginAttempts)
	require.NotNil(t, got.LockedUntil)
	require.WithinDuration(t, lockUntil, *got.LockedUntil, time.Second)

	require.NoError(t, repo.ResetLoginLockout(ctx, user.ID))

	got, err = repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, 0, got.FailedLoginAttempts)
	require.Nil(t, got.LockedUntil, "locked_until must be cleared back to NULL, not left stale")
}
