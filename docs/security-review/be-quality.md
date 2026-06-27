# Backend Code-Quality Review — Architecture, Error Handling, Context & Concurrency

Scope: **code quality, not security** (security is covered in the other `be-*.md` docs). Reviewed the
Go backend under `services/backend/internal/` (`service/`, `repository/`, sampled `handler/` and
`domain/`) plus `pkg/ai`, against the clean-architecture conventions in `services/backend/CLAUDE.md`.
Evaluates: clean-architecture adherence, idiomatic error handling, `context.Context` propagation,
concurrency safety, and general maintainability. `go vet ./...` is clean.

## Findings

### [High] `RunTx` provides no real transactional atomicity; the correct mechanism (`WithTypedTransaction`) is dead code

- **Location:** `internal/repository/user_repository.go:43-47` (and the identical `RunTx` in
  `recipe_repository.go:39-43`, `ai_config_repository.go:42-44`); callers
  `internal/service/user_service.go:70,176,212`, `recipe_service.go:135,229,302`,
  `ai_config_service.go:46,81`. Correct-but-unused mechanism: `WithTypedTransaction`
  (`user_repository.go:36-41`, `recipe_repository.go:32-37`, `ai_config_repository.go:35-40`).
- **Description:** `RunTx` opens a GORM transaction but **ignores the `tx` handle** and just calls
  `fn()`: `r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error { return fn() })`. The closure
  `fn` then calls repository methods on the **captured root repo** (e.g.
  `s.userRepo.Create(ctx, user)` → `r.DB.WithContext(ctx).Create(...)`), which run on a *separate
  pooled connection* and **auto-commit independently** of the outer transaction. The transaction the
  outer `RunTx` opened is therefore empty — it wraps nothing. `WithTypedTransaction` is the correct
  pattern (it rebinds a new repo to the `tx` session and passes it into the callback) but **no service
  uses it**.
- **Impact:** Multi-step writes that look atomic are not.
  - `userService.Register` (`user_service.go:70-98`): the `users` row is committed by its own inner
    statement before `CreateProfile` runs; if profile creation fails, the user persists **without a
    profile**.
  - `recipeService.Create` (`recipe_service.go:135-170`): `recipeRepo.Create` commits in its **own**
    inner `RunInTransaction` (`recipe_repository.go:46`) before the sub-recipe `Update` runs; if the
    `Update` fails, the recipe row is orphaned — and the failure path then deletes the uploaded image
    (`:162-168`), leaving a persisted recipe pointing at a deleted file.
  - `aiConfigService.Create` (`ai_config_service.go:46-66`): `ClearDefaultByUserID` can commit while
    the subsequent `Create` fails, leaving the user with **no default config**.
- **Recommendation:** Replace `RunTx(ctx, func() error {...})` with `WithTypedTransaction`, threading
  the transaction-bound repo into the closure so all inner operations execute on the same `tx`.
  Additionally, the inner repo methods that open their own `RunInTransaction` (e.g.
  `recipe_repository.go:46,52,116`) must accept and use the caller's `tx` rather than `r.DB`, or
  GORM nested-transaction savepoints must be used deliberately. Once migrated, delete `RunTx`.

### [Medium] Repository converts not-found to a plain `errors.New`, silently breaking the `IsNotFound` contract

- **Location:** `internal/repository/recipe_repository.go:149-154` (`return nil, errors.New("recipe
  not found")`, using the **stdlib** `errors` imported at `:5`); contract defined in
  `internal/errors/errors.go:45-54`.
- **Description:** Every other repository surfaces `gorm.ErrRecordNotFound` directly, which
  `errors.IsNotFound` recognizes (`errors.Is(err, gorm.ErrRecordNotFound)`). `RecipeRepository.GetByID`
  instead swallows the typed GORM error and returns a bare stdlib `errors.New("recipe not found")`,
  which is **neither** `gorm.ErrRecordNotFound` **nor** an `*AppError` with code `NOT_FOUND`. So
  `apperrors.IsNotFound(err)` returns `false` for a missing recipe.
