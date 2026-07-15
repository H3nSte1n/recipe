# Phase 3 — Fresh Auth/Session Review

**Subtask:** Re-examine `middleware/auth.go` and `user_service.go` token issuance/validation for
missing `iss`/`aud`/`nbf`/`iat`, empty-claim tolerance, missing token invalidation on reset/delete,
and any new auth route.

**Scope/method:** Static review of `internal/middleware/auth.go`, `internal/middleware/context.go`,
`internal/service/user_service.go`, `internal/router/router.go`, `pkg/config/config.go`, and
`env.development.yaml.sample`. No live probes needed beyond what Phase 2 (`02-jwt.md`) already
covered for signature/alg validation.

---

## Finding 1 — No server-side token revocation: password reset and account deletion do not invalidate previously-issued JWTs

- **Severity:** MEDIUM
- **Evidence:**
  - `internal/service/user_service.go:196-203` (`ValidateToken`) and
    `internal/middleware/auth.go:60-67` (`validateToken`) validate **only** signature + `exp` (via
    the `jwt/v5` library's built-in expiry check). There is no lookup against a revocation list,
    token version, or `password_changed_at`/`token_valid_after` column.
  - `ResetPassword` (`user_service.go:157-183`) updates the password hash and marks the reset token
    used, but never touches anything JWT-related — any JWT minted before the reset remains valid
    for the rest of its lifetime.
  - `Delete` (`user_service.go:205-216`) deletes the user row; the JWT itself has no server-side
    state, so a token minted before deletion keeps passing signature/exp checks in
    `AuthRequired()`. (Whether the *request itself* still succeeds afterwards depends on each
    handler's `GetByID`/`Where user_id = ?` failing — i.e., the token is still treated as
    "authenticated," even though the account is gone; some endpoints, e.g. ones that don't need a
    fresh user row, would still succeed.)
  - Token lifetime is `24h` (`env.development.yaml.sample:22`, `jwt.duration: 24h`).
- **Why it matters:** If a user's JWT is compromised (XSS, shoulder-surfing, leaked log, stolen
  device) and they respond by resetting their password — the standard advice for "I think I've
  been compromised" — the attacker's stolen token is **unaffected** and keeps working for up to 24h.
  The same gap applies to admin-initiated account deletion/ban: the token outlives the account
  action that was supposed to cut the user off.
- **Recommended control:** Add a lightweight revocation check: either (a) a `token_valid_after`
  (or `password_changed_at`) timestamp column on `users`, checked in `AuthRequired`/`ValidateToken`
  against the JWT's `iat` claim (requires adding `iat` to `generateToken`, see Finding 3), or (b) a
  short-lived denylist (Redis/DB) of revoked token IDs (`jti` claim). Bump it on password reset and
  on delete.

## Finding 2 — No `aud`/`iss` claims issued or checked

- **Severity:** LOW
- **Evidence:** `generateToken` (`user_service.go:185-194`) sets only `user_id`, `email`, `exp`.
  Neither `validateToken` (`middleware/auth.go:60-67`) nor `ValidateToken`
  (`user_service.go:196-203`) checks `aud`/`iss`/`nbf`.
- **Why it matters:** Low impact today (single API, single issuer, no other service consuming these
  tokens), but it's cheap defense-in-depth: if this JWT secret/format is ever reused by another
  service, or the app grows a second API surface, tokens minted for one audience could be replayed
  against the other with no code changes required to detect it.
- **Recommended control:** Add `iss` (e.g. `"recipe-api"`) and `aud` at mint time and validate both
  with `jwt.WithIssuer(...)`/`jwt.WithAudience(...)` parser options.

## Finding 3 — No `iat`/`nbf` claim, which blocks the token-revocation control in Finding 1

- **Severity:** LOW (compounding factor for Finding 1)
- **Evidence:** `generateToken` (`user_service.go:185-194`) does not set `iat` or `nbf`. The
  `jwt/v5` library will auto-populate `iat` if using its higher-level claim structs, but this code
  builds a raw `jwt.MapClaims{}` map with only `user_id`/`email`/`exp`, so no `iat` is present.
- **Why it matters:** An `iat`/`token_valid_after` comparison is the cheapest way to implement
  Finding 1's revocation-on-reset without adding a denylist store. Without `iat` on the token, the
  server has nothing to compare against a `password_changed_at` timestamp.
- **Recommended control:** Add `"iat": time.Now().Unix()` to `generateToken`'s claims map as part
  of fixing Finding 1.

## Finding 4 — Auth middleware tolerates a missing/non-string `user_id` claim rather than rejecting the request

- **Severity:** LOW (currently non-exploitable, flagged as a hardening gap)
- **Evidence:**
  - `internal/middleware/auth.go:51` — `userID, _ := claims["user_id"].(string)` discards the
    type-assertion failure; if `user_id` is absent or non-string, `userID` silently becomes `""`
    and the request proceeds to `c.Next()` (not aborted).
  - `internal/middleware/context.go:GetUserID` mirrors this: any non-string or missing value
    returns `""`, not an error.
- **Why it matters:** Today this is not exploitable — `""` does not match any real UUID primary
  key, so downstream `Where user_id = ?` queries return zero rows / not-found rather than an authz
  bypass. It is flagged because it is a silent-degrade pattern: a future handler or raw query that
  treats an empty string specially (e.g. an unscoped list, a `LIKE '%'`-style default, or a
  boolean/ownership check that treats `""` as falsy-meaning-"no owner restriction") would
  silently reintroduce an authz hole with no signal in the auth layer itself.
- **Recommended control:** Have `AuthRequired()` explicitly reject (401) when the `user_id` claim
  is missing, empty, or fails the string type-assertion, instead of forwarding an empty identity.

## New auth routes since prior review

- **None.** `internal/router/router.go:66-70` — `/auth/register`, `/auth/login`,
  `/auth/forgot-password`, `/auth/reset-password` are unchanged from the prior inventory
  (`docs/security-review/00-route-inventory.md`); cross-checked against `01-routes.md` from this
  audit's Phase 1, which reported no new/changed auth routes.

## Checks performed

1. Read `internal/middleware/auth.go` end-to-end for header parsing, alg enforcement, claim
   extraction, and abort/continue paths.
2. Read `internal/middleware/context.go` for how `user_id` is retrieved downstream.
3. Read `internal/service/user_service.go` in full: `Register`, `Login`, `ForgotPassword`,
   `ResetPassword`, `Delete`, `generateToken`, `ValidateToken`.
4. Grepped `pkg/config/config.go` and `env.development.yaml.sample` for `jwt.duration` /
   `expiration_hours` to establish the token lifetime window relevant to Finding 1's blast radius.
5. Cross-checked `internal/router/router.go` auth-route group against this audit's `01-routes.md`
   for new/changed endpoints.
6. Relied on Phase 2's `02-jwt.md` for already-verified signature/alg/boot-refusal behavior
   (not re-tested here to avoid duplicate live probes against the shared stack).

---

*No production code was modified. This file is the only artifact written.*
