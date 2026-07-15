# Phase 3 — Fresh Config / Error-Handling / DoS Review

**Subtask:** Scan handlers for raw `err.Error()` leakage, Gin debug mode, `fmt.Print` of
secrets/LLM output, missing request-size limits, and panics that could 500/crash.

**Scope/method:** Grepped every file in `internal/handler/` for `err.Error()` usage and classified
each by the HTTP status it's attached to (400 = client input error, generally fine to echo; 500 =
internal/service error, should never be echoed raw). Checked `cmd/api/main.go` for Gin mode
selection. Grepped for `fmt.Print`/`log.Print` and any logging of `APIKey`. Grepped for
`MaxBytesReader`/body-size limits across all routes, not just the known-hardened upload paths.
Live-verified the worst case (Finding 1) using the AI-config duplicate-create case hit during the
IDOR review (`03-idor.md`).

---

## Finding 1 — Internal/service errors are echoed verbatim to the client on several handlers (raw DB error confirmed live)

- **Severity:** MEDIUM
- **Evidence:**
  - `internal/handler/ai_config_handler.go:31,50,75,87,97,109` — `Create`, `Update`, `List`,
    `Delete`, `SetDefault` all do `c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})`,
    passing the raw service/repository error straight through.
  - `internal/handler/profile_handler.go:36` and `internal/handler/user_handler.go:61` — same
    pattern on profile update and (respectively) another 500-path.
  - **Live-confirmed** during this audit's IDOR probes (`03-idor.md`): re-creating an AI config for
    a model the user already has one for returned
    `{"error":"ERROR: duplicate key value violates unique constraint \"user_ai_configs_user_model_key\" (SQLSTATE 23505)"}`
    — the raw Postgres driver error, including the internal constraint name, reached the HTTP
    response body verbatim.
  - By contrast, `internal/handler/recipe_handler.go` (and parts of `ai_config_handler.go`, e.g.
    `:63,121`) instead map every error to a fixed **generic** message
    (`"failed to update recipe"`) but with the **wrong status code** (500 for what may be a 403/404)
    — see `03-idor.md` Finding 1 for that half of the problem. The two files show the codebase has
    two different broken patterns (leak-the-real-error vs. hide-the-real-status) rather than one
    consistent, correct error-mapping strategy.
- **Why it matters:** Raw database errors can reveal schema details (table/column/constraint
  names, driver/version fingerprints via `SQLSTATE` codes) to any authenticated caller — useful
  recon for an attacker probing for injection points or enumerating internal structure, and it's
  simply bad hygiene to let internal errors leave the trust boundary. Severity is MEDIUM rather
  than HIGH because no finding in this audit shows the leaked errors themselves granting access
  (see `03-idor.md` — all cross-tenant attempts were still correctly denied); the risk is
  information disclosure that lowers the cost of a subsequent attack.
- **Recommended control:** Introduce a single error-mapping helper (e.g.
  `apperrors.ToHTTPStatus(err) (int, string)`) that every handler calls: known `*AppError` types
  (`ErrUnauthorized`→403, `ErrNotFound`→404, validation errors→400) get their specific status and a
  safe message; anything else (including raw GORM/driver errors) is logged server-side at `Error`
  level and returns a fixed `500 {"error": "internal server error"}` with no wrapped detail. Apply
  it uniformly across `recipe_handler.go`, `ai_config_handler.go`, `profile_handler.go`,
  `shopping_list_handler.go`, and `user_handler.go`.

## Finding 2 — No global request-body-size limit on JSON endpoints

- **Severity:** LOW/MEDIUM (DoS-adjacent)
- **Evidence:** `http.MaxBytesReader` is applied **only** on the two multipart/binary upload paths
  — image upload (`recipe_handler.go:31`, 10 MiB) and PDF import (`recipe_handler.go:246`, 20 MiB).
  Every other JSON-bodied endpoint (`/auth/register`, `/auth/login`, `/recipes` create/update,
  `/shopping-lists`, `/ai-configs`, `/users` profile update, etc.) calls `c.ShouldBindJSON(&req)`
  with **no** body-size cap — `router.go` has no global body-limit middleware, and `gin.Default()`
  does not impose one itself.
