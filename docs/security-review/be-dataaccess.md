# Backend Data-Access Layer Review — Injection & Broken Object-Level Authorization (IDOR)

> Phase: deep review of the repository/data-access layer (GORM + PostgreSQL) and the services that
> call it. Scope: SQL/GORM injection, IDOR / broken object-level authorization, mass assignment, and
> missing tenant scoping on List queries. Every finding is grounded in real code with `file:line`.
>
> Files reviewed: all of `services/backend/internal/repository/*.go` and the calling services
> `internal/service/{shopping_list,recipe,ai_config,profile,user}_service.go`.

## Key architectural observation (read this first)

Repositories in this codebase are **deliberately tenant-unaware**. Single-object reads such as
`ShoppingListRepository.GetByID` (`repository/shopping_list_repository.go:31`),
`GetItemByID` (`:42`), `RecipeRepository.GetByID` (`repository/recipe_repository.go:121`),
`AIConfigRepository.GetByID` (`repository/ai_config_repository.go:56`), and
`StoreChainRepository.GetChain` fetch **by resource ID alone**, with no `user_id` predicate. All
object-level authorization therefore lives in the **service layer**, which re-checks ownership after
the repository returns the row. This pattern is sound *only as long as every service call site
performs the check.* The findings below are the places where a check is missing, plus a
defense-in-depth note on the pattern itself.

---

## Findings

### [Medium] IDOR — `AddRecipeToList` pulls any recipe's ingredients with no ownership/privacy check

- **Location:** `internal/service/shopping_list_service.go:263-272` (recipe fetch at `:269`);
  endpoint `POST /api/v1/shopping-lists/:id/add-recipe`
  (`internal/handler/shopping_list_handler.go:231-249`); request type
  `domain.AddRecipeToListRequest{ RecipeID string }` (`internal/domain/shopping_list.go:86-88`).
- **Description:** `AddRecipeToList` correctly verifies the caller owns the **target list**
  (`verifyListOwnership`, line 264), but then fetches the **source recipe** straight from the
  client-supplied body field `recipe_id` with `s.recipeRepo.GetByID(ctx, req.RecipeID, ...)`
  (line 269) and performs **no ownership and no `is_private` check** on that recipe. Contrast this
  with `recipeService.Create`/`Update`, which explicitly reject another user's private sub-recipe via
  `if subRecipe.IsPrivate && subRecipe.UserID != userID { return errors.ErrUnauthorized }`
  (`recipe_service.go:97-99`, `:201-203`). That guard is absent here.
- **Impact:** Any authenticated user who knows (or obtains) the UUID of **another user's private
  recipe** can add it to their own shopping list and then read the list back
  (`GET /shopping-lists/:id`), exposing the private recipe's ingredient names, amounts, and units.
  This is a confidentiality breach of private user data (broken object-level authorization). The
  ingredient names are also forwarded to the AI categorizer (`CategorizeItems`, line 296), a
  secondary egress of the private data. Practical exploitation requires knowing the target UUID
  (v4, not sequentially guessable), which is why this is rated Medium rather than High.
- **Recommendation:** After fetching the recipe, enforce the same rule used elsewhere:
  reject when `recipe.IsPrivate && recipe.UserID != userID` (return `ErrUnauthorized`/`ErrNotFound`).
  Mirror the existing sub-recipe guard in `recipe_service.go` for consistency.

### [Info] IDOR claim on shopping-list item routes (`UpdateItem`/`DeleteItem`/`ToggleItem`) — REFUTED

- **Location:** `internal/service/shopping_list_service.go:216-261` (`verifyItemOwnership` at
  `:216-229`); endpoints `PUT|DELETE /shopping-lists/:id/items/:itemId` and
  `PATCH /shopping-lists/:id/items/:itemId/toggle`.
- **Description:** The route inventory (`00-route-inventory.md`, item routes + Observation #5) flagged
  these as a likely IDOR because the handler binds only `:itemId` and ignores the parent `:id`.
  Verified against the actual code: the service **does** enforce ownership. `verifyItemOwnership`
  loads the item by ID (`GetItemByID`), then loads the item's **own parent list** via
  `GetByID(ctx, item.ListID)` and checks `if list.UserID != userID { return ErrUnauthorized }`
  (lines 221-227). All three mutators (`UpdateItem` `:231`, `DeleteItem` `:246`, `ToggleItem` `:254`)
  call it before mutating. A user therefore **cannot** modify another user's item by guessing its
  UUID. **The recon's suspected IDOR is not present** — ownership is correctly tied to the item's
  real parent list.
- **Impact:** None (informational). The claim is refuted.
- **Recommendation:** Minor hardening only: because the URL's parent `:id` is **decorative** (the
  service derives the list from the item, never validating that `:itemId` actually belongs to the
  `:id` in the path), a request like `PUT /shopping-lists/<list-A>/items/<item-of-list-B>` succeeds
  as long as both lists belong to the caller. This is not a cross-tenant vulnerability, but
  validating `item.ListID == :id` (returning 404 on mismatch) would make the nested route honest and
  prevent confusing client behavior.

### [Info] No SQL / GORM injection found — all queries are parameterized

