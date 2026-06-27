# Backend Auth & Session Review — JWT, Login/Register/Reset

Scope: JWT verification, token issuance, and the public auth flows (register / login / forgot-password / reset-password) of the Recipe Go backend (Gin + GORM + golang-jwt/v5). Grounded in `internal/middleware/auth.go`, `internal/service/user_service.go`, `internal/handler/user_handler.go`, `internal/domain/{auth,user}.go`, `pkg/config/config.go`, and the wiring in `cmd/api/main.go` / `internal/router/router.go`. Read-only review.

---

## Findings

### [High] No validation that the JWT signing secret is non-default, non-empty, or sufficiently strong

**Location:**
- `pkg/config/config.go:45-49` (`JWTConfig.Secret`, no constraints)
- `internal/router/router.go:25` (`middleware.NewAuthMiddleware(config.JWT.Secret)`)
- `internal/service/services.go:38` (`NewUserService(..., config.JWT.Secret, ...)`)
- `internal/middleware/auth.go:60-66` (`[]byte(m.secretKey)` used directly as HMAC key)
- `env.development.yaml.sample:18` / `env.development.yaml:15` (`secret: your-super-secret-key-here`)

**Description:** The HMAC signing key is read verbatim from config (`jwt.secret`) and passed straight into both the issuing service and the verification middleware. Nothing anywhere checks that the secret was changed from the shipped placeholder, that it is non-empty, or that it meets a minimum length. The committed sample config (`env.development.yaml.sample:18`) hard-codes the guessable placeholder `your-super-secret-key-here`, and an operator who copies the sample (the documented setup path in `CLAUDE.md`) inherits that exact value. If the secret were ever empty, `[]byte("")` would be used as the HMAC key with no guard, making tokens trivially forgeable.

Note on the env-var override path: config uses Viper `AutomaticEnv()` + `Unmarshal` (`pkg/config/config.go:86-95`). Viper's `AutomaticEnv` frequently does **not** populate struct fields through `Unmarshal` unless each key is explicitly `BindEnv`'d (none are here), so a `JWT_SECRET` environment variable may not actually override the file value at runtime — meaning the YAML default could be the only effective source. This was not verifiable under the read-only/no-build constraint, but if the override does not bind, the placeholder is the live key.

**Impact:** JWT identity is carried entirely by the `user_id` claim (`internal/service/user_service.go:185-189`). An attacker who knows or guesses the signing secret can forge a token for any `user_id` and fully impersonate any user (the project is published as `github.com/H3nSte1n/recipe`, so the placeholder default is public). This is a complete authentication bypass. Severity escalates to Critical if the default/placeholder secret is deployed to production.

**Recommendation:** Fail closed at startup: refuse to boot if `jwt.secret` is empty, equal to the known placeholder(s), or shorter than ~32 bytes. Generate the secret from a CSPRNG per environment and inject it only via a real secret store / env var, and verify the env override actually binds (use explicit `viper.BindEnv("jwt.secret")` or read `os.Getenv` directly). Never ship a usable default.

---

### [Medium] No rate limiting or lockout on login, forgot-password, or reset-password

**Location:** `internal/router/router.go:50-58` (public auth routes; only `cors` + `auth` middleware exist — `internal/middleware/` contains just `auth.go`, `context.go`, `cors.go`). No rate-limit dependency is present (`go.mod`/`go.sum` have no `x/time/rate`, tollbooth, or similar).

**Description:** `POST /auth/login`, `POST /auth/forgot-password`, and `POST /auth/reset-password` have no throttling, no per-account lockout, and no CAPTCHA. bcrypt cost 14 adds per-attempt cost but does not bound attempt volume.

**Impact:** Unlimited online password brute-force and credential-stuffing against `login`; reset-token brute-force against `reset-password` (token is a 64-hex-char value, so infeasible to guess in 1h, but still un-throttled); and password-reset email flooding / SMTP abuse against `forgot-password` (attacker can spam reset mails to any registered address).

**Recommendation:** Add IP- and account-scoped rate limiting and exponential backoff / temporary lockout on the auth endpoints (e.g. a limiter middleware in `internal/middleware/`). Throttle `forgot-password` per target email and per source IP.

---

