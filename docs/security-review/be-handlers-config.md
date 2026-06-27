# Backend Review — Handlers, CORS & Config (input validation, mass assignment, info leakage, secrets)

> Phase: handler / configuration security review of the Recipe backend (Go / Gin). Every finding
> below is grounded in real code with `file:line` anchors. Scope: CORS middleware, app/secret
> configuration, and a representative sample of HTTP handlers (request binding, validation, error
> responses). Service- and repository-layer authorization (IDOR), SSRF, and the AI/email packages
> are covered by their own phase documents and are referenced only where they intersect a handler.

Files reviewed:
- `internal/middleware/cors.go`
- `pkg/config/config.go`, `env.development.yaml`, `env.development.yaml.sample`
- `cmd/api/main.go`, `internal/router/router.go`
- Handlers: `user_handler.go`, `profile_handler.go`, `recipe_handler.go`, `ai_config_handler.go`,
  `shopping_list_handler.go`
- DTOs: `internal/domain/{user,auth,profile,recipe,ai_config}.go`

---

### [High] Weak/default JWT signing secret with no startup validation

**Location:** `pkg/config/config.go:45-49` (`JWTConfig.Secret`), `internal/router/router.go:25`
(`middleware.NewAuthMiddleware(config.JWT.Secret)`), `internal/middleware/auth.go:65`
(`return []byte(m.secretKey), nil`), `env.development.yaml:15` and `env.development.yaml.sample:18`
(`secret: your-super-secret-key-here`).

**Description:** The HS256 JWT signing key is loaded verbatim from config with **no validation of
length, entropy, or that it differs from the shipped default**. Both the committed sample and the
on-disk dev config use the guessable literal `your-super-secret-key-here`. `LoadConfig` enables
`viper.AutomaticEnv()` (`config.go:86`) so the secret *can* be overridden by an env var, but there
is no floor: if the env var is unset, the app silently boots with whatever the YAML contains
(including the default), and nothing fails closed.

**Impact:** Anyone who knows (or guesses) the signing secret can forge a JWT for any `user_id` and
fully impersonate any account — complete authentication bypass. Because the same default value
appears in version-controlled sample files, a deployment that forgets to override it is trivially
compromised. The `expiration_hours` is parsed as a `time.Duration` (`config.go:48`) from a YAML
string `24h`, which is fine, but does not mitigate a forgeable key.

**Recommendation:** Reject startup when `JWT.Secret` is empty, shorter than ~32 bytes, or equal to
any known sample default (`log.Fatal`). Generate per-environment secrets from a CSPRNG and inject
via secret manager / env only; never ship a real or placeholder secret in a tracked file. Consider
rotating to asymmetric (RS/ES) signing if multiple services must verify tokens.

---

### [Medium] Handlers leak raw internal error strings to clients (`err.Error()`)

**Location:** widespread. Examples:
- `user_handler.go:24,30` (Register bind + service error), `:46` (Login → 401 `err.Error()`),
  `:61` (ForgotPassword → 500 `err.Error()`), `:78` (ResetPassword).
- `ai_config_handler.go:31,41,75,87,97,109` (Create/Update/List/Delete/ListModels/SetDefault all
  return `gin.H{"error": err.Error()}` on 500).
- `profile_handler.go:36` (Update → 500 `err.Error()`).
- Every handler returns `c.JSON(400, gin.H{"error": err.Error()})` on `ShouldBindJSON` failure
  (`recipe_handler.go:45,82,192,228,251`; `shopping_list_handler.go:33,112,153,175,217,241`; etc.).

**Description:** Numerous handlers forward the raw Go error string straight to the HTTP client. For
binding errors this echoes go-playground/validator internals; for service/repository errors it can
surface GORM/PostgreSQL messages (constraint names, column names, SQL fragments), file-system
paths, SMTP errors, or AI-provider error text (`claude_model.go:38` wraps `Claude API error: %w`).
Note that several *other* handlers already do this correctly by returning a generic string and
logging the detail (e.g. `recipe_handler.go:59,140`; `shopping_list_handler.go:40,95`), so the
codebase is inconsistent.

**Impact:** Information disclosure that aids reconnaissance (schema/column discovery, internal
hostnames, library/version fingerprinting). `Login` (`:46`) and `ForgotPassword` (`:61`) are the
most sensitive: `ForgotPassword` deliberately returns a generic 200 body to prevent account
enumeration (`:65-67`), but the **500 branch leaks `err.Error()`**, undermining that control and
potentially confirming account/SMTP state.

