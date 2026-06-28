# Audit 02 — Account-Level Authorization (PR #36)

Date: 2026-06-28
Scope: Verify PR #36 hardened account-level authorization for (a) adding another
user's PRIVATE recipe to a shopping list, and (b) the members-visible users-list
endpoint leaking PII.

Live stack: backend API base `http://localhost:18080/api/v1`.

## Test setup

Two throwaway users registered + logged in for fresh JWTs (Phase-1 tokens expired):

- User A: `audit-authz-A-AE35CCC7-08FD-49D0-8A02-B38271220F09@example.com`
  - user_id `a80cb037-a344-44d9-8ad2-25fb546b9cdf`
- User B: `audit-authz-B-B7A98AC2-5E87-4C24-A010-5E106E8CB05C@example.com`

Password `Audit-Passw0rd!2026`. Register -> 201, login -> token (both).

Fixtures:
- User A PRIVATE recipe (`is_private: true`), id `5e645844-70ec-4a26-ae84-0843c3e1dfa7`,
  created via `POST /recipes` (DTO field is `is_private` — see
  `services/backend/internal/domain/recipe.go:128`). Response confirmed `is_private=true`.
- User B shopping list, id `544896d8-7fd5-4c8d-a552-bde1bba29c49`, via `POST /shopping-lists`
  (`sort_type` must be `CATEGORY|STORE`).

AddRecipeToList route: `POST /api/v1/shopping-lists/:id/add-recipe`
(`services/backend/internal/router/router.go:133`).
Users-list route: `GET /api/v1/users/list`
(`services/backend/internal/router/router.go:81`), behind the JWT-protected group.

---

## Item 1 — Cross-user AddRecipe with another user's PRIVATE recipe

Probe: as User B, add User A's PRIVATE recipe to B's own list.

```
POST /api/v1/shopping-lists/544896d8-7fd5-4c8d-a552-bde1bba29c49/add-recipe
Authorization: Bearer <User B>
{"recipe_id":"5e645844-70ec-4a26-ae84-0843c3e1dfa7","servings":2}
```

Response:
```
HTTP_STATUS=500
{"error":"failed to add recipe to list"}
```

Post-probe verification: `GET /shopping-lists/<B list>` -> `item_count = 0`
(A's private recipe was NOT added).

Code (service-layer authorization check confirmed):
`services/backend/internal/service/shopping_list_service.go:286-288`
```go
// Don't let a member pull another member's private recipe into their list
if recipe.IsPrivate && recipe.UserID != userID {
    return errors.ErrUnauthorized
}
```
(`errors.ErrUnauthorized` defined at `internal/errors/errors.go:58`.)

Note (defense-quality, not a bypass): the handler
`services/backend/internal/handler/shopping_list_handler.go:245-249` maps EVERY
service error — including `ErrUnauthorized` — to `500 Internal Server Error`
rather than `403/404`. The access-control outcome is correct (request denied,
nothing added), but the status code is imprecise. This is a minor hardening nit,
not an authorization failure.

VERDICT: **PASS** — A's private recipe could not be added by B. Service-layer guard
present at `shopping_list_service.go:286-288`. HTTP 500, body `{"error":"failed to
add recipe to list"}`, list remained empty (item_count=0). (Observation: 500 instead
of 403/404 due to handler error mapping at `shopping_list_handler.go:245-249`.)

---

## Item 2 — Members-visible users list (PII exposure)

Probe: as normal member User B:
```
GET /api/v1/users/list
Authorization: Bearer <User B>
```

Response:
```
HTTP_STATUS=200
[{"id":"...","first_name":"...","last_name":"..."}, ... ]  (6 users)
```

PII scan of response body:
- Union of all object keys: `["first_name", "id", "last_name"]`
- contains `email`: False
- contains `password` / `password_hash`: False
- contains `@`: False

Code (service-layer non-PII projection confirmed):
`services/backend/internal/service/user_service.go:218-235` maps `domain.User`
-> `domain.UserSummary`, copying only `ID`, `FirstName`, `LastName`.
`domain.UserSummary` (`internal/domain/user.go:21-25`) has no email/timestamp
fields; `domain.User.PasswordHash` is `json:"-"` (`internal/domain/user.go:11`).

VERDICT: **PASS** — endpoint returns a non-PII projection (id/first_name/last_name
only). No emails, password hashes, or other private data exposed. Service projection
at `user_service.go:218-235`, DTO `domain/user.go:21-25`. HTTP 200. (Endpoint is not
denied to members, but the non-PII projection satisfies the acceptance criterion.)

---

## Summary verdicts

- Item 1 (cross-user AddRecipe of private recipe): **PASS** —
  `shopping_list_service.go:286-288`; HTTP 500, `{"error":"failed to add recipe to
  list"}`, list item_count=0. Minor nit: handler maps ErrUnauthorized to 500
  (`shopping_list_handler.go:245-249`) instead of 403/404.
- Item 2 (users/list PII): **PASS** — `user_service.go:218-235` + `domain/user.go:21-25`;
  HTTP 200, body keys `[id, first_name, last_name]`, no email/password_hash.
