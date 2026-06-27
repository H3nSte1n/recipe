# 00 — Route Inventory & Auth Map

This document maps every HTTP API route of the Recipe backend to its handler function and records
whether the route is **PUBLIC** (no authentication) or **JWT-PROTECTED**. It is grounded in the
route definitions in `services/backend/internal/router/router.go` (the single source of truth for
routing) and the JWT middleware in `services/backend/internal/middleware/auth.go`. This is a
reconnaissance artifact only — it inventories the attack surface and flags items for later phases;
it does **not** perform the security analysis itself.

## How auth is applied

- All API routes live under the group `/api/v1` (`router.go:32`).
- **Public** routes are registered in `setupPublicRoutes` on the bare `v1` group with no middleware
  (`router.go:39`, `50-59`).
- **Protected** routes are registered on a child group that applies the JWT middleware:
  `protected := v1.Group(""); protected.Use(r.auth.AuthRequired())` (`router.go:42-44`).
- `AuthRequired()` (`middleware/auth.go:21-58`) requires an `Authorization: Bearer <token>` header,
  HMAC-validates the JWT against the configured secret, and injects `user_id` + `email` into the
  Gin context. Handlers retrieve the caller identity via `middleware.GetUserID(c)`.
- A static file server is conditionally mounted at `/uploads` when storage type is `local`
  (`router.go:34-36`) — this is **not** behind the JWT group (see Observations).

## Route table

All paths below are prefixed with `/api/v1`. Handler files live in
`services/backend/internal/handler/`.

| HTTP Method | Path | Handler (file:function) | Auth | Notes |
|---|---|---|---|---|
| POST | `/auth/register` | `user_handler.go:Register` | Public | Creates a user + profile. State-mutating, unauthenticated. |
| POST | `/auth/login` | `user_handler.go:Login` | Public | Returns JWT. Credential endpoint (brute-force surface). |
| POST | `/auth/forgot-password` | `user_handler.go:ForgotPassword` | Public | Sends reset email. Mutates state / triggers email. |
| POST | `/auth/reset-password` | `user_handler.go:ResetPassword` | Public | Consumes reset token, changes password. Unauthenticated state change. |
| GET | `/users` | `profile_handler.go:Get` | JWT | Returns caller's own profile (uses `GetUserID`). |
| PUT | `/users` | `profile_handler.go:Update` | JWT | Updates caller's own profile. |
| DELETE | `/users/me` | `user_handler.go:DeleteAccount` | JWT | Requires `?confirm=true`. Deletes caller's account. |
| GET | `/users/list` | `user_handler.go:ListAll` | JWT | Lists ALL users; no role/owner check (see Observations). |
| GET | `/ai-configs` | `ai_config_handler.go:List` | JWT | Lists caller's AI configs. |
| POST | `/ai-configs` | `ai_config_handler.go:Create` | JWT | Stores AI provider config (may hold API keys/secrets). |
| GET | `/ai-configs/:id` | `ai_config_handler.go:Get` | JWT | User-controlled `:id` → IDOR check needed in service. |
| PUT | `/ai-configs/:id` | `ai_config_handler.go:Update` | JWT | User-controlled `:id` → IDOR check needed. |
| DELETE | `/ai-configs/:id` | `ai_config_handler.go:Delete` | JWT | User-controlled `:id` → IDOR check needed. |
| GET | `/ai-configs/default` | `ai_config_handler.go:GetDefault` | JWT | Static segment; registered after `/:id` (routing note). |
| POST | `/ai-configs/:id/set-default` | `ai_config_handler.go:SetDefault` | JWT | User-controlled `:id` → IDOR check needed. |
| GET | `/ai-configs/models` | `ai_config_handler.go:ListModels` | JWT | Lists available AI models (reference data). |
| POST | `/recipes` | `recipe_handler.go:Create` | JWT | **File upload** (multipart `image`) + JSON. |
| GET | `/recipes/:id` | `recipe_handler.go:Get` | JWT | User-controlled `:id` → IDOR check needed. `nutrition_level` query. |
| PUT | `/recipes/:id` | `recipe_handler.go:Update` | JWT | **File upload** (multipart `image`); user-controlled `:id` → IDOR. |
| DELETE | `/recipes/:id` | `recipe_handler.go:Delete` | JWT | User-controlled `:id` → IDOR check needed. |
| GET | `/recipes` | `recipe_handler.go:ListMine` | JWT | Lists caller's recipes. |
| GET | `/recipes/public` | `recipe_handler.go:ListPublic` | JWT | Paginated public recipes (still requires a valid JWT). |
| POST | `/recipes/import/url` | `recipe_handler.go:ImportFromURL` | JWT | **Fetches an external, user-supplied URL** → SSRF risk. |
| POST | `/recipes/import/pdf` | `recipe_handler.go:ImportFromPDF` | JWT | **File upload** (multipart `file`); reads full file into memory. |
| POST | `/recipes/parser/instructions` | `recipe_handler.go:ParsePlainTextInstructions` | JWT | Sends user text to AI parser. |
| POST | `/shopping-lists` | `shopping_list_handler.go:Create` | JWT | Creates list for caller. |
| GET | `/shopping-lists` | `shopping_list_handler.go:List` | JWT | Lists caller's lists. |
| GET | `/shopping-lists/:id` | `shopping_list_handler.go:Get` | JWT | User-controlled `:id` → IDOR. Sort/store query params. |
| PUT | `/shopping-lists/:id` | `shopping_list_handler.go:Update` | JWT | User-controlled `:id` → IDOR. |
| DELETE | `/shopping-lists/:id` | `shopping_list_handler.go:Delete` | JWT | User-controlled `:id` → IDOR. |
| POST | `/shopping-lists/:id/items` | `shopping_list_handler.go:AddItem` | JWT | Nested resource; `:id` is the list. |
| PUT | `/shopping-lists/:id/items/:itemId` | `shopping_list_handler.go:UpdateItem` | JWT | Nested. Handler uses only `:itemId` + userID — IDOR/ownership check needed (`:id` ignored). |
| DELETE | `/shopping-lists/:id/items/:itemId` | `shopping_list_handler.go:DeleteItem` | JWT | Nested. Handler uses only `:itemId` + userID — IDOR check needed. |
| PATCH | `/shopping-lists/:id/items/:itemId/toggle` | `shopping_list_handler.go:ToggleItem` | JWT | Nested. Handler uses only `:itemId` + userID — IDOR check needed. |
| POST | `/shopping-lists/:id/add-recipe` | `shopping_list_handler.go:AddRecipe` | JWT | Pulls recipe ingredients into list; `:id` + recipe id in body. |
| GET | `/shopping-lists/:id/sorted` | `shopping_list_handler.go:SortByStore` | JWT | Requires `chain_id` query; `:id` → IDOR. |
| GET | `/store-chains` | `store_chain_handler.go:List` | JWT | Reference data; `country` query filter. |
| GET | `/store-chains/:id` | `store_chain_handler.go:Get` | JWT | Reference data lookup by id. |
| (static) | `/uploads/*` | Gin static file server (`router.go:35`) | **Public** (conditional) | Serves files from `config.Storage.LocalPath` when storage type is `local`. No JWT. |