- **Location:** entire `internal/repository/` tree (and services).
- **Description:** A sweep for `Raw(`, `Exec(`, string-concatenated `Where("..."+var)`, `fmt.Sprintf`
  into queries, and `Order(userInput)` / dynamic column names found **none**. Every `Where` uses
  bound `?` placeholders, e.g. `Where("user_id = ?", userID)`
  (`shopping_list_repository.go:67`), `Where("email = ?", email)` (`user_repository.go:59`),
  `Where("id = ? AND user_id = ?", configID, userID)` (`ai_config_repository.go:105`). The only
  function-call inside a predicate is `Where("LOWER(name) = LOWER(?)", name)`
  (`store_chain_repository.go:35`) — the user value is still a bound parameter; the `LOWER()` is on a
  literal column, so it is safe. Every `Order(...)` call uses **hard-coded literal column names**
  (`Order("created_at DESC")` `recipe_repository.go:182`; `Order("recipe_instructions.step_number")`
  `:167`); none takes user input. Shopping-list sorting by the user-supplied `sortBy` is done
  **in-memory in Go** via `sort.SliceStable` over a whitelist switch (`shopping_list_service.go:342-370`),
  never reaching SQL — so even the `sortBy` query param is not an injection vector.
- **Impact:** None (informational / negative result).
- **Recommendation:** Maintain the parameterized-query discipline; never interpolate user input into
  `Raw`, `Order`, or `Where` strings if these are added later.

### [Low] Tenant scoping is entirely service-enforced — no defense-in-depth at the repo layer

- **Location:** `repository/shopping_list_repository.go:31` (`GetByID`), `:42` (`GetItemByID`),
  `:54` (`Update`), `:59` (`Delete`), `:78` (`UpdateItem`), `:82` (`DeleteItem`);
  `repository/recipe_repository.go:115` (`Delete`), `:121` (`GetByID`);
  `repository/ai_config_repository.go:56` (`GetByID`), `:81` (`Delete`).
- **Description:** These mutating/reading repository methods accept a bare resource `id` and operate
  on it without any `user_id` filter. Today every live call path re-checks ownership in the service
  (`verifyListOwnership`/`verifyItemOwnership` in `shopping_list_service.go`; `existingRecipe.UserID
  != userID` in `recipe_service.go:183,298`; `config.UserID != userID` in `ai_config_service.go:114,
  144`), so there is **no currently exploitable IDOR** in these paths. The risk is structural: a
  future handler/service that forgets the check would expose a cross-tenant read/write with no second
  line of defense, because the repository will happily act on any UUID. The `AddRecipeToList` finding
  above is exactly this class of mistake already realized.
- **Impact:** No direct exploit today; elevated blast radius for future regressions. Verified clean
  call sites: shopping-list `GetByID`/`Update`/`Delete`/item mutators all go through `verify*Ownership`;
  recipe `Update`/`Delete`/`GetByID` all compare `UserID`; ai-config `GetByID`/`Update`/`Delete`/
  `SetDefault` all compare `config.UserID`.
- **Recommendation:** Add ownership to the data-access predicate for mutations (e.g.
  `Delete`/`Update` scoped `Where("id = ? AND user_id = ?", id, userID)` returning
  `RowsAffected == 0` as not-found/unauthorized — the pattern already used well in
  `AIConfigRepository.SetDefault` `:105` and `ClearDefaultByUserID` `:123`). This makes a missing
  service-layer check fail safe instead of silently leaking.

### [Info] `recipe GetByID` returns other users' *public* recipes by ID — intended, not a finding

- **Location:** `internal/service/recipe_service.go:319-333`; endpoint `GET /recipes/:id`.
- **Description:** `GetByID` only blocks access when `recipe.IsPrivate && recipe.UserID != userID`
  (line 328). A **non-private** recipe owned by another user is therefore readable by any
  authenticated caller. This matches the product's public-recipe model (the same rows are returned by
  `GET /recipes/public` / `ListPublic`, `repository/recipe_repository.go:190`). Recorded for
  completeness; not a vulnerability.
- **Impact:** None (by design).
- **Recommendation:** None.

### [Info] No mass-assignment at the repository layer

- **Location:** `repository/recipe_repository.go:68-72` (`Update` uses an explicit
  `Select(...)` field whitelist that **excludes** `user_id`/`id`); `recipe_service.go:230-249`
  (service builds the struct with `UserID: userID` from the JWT context, never from the request body);
  `ai_config_service.go:75-99` (`Update` copies only `APIKey`/`IsDefault`/`Settings` from the request
  onto a server-loaded `config`, leaving `UserID` untouched); `profile_repository.go:31` (`Updates`
  scoped by `Where("user_id = ?", profile.UserID)` with `userID` from context).
- **Description:** Reviewed for the classic GORM mass-assignment pitfall (binding a whole struct so a
  client can overwrite `user_id`/`id`/`is_default`). The recipe `Update` repo method's column
  whitelist (`Select("title", ..., "updated_at")`) means even if a request set a foreign `user_id`,
  the column would not be written. Tenant identifiers consistently originate from
  `middleware.GetUserID` (the JWT), not request JSON.
- **Impact:** None (informational / negative result).
- **Recommendation:** Keep using explicit `Select(...)` whitelists on `Updates`/`Save` for any new
  mutation that binds request structs.

---

## Summary

| Severity | Count | Findings |
|----------|-------|----------|
| Critical | 0 | — |
| High | 0 | — |
| Medium | 1 | IDOR in `AddRecipeToList` (private recipe ingredient disclosure) |
| Low | 1 | No defense-in-depth tenant scoping at repo layer (structural) |
| Info | 4 | Shopping-list-item IDOR **refuted**; no SQL injection; public-recipe-by-id is by design; no repo mass-assignment |

**Headline answers:**
- **SQL/GORM injection:** none found — all queries parameterized; `Order`/column names are literals;
  user-driven sort is in-memory.
- **Shopping-list-item IDOR (recon claim):** **refuted** — `verifyItemOwnership`
  (`shopping_list_service.go:216-229`) ties `:itemId` to a list owned by the caller before any mutation.
- **Real broken-object-level-authorization gap:** `AddRecipeToList`
  (`shopping_list_service.go:269`) fetches a user-supplied `recipe_id` with no ownership/privacy
  check, leaking other users' private recipe ingredients (Medium).
