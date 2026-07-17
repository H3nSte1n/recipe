package repository

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/H3nSte1n/recipe/internal/domain"
	"github.com/stretchr/testify/require"
)

// TestUserRepository_RecordFailedLogin_PersistsAndClears proves RecordFailedLogin's atomic
// UPDATE...RETURNING correctly increments failed_login_attempts, sets locked_until once the
// threshold is reached, and that ResetLoginLockout's map-based Updates writes a NULL
// locked_until back — GORM's map-based Updates (unlike struct-based Updates) must not skip the
// nil value or the account would stay locked forever after a reset.
func TestUserRepository_RecordFailedLogin_PersistsAndClears(t *testing.T) {
	db := openTestDB(t)
	require.NoError(t, db.AutoMigrate(&domain.User{}))

	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &domain.User{ID: "u1", Email: "a@b.com", PasswordHash: "x"}
	require.NoError(t, repo.Create(ctx, user))

	lockUntil := time.Now().Add(15 * time.Minute).Truncate(time.Second)
	attempts, locked, err := repo.RecordFailedLogin(ctx, user.ID, 5, lockUntil)
	require.NoError(t, err)
	require.Equal(t, 1, attempts, "first failure should not yet reach the threshold")
	require.Nil(t, locked)

	got, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, 1, got.FailedLoginAttempts)
	require.Nil(t, got.LockedUntil)

	// Four more failures reach the threshold (5) and should set locked_until.
	for i := 0; i < 4; i++ {
		attempts, locked, err = repo.RecordFailedLogin(ctx, user.ID, 5, lockUntil)
		require.NoError(t, err)
	}
	require.Equal(t, 5, attempts)
	require.NotNil(t, locked)

	got, err = repo.GetByID(ctx, user.ID)
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

// TestUserRepository_RecordFailedLogin_ConcurrentAttemptsDoNotLoseIncrements proves the fix for
// the read-modify-write race: firing many concurrent failed logins for the same account must
// accumulate the full count, not collapse to ~1 the way a stale in-memory read-then-write would.
func TestUserRepository_RecordFailedLogin_ConcurrentAttemptsDoNotLoseIncrements(t *testing.T) {
	db := openTestDB(t)
	require.NoError(t, db.AutoMigrate(&domain.User{}))
	// A single connection is required here: SQLite serializes writes at the connection/file
	// level regardless of the UPDATE's own atomicity, so this test's job is to prove the SQL
	// statement itself is correct (a single atomic increment expression), not to reproduce
	// Postgres's row-level concurrency. The concurrent goroutines still exercise that no
	// value gets silently dropped when RecordFailedLogin is called back-to-back rapidly.
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &domain.User{ID: "u1", Email: "a@b.com", PasswordHash: "x"}
	require.NoError(t, repo.Create(ctx, user))

	const n = 20
	lockUntil := time.Now().Add(15 * time.Minute)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := repo.RecordFailedLogin(ctx, user.ID, 1000, lockUntil)
			require.NoError(t, err)
		}()
	}
	wg.Wait()

	got, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, n, got.FailedLoginAttempts, "every concurrent failed login must be counted, none lost to a stale read-modify-write")
}
