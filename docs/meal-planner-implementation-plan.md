> **Update:** the backend was realigned to the repo's layered pattern (`domain` → `repository` → `service` → `handler`)
> instead of the separate `internal/mealplan` hexagonal package described below. File paths in this document
> are historical. The hexagonal ideas that were deferred are tracked in `docs/meal-planner-hexagonal-followup.md`.

# Meal planner MVP implementation plan

## Goal and scope

Let an authenticated user schedule their own recipes on calendar dates, edit or remove each scheduled meal, and see the same entries in calendar and chronological list views. Multiple entries on one date and repeated uses of one recipe are allowed. This plan covers the backend, frontend, migrations, and automated tests. Recurrence, shared plans, shopping-list automation, reminders, and drag and drop are outside this MVP.

## Repository findings that shape the work

- The Go API uses Gin, GORM, PostgreSQL, sequential `golang-migrate` SQL migrations, and composition in `services/backend/cmd/api/main.go`. The current latest migration is `000018`.
- Existing recipe records have `recipes.user_id`, `is_private`, and `status`; recipe deletion physically deletes the row. Existing recipe GET permits another user's public recipe, but the home UI loads only the current user's recipes and displays published ones.
- Protected routes use JWT auth. State-changing routes use `RequireVerified`; reads remain available to unverified users. Apply the same policy to the planner.
- Existing `internal/domain` structs contain GORM/HTTP tags and existing services sometimes import `internal/repository`. The new planner slice should live in a separate package tree so these dependencies do not enter its core.
- The frontend is React 18 with local screen state in `App.tsx`, relative `/api/v1` calls, and no configured test runner. Existing Go tests are largely mocks; there is no PostgreSQL integration test harness today.

## Product and API decisions

1. **Ownership and visibility:** A planned meal belongs to the authenticated user and may reference only a recipe with the same `user_id`, whether that recipe is public or private. Do not accept another user's public recipe in this MVP. This avoids a plan exposing a recipe after its owner makes it private. The picker should show the current user's published recipes, matching the current home UI; existing plans remain readable if a recipe later becomes draft or archived. The backend accepts any status for an owned recipe so status edits do not invalidate the plan.
2. **Deletion and edits:** `recipe_id` is a foreign key with `ON DELETE CASCADE`; deleting a recipe removes its planned entries. Deleting a user also removes their entries. The list response joins the current recipe title (and status), so recipe title/status changes appear immediately. The UI refreshes the visible range after recipe deletion and shows a normal empty state if an entry disappeared.
3. **Dates:** `cook_date` is a PostgreSQL `DATE` and JSON `YYYY-MM-DD`, with no clock time or timezone conversion. Validate exact ISO calendar dates, including leap days; reject timestamps and invalid dates. Compare and render dates as local calendar dates in the browser without parsing them as UTC instants. Past dates are allowed for history and correction. `created_at`/`updated_at` are UTC instants.
4. **Fields:** Required `recipe_id` (UUID) and `cook_date`; optional `meal_type` enum `breakfast|lunch|dinner|snack`, `servings` positive integer (1–100), and `notes` string (max 1000 Unicode characters). Empty optional values become SQL NULL. No unique constraint on user/date/recipe/meal type. `servings` NULL means use the recipe's current servings; the API should return it as `null`, not silently copy the recipe value.
5. **Range and paging:** `GET /planned-meals?from=YYYY-MM-DD&to=YYYY-MM-DD&page=1&page_size=100` requires both inclusive dates, with `from <= to` and at most 93 calendar days inclusive. `page` defaults to 1; `page_size` defaults to 100 and must be 1–100. Return `{items, total, page, page_size}` sorted by `cook_date ASC, id ASC`. Calendar and list clients fetch all pages for the requested range; this avoids silently dropping meals when a day has many entries. Query uses user and date predicates before paging and has a supporting `(user_id, cook_date, id)` index. Reject invalid or oversized ranges with 400.
6. **Endpoints:** `POST /api/v1/planned-meals` creates and returns the entry with 201. `GET /api/v1/planned-meals` lists a range with 200. `GET /api/v1/planned-meals/:id` returns one entry with 200. `PUT /api/v1/planned-meals/:id` replaces editable fields and returns 200. `DELETE /api/v1/planned-meals/:id` returns 204. PUT requires `recipe_id` and `cook_date`; optional fields can be omitted/cleared. This simple replacement contract avoids ambiguity over clearing NULL fields.
7. **Errors and auth:** Missing/invalid JWT is 401 via existing middleware. An unverified user may GET, while POST/PUT/DELETE return 403 via `RequireVerified`. Malformed UUID/date/body, invalid enum/servings/notes, missing fields, and bad range/page return 400. Missing, inaccessible, or deleted recipe/plan returns 404 without revealing another user's IDs. Unexpected database errors return 500 with a generic message; log detail server-side. Scope every read, update, and delete by `user_id` in SQL, and check recipe ownership within the write transaction to avoid a validation/write race. Database FK/check failures should map to safe 400/404 only when identified; otherwise 500.

