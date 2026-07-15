# Phase 3 — Fresh Data-Access / IDOR Review

**Subtask:** Audit every repository query for tenant scoping (`Where user_id = ?`) and probe each
owned resource (recipes, shopping lists, items, ai_configs, nested item routes) for cross-user
read/write as user B.

**Scope/method:** Static review of `internal/service/{recipe,shopping_list,ai_config}_service.go`
and their repository implementations, followed by live cross-tenant probes against the shared
stack (`http://localhost:18080`) using the two throwaway audit users from `00-notes.md` (tokens
re-minted — the originals had expired after two weeks). User A created a private recipe, a
shopping list with one item, and reused an existing AI config; user B then attempted read/write on
every one of user A's resource IDs.

---

## Result: no IDOR found — every probed cross-tenant access was denied

| Resource / route | Probe (as user B against user A's object) | Result |
|---|---|---|
| `GET /recipes/:id` (private) | Read | **Denied** (`ErrUnauthorized`, HTTP 500 — see Finding 1) |
| `PUT /recipes/:id` | Overwrite title | **Denied** (500) |
| `DELETE /recipes/:id` | Delete | **Denied** (500) |
| `GET /shopping-lists/:id` | Read | **Denied** (404 "shopping list not found") |
| `PUT /shopping-lists/:id/items/:itemId` | Overwrite item | **Denied** (500) |
| `DELETE /shopping-lists/:id/items/:itemId` | Delete item | **Denied** (500) |
| `PATCH /shopping-lists/:id/items/:itemId/toggle` | Toggle checked | **Denied** (500) |
| `GET /ai-configs/:id` | Read (incl. decrypted API key) | **Denied** (404) |
| `PUT /ai-configs/:id` | Modify | **Denied** (500, "unauthorized") |
| `DELETE /ai-configs/:id` | Delete | **Denied** (500, "unauthorized") |
| `POST /ai-configs/:id/set-default` | Hijack another user's default model config | **Denied** (500, "unauthorized") |

Every code path traced back to an explicit ownership check before any read/write:

- `recipe_service.go:195` (`Update`), `:315` (`Delete`), `:345` (`GetByID`) — compares
  `existingRecipe.UserID`/`recipe.UserID` against the caller's `userID` and returns
  `errors.ErrUnauthorized` on mismatch (private-recipe read; owner-only write/delete).
- `shopping_list_service.go:114-123` (`verifyListOwnership`) and `:226-239`
  (`verifyItemOwnership`, which resolves the item → its parent list → the list's `user_id`) gate
  every list and item operation (`Update`, `Delete`, `GetByID`, `AddItem`, `UpdateItem`,
  `DeleteItem`, `ToggleItem`, `AddRecipeToList`, both sorted-view methods).
- `ai_config_service.go:129-141` (`GetByID`) and `:168-179` (`SetDefault`) compare
  `config.UserID` to the caller; `Update`/`Delete` both route through `GetByID` first
  (`:87-91`, `:155-159`), so they inherit the same check.
- At the repository layer, `Delete`/`UpdateItem`/`DeleteItem` for shopping lists and
  `Delete`/`Update` for AI configs and recipes take **only** the resource `id` with no `user_id`
  predicate (e.g. `shopping_list_repository.go:66-67,90-91`) — tenant isolation is enforced
  entirely at the service layer, one layer up from the DB. This was double-checked live (table
  above) rather than trusted from reading the code, since a repository-only defense is a common
  place for this kind of bug to hide.
- List-returning queries (`ListByUserID` for recipes, shopping lists, and AI configs) all carry
  `Where("user_id = ?", userID)` at the repository layer (`recipe_repository.go:169`,
  `shopping_list_repository.go:75`, `ai_config_repository.go:69`) — confirmed no unscoped
  `Find`/`Preload` that could return another user's rows.

## Finding 1 — Denied requests return HTTP 500 instead of 403/404, several with a leaked error string

- **Severity:** LOW (authorization itself is correctly enforced — no data crossed the tenant
  boundary in any probe above; this is a status-code/observability defect, not a bypass)
- **Evidence:** `internal/handler/recipe_handler.go:130,147,173` map **every** service error
  (including `errors.ErrUnauthorized` and `errors.ErrNotFound`) to a flat
  `http.StatusInternalServerError` with a generic message — confirmed live: user B's denied
  `GET`/`PUT`/`DELETE` on user A's private recipe all returned `500`, not `401`/`403`/`404`.
  `internal/handler/ai_config_handler.go:50,87,109` go further and echo the raw `err.Error()`
  string into the 500 body — confirmed live: `PUT/DELETE/POST .../set-default` on another user's AI
  config returned `{"error":"unauthorized"}` with a `500` status (benign message here, but the
  pattern is what produced the raw Postgres constraint-violation string
  `ERROR: duplicate key value violates unique constraint "user_ai_configs_user_model_key" (SQLSTATE 23505)`
  seen during setup for this probe — full analysis in `03-config.md`).
- **Why it matters:** Not an IDOR — access is correctly denied in all cases. But (a) a `500` on a
  routine, expected "you don't own this" denial is indistinguishable from a real server fault to
  monitoring/alerting, and (b) `ai_config_handler.go`'s `err.Error()` pass-through is the same
  pattern responsible for the raw-SQL leak tracked as the primary finding in `03-config.md`.
- **Recommended control:** Map `errors.ErrUnauthorized` → 403 and `errors.ErrNotFound` → 404
  consistently across all handlers (a shared `apperrors`→HTTP-status helper, since
  `ai_config_handler.go` and `shopping_list_handler.go` already do this correctly in places — e.g.
  `ai_config_handler.go:63,121`, `shopping_list_handler.go:78` — but `recipe_handler.go` and the
  rest of `ai_config_handler.go` do not). Never serialize `err.Error()` directly into a client
  response; see `03-config.md` for the full recommendation.

## Checks performed

1. Read `recipe_service.go`, `shopping_list_service.go`, `ai_config_service.go` in full for
   ownership-check placement (before vs. after any mutating repo call).
2. Read `recipe_repository.go`, `shopping_list_repository.go`, `ai_config_repository.go` for
   `Where user_id = ?` scoping on every list/get/update/delete method.
3. Live probes: registered/reused two audit users (A, B); as A, created a private recipe, a
   shopping list with one item, and confirmed an existing AI config; as B, attempted GET/PUT/DELETE
   (and item toggle, set-default) against every one of A's resource IDs.
4. Cross-checked recipe sub-resource guard (`AddRecipeToList` pulling another user's private
   recipe into a list) — code-level only here; already live-verified in Phase 2's `02-authz.md`
   (PR #36), not re-probed to avoid duplicate destructive state on the shared stack.

---

*No production code was modified. This file is the only artifact written.*
