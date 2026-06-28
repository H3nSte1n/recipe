# 01 — API Attack-Surface Re-Inventory (Audit 2026-06)

Subtask **P1.2** of the Phase-1 post-remediation audit. This re-inventories every HTTP route
from `services/backend/internal/router/router.go` (single source of truth for routing) and diffs
it against the prior recon artifact `docs/security-review/00-route-inventory.md`. Read-only audit;
no production code modified.

Source of truth: `services/backend/internal/router/router.go` (147 lines, read in full).
Cross-checked: `services/backend/internal/handler/uploads_handler.go`,
`services/backend/pkg/signedurl/signedurl.go`.

## Middleware chains (how requests are wrapped)

- **Global (every request):** `gin.Default()` (Logger + Recovery) and `middleware.CORS(config.CORS.AllowedOrigins)` applied on the engine (`router.go:21`, `router.go:27`).
- **Multipart bound:** `engine.MaxMultipartMemory = 10 << 20` (10 MiB in-memory cap; larger parts spill to temp files, with per-handler `MaxBytesReader` enforcing hard limits per the code comment) (`router.go:24-25`).
- **API group:** all API routes under `/api/v1` (`router.go:40`).
- **Public group:** `setupPublicRoutes(v1)` — registered on the bare `v1` group, no auth (`router.go:52`, `63-72`).
- **Protected group:** `protected := v1.Group(""); protected.Use(r.auth.AuthRequired())` — JWT middleware (`router.go:55-57`). `AuthRequired()` requires `Authorization: Bearer <token>`, HMAC-validates the JWT, injects `user_id`/`email` into context.
- **Uploads mount (non-API):** `GET /uploads/:filename` registered directly on the engine (NOT inside `/api/v1`, NOT inside the JWT group), conditional on `config.Storage.Type == "local"` (`router.go:42-49`). It is wrapped by a **signed-URL signature check** inside the handler instead of JWT — see Diff section. Gets global CORS only as middleware.
- No dedicated security-headers middleware is registered at the router level (nosniff/attachment are set inside `UploadsHandler.Serve`, not globally). INCONCLUSIVE whether other handlers set security headers — out of scope for P1.2.

## Current route table

All `/api/v1/...` paths below are prefixed `/api/v1`. Handler files in `services/backend/internal/handler/`.

