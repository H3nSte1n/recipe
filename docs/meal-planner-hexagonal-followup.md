# Meal planner: hexagonal architecture follow-up (separate scope)

## Background

PR #53 first shipped the meal planner as a self-contained hexagonal module (`internal/mealplan/{domain,application,adapter}`).
It was then realigned to the repo's layered pattern so it matches `CLAUDE.md` (`domain` → `repository` → `service` → `handler`,
aggregated in `repositories.go`, `services.go`, `handlers.go`). This document captures what a proper move to hexagonal
architecture would involve, so it can be decided and done for the whole backend rather than for one feature.

Nothing here is required for the planner to work. It is a design decision about the codebase, so per `CLAUDE.md`
("ask before architecture changes") it needs an explicit go-ahead.

## Goal

Business rules depend on nothing but the domain and on interfaces the business layer owns. HTTP, GORM and Postgres sit
at the edges and can be replaced or tested without touching the rules.

## What the current layered code leaves on the table

| Gap | Where | Effect |
| --- | --- | --- |
| Domain model carries GORM and JSON tags | `internal/domain/*` | Storage and wire formats shape the domain |
| Services depend on `repository` package types (`repository.PlannedMealRepository` inside `WithTypedTransaction`) | `internal/service/*` | Service cannot be built or tested without the repository package |
| Repository returns `AppError` values and maps Postgres codes | `internal/repository/*` | Persistence adapter decides HTTP-relevant error meaning |
| Service receives the whole `repository.RecipeRepository` for one lookup (`GetOwnedSummary`) | `services.go` | Port is narrow in the service, but the adapter is still the shared GORM repository |
| IDs are raw strings end to end | all layers | Parsing and validation are scattered; invalid IDs surface late |
| `userID == ""` check repeated in every handler | `internal/handler/*` | Auth identity is a loose string, not a typed value |

## Proposed target (per domain, applied incrementally)

1. **Pure domain types.** Remove GORM/JSON tags from domain entities. Add separate persistence rows (in the repository
   adapter) and response DTOs (in the handler). Use `uuid.UUID` and a date value type instead of strings.
2. **Domain errors.** Define `ErrNotFound`, `ErrInvalid`, `ErrConflict` in the domain. Adapters translate driver errors into
   them; handlers translate them into HTTP statuses. `internal/errors` stays as the HTTP-facing helper.
3. **Ports owned by the business layer.** The service package declares the repository interface it needs (it already does
   this with unexported interfaces) but with domain types only, and takes `uuid.UUID` identities parsed at the HTTP edge.
4. **Recipe catalog port.** Done: the service declares `plannedMealRecipeRepository.GetOwnedSummary`, implemented by the recipe repository. The explicit `FOR KEY SHARE` lock was dropped; the composite foreign key `(recipe_id, user_id)` stops a recipe delete racing an insert, and a violation is reported as "recipe not found".
5. **Typed auth identity.** Middleware puts a parsed user ID in the context; handlers read it through one helper that
   returns 401, instead of repeating the empty-string check.
6. **Move to a package-per-feature layout only if wanted.** Layer-per-package (today) and feature-per-package
   (`internal/mealplan/...`) are both compatible with the above; pick one for the whole repo.

## Suggested sequence

1. Decide the target layout and error model (one short ADR in `docs/`).
2. Pilot on one small domain (planned meals is a good candidate: it is new, small, and has integration tests).
3. Introduce the typed identity helper and domain errors, keeping existing handlers working.
4. Convert one existing domain (store chains is the smallest) to confirm the pattern works with the older code.
5. Update `CLAUDE.md` so "every domain follows the same pattern" describes the new pattern, and list which domains are migrated.

## Risks and costs

- It touches every layer of every domain it is applied to; do it per domain, not in one change.
- The repo's existing tests use `testify/mock` against `repository.*` interfaces. Moving ports into the service package
  changes those mocks.
- The composite foreign key stops a recipe deletion racing a plan write. Any change to it needs the integration test
  (`internal/repository/planned_meal_repository_integration_test.go`) to keep passing.

## Done when

- Domain packages import only the standard library and `uuid`.
- Services import no `gorm`, `pgconn`, `gin`, or `repository` packages.
- Adapters (HTTP, Postgres) are the only places that know status codes or SQL error codes.
- `CLAUDE.md` describes one architecture, and every domain follows it or is listed as pending.