### JSON shape

```json
{
  "id": "uuid",
  "recipe_id": "uuid",
  "cook_date": "2026-10-03",
  "meal_type": "dinner",
  "servings": 2,
  "notes": "Prepare sauce earlier",
  "recipe": { "id": "uuid", "title": "Pasta", "status": "published" },
  "created_at": "2026-10-03T12:00:00Z",
  "updated_at": "2026-10-03T12:00:00Z"
}
```

`meal_type`, `servings`, and `notes` may be `null`. Do not return another user's recipe metadata. `user_id` is taken only from the JWT and need not be exposed in responses.

## Architecture and file-level work

Dependency direction: Gin adapter → application use case → pure planner domain and application-owned ports. PostgreSQL adapter implements the ports. Only `cmd/api/main.go` constructs the adapters and use case. The new core imports no Gin, GORM, SQL driver, HTTP binding, or `internal/repository` packages.

### Phase 1: schema and pure core

1. Add `services/backend/migrations/000019_create_planned_meals.up.sql` and `.down.sql`: UUID PK, `user_id` and `recipe_id` FKs with cascade, `cook_date DATE NOT NULL`, nullable enum/check fields, positive bounded servings and note length checks, timestamps, and `(user_id, cook_date, id)` index. Down drops only this table. Use existing UUID extension from migration `000001`.
2. Add `services/backend/internal/mealplan/domain/planned_meal.go` and `date.go`: plain structs/value types; exact date parsing/formatting, field validation, and domain errors. Keep persistence rows and request DTOs outside this package. Avoid `time.Time` JSON encoding for `DATE` to prevent timezone shifts; a date value may internally use validated year/month/day or an ISO string.
3. Add `services/backend/internal/mealplan/application/ports.go` and `service.go`: use-case commands and queries, repository and recipe-lookup interfaces owned by the application, typed input/not-found/forbidden/internal errors, create/get/list/replace/delete methods. The use case checks auth identity, bounds, and field rules and returns caller-safe errors. Decide exact transaction boundary in the port so checking owned recipe and writing the entry are atomic.

### Phase 2: adapters and API wiring

4. Add `services/backend/internal/mealplan/adapter/postgres/` repository files with private GORM/SQL row structs. Implement owner-filtered CRUD and range/count queries, transaction-based owned-recipe checks, explicit mapping between SQL DATE and domain date, and translation of `record not found` and identified constraint violations. A single joined query should fetch the minimal recipe summary for result rows; do not preload full recipe graphs or image URLs.
5. Add `services/backend/internal/mealplan/adapter/http/handler.go` and `dto.go` with Gin request/response types and strict JSON decoding. Reject unknown JSON fields and multiple JSON objects; parse UUID/date/optional values explicitly. Map typed application errors to the status contract above; keep driver errors out of responses.
6. Wire the new adapter and use case in `services/backend/cmd/api/main.go`. Add a planner handler dependency to `services/backend/internal/router/router.go` and register `planned-meals` under protected routes. GET methods omit `requireVerified`; mutating methods include it. Keep existing recipe/service/repository aggregators untouched unless routing tests require a narrow constructor adjustment.