**Counts:** 38 API routes (4 public auth routes + 34 JWT-protected) plus 1 conditional public
static file mount at `/uploads`.

## Observations (flags for later phases — not analyzed here)

1. **Public state-mutating routes.** All four `/auth/*` endpoints are unauthenticated and mutate
   state: `register` (account creation / user enumeration), `login` (brute-force / credential
   stuffing surface), `forgot-password` (email send + possible account enumeration), and
   `reset-password` (password change gated only by a token — token strength/expiry/single-use must
   be reviewed).

2. **Public static file serving at `/uploads`.** Mounted outside the JWT group (`router.go:34-36`).
   Anything written to local storage (uploaded recipe images, imported PDFs) is publicly readable
   with no auth. Review for: object-reference predictability/enumeration, path traversal in stored
   filenames, and whether sensitive uploads (PDFs) end up here.

3. **SSRF — `POST /recipes/import/url`.** `ImportFromURL` fetches a user-supplied URL server-side
   (`recipe_handler.go:183-204`). Classic SSRF vector (internal metadata endpoints, internal
   services, file:// schemes). Needs URL allow-listing / scheme + IP validation review in the
   `urlparser` package.

4. **File-upload routes.** `POST /recipes` and `PUT /recipes` accept a multipart `image`
   (`recipe_handler.go:32-48, 70-85`); `POST /recipes/import/pdf` accepts a multipart `file` and
   reads the **entire file into memory** sized by the client-provided `file.Size`
   (`recipe_handler.go:206-241`). Review for: size limits / DoS, content-type validation, stored
   filename sanitization, and the partial-read bug pattern (`f.Read` may not read the full file).

5. **IDOR on user-controlled IDs.** Every `/:id` and `/:itemId` route accepts a client-supplied
   UUID. Handlers forward `userID` to the service layer, so ownership enforcement lives in the
   services/repositories and must be verified there for each resource (ai-configs, recipes,
   shopping-lists, shopping-list items). **Highest-risk subset:** the shopping-list *item* routes
   (`UpdateItem`, `DeleteItem`, `ToggleItem`) bind only `:itemId` and ignore the parent `:id`
   (`shopping_list_handler.go:166-229`) — confirm the service ties the item to a list owned by the
   caller, otherwise any user could mutate any item by guessing its UUID.

6. **`GET /users/list` exposes all users.** `ListAll` (`user_handler.go:109-117`) returns every
   user with no role or ownership restriction — any authenticated user can enumerate the full user
   directory. Review what fields are returned (PII / password hashes / emails).

7. **AI config may store secrets.** `POST/PUT /ai-configs` likely persists provider API keys.
   Confirm such secrets are never returned by `GET /ai-configs*` responses and are encrypted at
   rest.

8. **Routing-order note (low priority, not a vuln yet).** `/ai-configs/default` and
   `/ai-configs/models` are registered *after* `/ai-configs/:id` (`router.go:75-82`); likewise
   `/recipes/public` after `/recipes/:id`. Gin's radix router gives static segments priority, so
   this resolves correctly, but it's worth confirming no shadowing during the review.

9. **`/recipes/public` requires auth.** Despite the name, listing "public" recipes still needs a
   valid JWT (it is in the protected group). Note for completeness; not a finding.