### [Medium] Stateless tokens are not revocable; password reset and account deletion do not invalidate outstanding JWTs

**Location:** `internal/service/user_service.go:156-182` (`ResetPassword`), `204-215` (`Delete`), `184-193` (`generateToken`); token lifetime `jwt.duration: 24h` (`env.development.yaml:16`).

**Description:** JWTs are self-contained with a 24h `exp` and no server-side session/denylist. `ResetPassword` updates the password hash and marks the single reset token used, but does nothing to existing access tokens. `Delete` removes the user row but issued tokens remain cryptographically valid until `exp`. There is also no `iat`/token-version mechanism to bound this.

**Impact:** A stolen or previously-issued token stays valid for up to 24 hours even after the victim resets their password (the standard "I changed my password to lock out the attacker" expectation fails) or after the account is deleted. Combined with the High secret finding, forged tokens cannot be revoked either.

**Recommendation:** Either shorten access-token lifetime (e.g. 15 min) with a refresh-token flow, or add a server-side revocation signal (per-user token version / `tokenVersion` claim bumped on password reset and delete, or a denylist keyed by `jti`). At minimum, invalidate sessions on password reset.

---

### [Medium] User enumeration via register response, and via login/forgot-password timing & status

**Location:**
- `internal/service/user_service.go:60-67` + `internal/handler/user_handler.go:28-31` (register returns `"email already registered"`, HTTP 400)
- `internal/service/user_service.go:107-115` (login: missing user returns immediately at `GetByEmail`, before bcrypt)
- `internal/service/user_service.go:128-153` + `internal/handler/user_handler.go:60-67` (forgot-password)

**Description:** Three separate oracles reveal whether an email is registered:
1. **Register** returns the explicit message `"email already registered"` for existing addresses, directly confirming account existence.
2. **Login** short-circuits to `"invalid credentials"` *without* running `bcrypt.CompareHashAndPassword` when the email is unknown (`GetByEmail` errors first). For a known email with a wrong password it runs bcrypt at cost 14 (~hundreds of ms). The measurable timing delta distinguishes registered from unregistered emails despite the identical message.
3. **Forgot-password** returns a generic `"if the email exists..."` message (good intent), but the existing-email path performs a DB insert **plus a synchronous SMTP send** (`SendPasswordResetEmail`, line 153) while the nonexistent path returns immediately (line 134) — a large network-latency timing delta. Worse, if SMTP send fails, the handler returns **HTTP 500** (`user_handler.go:60-62`) only on the existing-email branch, so status code can also leak existence.

**Impact:** An attacker can enumerate valid accounts (for targeted phishing, credential stuffing, or reset spam) directly from register, and via timing/status from login and forgot-password even though their messages are generic.

**Recommendation:** Make register's response generic (e.g. "if this email is available, your account was created" / always 200 then send a verification mail), or accept the tradeoff consciously. For login, always run a bcrypt comparison against a dummy hash when the user is absent to equalize timing. For forgot-password, send email asynchronously (enqueue) so the response time is constant, and return 200 regardless of SMTP outcome (log failures server-side).

---

### [Low] JWT has no audience, issuer, not-before, or issued-at claims and none are validated

**Location:** `internal/service/user_service.go:185-189` (claims: only `user_id`, `email`, `exp`); `internal/middleware/auth.go:60-66` (parse validates signature + default `exp`, no `aud`/`iss` options).

**Description:** Issued tokens contain no `iss`, `aud`, `nbf`, or `iat`. Verification does not assert any issuer/audience. golang-jwt/v5 enforces `exp` automatically when present (and `exp` is set — see positive below), but there is no binding of the token to this service.

**Impact:** Low for a single self-contained service, but if the same signing secret were ever reused across services/environments, a token minted for one would be accepted by another. No defense-in-depth against token reuse across contexts.

**Recommendation:** Add `iss` and `aud` claims at issuance and validate them in the middleware (`jwt.WithAudience`, `jwt.WithIssuer`). Add `iat` for auditability.

---

### [Low] Middleware silently tolerates missing/empty identity claims

**Location:** `internal/middleware/auth.go:44-54`.

**Description:** After `token.Valid` is checked, `user_id` and `email` are extracted with the comma-ok form but the failure is discarded (`userID, _ := claims["user_id"].(string)`). A validly-signed token lacking a `user_id` claim passes the middleware with `userID == ""`, which is then `c.Set` and propagated to handlers.