- **Impact:** A latent defect that misleads *every* caller of `recipeRepo.GetByID`. In
  `recipeService.GetByID/Update/Delete` (`recipe_service.go:319-333`, etc.) the
  `if errors.IsNotFound(err)` branch never fires for a genuinely missing recipe, so the service
  returns the raw "recipe not found" error instead of the canonical `ErrNotFound`. It also corrupts
  data-flow for indirect callers such as `shoppingListService.AddRecipeToList`
  (`shopping_list_service.go:269`). (The *user-facing* status for this path is determined separately
  by the handler — see next finding — so the two should not be conflated: this is the underlying
  translation bug, independent of the handler.)
- **Recommendation:** Mirror the other repos — either return `gorm.ErrRecordNotFound` unchanged or
  return `apperrors.ErrNotFound.Wrap("recipe not found")` so the `NOT_FOUND` code is preserved.

### [Medium] Error translation at the HTTP boundary is ad-hoc; `AppError.Code` is never consulted

- **Location:** `internal/handler/recipe_handler.go` (all methods, e.g. `:56-60`, `:137-141` map
  *any* service error to `500`); `internal/handler/shopping_list_handler.go:76-80` maps *any* error
  from `Get` to `404`. `AppError` codes are defined in `internal/errors/errors.go:10-43,56-59`.
- **Description:** Services carefully build typed errors (`ErrNotFound`, `ErrUnauthorized`,
  `INVALID_INPUT`, etc.), but no handler inspects `*AppError`/`Code` (confirmed: no `errors.As`,
  `IsNotFound`, `StatusForbidden`, or `StatusConflict` usage anywhere in `internal/handler`). Instead
  each endpoint hardcodes a single status: `recipe_handler` returns `500` for everything (so an
  unauthorized access or a missing recipe both become `500 "failed to get recipe"`), while
  `shopping_list_handler.Get` returns `404` for everything (so an `ErrUnauthorized` from
  `verifyListOwnership` and a real DB error both become `404 "shopping list not found"`). The whole
  `AppError.Code` machinery is unused at the boundary.
- **Impact:** Wrong/misleading HTTP status codes (404s reported as 500s and vice-versa), inconsistent
  behavior across domains, and masked infrastructure failures. This is the *quality/correctness* angle
  on error handling and is distinct from the security finding about leaking `err.Error()` strings
  (`be-handlers-config.md`, Medium).
- **Recommendation:** Add one shared helper (e.g. `respondError(c, err)`) that maps `AppError.Code` →
  HTTP status via `errors.As`, and call it from every handler so layer-to-transport translation is
  centralized and consistent.

### [Medium] Three overlapping transaction abstractions; one is the dead-but-correct one

- **Location:** `internal/repository/base_repository.go:16` (`RunInTransaction`); per-repo `RunTx` and
  `WithTypedTransaction` (`recipe_repository.go:18-19,32-43`, `user_repository.go:22-23,36-47`,
  `ai_config_repository.go:21-22,35-44`).
- **Description:** Each repo exposes three ways to run a transaction: `RunInTransaction` (base,
  passes `*gorm.DB`), `WithTypedTransaction` (passes a tx-bound repo interface — the idiomatic one),
  and `RunTx` (passes nothing — the broken one from the High finding). Services use only `RunTx`;
  `WithTypedTransaction` is implemented on three repos and **called by none**, so it is dead code.
- **Impact:** Cognitive overhead and a trap: the abstraction that looks ergonomic (`RunTx`) is the
  unsafe one, and the safe one is invisible. Inconsistent across the codebase (some repos define
  the typed variant, `shopping_list_repository.go` defines none and has no transactional writes at
  all despite `Create`+`AddItems` being a two-step operation in `shoppingListService.Create`,
  `shopping_list_service.go:74-99`).
- **Recommendation:** Settle on one transaction pattern (`WithTypedTransaction`), migrate callers to
  it, and remove `RunTx`. Wrap `shoppingListService.Create`'s list+items writes in it too.

### [Medium] `ImportFromPDF` reads the upload with a single `f.Read`, which is not guaranteed to fill the buffer

- **Location:** `internal/handler/recipe_handler.go:206-241` (read at `:220-224`; missing guard at
  `:232`).