**Recommendation:** Return generic, stable client messages (e.g. `"invalid request"`,
`"internal error"`) and log the real error server-side with `zap` (as the recipe/shopping handlers
already do). For bind failures, map validator errors to a curated field-level message rather than
echoing the raw string. Audit `ForgotPassword` so neither success nor failure paths leak state.

---

### [Medium] Plaintext live secret stored in on-disk config; secrets sourced from YAML

**Location:** `env.development.yaml:38` (a real-looking Anthropic key
`anthropic_api_key: sk-ant-api03-…`), plus `db.password` (`:10`), `smtp.password` (`:23`),
`jwt.secret` (`:15`), and AWS keys in the sample (`env.development.yaml.sample:36-37`).
Loader: `pkg/config/config.go:78-98`.

**Description:** Secrets are read from per-environment YAML files. `env.development.yaml` is
git-ignored (confirmed: `git ls-files` returns nothing for it) so it is **not committed**, but it
currently contains what appears to be a **real, active Anthropic API key in cleartext on disk**.
The tracked `.sample` carries placeholder secrets that double as the effective defaults if an
operator copies the file without changing them. `viper.AutomaticEnv()` (`config.go:86`) permits env
overrides, but there is no enforcement that production secrets come from env rather than the file.

**Impact:** A real provider key in a plaintext working-tree file is exposed to anyone with host/
repo-clone/backup access and is at risk of accidental commit (one `.gitignore` change away).
Disclosure enables unauthorized, billable use of the AI provider account. The pattern also
encourages shipping placeholder DB/SMTP/JWT secrets into running environments.

**Recommendation:** Rotate/revoke the Anthropic key immediately and treat it as compromised. Move
all secrets out of YAML into a secret manager or injected env vars; keep only non-secret structure
in files. Add a pre-commit/secret-scanning hook. Never store live keys in dev config files.

---

### [Medium] Gin runs in debug mode outside production; default env is `development`

**Location:** `cmd/api/main.go:19-39` (env defaults to `development`; `gin.SetMode(gin.ReleaseMode)`
only when `cfg.App.Env == "production"`), `internal/router/router.go:18` (`gin.Default()`).

**Description:** Gin mode is tied to `App.Env`, which defaults to `development` when `APP_ENV` is
unset (`main.go:19-21`). Unless `env == "production"`, the engine stays in debug mode, which emits
`[GIN-debug]` route dumps and warnings and is explicitly flagged by Gin as not for production.
`gin.Default()` installs the Logger and Recovery middleware; Recovery returns a 500 without a stack
to the client, but debug-mode verbosity and the reliance on a single env string for the security
posture are fragile (a misconfigured/blank `APP_ENV` silently downgrades to debug).

**Impact:** Verbose diagnostics and route enumeration in logs; an operational misconfiguration
(`APP_ENV` unset/typo) silently runs a "production" deployment in debug mode. Low direct exploit.

**Recommendation:** Default to release mode and require an explicit opt-in for debug; fail closed if
`APP_ENV` is unrecognized. Confirm the Recovery handler never returns stack traces to clients and
that panics are logged structurally.

---

### [Low] Validation bypassed on multipart recipe submissions

**Location:** `recipe_handler.go:32-42` (Create) and `:70-79` (Update) — the `recipe` form field is
parsed with `json.Unmarshal`, **not** `ShouldBindJSON`. DTO validators that are skipped:
`CreateRecipeRequest.SourceType binding:"required,oneof=URL MANUAL PDF IMAGE"`,
`Servings binding:"required,min=1"`, `Rating binding:"omitempty,min=0,max=5"`,
`Status binding:"omitempty,oneof=…"` (`internal/domain/recipe.go:124-141`).

**Description:** `json.Unmarshal` ignores Gin/`binding` tags, so on the `multipart/form-data` branch
(the branch used whenever an image is attached) none of the field-level constraints are enforced.
The JSON branch (`:44`, `:81`) validates correctly, producing inconsistent validation depending on
content type.

**Impact:** Clients can persist recipes with out-of-range/invalid values (`servings <= 0`,
`rating` outside 0–5, arbitrary `source_type`/`status`) by submitting multipart. Mostly a data-
integrity issue; downstream code that assumes validated ranges could misbehave.

**Recommendation:** After `json.Unmarshal`, run the same validation explicitly (e.g.
`binding.Validator.ValidateStruct(&req)`), or restructure so both branches converge on one
validation routine. Apply the same fix to PDF/image multipart paths.

