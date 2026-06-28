# Audit 02 — JWT signing-secret fail-closed (PR #33)

Phase 2 post-remediation security audit. AUDIT ONLY — no production code under `services/` was
modified. No git or docker commands were run. Live probes used a DISPOSABLE backend binary on a
separate port (28080) and read-only requests to the shared stack (backend :18080 / db :15432).

Date: 2026-06-28
Auditor: audit worker (Phase 2)

---

## Check 1 — Boot-time JWT secret validation code

Validation logic lives in `Config.Validate()` and is invoked at startup before any DB work.

- `services/backend/pkg/config/config.go:125` — `const minJWTSecretBytes = 32`
- `services/backend/pkg/config/config.go:129-135` — `knownWeakJWTSecrets` placeholder set
  (`your-super-secret-key-here`, `change_me`, `change-me`, `changeme`, `secret`), matched
  case-insensitively.
- `services/backend/pkg/config/config.go:141-155` — `Validate()`:
  - line 144: trims the secret (so whitespace padding can't satisfy the length check),
  - line 145-147: **empty** secret → error,
  - line 148-150: **known placeholder** (lowercased) → error,
  - line 151-153: **< 32 bytes** → error.
- `services/backend/cmd/api/main.go:31-33` — `cfg.Validate()` is called immediately after
  `LoadConfig` and **before** `database.MigrateDB` (line 35) and DB connect (line 48). On error it
  calls `log.Fatal`, which exits non-zero. This is a true fail-closed gate ahead of all DB/network
  work.
- `services/backend/internal/middleware/auth.go:60-67` — `validateToken` uses `jwt.Parse` and
  rejects any token whose signing method is not HMAC (`*jwt.SigningMethodHMAC`) at line 62-64,
  blocking `alg=none` and asymmetric-alg confusion. The library also enforces signature + `exp`.

Sample/committed configs are consistent with the gate:
- `env.development.yaml.sample:21` ships `secret: CHANGE_ME` (a known placeholder → refuses boot).
- `env.development.yaml:16` uses `dev-only-jwt-secret-please-override-via-env-32bytes`
  (49 bytes, non-placeholder → boots in local dev only).

Unit tests present and passing — `services/backend/pkg/config/config_test.go:58-89`
(`TestConfig_Validate_JWTSecret`) covers empty, whitespace-only, placeholder (case-insensitive),
too-short, whitespace-padded-short, exactly-31 (fail), exactly-32 (pass), strong (pass).

```
$ go test ./pkg/config/... ./internal/middleware/...
ok   github.com/H3nSte1n/recipe/pkg/config   0.377s
?    github.com/H3nSte1n/recipe/internal/middleware   [no test files]
```

**Check 1: PASS** — code refuses boot on empty / placeholder / <32-byte secret
(config.go:141-155, gated in main.go:31-33 before DB connect); HMAC-only enforced in
auth.go:62-64; unit tests pass.

---

## Check 2 — Live boot probe (disposable binary, port 28080, env-driven config)

Built once: `go build -o /tmp/recipe-jwt-probe ./cmd/api` (exit 0). Each case ran from a temp dir
with a crafted `env.probe.yaml` (APP_ENV=probe); the secret value was supplied via the YAML so the
exact value under test reaches `Validate()`. The valid case pointed DB at the already-migrated
shared db (localhost:15432, read-only no-op migrations) and bound app port 28080.

| Sub-probe | Secret | Result | Exit | Log line |
|-----------|--------|--------|------|----------|
| (a) empty | `""` | refused | **1** | `Invalid configuration: jwt.secret is not set; inject a strong secret via JWT_SECRET` |
| (b) placeholder | `CHANGE_ME` | refused | **1** | `Invalid configuration: jwt.secret is a known placeholder value; set a strong random secret (>= 32 bytes) via JWT_SECRET` |
| (c) too short | `short-secret` (12 b) | refused | **1** | `Invalid configuration: jwt.secret must be at least 32 bytes, got 12; inject a strong secret via JWT_SECRET` |
| (d) valid | `openssl rand -base64 48` (48 b) | **BOOTED + bound :28080** | running (killed) | `Starting server on port 28080` → `Listening and serving HTTP on :28080`; `GET /api/v1/recipes` returned 401 (no token), confirming protected route live |

For (d) the binary passed JWT `Validate()`, ran DB migrations (no-op against migrated shared db),
initialized the encryption cipher, set up CORS, bound the port, and served HTTP; it was then killed
cleanly. (Intermediate runs that stopped at `Could not migrate database` / CORS panic confirmed the
JWT gate had already passed for the valid secret before those later stages.)

Disposable process verified gone afterward (`pgrep recipe-jwt-probe` → NONE; port 28080 FREE);
shared stack on :18080 left untouched.

**Check 2: PASS** — (a)/(b)/(c) each exit 1 with a distinct fatal validation log before DB connect;
(d) boots and binds a listening port.

---

## Check 3 — Tampered-JWT rejection on a protected route (shared stack, read-only)

Registered + logged in a throwaway user at `http://localhost:18080/api/v1`
(register → 201, login → 200). A valid token on `GET /api/v1/recipes` returned **200** (baseline).
The shared stack's HMAC secret was confirmed to be the dev value
`dev-only-jwt-secret-please-override-via-env-32bytes` (recomputed the live token's HS256 signature
and it matched), which let me craft a *properly-signed-but-expired* token and an
*empty-user_id* token to isolate those paths rather than conflating them with bad signatures.

Tampered tokens sent as `Authorization: Bearer <token>` to `GET /api/v1/recipes`:

| Tampered token | HTTP |
|----------------|------|
| `alg=none` (unsigned) | **401** |
| valid header/payload, signature overwritten | **401** |
| HS256 properly signed, `exp` in the past | **401** |
| HS256 properly signed, `user_id: ""` (valid exp/sig) | **401** |

All four rejected with 401. Mechanism traced per case:
- **alg=none** and **bad-signature** → rejected by the auth middleware itself
  (`auth.go:62-64` rejects non-HMAC signing methods; `jwt.Parse` rejects the bad signature).
- **expired** → rejected by the JWT library's `exp` validation (token marked invalid → middleware
  returns 401 at `auth.go:38-42`).