- **Why it matters:** Once the app is reachable without the VPN's implicit rate-limiting
  (`03-vpn-deps.md` Finding 2), an unauthenticated caller can POST an arbitrarily large body to
  `/auth/register` or `/auth/login` (both public, pre-auth) and force the server to buffer/parse
  the whole thing before validation fails — a low-cost memory/CPU exhaustion vector. This compounds
  with `03-injection.md` Finding 2 (uncapped nested-array sizes on authenticated recipe writes).
- **Recommended control:** Add a small global body-size-limit middleware (e.g. 1–2 MiB, well above
  any legitimate JSON payload) ahead of all routes in `router.go`, in addition to the existing
  per-upload caps.

## Finding 3 — Gin release mode is conditional on `APP_ENV=production` being set correctly

- **Severity:** LOW (config-correctness, not a code defect)
- **Evidence:** `cmd/api/main.go:39-44` only calls `gin.SetMode(gin.ReleaseMode)` inside
  `if cfg.App.Env == "production"`; any other value (including an unset/misconfigured `APP_ENV` at
  deploy time) leaves Gin in its default `DebugMode`.
- **Why it matters:** Gin's own documentation advises against running in debug mode in
  production — it prints a verbose startup banner/route table to the process's stdout and is
  "not recommended" for prod, though it does not by itself leak stack traces into HTTP responses
  (panics are still caught by `gin.Recovery()`, included in `gin.Default()`, which returns a
  generic 500 to the client either way). The risk here is purely operational: if the real
  deployment doesn't explicitly set `APP_ENV=production`, this silently degrades rather than
  failing loudly the way the JWT-secret check does (`pkg/config/config.go` `Validate()`).
- **Recommended control:** Treat `APP_ENV` the same way `jwt.secret` is treated — fail closed (or
  at minimum log a prominent warning) if `APP_ENV` isn't exactly `"production"` at boot in a
  non-development build, rather than silently defaulting to debug mode.

## Checked and clean

- **No `fmt.Print`/`log.Print` anywhere in `internal/` or `pkg/`** (grep confirmed) — all logging
  goes through the injected `zap.Logger`, so there's no unstructured stdout path that could leak
  secrets outside the structured log pipeline.
- **No logging of AI API keys**: every log site touching `APIKey` either logs only the *fact* of a
  decrypt failure (`apikey_crypto.go:22`, `zap.Error(err)` only, no key value) or doesn't log the
  key at all. Grep for `zap.*APIKey` / `zap.*apiKey` found no call that logs the key value itself.
- **No obvious panic surface**: grepped handlers/services for unchecked type assertions
  (`.(string)` without `, ok`), raw slice indexing after unchecked parsing, and similar; found none
  outside the already-covered auth middleware pattern (`03-auth.md` Finding 4, which fails safe to
  an empty string rather than panicking).
- **PDF/image upload paths already have hard size caps** (`maxImageUploadBytes` 10 MiB,
  `maxPDFUploadBytes` 20 MiB) enforced via both `MaxBytesReader` and an explicit post-read length
  check (`recipe_handler.go:277`) — this half of the DoS surface is handled correctly; Finding 2
  is specifically about the *other* (JSON) routes.

## Checks performed

1. Grepped every `internal/handler/*.go` for `err.Error()`, cross-referencing each call site's
   HTTP status code to determine if it exposes internal error detail.
2. Live-reproduced a 500 with a leaked raw Postgres error via the AI-config duplicate-create case
   encountered during the IDOR probes.
3. Read `cmd/api/main.go` boot sequence for `gin.SetMode` conditions.
4. Grepped for `fmt.Print`, `log.Print`, and any `zap.*` call referencing `APIKey`/`apiKey`.
5. Grepped for `MaxBytesReader` / body-size limits across the whole router surface, confirming
   only the two known upload paths are capped.
6. Grepped handlers/services for unchecked type assertions and unbounded slice indexing that could
   panic.

---

*No production code was modified. This file is the only artifact written.*