**Impact:** Low. Any validly-signed token (which already requires the secret) is accepted even if malformed; downstream handlers receive an empty `user_id` and would operate as a "user" owning no resources. No privilege gain on its own, but it removes a sanity check and depends on every handler treating empty `user_id` safely. Some handlers (e.g. `DeleteAccount`, `user_handler.go:88-92`) do guard `userID == ""`; not all are guaranteed to.

**Recommendation:** Reject the request (401) if `user_id` is absent or empty after claim extraction, rather than proceeding with a blank identity.

---

### [Info — positive] Signing algorithm is pinned to HMAC; `none`/alg-confusion is blocked at both verification sites

**Location:** `internal/middleware/auth.go:60-66` and `internal/service/user_service.go:195-202`.

**Description:** Both verifiers assert `token.Method.(*jwt.SigningMethodHMAC)` inside the keyfunc and reject anything else, so `alg: none` and RS256↔HS256 confusion (passing an RSA public key as an HMAC secret) are not exploitable. Tokens are issued with `jwt.SigningMethodHS256` (`user_service.go:191`). No issue found in this sub-area. (Note: `userService.ValidateToken` appears unused — the middleware reimplements verification — harmless duplication.)

---

### [Info — positive] Signature is actually validated and `token.Valid` / `exp` are enforced

**Location:** `internal/middleware/auth.go:37-49`; `exp` set at `internal/service/user_service.go:188`.

**Description:** `jwt.Parse` performs real signature verification with the configured key, and the middleware additionally checks `token.Valid` (line 45) before trusting claims. The `exp` claim is set at issuance and golang-jwt/v5 validates expiry by default, so expired tokens are rejected. The token-verification path has no "parse-but-don't-verify" gap. (Lifetime length is addressed separately as Medium.)

---

### [Info — positive] Password hashing uses bcrypt cost 14 and the hash is never serialized

**Location:** `internal/domain/user.go:11` (`PasswordHash ... json:"-"`), `36-45` (`bcrypt.GenerateFromPassword(..., 14)`, `CompareHashAndPassword`).

**Description:** Passwords are hashed with bcrypt at cost factor 14 (above the common 10–12 baseline; good) and compared in constant time by bcrypt. The hash field carries `json:"-"`, so it is never returned in `LoginResponse`, `Register`, or `GET /users/list` responses. No plaintext storage, no weak/fast hash. No issue found.

---

### [Info — positive] Password-reset tokens have strong entropy, short expiry, and are single-use

**Location:** `internal/service/user_service.go:137-153` (generation) and `156-182` (consumption); `internal/repository/user_repository.go:81-87, 99` (lookup + mark-used).

**Description:** Reset tokens are 32 bytes from `crypto/rand` (256-bit), hex-encoded, with a 1-hour expiry (`ExpiresAt: now+1h`). `ResetPassword` checks `Used || now.After(ExpiresAt)` and, within a transaction, updates the password and marks the token used — enforcing single use atomically. Entropy, expiry, and single-use are all correct. Minor residual notes (not findings): other outstanding reset tokens for the same user are not invalidated on a successful reset (each still individually expires/one-shot), and lookup is a plain DB equality match (not constant-time, but the 256-bit space makes timing irrelevant).

---

### [Info] Out-of-scope secret observed: live-looking Anthropic API key in env file

**Location:** `env.development.yaml:39` (`anthropic_api_key: sk-ant-api03-...`).

**Description:** The gitignored local env file contains what appears to be a real Anthropic API key. This is not committed to git (the file is `.gitignore`'d) and is outside JWT/session scope. Flagging for the AI/secrets reviewer to confirm rotation; no auth impact.

---

## Summary

Critical: 0 · High: 1 · Medium: 3 · Low: 2 · Info: 5 (4 positive, 1 cross-reference)

Most serious auth finding: **[High] No validation that the JWT signing secret is non-default/non-empty/strong** — the codebase ships a public placeholder secret (`env.development.yaml.sample:18`) and never enforces that it is changed; whoever knows the secret can forge a token for any `user_id` and fully impersonate any user (escalates to Critical if the default reaches production).