| Method | Path | Handler | Auth | Notes (router.go anchor) |
|---|---|---|---|---|
| POST | `/auth/register` | `UserHandler.Register` | Public | State-mutating, unauth. `:66` |
| POST | `/auth/login` | `UserHandler.Login` | Public | Returns JWT; brute-force surface. `:67` |
| POST | `/auth/forgot-password` | `UserHandler.ForgotPassword` | Public | Email send / enumeration. `:69` |
| POST | `/auth/reset-password` | `UserHandler.ResetPassword` | Public | Token-gated password change. `:70` |
| GET | `/users` | `ProfileHandler.Get` | JWT | Caller's own profile. `:78` |
| PUT | `/users` | `ProfileHandler.Update` | JWT | Updates own profile. `:79` |
| DELETE | `/users/me` | `UserHandler.DeleteAccount` | JWT | Deletes caller's account. `:80` |
| GET | `/users/list` | `UserHandler.ListAll` | JWT | Lists ALL users (enumeration). `:81` |
| GET | `/ai-configs` | `AIConfigHandler.List` | JWT | Caller's AI configs. `:86` |
| POST | `/ai-configs` | `AIConfigHandler.Create` | JWT | May store provider API keys. `:87` |
| GET | `/ai-configs/:id` | `AIConfigHandler.Get` | JWT | User-controlled `:id` → IDOR. `:88` |
| PUT | `/ai-configs/:id` | `AIConfigHandler.Update` | JWT | User-controlled `:id` → IDOR. `:89` |
| DELETE | `/ai-configs/:id` | `AIConfigHandler.Delete` | JWT | User-controlled `:id` → IDOR. `:90` |
| GET | `/ai-configs/default` | `AIConfigHandler.GetDefault` | JWT | Static seg after `/:id`. `:92` |
| POST | `/ai-configs/:id/set-default` | `AIConfigHandler.SetDefault` | JWT | User-controlled `:id` → IDOR. `:93` |
| GET | `/ai-configs/models` | `AIConfigHandler.ListModels` | JWT | Reference data. `:95` |
| POST | `/recipes` | `RecipeHandler.Create` | JWT | **File upload** (multipart `image`). `:100` |
| GET | `/recipes/:id` | `RecipeHandler.Get` | JWT | User-controlled `:id` → IDOR. `:101` |
| PUT | `/recipes/:id` | `RecipeHandler.Update` | JWT | **File upload**; `:id` → IDOR. `:102` |
| DELETE | `/recipes/:id` | `RecipeHandler.Delete` | JWT | `:id` → IDOR. `:103` |
| GET | `/recipes` | `RecipeHandler.ListMine` | JWT | Caller's recipes. `:105` |
| GET | `/recipes/public` | `RecipeHandler.ListPublic` | JWT | Still requires JWT. `:106` |
| POST | `/recipes/import/url` | `RecipeHandler.ImportFromURL` | JWT | **Fetches user-supplied URL** → SSRF. `:110` |
| POST | `/recipes/import/pdf` | `RecipeHandler.ImportFromPDF` | JWT | **File upload** (multipart `file`). `:111` |
| POST | `/recipes/parser/instructions` | `RecipeHandler.ParsePlainTextInstructions` | JWT | User text → AI parser. `:116` (registered as `parser.POST("instructions", ...)` — no leading slash; resolves to `/recipes/parser/instructions`) |
| POST | `/shopping-lists` | `ShoppingListHandler.Create` | JWT | `:122` |
| GET | `/shopping-lists` | `ShoppingListHandler.List` | JWT | `:123` |
| GET | `/shopping-lists/:id` | `ShoppingListHandler.Get` | JWT | `:id` → IDOR. `:124` |
| PUT | `/shopping-lists/:id` | `ShoppingListHandler.Update` | JWT | `:id` → IDOR. `:125` |
| DELETE | `/shopping-lists/:id` | `ShoppingListHandler.Delete` | JWT | `:id` → IDOR. `:126` |
| POST | `/shopping-lists/:id/items` | `ShoppingListHandler.AddItem` | JWT | Nested. `:128` |
| PUT | `/shopping-lists/:id/items/:itemId` | `ShoppingListHandler.UpdateItem` | JWT | Nested; verify parent ownership. `:129` |
| DELETE | `/shopping-lists/:id/items/:itemId` | `ShoppingListHandler.DeleteItem` | JWT | Nested; verify parent ownership. `:130` |
| PATCH | `/shopping-lists/:id/items/:itemId/toggle` | `ShoppingListHandler.ToggleItem` | JWT | Nested; verify parent ownership. `:131` |
| POST | `/shopping-lists/:id/add-recipe` | `ShoppingListHandler.AddRecipe` | JWT | `:id` + recipe id in body. `:133` |
| GET | `/shopping-lists/:id/sorted` | `ShoppingListHandler.SortByStore` | JWT | `chain_id` query; `:id` → IDOR. `:134` |
| GET | `/store-chains` | `StoreChainHandler.List` | JWT | Reference data; `country` filter. `:139` |
| GET | `/store-chains/:id` | `StoreChainHandler.Get` | JWT | Reference data lookup. `:140` |
| GET | `/uploads/:filename` | `UploadsHandler.Serve` | **Signed-URL** (conditional, storage=local) | NOT JWT, NOT under `/api/v1`. Signature-checked. `:48` |

**Current counts:** 38 API routes (4 public `/auth/*` + 34 JWT-protected) + 1 conditional
`/uploads/:filename` mount. This exactly reconciles with the prior review's "38 API routes" count.

## Diff vs `00-route-inventory.md`

### New routes
- **None.** No API endpoint was added between the prior recon and this audit. The 38 API routes are identical in method/path/auth-group.

### Removed routes
- **None.** No API endpoint was removed.