---

### [Low] Nested domain models in request DTO accept client-set identity fields (mass-assignment surface)

**Location:** `internal/domain/recipe.go:133-139` — `CreateRecipeRequest` embeds the **full GORM
domain structs** `Ingredients []RecipeIngredient`, `Instructions []RecipeInstruction`, and
`Nutrition *RecipeNutrition`. Those structs expose writable JSON fields including
`RecipeIngredient.ID`/`RecipeID` (`recipe.go:34-35`), `RecipeInstruction.ID`/`RecipeID`
(`:45-46`), and `RecipeNutrition.ID`/`RecipeID` (`:53-54`).

**Description:** Top-level mass assignment is handled well across the app — handlers bind to
dedicated request DTOs (`RegisterRequest`, `UpdateProfileRequest`, `CreateUserAIConfigRequest`,
etc.) rather than to domain models, and ownership fields are taken from the JWT context
(`middleware.GetUserID(c)`), never from the body. `User.PasswordHash` and `UserAIConfig.APIKey` are
correctly `json:"-"` (`user.go:11`, `ai_config.go:12`), so secrets cannot be set or read via JSON.
The gap is the **nested** recipe objects: a client can supply `ingredients[].id`,
`instructions[].recipe_id`, `nutrition.recipe_id`, etc. Whether this is exploitable (e.g. attaching
a child row to another user's recipe, or overwriting an existing row by guessing its UUID) depends
entirely on how `recipeService.Create/Update` treats those IDs — verify the service ignores or
re-derives them rather than trusting client-supplied IDs in GORM upserts/associations.

**Impact:** Potential cross-tenant row association or unintended upsert if the service persists
client-supplied nested IDs. Bounded by service-layer behavior (out of scope here, flagged for the
service phase).

**Recommendation:** Define explicit nested *request* types (e.g. `RecipeIngredientRequest`) that
omit `ID`/`RecipeID`, mirroring the pattern already used for `SubRecipeRequest`
(`recipe.go:144-147`). Ensure the service never trusts client-provided child IDs/foreign keys.

---

### [Low] CORS: `AllowCredentials: true` is hardcoded regardless of configured origins

**Location:** `internal/middleware/cors.go:9-18`; origins wired at `internal/router/router.go:20`
from `config.CORS.AllowedOrigins`; current value `env.development.yaml:40-42` →
`["http://localhost:5173"]`.

**Description:** Verified against the actual `cors.go`: `AllowOrigins` is **not** a wildcard `*`,
and it is **not reflected arbitrarily** — it is driven by the explicit `cors.allowed_origins` list
from config (currently a single localhost origin). `AllowMethods` and `AllowHeaders` are a
reasonable closed set (`GET/POST/PUT/PATCH/DELETE/OPTIONS`; `Origin/Content-Type/Authorization`).
The one rigid choice is `AllowCredentials: true`, which is hardcoded independent of the origin list.
The serious misconfiguration the brief asks about — wildcard origin **combined with** credentials —
**does not exist in the current configuration**. The residual risk is latent: nothing prevents an
operator from setting `allowed_origins: ["*"]` in YAML, at which point the hardcoded
`AllowCredentials: true` becomes a credential-exposing wildcard (gin-contrib/cors would either
reject the combination at startup or, depending on version, behave unsafely — do not rely on it).

**Impact:** None today (origin allowlist is restrictive). Latent: a future wildcard origin would,
together with the fixed `AllowCredentials: true`, expose authenticated responses cross-origin.

**Recommendation:** Keep the explicit origin allowlist (good). Add a guard that refuses to start if
`allowed_origins` contains `*` while credentials are enabled, and consider deriving the
`AllowCredentials` flag from config so the two settings can never drift into an unsafe pair.

---

### [Low] No request-body/upload size limits; PDF read sized by client-declared length

**Location:** `recipe_handler.go:206-224` (`ImportFromPDF`): `fileBytes := make([]byte, file.Size)`
then `f.Read(fileBytes)`. No `engine.MaxMultipartMemory` set (`router.go:18` uses bare
`gin.Default()`), no `MaxBytesReader`, and no per-handler size cap on JSON/multipart bodies.

**Description:** The PDF importer allocates a byte slice sized directly from the client-supplied
`file.Size`, and there is no global body-size limit. (The single `f.Read` is also a correctness bug
— it may short-read large files — noted in the threat model.) No content-type/magic-byte check
confirms the upload is actually a PDF/image.

**Impact:** Memory-exhaustion / DoS via a large declared `file.Size` or large JSON bodies; an
attacker can drive allocation and bandwidth with a single authenticated request.

**Recommendation:** Set `engine.MaxMultipartMemory` and wrap `c.Request.Body` with
`http.MaxBytesReader`; enforce explicit max sizes for image and PDF uploads; validate content type;
stream/`io.ReadFull` with a hard cap instead of `make([]byte, file.Size)` + `f.Read`.

---

### [Low] Provider response logged to stdout via `fmt.Print`

**Location:** `pkg/ai/claude_model.go:41` (`fmt.Print(message)`).

**Description:** The full Anthropic API response object is printed to stdout on every parse. While
not a handler, it is reachable from `RecipeHandler.ImportFromURL/ImportFromPDF/ParsePlainText…` and
bypasses the structured `zap` logger. It can place untrusted/LLM-generated content and metadata into
process logs unstructured.

**Impact:** Log hygiene / potential info leakage into stdout logs; aids prompt-injection debugging
for an attacker who can read logs. Low.

**Recommendation:** Remove the `fmt.Print` or replace with a gated `logger.Debug` that redacts
content; never log full provider payloads at info level.

---

### [Info] No security/transport headers; user files served same-origin under `/uploads`

**Location:** `internal/router/router.go:18-36` — `gin.Default()` adds no security headers; static
files served at `engine.Static("/uploads", …)` when `storage.type == local`.

**Description:** No middleware sets `X-Content-Type-Options: nosniff`, `X-Frame-Options`/CSP,
`Referrer-Policy`, or HSTS, and the app terminates plain HTTP (TLS is assumed to be handled by an
upstream proxy — not present in this repo). User-uploaded files are served from the same origin
under `/uploads` with no auth (see route inventory), so missing `nosniff`/CSP compounds the stored-
content risks flagged elsewhere.

**Impact:** Increases impact of any stored-content/XSS issue (MIME sniffing of uploaded files),
allows clickjacking, and offers no transport hardening at the app layer.

**Recommendation:** Add a small security-headers middleware (`nosniff`, `X-Frame-Options: DENY` /
frame-ancestors CSP, `Referrer-Policy`, and HSTS when behind TLS). Serve user uploads from a
separate origin or with a restrictive `Content-Security-Policy`/`Content-Disposition: attachment`.

---

### [Info] `GET /users/list` returns every user's email to any authenticated caller

**Location:** `user_handler.go:109-117` → `service/user_service.go:217-218` (`ListAll`) returning
`[]domain.User`. `PasswordHash` is correctly hidden (`json:"-"`, `user.go:11`).

**Description:** Any authenticated user receives the full user directory including emails and names.
Password hashes are not exposed (good), but the endpoint has no role/ownership gate. This is
primarily an authorization issue (covered in the route inventory) but is noted here for the PII /
info-exposure dimension of the response shape.

**Impact:** User enumeration / PII harvesting (emails) by any low-privilege authenticated account.

**Recommendation:** Restrict to admin/role-gated access, or remove; if a directory is required,
return a minimal, non-PII projection.

---

## Summary

| Severity | Count |
|----------|-------|
| Critical | 0 |
| High     | 1 |
| Medium   | 3 |
| Low      | 5 |
| Info     | 2 |
| **Total**| **11** |

**Most serious:** Weak/default JWT signing secret (`your-super-secret-key-here`) loaded from config
with no startup validation (High) — a known/guessable HS256 key allows forging tokens for any user
and complete authentication bypass.

**CORS posture:** **Not** a wildcard. `AllowOrigins` is the explicit configured list
(currently `["http://localhost:5173"]`), origins are not arbitrarily reflected, and methods/headers
are a sane closed set. `AllowCredentials: true` is hardcoded but paired with a restrictive origin
allowlist, so the dangerous `*`-plus-credentials combination is **not present today**; the only
residual risk is latent if an operator later sets `allowed_origins: ["*"]` (Low).

**Cross-cutting:** Pervasive `err.Error()` leakage to clients (Medium) and a live Anthropic key in a
plaintext on-disk dev config (Medium, rotate immediately) are the next priorities after the JWT
secret. Mass assignment is well-controlled at the top level (dedicated DTOs; `user_id` from JWT;
`PasswordHash`/`APIKey` are `json:"-"`); the only gap is client-settable IDs on nested recipe
objects (Low, verify in the service layer).