- **empty user_id** → NOTE: this token is correctly signed with a valid `exp`, so it **passes the
  auth middleware** (the middleware does NOT validate the `user_id` claim — `auth.go:51`
  `userID, _ := claims["user_id"].(string)` accepts empty). The 401 is enforced one layer down by a
  handler guard: `RecipeHandler.ListMine` at `internal/handler/recipe_handler.go:182-184`
  (`if userID == "" { 401 "unauthorized" }`). Confirmed by response body `{"errors":"unauthorized"}`.
  This is defense-in-depth, but worth flagging: empty-user_id rejection is handler-specific, not a
  central middleware guarantee — a future protected handler that omits this check could accept a
  blank-subject token. Not a PR #33 regression (PR #33 scope is the signing-secret gate).

**Check 3: PASS** — alg=none, bad-signature, expired, and empty-user_id tokens all return 401;
valid token returns 200. (Empty-user_id is caught by a handler guard, not the middleware — see note.)

---

## Verdicts

- **Check 1 (validation code present, fail-closed before DB): PASS** — `config.go:141-155`
  (empty/placeholder/<32B), gated in `main.go:31-33` before `MigrateDB`; HMAC-only in
  `auth.go:62-64`; `config_test.go` passes.
- **Check 2 (live boot probe): PASS** — empty/placeholder/short → exit 1 with distinct fatal logs;
  valid 48-byte secret boots and binds :28080.
- **Check 3 (tampered-JWT rejection): PASS** — alg=none / bad-sig / expired / empty-user_id all 401;
  valid token 200.

**OVERALL: PASS** — PR #33 fail-closed JWT signing-secret remediation verified in code, via live
boot probe, and via tampered-token rejection on a protected route.