- **Description:** `fileBytes := make([]byte, file.Size)` followed by a single `f.Read(fileBytes)`.
  `io.Reader.Read` may return fewer bytes than requested without error, so larger PDFs can be read
  **partially** and silently truncated before being handed to the parser. Separately, unlike every
  other recipe handler, this method does not guard `userID == ""` (`:232`) before using it.
- **Impact:** Intermittent, size-dependent parse failures/corruption that are hard to reproduce; and
  an unauthenticated-context call path that skips the standard guard. (The client-declared `file.Size`
  sizing is also flagged from the security angle in `be-handlers-config.md`; the partial-read bug and
  the missing guard are the quality concerns here.)
- **Recommendation:** Use `io.ReadAll(f)` (or `io.ReadFull`) instead of a bare `Read`, and add the
  `userID == ""` guard for consistency with the other handlers.

### [Low] Non-idiomatic `err == gorm.ErrRecordNotFound` equality instead of `errors.Is`

- **Location:** `internal/repository/recipe_repository.go:151`, `:239`.
- **Description:** Direct `==` comparison against `gorm.ErrRecordNotFound`. This works only because
  GORM currently returns that sentinel unwrapped; it breaks the moment the error is wrapped anywhere
  in the chain. Idiomatic Go uses `errors.Is`.
- **Impact:** Low today, fragile under refactor. Inconsistent with the wrapped-error handling used
  elsewhere.
- **Recommendation:** Use `errors.Is(err, gorm.ErrRecordNotFound)`.

### [Low] Duplicated `domain.Recipe` construction and duplicated sort-field knowledge

- **Location:** `internal/service/recipe_service.go:115-133` vs `:230-249` (≈18 lines of identical
  field mapping in `Create` and `Update`); `internal/service/shopping_list_service.go:151` (valid-sort
  map) vs `:350-363` (the `switch` in `sortItems`).
- **Description:** The request→`domain.Recipe` mapping is copy-pasted between `Create` and `Update`.
  The set of valid sort fields is encoded twice — once as a `map` used only to log a warning, once as
  a `switch` with a `default` — so they can drift independently.
- **Impact:** Maintenance hazard (a new field must be added in two places). Minor.
- **Recommendation:** Extract a `buildRecipe(userID, req)` helper; derive the valid-sort set from a
  single source (or have `sortItems` report whether the field was recognized).

### [Low] `Login` collapses all `GetByEmail` errors into "invalid credentials"

- **Location:** `internal/service/user_service.go:107-115`.
- **Description:** Any error from `GetByEmail` (including a DB connection failure) is mapped to
  `apperrors.New("invalid credentials")`. While returning a uniform message to the *client* is good
  for auth, swallowing the underlying error means a database outage is logged/handled as a routine bad
  login, with no error wrapping (`%w`) or warn-level log of the real cause.
- **Impact:** Operational blind spot — infra failures during login are invisible. Low.
- **Recommendation:** Distinguish `IsNotFound`/wrong-password (return the uniform message) from other
  errors (log + return the wrapped error), as `ForgotPassword` already does (`user_service.go:128-134`).

### [Info] Minor nits

- **`errors.New(fmt.Sprintf(...))`** at `recipe_service.go:458` — the custom `errors.New` takes a
  plain message; using `fmt.Sprintf` to build it is slightly awkward (and `go vet`/linters can't see
  the format string). A small `Newf`-style helper would read better.
- **Leftover debug print** `fmt.Print(message)` at `pkg/ai/claude_model.go:43` — stray debug output
  in a hot path (also noted from the security angle in `be-handlers-config.md`). Should be removed or
  switched to the injected `zap.Logger`.
- **Domain not fully transport-agnostic** — `internal/domain/recipe.go:4` imports `mime/multipart`
  because request DTOs carry `*multipart.FileHeader`. It's the one spot where the domain package
  knows about an HTTP transport detail; otherwise domain imports are clean (no `gorm`/`gin`/`pkg`).
  Consider moving file-upload DTOs to a transport/request package if this grows.