### Phase 3: frontend data and views

7. Add `services/frontend/src/types/plannedMeal.ts` and `services/frontend/src/services/plannedMealService.ts` with typed relative API calls, `getAuthHeaders`, useful error handling, and AbortSignal support for list requests. Add `src/hooks/usePlannedMeals.ts` to request all pages for a visible range, cancel stale requests, and refresh after mutations.
8. Add `services/frontend/src/pages/PlannerPage.tsx`, `src/components/PlannerCalendar.tsx`, `PlannerList.tsx`, and `PlannedMealForm.tsx` plus corresponding CSS. Calendar uses a Monday-first month grid and a 42-day visible range when necessary; list groups the same fetched entries by date. Both views share one data array and the same create/edit/delete actions. Controls: previous/next month, Today, calendar/list switch, add entry, and recipe picker. Show loading, empty, error, and save/delete failure states. Use semantic buttons and labels, keyboard-accessible form controls, and a confirmation before delete.
9. Add a planner screen to `services/frontend/src/App.tsx` and a navigation control in `HomeHeader.tsx` (plus a route back to recipes). Keep planner view/range while switching between its calendar and list views. Fetch recipes through existing `getMyRecipes`; show published owned recipes in the picker. Handle deleted recipes by refetching the range if an edit returns 404. Format calendar dates with explicit year/month/day helpers, not `new Date('YYYY-MM-DD')`.

### Phase 4: tests and verification

10. Add Go unit tests alongside the new domain and application files: strict date/leap-year/range validation, field bounds and NULL semantics, CRUD, multiple entries per date, ownership, missing recipe/entry, recipe reassignment, service error propagation, and inclusive range/paging. Use fake ports; assert no cross-user data is returned.
11. Add Gin handler/router tests for response shape/status, malformed JSON/UUID/date/range/page, verification gate, JWT requirement, and user ID sourced from auth context. Test 204 with empty body and safe 500 responses.
12. Add PostgreSQL-backed integration tests behind `-tags=integration` for migration up/down, table constraints and FK cascades, date round trip across timezone settings, stable inclusive range order/page boundaries, transaction/owner filtering, and recipe title/status changes. Introduce an isolated `TEST_DATABASE_URL` and a reproducible `docker-compose.test.yml` or equivalent test-only PostgreSQL service; never run integration tests against the normal `recipe_db`. Do not silently skip these tests when the required DB URL is absent.
13. Add Vitest, React Testing Library, and jsdom to `services/frontend/package.json`/lockfile with a `test` script and setup file. Test date helpers, service paging/error behavior, calendar/list parity, add/edit/delete flows, same-day duplicates, navigation across month boundaries, stale-request handling, empty/error states, and accessibility labels using mocked API calls. Tests should assert visible user behavior, not component internals.

## Acceptance criteria

- A verified user can add, edit, and remove their own planned entries and see them immediately in both views; two entries on one day appear independently.
- All entries survive reload, and changing the view does not create or lose records.
- A different user's recipe or planned entry cannot be accessed or altered; public visibility does not bypass plan ownership.
- Deleting a recipe or user removes related entries; edits to a recipe title/status appear in the plan.
- Date-only values retain the chosen calendar day across browser/backend/database timezone differences, including DST boundaries.
- Invalid input returns documented 400/404 responses; JWT and verification behavior matches other protected routes; no SQL detail leaks.
- Pure core has no framework/persistence imports. Required Go unit, PostgreSQL integration, frontend component, type, lint, and build checks pass.

## Verification commands

```bash
cd services/backend && go test ./...
cd services/backend && go test -tags=integration ./...  # with isolated TEST_DATABASE_URL
cd services/backend && go vet ./...
cd services/frontend && npm test -- --run
cd services/frontend && npm run type-check
cd services/frontend && npm run lint
cd services/frontend && npm run build
```

For integration tests, start the dedicated test PostgreSQL service first, then set `TEST_DATABASE_URL` to that service's disposable database. Verify migration up/down and rerun up within that test database. Finish with a manual smoke check of adding, editing, and deleting meals in both views and reloading the page.
