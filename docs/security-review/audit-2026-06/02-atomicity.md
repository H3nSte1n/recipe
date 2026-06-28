# Audit 02 — Transactional Atomicity (PR #35)

**Scope:** Verify `RunTx` was removed, `WithTypedTransaction` is the typed-transaction helper used by all transactional callers, and that mid-transaction failures cannot leave orphaned rows.

**Method:** Static code read + grep + Go unit tests (sqlite-backed, no external DB). No production code modified.

---

## Check 1 — `RunTx` is DELETED

`grep -rn "RunTx" services/backend/` returns **3 hits, all in test files, all referring to the OLD removed helper by name in comments / test-case descriptions** — there is NO definition and NO caller of `RunTx` in production code:

```
internal/repository/user_repository_tx_test.go:31:// the non-transactional connection (the old broken RunTx), and PASSES with
internal/service/user_service_test.go:175:  name: "returns error when RunTx returns err",
internal/service/user_service_test.go:498:  name: "returns error when RunTx fails",
```

Confirmed via `grep -rn "RunTx\|func.*Transaction" internal/repository/*.go` (excluding tests): the only transaction primitives are `BaseRepository.RunInTransaction` and the per-repo `WithTypedTransaction`. No `RunTx` symbol exists.

**VERDICT: PASS** — `RunTx` is gone from production code; remaining textual references are historical comments/test names.

---

## Check 2 — `WithTypedTransaction` used by ALL transactional callers

The typed helper is defined on every repository that does multi-write operations and is backed by `BaseRepository.RunInTransaction`:

- `internal/repository/base_repository.go:16-20` — `RunInTransaction` → `r.DB.WithContext(ctx).Transaction(fn)` (GORM managed tx)
- `internal/repository/user_repository.go:35-40` — `WithTypedTransaction` threads `NewBaseRepository(tx)` into the closure
- `internal/repository/shopping_list_repository.go:32-37` — same pattern
- `internal/repository/recipe_repository.go:31-36` — same pattern
- `internal/repository/ai_config_repository.go:34-39` — same pattern

All three required multi-write call sites run inside ONE transaction via the tx-scoped repo:

1. **User register (user + profile)** — `internal/service/user_service.go:71-99`
   Inside the closure: `txRepo.Create(ctx, user)` (line 85) then `txRepo.CreateProfile(ctx, profile)` (line 98). Both writes go through the tx-scoped repo.

2. **Shopping-list create (list + items)** — `internal/service/shopping_list_service.go:78-106`
   Inside the closure: `txRepo.Create(ctx, list)` (line 79) then `txRepo.AddItems(ctx, items)` (line 100). Both write through `r.DB` which IS the tx connection (`shopping_list_repository.go:78 Create`, `:82-84 AddItems` both use `r.DB.WithContext(ctx).Create(...)`).

3. **Recipe create (recipe + nested sub-recipes)** — `internal/service/recipe_service.go:142-168`
   Inside the closure: `txRepo.Create(ctx, recipe)` (line 143) then, when sub-recipes present, `txRepo.Update(ctx, recipe)` (line 159) to persist the nested `SubRecipes`. Both via the tx-scoped repo.

   (Additional transactional callers also use the helper: `recipe_service.go:241, :319`, `ai_config_service.go:57, :93`, `user_service.go:177, :213` — all `WithTypedTransaction`.)

Note (recipe path only): `RecipeRepositoryImpl.Create`/`Update` (`recipe_repository.go:38-45`) themselves re-enter `RunInTransaction`. When invoked through the outer `WithTypedTransaction` closure, GORM opens these as **savepoints nested inside the outer transaction**, not a second independent transaction — so atomicity holds (an outer rollback discards the savepoint). The user-register and shopping-list paths write directly through the tx connection with no nesting.