- **`CLAUDE.md` drift** — both `CLAUDE.md` files describe a generic `BaseRepository[T]` with
  `GetDB()`/`withDB()`/`WithTransaction()`, but the real `BaseRepository`
  (`base_repository.go:8-20`) is non-generic and exposes only `DB` and `RunInTransaction`. Docs
  should be updated to match.

## Concurrency

No concurrency defects found. There are **no** `go func`, channels, or `sync` primitives anywhere in
`internal/` or `pkg/` (and none in `cmd/`, e.g. no graceful-shutdown goroutine). Request handling is
Gin's per-request model. Services and repositories are constructed once and shared across concurrent
requests, but they hold only immutable dependencies (repos, logger, config, an `*anthropic.Client`);
e.g. `ClaudeModel` (`pkg/ai/claude_model.go:11-15`) carries only a client, a version string, and a
logger — no mutable shared state — so singleton sharing is safe. The in-memory sort/organize helpers
(`shopping_list_service.go:342-377`, `store_chain_service.go:60-85`) operate only on caller-owned
slices. No data races are possible given the current code.

## Context propagation

Solid. `context.Context` is threaded handler → service → repository everywhere, repositories
consistently call `.WithContext(ctx)`, and there are **no** `context.Background()`/`context.TODO()`
calls buried in request call chains. The only context-free service method is
`UserService.ValidateToken` (`user_service.go:195`), which is pure JWT verification with no I/O — an
acceptable exception.

## Positives

- **Consumer-defined interfaces (textbook Go).** Each service declares its own *minimal* dependency
  interface at the point of use (e.g. `recipeUserRepository{ GetByID }`, `recipeAIConfigRepository`,
  `shoppingListRecipeRepository`) instead of importing the fat exported repo interface. This is the
  idiomatic "accept interfaces" pattern and directly answers "interfaces at the right boundaries" —
  it keeps services decoupled and trivially mockable (and the test files confirm this is exploited).
- **Clean layering.** Dependency flow respects handler → service → repository; handlers never touch
  GORM/the DB (verified — no `gorm`/`WithContext` references in `internal/handler`), repositories
  carry no business rules beyond data shaping, and the domain package imports no infrastructure (the
  lone `mime/multipart` import noted above is the only blemish). Wiring is centralized in `cmd/api`
  and the `repositories.go`/`services.go`/`handlers.go` aggregators.
- **Ownership-verification helpers.** `verifyListOwnership`/`verifyItemOwnership`
  (`shopping_list_service.go`) and the inline `UserID != userID` checks centralize authorization in
  the service layer consistently.
- **Resource cleanup on failure.** `recipeService.Create`/`Update` delete uploaded images when the
  surrounding operation fails (`recipe_service.go:162-168`, `:272-279`) and log cleanup failures
  rather than ignoring them — careful, defensive code.
- **Structured logging & DI throughout** — `zap.Logger` and dependencies are constructor-injected
  everywhere; no global state. The PR #24 services refactor (consumer-defined interfaces + per-service
  files) is a clear, well-organized structure to build on.

## Summary

| Severity | Count | Findings |
|----------|-------|----------|
| High | 1 | `RunTx` gives no real atomicity; correct `WithTypedTransaction` is unused |
| Medium | 4 | Recipe repo breaks `IsNotFound` contract; ad-hoc HTTP error translation ignores `AppError.Code`; three overlapping tx abstractions (dead `WithTypedTransaction`); `ImportFromPDF` partial-read + missing guard |
| Low | 3 | `== gorm.ErrRecordNotFound` vs `errors.Is`; duplicated recipe/sort logic; `Login` masks all errors |
| Info | 1 | Minor nits (errors.New+Sprintf, stray `fmt.Print`, domain `multipart` import, CLAUDE.md drift) + concurrency/context/positives |

**Most impactful:** the `RunTx` transaction wrapper is illusory — multi-step writes
(`Register`, recipe `Create`, AI-config `Create`) are not atomic, so partial failures leave orphaned
rows; the idiomatic fix (`WithTypedTransaction`) is already written but unused.

**Overall architectural health: good.** Layering, dependency injection, consumer-defined interfaces,
context propagation, and concurrency are all sound and idiomatic. The defects are concentrated in two
seams — the transaction abstraction and cross-layer error translation — both fixable without
structural change.