### Changed routes
- **`/uploads` — CHANGED (the one material change; remediation PR work).**
  - **Prior (`00-route-inventory.md:68`, `:81-84`):** Gin **static file server** (`engine.Static`) mounted at `/uploads/*`, **fully public, no auth, no signature** — any file in `config.Storage.LocalPath` publicly readable, with path-traversal / enumeration concerns flagged.
  - **Now (`router.go:42-49`):** route is `GET /uploads/:filename` served by **`handler.UploadsHandler.Serve`**, gated by a **short-lived signed URL**. A `signedurl.Signer` is built from `config.JWT.Secret` with `signedurl.DefaultTTL` (`router.go:46`), passed to `NewUploadsHandler(localPath, signer, logger)` (`router.go:47`, confirmed in `uploads_handler.go:24`/`:28`). Comment states files now require a valid short-lived link and are served with `nosniff` + `attachment`.
  - **Net effect on attack surface:** the open static directory is closed; the new surface is (a) the **signature scheme** (HMAC over `config.JWT.Secret`, TTL = `DefaultTTL`) — verify it cannot be forged/replayed and TTL is sane; and (b) the **`:filename` path parameter** → still a path-traversal candidate (`../`, absolute paths, symlink) to be validated inside `Serve`. Flagged for the uploads/path-traversal phase.
  - Path shape also narrowed: prior wildcard `/uploads/*` → now a single named param `/uploads/:filename` (no nested subpaths). Minor, reduces traversal depth but does not by itself prevent `..` segments — confirm in handler.

### Unchanged but worth re-flagging
- All IDOR `/:id` / `/:itemId` routes, SSRF `import/url`, file-upload routes, `GET /users/list`, and AI-config secret storage are **unchanged at the routing layer**. Whether the remediation PRs (#31-#38) fixed the *handler/service* behavior behind these unchanged routes is OUT OF SCOPE for P1.2 (routing inventory) and must be checked by the per-vuln phases.

## Attack-surface notes for later phases

1. **Signed-URL uploads (`GET /uploads/:filename`, `router.go:48`).** New since prior recon. Audit the signature/TTL design in `pkg/signedurl/signedurl.go` and the `:filename` handling in `uploads_handler.go:Serve` for path traversal, signature forgery/replay, and whether `nosniff`+`attachment` are actually emitted. NOT behind JWT — security relies entirely on the signature.
2. **SSRF (`POST /recipes/import/url`, `router.go:110`).** Takes a user-supplied URL fetched server-side. Recheck `pkg/urlparser` for scheme/IP allow-listing post-remediation.
3. **File uploads (`POST /recipes`, `PUT /recipes`, `POST /recipes/import/pdf`; `router.go:100,102,111`).** Multipart `image`/`file`. With `MaxMultipartMemory=10MiB` + per-handler `MaxBytesReader` now in place (`router.go:24-25`), verify hard size caps, content-type validation, and stored-filename sanitization (the sanitized name feeds the `/uploads/:filename` mount above).
4. **IDOR — nested item routes (`router.go:129-131`).** `UpdateItem`/`DeleteItem`/`ToggleItem` historically bound only `:itemId` and ignored parent `:id`. Routing unchanged; confirm the service now ties item → list → caller.
5. **IDOR — generic `/:id` resources** across ai-configs, recipes, shopping-lists, store-chains. Ownership enforcement lives in services; verify per resource.
6. **User enumeration (`GET /users/list`, `router.go:81`).** Any authenticated user lists all users; review returned fields for PII/secrets.
7. **AI-config secret handling (`POST/PUT /ai-configs`, `router.go:87,89`).** Confirm API keys are never echoed by `GET /ai-configs*` and are encrypted at rest.
8. **Public unauth `/auth/*` (`router.go:66-70`).** register/login/forgot/reset — brute-force, enumeration, reset-token strength.
9. **Routing-order (low pri).** Static segments `/ai-configs/default`, `/ai-configs/models`, `/recipes/public` registered after `/:id` siblings; Gin radix prioritizes static segments, but confirm no shadowing. `parser.POST("instructions", ...)` lacks a leading slash (`router.go:116`) — harmless (resolves to `/recipes/parser/instructions`) but noted.

## INCONCLUSIVE / out of scope for P1.2
- Whether `UploadsHandler.Serve` actually prevents path traversal and whether the signature is sound — flagged for the uploads phase (only the route wiring and constructor signature were verified here).
- Whether handler/service-layer fixes from PRs #31-#38 closed the IDOR/SSRF/upload issues — those are behavioral, checked by later phases; the routing surface for them is unchanged.
- Presence of global security-headers middleware — none seen at router level; not exhaustively traced.
</content>
</invoke>