Build-confirmation: `cd services/backend && go test ./...` passes for all packages (handler, service, repository, pkg/*), proving the whole backend compiles against the new typed-transaction API and every service-layer mock satisfies the `WithTypedTransaction` interface — i.e. no caller was stranded by the `RunTx` removal.

**VERDICT: PASS** — `WithTypedTransaction` is the single typed-transaction helper and is used by every multi-write call site; each runs its writes through the tx-scoped repository inside one transaction. Build-confirmed by a clean full `go test ./...`.

---

## Check 3 — No orphaned rows on mid-transaction failure

### Code-level rollback guarantee (cited)

- `internal/repository/base_repository.go:16-20`:
  ```go
  func (r *BaseRepository) RunInTransaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
      return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
          return fn(tx)
      })
  }
  ```
  GORM's `db.Transaction(fn)` **commits only if `fn` returns nil and automatically rolls back the entire transaction if `fn` returns any non-nil error** (or panics). Every `WithTypedTransaction` wrapper (`user_repository.go:35`, `shopping_list_repository.go:32`, `recipe_repository.go:31`) builds the closure repo from the SAME `tx` (`NewBaseRepository(tx)`), so every write inside the closure participates in that one managed transaction. Any step returning `err` propagates out of `fn` → GORM aborts the commit and rolls back all prior writes.

### Traced paths

- **User register (user-without-profile impossible):** `user_service.go:71` — if `CreateProfile` (line 98) errors, the closure returns the error, `RunInTransaction` rolls back the already-inserted user row. No user-without-profile can persist.
- **Shopping-list (list-without-items impossible):** `shopping_list_service.go:78` — if `AddItems` (line 100) errors, the closure returns the error and the already-inserted list row is rolled back. No empty orphaned list can persist.

### Empirical test evidence

`go test ./internal/repository/ -run TestUserRepository_WithTypedTransaction -v`:

```
--- PASS: TestUserRepository_WithTypedTransaction_RollsBackOnFailure (0.00s)
--- PASS: TestUserRepository_WithTypedTransaction_CommitsOnSuccess (0.00s)
PASS
ok  github.com/H3nSte1n/recipe/internal/repository 0.620s
```

`TestUserRepository_WithTypedTransaction_RollsBackOnFailure` (`user_repository_tx_test.go:33-56`) migrates ONLY the `users` table, forcing `CreateProfile` to fail with `no such table: profiles`; it then asserts the user row count is `0` — proving the earlier `Create(user)` write was rolled back, not orphaned. It uses a temp-FILE sqlite DB (not `:memory:`) so the assertion connection sees the same DB the tx wrote to, making the rollback proof valid. Test passes. The commit-on-success counterpart also passes (both rows persist).

Live fault-injection against the real Postgres path was NOT performed (no running DB in the audit harness); the guarantee rests on the GORM-managed-transaction code proof + the sqlite rollback test, which exercises the identical `WithTypedTransaction`/`RunInTransaction` code path.

**VERDICT: PASS (code-verified + unit-test-proven; live Postgres fault-injection INCONCLUSIVE-but-code-verified)** — rollback is guaranteed by GORM managed transactions (`base_repository.go:16-20`), traced on both the user-register and shopping-list paths, and empirically demonstrated by a passing rollback unit test.

---

## Summary Verdicts

- Check 1 (RunTx deleted): **PASS** — zero production references; 3 grep hits are historical test comments/names only.
- Check 2 (WithTypedTransaction used by all transactional callers): **PASS** — user register (`user_service.go:71`), shopping-list create (`shopping_list_service.go:78`), recipe create (`recipe_service.go:142`), plus ai_config/reset paths, all run multi-writes in one tx.
- Check 3 (no orphaned rows on mid-tx failure): **PASS** — GORM managed tx rolls back on any closure error (`base_repository.go:16-20`); user-without-profile and list-without-items paths traced; `TestUserRepository_WithTypedTransaction_RollsBackOnFailure` passes. Live Postgres fault-injection: INCONCLUSIVE-but-code-verified.
