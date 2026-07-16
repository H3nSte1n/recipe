# Phase 3 — Fresh Input-Validation / Injection Review

**Subtask:** Check GORM for raw/string-built SQL, `binding:` validators on every DTO,
mass-assignment on nested recipe writes, and sort/pagination-field injection in
`shopping_list_service`.

**Scope/method:** Grepped all of `internal/repository/` and `pkg/` for `Raw(`, `Exec(`, and
`fmt.Sprintf` feeding into query strings. Read every request DTO in `internal/domain/` for
`binding:` coverage. Traced nested `Ingredients`/`Instructions`/`SubRecipes` from request bind
through to persistence for mass-assignment of `id`/`recipe_id`. Read `shopping_list_service.go`'s
sort implementation and `recipe_handler.go`'s pagination parsing.

---

## Result: no SQL injection found

Every repository query (`internal/repository/*.go`) uses GORM's parameterized `Where("... = ?",
arg)` form or struct-based `Create`/`Updates`/`Delete`. Grep for `Raw(`, `Exec(`, and
`fmt.Sprintf` anywhere near a query builder returned no matches in `internal/` or `pkg/` outside
of unrelated string formatting (log messages, error text, AI prompt construction). No
string-concatenated or `fmt.Sprintf`-built SQL exists anywhere in the codebase.

## Finding 1 — Mass-assignment surface on nested recipe writes: domain models double as bind targets, but is not currently exploitable

- **Severity:** LOW (defense-in-depth gap, not a live vulnerability)
- **Evidence:**
  - `internal/domain/recipe.go:123-141` — `CreateRecipeRequest.Ingredients []RecipeIngredient` and
    `.Instructions []RecipeInstruction` bind directly from client JSON via `ShouldBindJSON`, but
    `RecipeIngredient`/`RecipeInstruction` (`recipe.go:33-50`) are the **same structs used for
    GORM persistence** — including their `ID` and `RecipeID` primary-key/foreign-key fields, which
    have ordinary `json:"id"` / `json:"recipe_id"` tags (not `json:"-"` or `binding:"-"`). A client
    can therefore include `"id"` and `"recipe_id"` values for each ingredient/instruction in the
    request body.
  - **Why it's not exploitable today:** `RecipeID` is silently overwritten regardless of client
    input — GORM's auto-save-association behavior sets each child's FK to match the actual parent
    on `Create` (recipe_repository.go:38-42), and `Update` explicitly overwrites it in a loop
    (`recipe_service.go:242-244,268-273`) before persisting. A client-supplied `id` that collides
    with an existing row (another user's ingredient, or one of the same user's) hits Postgres's
    primary-key uniqueness constraint and the whole transaction fails — it cannot overwrite or
    reassign someone else's row. Confirmed no `binding:"dive"` is needed for the FK to leak since
    it's structurally overwritten, not merely unvalidated.
  - **Residual risk:** if a future refactor ever removes the explicit FK reassignment (e.g. someone
    "simplifies" the Update loop) or switches from `Create` to an `Upsert`/`Save` that honors a
    client-supplied PK on conflict, this becomes a live cross-tenant write primitive with no
    additional code change needed to trigger it — the vulnerable shape (attacker-controlled PK/FK
    in a bind-target-doubling-as-persistence-model) is already in place.
- **Recommended control:** Use dedicated request DTOs for nested ingredients/instructions (only
  `name`/`description`/`amount`/`unit`/`notes` for ingredients; `step_number`/`instruction` for
  instructions) instead of binding directly into the GORM model, so `id`/`recipe_id` cannot be set
  by the client even in principle. Lower priority than the confirmed findings elsewhere in this
  audit given the current code is not exploitable.

## Finding 2 — No cap on nested-array size for ingredients/instructions/sub-recipes

- **Severity:** LOW (storage/DoS-adjacent, not injection)
- **Evidence:** `CreateRecipeRequest.Ingredients`, `.Instructions`, `.SubRecipes`
  (`recipe.go:133-141`) have no `binding:"max=N"`/`dive` length cap, and
  `RecipeIngredient.Name`/`RecipeInstruction.Instruction` have no `max` length validator either.
  `engine.MaxMultipartMemory` (`router.go:25`) and per-file upload caps exist, but nothing bounds
  the number of JSON array elements or their string field lengths in a single recipe write.
- **Why it matters:** An authenticated user (any self-registered account once the VPN is removed —
  see `03-vpn-deps.md` Finding 3) can submit a single `POST /recipes` with an unbounded number of
  ingredients/instructions or arbitrarily long text fields, each triggering its own `Create`
  statement inside the transaction — a cheap way to bloat the database or degrade request latency
  for other users. Bounded in severity by Postgres's own row/statement limits and the request body
  size (see request-size-limit finding in `03-config.md`), but no application-level guard exists.
- **Recommended control:** Add `binding:"max=200"` (or similar) on the three slices and
  `binding:"max=2000"` on free-text fields (`name`, `description`, `instruction`, `notes`).

## Finding 3 — Shopping-list sort field is validated but not enforced (falls back silently rather than rejecting)

- **Severity:** INFORMATIONAL (not injection — no SQL/query-string construction is involved)
- **Evidence:** `shopping_list_service.go:154-169` (`GetSorted`) checks `sortBy` against an
  allow-list (`validSortFields`) but only **logs a warning** on a miss
  (`s.logger.Warn("unknown sortBy field, defaulting to name", ...)`) rather than rejecting the
  request; `sortItems` (`:358-386`) is a pure in-memory `sort.SliceStable` with a `switch`/`case`
  over known field names and a `default` fallback to name-sort — there is no place where `sortBy`
  or `sortDirection` reaches a query string, ORDER BY clause, or any interpolation point.
- **Why this is NOT an injection vector:** Unlike a typical "sort field injection" (where an
  unvalidated field name gets interpolated into `ORDER BY <field>`), this code never builds a
  query — the "sort" happens entirely in Go after the data is already fetched with a fixed
  `Where("user_id = ?")`. An attacker-controlled `sortBy` value has no path to becoming SQL.
- **Recommended control:** None required for injection. As a minor UX/correctness cleanup
  (optional), reject unknown `sortBy` with a 400 instead of silently defaulting — but this is a
  code-quality nit, not a security finding.

## Finding 4 — Public-recipe pagination parameters are unvalidated

- **Severity:** LOW
- **Evidence:** `internal/handler/recipe_handler.go:198-199` —
  `page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))` and
  `pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))` discard the parse error (so a
  non-numeric value silently becomes `0`) and apply no `min`/`max` clamp before
  `recipe_repository.go:212-213` computes `Offset((page - 1) * pageSize).Limit(pageSize)`.
  `page=0` (or any non-numeric input) produces a negative `Offset`, which Postgres rejects
  ("OFFSET must not be negative") — surfacing as an unhandled-error 500 (see `03-config.md`
  request-validation findings). An arbitrarily large `page_size` (e.g. `999999999`) is passed
  straight through to `LIMIT`, which Postgres accepts and would return/scan the entire public-recipes
  table in one response.
- **Why it matters:** No SQL injection (values are bound as query parameters, not concatenated),
  but it is a resource-exhaustion / minor-DoS surface once anonymous registration is possible
  (`03-vpn-deps.md` Finding 3): repeated large-`page_size` requests can be used to pull the entire
  public dataset repeatedly or trigger the negative-offset error path.
- **Recommended control:** Clamp `page` to `>= 1` and `page_size` to a sane range (e.g. `1–100`)
  in the handler before calling the service.

## Checks performed

1. Grepped `internal/repository/`, `internal/service/`, and `pkg/` for `Raw(`, `Exec(`,
   `fmt.Sprintf` near query construction — none found; all queries are parameterized
   `Where("... = ?", arg)`.
2. Read every `binding:` tag in `internal/domain/recipe.go` and `internal/domain/shopping_list.go`
   for required/oneof/min/max coverage.
3. Traced `CreateRecipeRequest.Ingredients`/`.Instructions`/`.SubRecipes` from JSON bind through
   `recipe_service.go`'s `Create`/`Update` to `recipe_repository.go`'s GORM calls, checking whether
   client-supplied `id`/`recipe_id` on nested structs could reassign or overwrite another
   user's/recipe's rows.
4. Read `shopping_list_service.go`'s `GetSorted`/`sortItems` end-to-end to confirm `sortBy`/
   `sortDirection` never reach a query string.
5. Read `recipe_handler.go`'s `ListPublic` pagination parsing and `recipe_repository.go`'s
   `ListPublic` `Offset`/`Limit` construction for missing bounds.

---

*No production code was modified. This file is the only artifact written.*
