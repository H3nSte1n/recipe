# Recipe App — Consolidated Security & Code-Quality Review

**Date:** 2026-06-27
**Scope:** Full-stack review of the Recipe application — Go/Gin backend (`services/backend/`) and React 18 + TypeScript frontend (`services/frontend/`).
**Methodology:** Six-phase review consolidated from 14 per-area findings documents (see Appendix). Phase 0 = reconnaissance (route inventory, dependency audit, threat model); Phases 1–5 = backend security (auth, SSRF, storage, AI, data-access, handlers/config), backend quality, and frontend security (auth, XSS/API-client, data-exposure) + frontend quality. All findings are grounded in real code at cited `file:line` anchors. This document is a read-only synthesis; it de-duplicates overlapping findings, reconciles cross-phase disagreements, and normalizes severities onto a single scale (Critical / High / Medium / Low / Info).

---

## 1. Executive Summary

The Recipe application is **architecturally healthy** but carries one **Critical**, internet-facing security defect and a cluster of **High**/**Medium** issues that should be addressed before any production exposure. The backend follows clean architecture with consistently parameterized queries (no SQL injection found), correct bcrypt password hashing (cost 14), strong/single-use password-reset tokens, and JWT algorithm pinning that blocks `alg:none`/RS↔HS confusion. The frontend enforces strict TypeScript, contains **no** `dangerouslySetInnerHTML`/`innerHTML`/`eval` sinks (React auto-escaping is correctly relied upon for untrusted AI/imported recipe content), keeps all secrets server-side, and uses relative-path, Bearer-header API calls that structurally avoid classic CSRF.

The **single most urgent item** is **Server-Side Request Forgery (SSRF) on the recipe URL-import endpoint** (`POST /api/v1/recipes/import/url`): the user-supplied URL is fetched with no allowlist, no private/loopback/link-local IP filtering, and no scheme restriction. Any registered user (registration is open) can make the server fetch cloud-metadata (`169.254.169.254`), loopback, RFC1918 hosts, and Docker-internal services (`app:8080`, `db:5432`) — a directly exploitable credential-theft and internal-reconnaissance primitive that fires before any AI processing. Close behind are a **weak/default JWT signing secret** shipped in version-controlled sample config with no startup validation (a complete authentication-bypass if the placeholder reaches production), **stored-XSS via the unauthenticated `/uploads` mount** (unrestricted upload type + MIME sniffing), and **redirect-based SSRF** that defeats any front-door URL check. A live-looking Anthropic API key currently sits in the on-disk dev config (`env.development.yaml:38`, git-ignored) and should be **rotated immediately** as an incident-response action. The remaining body of work is well-bounded: rate-limiting/revocation gaps on auth, plaintext secret storage at rest, AI cost/DoS and prompt-injection hardening, non-atomic transactions (`RunTx`), a logout button that never clears the JWT, and a long tail of quality/accessibility/dependency-hygiene items.

---

## 2. Severity Summary Table

De-duplicated totals across all areas. (Info tier counts substantive informational findings and reconciliations; positive controls are consolidated separately in Section 5.)

| Severity | Count | Examples |
|----------|-------|----------|
| **Critical** | **1** | SSRF — no egress filtering on URL import |
| **High** | **3** | Weak/default JWT secret; stored-XSS via `/uploads`; redirect-based SSRF |
| **Medium** | **27** | Plaintext key storage at rest; `AddRecipeToList` IDOR; `err.Error()` leakage; non-atomic `RunTx`; logout doesn't clear JWT; no auth rate-limiting; AI cost/DoS & prompt injection; upload DoS; vulnerable build deps |
| **Low** | **27** | Missing `iss`/`aud` claims; CORS `AllowCredentials` hardcoded; `/users/list` email exposure; dead `axios` dependency; image `referrerPolicy`; numerous quality/a11y items |
| **Info** | **8** | Shopping-list-item IDOR **refuted**; SSRF blind-oracle bound; missing security headers; S3 stub; by-design public recipes |
| **Total** | **66** | |

---

## 3. Findings by Severity

### Critical

#### [CRITICAL] SSRF — user-supplied import URL fetched against any host (cloud metadata / internal services)
- **Area:** Backend / SSRF (egress boundary B3) · **Source doc:** be-ssrf.md
- **Location:** `pkg/urlparser/fetcher.go:27-35` (request build + `client.Do`); `pkg/urlparser/service.go:35-37` (raw URL passed in); `internal/handler/recipe_handler.go:190-196` (binding/dispatch); `internal/domain/recipe.go:150` (validator).
- **Description / Impact:** The only validation on the import URL is Gin's `binding:"required,url"`, a purely *syntactic* check. There is **no allowlist, no destination-host validation, and no private/loopback/link-local IP filtering** anywhere in the fetch path; the URL flows unchanged into `http.NewRequestWithContext`/`client.Do`. Any authenticated user (open registration ⇒ low auth bar) can force the server to issue arbitrary outbound GETs to AWS/GCP metadata (`169.254.169.254`, `metadata.google.internal` → IAM credentials), loopback (`127.0.0.1`, `[::1]`), RFC1918 ranges, and Docker-internal service names (`app:8080`, `db:5432`). This is the highest-ranked risk in the threat model, is directly exploitable with no LLM cooperation (the request fires before any parsing), and yields credential theft plus internal host/port enumeration. Raw-body exfiltration is *partially* constrained because fetch errors are masked and success content is filtered through the LLM (see Info finding), but the request-forgery and enumeration primitives are fully available.
- **Remediation:** Validate the destination before and during the fetch: (1) restrict scheme to `http`/`https`; (2) resolve the hostname and reject any request whose **resolved IP** falls in loopback / private (RFC1918) / link-local (`169.254.0.0/16`) / unique-local (`fc00::/7`) / `::1` / multicast / unspecified ranges — reject by resolved IP, not hostname string, to also block `app`/`db`; (3) pin the validated IP for the actual connection via a custom `Transport.DialContext` (re-validate in `DialContext`) to close the TOCTOU/DNS-rebinding gap; (4) prefer an explicit egress allowlist if the product can tolerate it. Combine with the redirect-validation fix (High) below.

### High

#### [HIGH] Weak/default JWT signing secret with no startup validation
- **Area:** Backend / Auth & Config · **Source docs:** be-auth.md, be-handlers-config.md *(de-duplicated — reported once)*
- **Location:** `pkg/config/config.go:45-49` (`JWTConfig.Secret`, no constraints); `internal/router/router.go:25`; `internal/service/services.go:38`; `internal/middleware/auth.go:60-66` (`[]byte(m.secretKey)`); `env.development.yaml.sample:18` / `env.development.yaml:15` (placeholder secret literal).
- **Description / Impact:** The HS256 signing key is read verbatim from config and passed straight into both the issuer and the verifier with **no check** that it is non-empty, of minimum length, or changed from the shipped placeholder. The committed sample hard-codes a guessable placeholder literal, and the documented setup path (copy the sample) inherits it. The project is published publicly, so the placeholder is public. JWT identity is carried entirely by the `user_id` claim — anyone who knows/guesses the secret can forge a token for any `user_id` and **fully impersonate any user (complete authentication bypass)**. `viper.AutomaticEnv()` *may* allow an env override, but nothing fails closed if it is unset, and the override may not even bind through `Unmarshal` without explicit `BindEnv`. Escalates to effectively Critical if the default reaches production.
- **Remediation:** Fail closed at startup: refuse to boot if `jwt.secret` is empty, equals any known sample placeholder, or is shorter than ~32 bytes. Generate per-environment secrets from a CSPRNG, inject only via a secret store / env var, and verify the override actually binds (`viper.BindEnv("jwt.secret")` or read `os.Getenv` directly). Never ship a usable default.

#### [HIGH] Stored XSS via unrestricted upload file type + MIME sniffing served from public `/uploads`
- **Area:** Backend / Storage · **Source doc:** be-storage.md
- **Location:** `pkg/storage/local_storage.go:29-52` (no type validation; `filename := uuid + filepath.Ext(file.Filename)` preserves attacker extension); `internal/handler/recipe_handler.go:40-42,77-79` (no validation); `internal/router/router.go:34-36` (public static serve, outside the JWT group).
- **Description / Impact:** `UploadFile` accepts any uploaded file and writes it with the **attacker-controlled extension preserved** — no Content-Type check, no magic-byte validation, no extension allowlist. The file is then served by Gin's `engine.Static("/uploads", ...)` (`http.FileServer`), which sets `Content-Type` from the extension (`.html` → `text/html`, `.svg` → `image/svg+xml`) and falls back to `http.DetectContentType` sniffing for unknown/absent extensions. No `X-Content-Type-Options: nosniff` and no `Content-Disposition: attachment` are set. An attacker can upload an `.html`/`.svg` "image" served as active content. Honest exploit path: an `<img src>` will **not** execute it — the victim must navigate directly to the (unguessable UUID) URL — so this is not automatic token theft in the sample config where `/uploads` (`:8080`) is a different origin from the SPA (`:5173`). It **escalates to full account takeover (Critical) if `/uploads` shares an origin with the SPA** (the typical single-origin reverse-proxy production deployment), where injected script can read the `localStorage` JWT. Independent of origin, script runs on a trusted app domain (phishing/CSRF pivot).
- **Remediation:** Validate uploads against an allowlist of image content types by sniffing the actual bytes (`http.DetectContentType` / decode the image), not the client extension; reject non-raster-image files; normalize the stored extension from the detected type. On serving, always send `X-Content-Type-Options: nosniff` and `Content-Disposition: attachment` (or serve uploads from a separate, cookie-less, sandboxed origin/CDN). Consider re-encoding images server-side to strip embedded scripts/metadata.

#### [HIGH] Redirects followed without per-hop destination validation (redirect SSRF / DNS rebinding)
- **Area:** Backend / SSRF · **Source doc:** be-ssrf.md
- **Location:** `pkg/urlparser/helpers.go:10-15` (`defaultRedirectPolicy`); `pkg/urlparser/service.go:24` (wired as `CheckRedirect`).
- **Description / Impact:** `defaultRedirectPolicy` caps the redirect *count* at 10 but performs **no validation of the redirect target**. Even if a host/IP allowlist were added to the *initial* URL, an attacker-controlled public server can `302`-redirect the fetcher to `http://169.254.169.254/...` or `http://app:8080/...`, bypassing the front-door check. This is the classic redirect-SSRF / DNS-rebinding bypass and makes any single-point URL validation insufficient on its own.
- **Remediation:** In `CheckRedirect`, apply the *same* IP/host validation from the Critical finding to every hop's `req.URL`, rejecting redirects to disallowed addresses, and combine with per-connection IP pinning so the dialed address is always the validated one. (Fix this together with the Critical SSRF finding — they share one validation routine.)

### Medium

> Concise treatment; each item is fully detailed in its source doc.

#### [MEDIUM] Plaintext storage of API keys / secrets at rest (DB column + on-disk config)
- **Area:** Backend / AI & Config · **Source docs:** be-ai.md, be-handlers-config.md *(merged — same class)*
- **Location:** `internal/domain/ai_config.go:12`; `migrations/000006_create_user_ai_configs.up.sql:5` (`api_key VARCHAR(255) NOT NULL`); read at `internal/service/recipe_service.go:435`; server keys `pkg/config/config.go:67-68`; on-disk live key `env.development.yaml:38`.
- **Description / Impact:** User-provided Anthropic/OpenAI API keys are stored in `user_ai_configs.api_key` as **plaintext** — no encryption-at-rest, hashing, or KMS indirection. `json:"-"` correctly hides keys from HTTP responses but does nothing for storage. Server-wide keys are likewise read from plaintext YAML, and `env.development.yaml:38` currently holds a **live-looking Anthropic key in cleartext on disk** (git-ignored, so not committed, but one `.gitignore` change from exposure and readable to anyone with host/backup access). Any DB read-access compromise (injection elsewhere, leaked backup, over-broad `SELECT`) yields every user's billable provider credential.
- **Remediation:** **Rotate/revoke the on-disk Anthropic key immediately and treat it as compromised.** Encrypt the key column at the application layer (AES-GCM with a key from a secret manager/KMS, envelope encryption) or persist only a reference into a secrets backend. Move all server secrets (db/smtp/jwt/AI) out of YAML into a secret manager or injected env; add secret-scanning pre-commit hooks; ensure backups are encrypted.

#### [MEDIUM] No real transactional atomicity — `RunTx` wraps nothing; correct `WithTypedTransaction` is dead code
- **Area:** Backend / Quality · **Source doc:** be-quality.md *(be-quality rated this High; normalized to Medium — it is a data-integrity/correctness defect with no direct security exploit, and the task's expected mapping places it in the Medium tier. Retained at the top of the roadmap due to its foundational nature and cheap fix.)*
- **Location:** `internal/repository/user_repository.go:43-47` (and identical `RunTx` in `recipe_repository.go:39-43`, `ai_config_repository.go:42-44`); callers in `user_service.go:70,176,212`, `recipe_service.go:135,229,302`, `ai_config_service.go:46,81`; correct-but-unused `WithTypedTransaction` (`user_repository.go:36-41`, etc.).
- **Description / Impact:** `RunTx` opens a GORM transaction but **ignores the `tx` handle** and calls `fn()`, whose inner repo calls run on the captured root repo (a separate pooled connection) and **auto-commit independently**. The outer transaction wraps nothing. Result: `Register` can commit a user row without a profile; recipe `Create` can orphan a recipe row (and then delete the uploaded image, leaving a persisted recipe pointing at a deleted file); ai-config `Create` can leave a user with no default. The idiomatic fix already exists but is uncalled.
- **Remediation:** Replace `RunTx(ctx, func() error {…})` with `WithTypedTransaction`, threading the tx-bound repo into the closure so inner ops share one `tx`. Inner methods that open their own `RunInTransaction` must accept/use the caller's `tx`. Wrap `shoppingListService.Create`'s list+items writes too. Once migrated, delete `RunTx`.

#### [MEDIUM] IDOR — `AddRecipeToList` discloses other users' private-recipe ingredients
- **Area:** Backend / Data-access · **Source doc:** be-dataaccess.md
- **Location:** `internal/service/shopping_list_service.go:263-272` (recipe fetch at `:269`); endpoint `POST /shopping-lists/:id/add-recipe`; `internal/domain/shopping_list.go:86-88`.
- **Description / Impact:** The method verifies the caller owns the **target list** but fetches the **source recipe** straight from the client-supplied `recipe_id` with **no ownership and no `is_private` check** — unlike `recipeService.Create/Update`, which reject another user's private sub-recipe (`recipe_service.go:97-99`). Any authenticated user who knows another user's private-recipe UUID can add it to their own list and read back ingredient names/amounts/units (also forwarded to the AI categorizer — secondary egress). Rated Medium because exploitation requires knowing a UUIDv4 (not sequentially guessable).
- **Remediation:** After fetching, enforce `if recipe.IsPrivate && recipe.UserID != userID { return ErrUnauthorized }`, mirroring the existing sub-recipe guard.

#### [MEDIUM] Handlers leak raw internal error strings to clients (`err.Error()`)
- **Area:** Backend / Handlers · **Source doc:** be-handlers-config.md
- **Location:** widespread — `user_handler.go:24,30,46,61,78`; `ai_config_handler.go:31,41,75,87,97,109`; `profile_handler.go:36`; every `ShouldBindJSON` failure path (`recipe_handler.go:45,82,…`; `shopping_list_handler.go:33,112,…`).
- **Description / Impact:** Raw Go error strings are forwarded to clients, surfacing GORM/Postgres messages (constraint/column names, SQL fragments), filesystem paths, SMTP errors, or AI-provider text. `Login` (`:46`) and `ForgotPassword` (`:61`) are most sensitive: the 500 branch of `ForgotPassword` leaks `err.Error()`, undermining its deliberately generic anti-enumeration response. The codebase is inconsistent — several handlers already return generic strings and log detail via `zap`.
- **Remediation:** Return generic, stable client messages and log the real error server-side with `zap`. Map validator errors to curated field messages. Audit `ForgotPassword` so neither branch leaks state.

#### [MEDIUM] Logout button never clears the JWT — token persists in `localStorage` and stays usable
- **Area:** Frontend / Auth · **Source doc:** fe-auth.md
- **Location:** `App.tsx:12` (`onLogout={() => setScreen('landing')}`); `HomePage.tsx:16,65`; `HomeHeader.tsx:52-57`; working-but-bypassed `useAuth.ts:20-23` / `authService.ts:51-53`.
- **Description / Impact:** A correct logout exists (`authService.logout()` removes the token) but the header sign-out button is wired to a handler that only flips screen state — it never touches `localStorage`. After "Sign out", the JWT remains; a refresh re-runs `checkAuth()` and silently signs the user back in, and the token is still a valid Bearer credential. Compounds with the non-revocable 24h backend token: a user cannot actually end their session from the UI for up to 24h (tips toward High on shared devices).
- **Remediation:** Wire the button through `useAuth`/`authService.logout()` so the token is removed before switching screens — mirror the 401 path in `apiClient.ts`.

#### [MEDIUM] JWT stored in `localStorage` — exfiltratable by any future XSS
- **Area:** Frontend / Auth & XSS · **Source docs:** fe-auth.md, fe-xss-apiclient.md *(de-duplicated)*
- **Location:** `services/authService.ts:3,20,42,56,84`.
- **Description / Impact:** The session JWT is persisted in `localStorage`, readable by any JS on the origin. With **no XSS sink observed today** (no `dangerouslySetInnerHTML`/`innerHTML`/`eval`) this is latent rather than active — keeping it Medium — but any future injected/compromised script could exfiltrate a full Bearer credential, and combined with the non-revocable 24h backend token grants full account access with no revocation path. An accepted-but-documented tradeoff of the Bearer-header design.
- **Remediation:** Prefer an `HttpOnly`/`Secure`/`SameSite` cookie (removes JS read access; then add CSRF protection). If keeping `localStorage`: hold the XSS surface at zero, add a strict CSP, and shorten token lifetime / add server-side revocation.

#### [MEDIUM] No rate limiting or lockout on login / forgot-password / reset-password
- **Area:** Backend / Auth · **Source doc:** be-auth.md
- **Location:** `internal/router/router.go:50-58` (public auth routes; `internal/middleware/` has only `auth.go`/`context.go`/`cors.go`; no limiter dependency in `go.mod`).
- **Description / Impact:** No throttling, lockout, or CAPTCHA on the public auth endpoints. bcrypt cost 14 raises per-attempt cost but does not bound volume. Enables online password brute-force / credential-stuffing on `login`, un-throttled reset-token attempts, and password-reset email flooding / SMTP abuse on `forgot-password`.
- **Remediation:** Add IP- and account-scoped rate limiting with exponential backoff / temporary lockout; throttle `forgot-password` per target email and source IP.

#### [MEDIUM] Stateless JWTs are not revocable; password reset and account deletion don't invalidate outstanding tokens
- **Area:** Backend / Auth · **Source doc:** be-auth.md
- **Location:** `internal/service/user_service.go:156-182` (ResetPassword), `204-215` (Delete), `184-193` (generateToken); `jwt.duration: 24h`.
- **Description / Impact:** JWTs are self-contained with a 24h `exp` and no server-side session/denylist. A reset updates the hash but leaves existing tokens valid; deletion removes the user row but issued tokens stay cryptographically valid until `exp`. A stolen/old token survives a password change for up to 24h.
- **Remediation:** Shorten access-token lifetime (e.g. 15 min) with a refresh flow, or add a revocation signal (per-user `tokenVersion` bumped on reset/delete, or a `jti` denylist). At minimum, invalidate sessions on password reset.

#### [MEDIUM] User enumeration via register response and via login/forgot-password timing & status
- **Area:** Backend / Auth · **Source doc:** be-auth.md
- **Location:** `user_service.go:60-67` + `user_handler.go:28-31` (register `"email already registered"`); `user_service.go:107-115` (login short-circuits before bcrypt); `user_service.go:128-153` + `user_handler.go:60-67` (forgot-password timing + 500 status leak).
- **Description / Impact:** Three oracles reveal whether an email is registered: register's explicit message, login's timing delta (bcrypt only runs for known emails), and forgot-password's DB-insert + synchronous SMTP send (plus a 500 only on the existing-email branch).
- **Remediation:** Make register's response generic; for login, always run a bcrypt comparison against a dummy hash when the user is absent; send forgot-password mail asynchronously and return 200 regardless of SMTP outcome.

#### [MEDIUM] Unbounded untrusted input sent to a paid LLM API (cost / DoS)
- **Area:** Backend / AI · **Source doc:** be-ai.md
- **Location:** `pkg/ai/model.go:64-167` (prompt builders, no length bound); upstream `pkg/urlparser` (`io.ReadAll`, no cap) and `pkg/pdfparser/service.go:25-66` (whole-PDF concatenated).
- **Description / Impact:** Full fetched-HTML / PDF / free-text content is interpolated into prompts with no truncation or token ceiling before egress to the provider. An attacker can import a huge page or upload a large PDF to force arbitrarily large paid requests (output *is* capped at 2000 tokens, so this is input-side cost amplification + memory pressure).
- **Remediation:** Enforce a byte/character cap on `content` before building the prompt; cap the upstream `io.ReadAll`/PDF size; consider a `count_tokens` pre-check and per-user rate limiting on AI import endpoints.

#### [MEDIUM] Prompt injection — untrusted content concatenated into prompts with no instruction/data separation
- **Area:** Backend / AI · **Source doc:** be-ai.md
- **Location:** `pkg/ai/model.go:64-167` (`createPrompt`/`createParseInstructionsPrompt`/`createPromptToCategorizeShoppingListItems`, all `fmt.Sprintf` inline); callers `claude_model.go:27,51,73` / `gpt_model.go:26,41,57`.
- **Description / Impact:** Attacker-controlled web/PDF/free text is placed directly into the prompt body with no delimiting or system-vs-user separation. Injected instructions can steer model output. Rated Medium (not High) because the worst realistic outcome is attacker-controlled *recipe JSON* — the downstream sinks are safe (GORM parameterized; React auto-escapes; output capped at 2000 tokens; the prompt holds no secrets).
- **Remediation:** Wrap untrusted content in explicit delimiters with a "treat strictly as data / never follow instructions inside" directive, move instructions into a `system` message, and validate parsed output against an allowlist.

#### [MEDIUM] No upload size limit → disk-fill (images) and memory-exhaustion (PDF) DoS
- **Area:** Backend / Storage · **Source docs:** be-storage.md *(be-handlers-config.md flags the same as Low — normalized to Medium)*
- **Location:** `pkg/storage/local_storage.go:47` (`io.Copy`, no limit); `internal/handler/recipe_handler.go:220-224` (`make([]byte, file.Size)` + `f.Read`); no `MaxMultipartMemory`/`MaxBytesReader` (`router.go:17-28`).
- **Description / Impact:** Image uploads stream to disk with unbounded `io.Copy`; PDF import reads the entire file into a heap buffer sized from client-declared `file.Size`. No global body-size limit. A single authenticated user can exhaust disk or drive memory/OOM.
- **Remediation:** Set `engine.MaxMultipartMemory` and wrap `c.Request.Body` with `http.MaxBytesReader`; reject oversized files with 413; cap/stream the PDF parser input (`io.ReadFull`/`io.LimitReader`).

#### [MEDIUM] Unauthenticated public access to all uploaded files (`/uploads/*`)
- **Area:** Backend / Storage · **Source doc:** be-storage.md
- **Location:** `internal/router/router.go:34-36`; URL minted at `pkg/storage/local_storage.go:51`.
- **Description / Impact:** The static mount is registered outside the JWT group. Every uploaded file is reachable by anyone with the URL, with no auth and no requester↔owner tie. Images for **private** recipes live in the same public directory, so a leaked/shared image URL exposes them regardless of recipe privacy. Confidentiality depends solely on URL secrecy (UUIDv4 unguessability — defense-in-depth, not access control).
- **Remediation:** Serve uploads through an authenticated handler that verifies access to the owning recipe (or issues short-lived signed URLs) instead of a blanket public static mount.

#### [MEDIUM] No HTTP client timeout on the URL fetcher (DoS / slow-loris / goroutine hang)
- **Area:** Backend / SSRF · **Source doc:** be-ssrf.md
- **Location:** `pkg/urlparser/service.go:22-25` (no `Timeout`); `pkg/urlparser/fetcher.go:35` (`client.Do`).
- **Description / Impact:** The shared `http.Client` sets no `Timeout` and uses the default transport (no response-header/dial timeouts). A slow/malicious target — or an internal host reached via SSRF that accepts but never responds (e.g. `db:5432`) — can hold a request open indefinitely, tying up goroutines/connections.
- **Remediation:** Set an explicit `http.Client.Timeout` (10–15s) and transport dial/response-header timeouts, and/or wrap the fetch in `context.WithTimeout`.

#### [MEDIUM] Unbounded response-body read on URL fetch (memory exhaustion)
- **Area:** Backend / SSRF · **Source doc:** be-ssrf.md
- **Location:** `pkg/urlparser/fetcher.go:50` (`io.ReadAll(resp.Body)`).
- **Description / Impact:** The full response body is read into memory with no size cap; a malicious or internal endpoint can return an arbitrarily large/endless body buffered entirely before goquery parsing.
- **Remediation:** Wrap the body with `io.LimitReader(resp.Body, maxBytes)` before `io.ReadAll`; reject/truncate oversized responses; consider checking `Content-Type`.

#### [MEDIUM] Gin runs in debug mode outside production; default env is `development`
- **Area:** Backend / Config · **Source doc:** be-handlers-config.md
- **Location:** `cmd/api/main.go:19-39` (release mode only when `Env == "production"`); `internal/router/router.go:18` (`gin.Default()`).
- **Description / Impact:** Gin mode is tied to `App.Env`, which defaults to `development`. A blank/typo'd `APP_ENV` silently runs a production deployment in debug mode (verbose `[GIN-debug]` route dumps/warnings). Low direct exploit but fragile posture-by-single-string.
- **Remediation:** Default to release mode with explicit debug opt-in; fail closed on unrecognized `APP_ENV`; confirm the Recovery handler never returns stack traces.

#### [MEDIUM] Recipe repository breaks the `IsNotFound` contract
- **Area:** Backend / Quality · **Source doc:** be-quality.md
- **Location:** `internal/repository/recipe_repository.go:149-154` (`errors.New("recipe not found")` using stdlib `errors`); contract in `internal/errors/errors.go:45-54`.
- **Description / Impact:** Unlike every other repo, `RecipeRepository.GetByID` swallows `gorm.ErrRecordNotFound` and returns a bare stdlib error, so `apperrors.IsNotFound(err)` returns `false` for a missing recipe. The `IsNotFound` branch never fires in `recipeService.GetByID/Update/Delete`, and the bug propagates to `shoppingListService.AddRecipeToList`.
- **Remediation:** Return `gorm.ErrRecordNotFound` unchanged or `apperrors.ErrNotFound.Wrap("recipe not found")`.

#### [MEDIUM] Ad-hoc HTTP error translation ignores `AppError.Code`
- **Area:** Backend / Quality · **Source doc:** be-quality.md
- **Location:** `internal/handler/recipe_handler.go` (all methods map any error to 500, e.g. `:56-60`, `:137-141`); `shopping_list_handler.go:76-80` (maps any `Get` error to 404); codes in `internal/errors/errors.go:10-43`.
- **Description / Impact:** Services build typed errors, but no handler inspects `*AppError`/`Code` (no `errors.As`/`IsNotFound` anywhere in `internal/handler`). Each endpoint hardcodes one status, so unauthorized/not-found/infra errors are mislabeled (404s as 500s and vice-versa).
- **Remediation:** Add a shared `respondError(c, err)` helper that maps `AppError.Code` → HTTP status via `errors.As`, and call it from every handler.

#### [MEDIUM] Three overlapping transaction abstractions; the correct one is dead code
- **Area:** Backend / Quality · **Source doc:** be-quality.md
- **Location:** `base_repository.go:16` (`RunInTransaction`); per-repo `RunTx` and `WithTypedTransaction` (`recipe_repository.go:18-19,32-43`, etc.); `shopping_list_repository.go` defines none despite a two-step `Create`+`AddItems`.
- **Description / Impact:** Each repo exposes three transaction mechanisms; services use only the broken `RunTx`, while the idiomatic `WithTypedTransaction` is implemented on three repos and called by none. The ergonomic-looking abstraction is the unsafe one — a maintenance trap (root cause shared with the non-atomic `RunTx` Medium above).
- **Remediation:** Settle on `WithTypedTransaction`, migrate callers, remove `RunTx`, and wrap `shoppingListService.Create`'s writes.

#### [MEDIUM] `ImportFromPDF` reads the upload with a single `f.Read` (short-read) + missing `userID` guard
- **Area:** Backend / Quality · **Source doc:** be-quality.md
- **Location:** `internal/handler/recipe_handler.go:206-241` (read at `:220-224`; missing guard at `:232`).
- **Description / Impact:** `make([]byte, file.Size)` + a single `f.Read` can short-read large PDFs (silent truncation → intermittent parse corruption), and unlike other handlers this method omits the `userID == ""` guard.
- **Remediation:** Use `io.ReadAll`/`io.ReadFull` (with a size cap per the storage Medium) and add the `userID == ""` guard.

#### [MEDIUM] `RecipeGraph` recomputes the entire layout/edge set on every render (incl. each drag/pan frame)
- **Area:** Frontend / Quality (performance) · **Source doc:** fe-quality.md
- **Location:** `components/RecipeGraph.tsx:94-107` (unmemoized in the render body); pan/zoom `setTransform` at `:141-148`, `:109-126`.
- **Description / Impact:** `computeLayout` (recursive longest-path, O(n²) worst case), `nodeMap`, `edges`, and canvas dimensions are computed unconditionally every render; every `onMouseMove`/`handleWheel` re-runs the full layout though it depends only on `recipes`. Visible jank on larger recipe sets.
- **Remediation:** `useMemo(() => …, [recipes])` for the layout-dependent values; only `transform` should change per mouse move.

#### [MEDIUM] `AddRecipeModal` is a ~690-line component mixing parsing, fetching, orchestration, and presentation
- **Area:** Frontend / Quality · **Source doc:** fe-quality.md
- **Location:** `components/AddRecipeModal.tsx` (`:253-689`; `handleSave` `:450-514`).
- **Description / Impact:** One file holds free-text parsers, two subcomponents, ~20 `useState` hooks, all CRUD API calls, and the full JSX; `handleSave` both creates child recipes in a loop and builds/sends the parent payload. Hard to test/read, high regression surface — counter to "readability over cleverness".
- **Remediation:** Extract pure parse/format helpers to `utils/`, lift sub-recipe state into a hook, move CRUD orchestration into `recipeService`, split out the subcomponents.

#### [MEDIUM] Duplicated "open recipe by id, then set serves" async logic in `HomePage`
- **Area:** Frontend / Quality · **Source doc:** fe-quality.md
- **Location:** `pages/HomePage.tsx:46-55, 92-103, 119-127, 128-136` (and partially `:157-165`).
- **Description / Impact:** The same fetch-then-set-state pattern is copy-pasted ≥4 times with inconsistent `catch` behavior (fallback vs. silent ignore) — the classic copy-paste drift.
- **Remediation:** Extract one `openRecipe(recipe)` helper/hook used by every handler.

#### [MEDIUM] Dead, duplicated, drifted auth pages — `LoginPage.tsx` / `RegisterPage.tsx` are never imported
- **Area:** Frontend / Quality · **Source doc:** fe-quality.md
- **Location:** `pages/LoginPage.tsx`, `pages/RegisterPage.tsx` (+ their CSS); live forms in `pages/LandingPage.tsx:36-111`.
- **Description / Impact:** Two full page components + stylesheets are unreachable; the live forms live in `LandingPage`. The implementations have drifted — `RegisterPage`/`LandingPage.RegisterView` call the `register` service directly rather than through `useAuth`, so registration never updates `isAuthenticated`. Dead CSS ships in the bundle.
- **Remediation:** Delete the unused pages (+CSS) or consolidate into one reusable `AuthForm`; standardize on `useAuth` for both login and register.

#### [MEDIUM] Clickable text elements (`<p>`/`<span>`) are not keyboard-accessible
- **Area:** Frontend / Quality (a11y) · **Source doc:** fe-quality.md
- **Location:** `pages/LandingPage.tsx:69, 108` (`<p … onClick>`); `pages/RegisterPage.tsx:73` (dead file).
- **Description / Impact:** Login/register switch affordances use bare `onClick` on non-interactive elements with no `role`/`tabIndex`/key handler — not focusable or screen-reader-announced. (`RecipeCard`/`AddRecipeModal` do this correctly, showing the contrast.)
- **Remediation:** Use `<button type="button">` styled as a link, or add `role`/`tabIndex`/`onKeyDown`; drop inline `style` for a BEM class.

#### [MEDIUM] Developer/debug tooling shipped on the public landing page
- **Area:** Frontend / Quality · **Source doc:** fe-quality.md
- **Location:** `components/ThemeExplorer.tsx` (header literally `// TEMPORARY`), `components/TunnelControls.tsx`; rendered at `pages/LandingPage.tsx:147-149`.
- **Description / Impact:** A self-described temporary color-picker (mutates `document.documentElement` CSS vars) and a 138-line animation slider panel render on the unauthenticated landing page for every visitor. Debug UI in production (polish/quality, not security).
- **Remediation:** Gate both behind `import.meta.env.DEV` or remove `ThemeExplorer` per its own note.

#### [MEDIUM] Vulnerable build/dev npm dependencies (vite / rollup / esbuild chain)
- **Area:** Dependencies (frontend toolchain) · **Source doc:** 00-dependency-audit.md
- **Location:** `services/frontend/package.json` (`vite ^5.1.0`, installed `5.4.21`); transitive `esbuild`, `rollup`, `form-data`, `follow-redirects`, etc.
- **Description / Impact:** `npm audit` reports 7 high / 6 moderate advisories. Mixed exposure: **esbuild** is dev-server-only (a website can read dev-server responses); **rollup** is the *production build bundler* with an Arbitrary File Write via Path Traversal advisory (a build-environment risk). The audit's only offered fix is a semver-major bump to vite 8. These are build/dev-time, not runtime-exposed (the app ships no axios — see Low), so normalized to a single Medium rather than per-advisory High.
- **Remediation:** Plan a deliberate vite 8 upgrade (test `vite build` + `@vitejs/plugin-react` compatibility); verify resolved transitive versions of `form-data`/`follow-redirects` after. Run `npm audit` in CI.

### Low

> Grouped by area; one line each. All are detailed in their source docs.

**Backend**
1. **[LOW] JWT lacks `iss`/`aud`/`nbf`/`iat` claims** — *be-auth.md* — `user_service.go:185-189`; no binding of the token to this service (defense-in-depth). Add and validate `iss`/`aud`.
2. **[LOW] Middleware tolerates missing/empty identity claims** — *be-auth.md* — `middleware/auth.go:44-54`; a validly-signed token lacking `user_id` proceeds with `userID == ""`. Reject 401 if absent.
3. **[LOW] URL scheme not explicitly restricted** — *be-ssrf.md* — `recipe.go:150`, `fetcher.go:28`; non-HTTP schemes only incidentally blocked by `http.Transport`. Require `http`/`https` explicitly (fold into the SSRF fix).
4. **[LOW] Path traversal in stored/deleted filenames — investigated, mitigated** — *be-storage.md* — `local_storage.go:30-33,54-56`; UUID rename + `filepath.Base`/`Ext` + non-bindable `ImageURL` neutralize it. Keep discarding the client base name.
5. **[LOW] Lenient/unvalidated LLM output persisted** — *be-ai.md* — `parser.go:10-18,35-53`; first-`{`/last-`}` extraction, `domain.Category(cat)` cast with no whitelist. Validate/bound parsed fields; check the category enum.
6. **[LOW] Full LLM response logged to stdout via `fmt.Print`** — *be-ai.md / be-handlers-config.md / be-quality.md (deduped)* — `pkg/ai/claude_model.go:41`. Remove or gate behind `logger.Debug`.
7. **[LOW] Retired/outdated AI model IDs** — *be-ai.md* — `pkg/ai/model.go:14-24`; configured Claude/OpenAI IDs are retired (default 404s) — availability/correctness. Migrate to current IDs.
8. **[LOW] No defense-in-depth tenant scoping at the repo layer** — *be-dataaccess.md* — repos fetch by bare ID; ownership is service-enforced only. No exploit today; add `Where("id = ? AND user_id = ?")` to mutations as a fail-safe.
9. **[LOW] Multipart branch bypasses request-body validators** — *be-handlers-config.md (be-storage Info dup)* — `recipe_handler.go:32-42,70-79`; `json.Unmarshal` skips Gin `binding:` validators on uploads (`servings`/`rating`/`source_type` unchecked). Run struct validation in the multipart branch.
10. **[LOW] Nested recipe DTOs accept client-set identity fields (mass-assignment surface)** — *be-handlers-config.md* — `recipe.go:133-139`; `ingredients[].id`/`recipe_id` etc. are client-settable. Define explicit nested request types; never trust client child IDs.
11. **[LOW] CORS `AllowCredentials: true` hardcoded regardless of origins (latent)** — *be-handlers-config.md* — `cors.go:9-18`; origin allowlist is restrictive today, so no current issue; guard against a future `["*"]` origin paired with credentials.
12. **[LOW] `== gorm.ErrRecordNotFound` instead of `errors.Is`** — *be-quality.md* — `recipe_repository.go:151,239`; fragile under error wrapping.
13. **[LOW] Duplicated recipe-build / sort-field logic** — *be-quality.md* — `recipe_service.go:115-133` vs `:230-249`; `shopping_list_service.go:151` vs `:350-363`. Extract a `buildRecipe` helper; single-source the sort set.
14. **[LOW] `Login` masks all `GetByEmail` errors as "invalid credentials"** — *be-quality.md* — `user_service.go:107-115`; a DB outage looks like a routine bad login. Distinguish not-found from infra errors (log + wrap).
15. **[LOW] `GET /users/list` returns every user's email to any authenticated caller** — *be-handlers-config.md / 00-route-inventory.md (elevated from Info — authenticated PII enumeration / missing authorization)* — `user_handler.go:109-117`. Restrict to admin/role-gated, or return a minimal non-PII projection.

**Frontend**
16. **[LOW] Auth gating is client-side only and based on an unverified `exp`** — *fe-auth.md* — `App.tsx:9`, `authService.ts:59-77`; a forged/garbage token briefly renders protected chrome (backend independently enforces signatures, so no data leaks). Gate initial render behind a lightweight authenticated check.
17. **[LOW] 401 session-expiry handling lives only in `apiFetch`** — *fe-auth.md* — `apiClient.ts:3-11`; a future raw-`fetch` authenticated service would bypass the 401 redirect. Make `apiFetch` the single mandatory entry point.
18. **[LOW] Remote `image_url` rendered into `<img src>` without scheme validation / `referrerPolicy`** — *fe-xss-apiclient.md / fe-data-exposure.md (deduped)* — `RecipeCard.tsx:28`, `RecipeModal.tsx:207`, `RecipeGraph.tsx:200`; not script-executing, but an attacker-controlled URL on a shared recipe is a viewer IP/referrer beacon. Add `referrerPolicy="no-referrer"` + scheme allowlist.
19. **[LOW] Bearer token attached per-call, not via a central interceptor** — *fe-xss-apiclient.md* — `authService.ts:79-85` spread into each `recipeService` call; safe only because all URLs are relative. Centralize attachment and restrict to same-origin.
20. **[LOW] `axios` is a declared dependency but never imported — remove it** — *fe-xss-apiclient.md (reconciles 00-dependency-audit.md)* — `package.json:14`; see Reconciliation note below. Removing it clears every axios/`form-data`/`follow-redirects` advisory (non-breaking).
21. **[LOW] Landing page preloads hardcoded third-party Unsplash images** — *fe-data-exposure.md* — `ScatteredBackground.tsx:5-17,135-136`; third-party requests + referrer/IP exposure before login. Self-host or add `referrerPolicy="no-referrer"`.
22. **[LOW] API call swallows its error silently** — *fe-quality.md* — `AddRecipeModal.tsx:292` (`.catch(() => {})`). Keep a justifying comment or set a non-blocking state.
23. **[LOW] Inconsistent ID generation and index-based list keys** — *fe-quality.md* — `AddRecipeModal.tsx:115` (`Date.now()+i`) vs `crypto.randomUUID()`; index keys in `RecipeModal`/`RecipeGraph`. Use `crypto.randomUUID()` uniformly.
24. **[LOW] `LandingPage` subcomponents use anonymous inline prop types** — *fe-quality.md* — `LandingPage.tsx:17,36,74`; declare `…Props` interfaces per convention.
25. **[LOW] Sub-recipe children inherit parent cook time; mid-loop failure orphans recipes** — *fe-quality.md* — `AddRecipeModal.tsx:462-482`; child `cook_time`/`shelf_life` copied from the parent form, sequential creates with no rollback. Source child timing from the sub-section; batch or clean up on partial failure.
26. **[LOW] Modal dialogs lack dialog semantics / focus management** — *fe-quality.md* — `RecipeModal.tsx:186-187`, `AddRecipeModal.tsx:517-518`; add `role="dialog"`/`aria-modal`/labelling + focus trap/restore.

**Dependencies**
27. **[LOW] ESLint v8 is End-of-Life; `@typescript-eslint` on v7 (maintenance)** — *00-dependency-audit.md* — dev/lint tooling, not a live CVE. Modernize to ESLint 9 / typescript-eslint 8.

### Info

1. **[INFO] Shopping-list-item IDOR (`UpdateItem`/`DeleteItem`/`ToggleItem`) — REFUTED** — *be-dataaccess.md (refutes 00-route-inventory.md Obs #5)* — `verifyItemOwnership` (`shopping_list_service.go:216-229`) loads the item's own parent list and checks `list.UserID != userID` before every mutation; a user **cannot** modify another user's item by guessing its UUID. Optional hardening: validate `item.ListID == :id` to make the decorative nested route honest.
2. **[INFO] SSRF is partially blind — errors masked, but a success/timing oracle and LLM-echoed content remain** — *be-ssrf.md* — `recipe_handler.go:197-201`; bounds *data-theft* severity (not the request-forgery primitive). No change needed beyond fixing the SSRF.
3. **[INFO] No security/transport headers (`nosniff`/CSP/`X-Frame-Options`/HSTS); uploads served same-origin** — *be-handlers-config.md* — `router.go:18-36`; compounds the stored-content risks. Add a security-headers middleware.
4. **[INFO] `recipe GetByID` returns other users' *public* recipes by ID — by design** — *be-dataaccess.md* — `recipe_service.go:319-333`; matches the public-recipe product model. Not a vulnerability.
5. **[INFO] S3 storage backend is an unreachable stub** — *be-storage.md* — `s3_storage.go:23-31`, `factory.go:15-16`; selecting `type: s3` is fatal at startup, so no silent insecure storage. Apply the same controls if implemented.
6. **[INFO] Imported PDFs are not persisted to `/uploads`** — *be-storage.md* — `recipe_handler.go:206-241`; PDFs are parsed in-memory, never written to the public mount (negative result).
7. **[INFO] `VITE_`-prefixed env vars are inlined into the client bundle (by design)** — *fe-data-exposure.md* — only non-sensitive `VITE_API_URL`/`VITE_ENV` exist; forward-looking warning never to put a secret behind a `VITE_` var.
8. **[INFO] API responses are typed by annotation, not runtime-validated** — *fe-quality.md* — `recipeService.ts:30` etc.; consistent with the project's own pattern (snake_case interfaces match Go structs). Checked and acceptable.

---

## Cross-Phase Reconciliations

These three items were corrected/clarified during consolidation because earlier phases and later deep-dives disagreed:

- **axios is NOT the app's live HTTP client (Phase 1 vs Phase 4).** The Phase 1 dependency audit (`00-dependency-audit.md`) called axios the "frontend's actual HTTP client" and rated its 20+ advisories (SSRF, prototype-pollution MITM, etc.) **High/runtime-exposed**. The Phase 4 frontend XSS/API review (`fe-xss-apiclient.md`) verified the app uses the **native `fetch` API exclusively** and never imports axios. **Corrected conclusion:** axios is an **unused/dead dependency**; because Vite/Rollup tree-shake unused modules, the axios advisories (and its `form-data`/`follow-redirects` transitives) almost certainly do **not** reach the production bundle — real-world runtime impact is **Low**, not High. axios remains a supply-chain liability in the lockfile and **should be removed** (non-breaking, clears every related advisory). Recorded as Low #20.
- **Shopping-list-item IDOR — refuted.** `00-route-inventory.md` (Obs #5) flagged `UpdateItem`/`DeleteItem`/`ToggleItem` as a likely IDOR because the handler binds only `:itemId` and ignores the parent `:id`. The Phase 3 data-access review (`be-dataaccess.md`) verified the service enforces ownership via `verifyItemOwnership` (ties the item to its real parent list, checks `list.UserID`). **This report reflects the refutation** (Info #1) — there is no such IDOR. (The genuine object-level-authorization gap is the *separate* `AddRecipeToList` finding, Medium.)
- **JWT-secret finding de-duplicated.** The weak/default JWT signing secret appears in both `be-auth.md` (rated High) and `be-handlers-config.md` (rated High). It is listed **once** here as High #1.

---

## 4. Remediation Roadmap

Prioritized, numbered. Items map to the findings above.

### Immediate (do now / before any production exposure)
1. **Rotate the live-looking Anthropic API key** in `env.development.yaml:38` — treat as compromised (incident response; part of Medium "plaintext secrets at rest").
2. **Fix the SSRF (Critical) + redirect-SSRF (High) together.** Add scheme restriction, resolved-IP allow/deny validation (block loopback/RFC1918/link-local/unique-local/`169.254.169.254`/`app`/`db`), per-connection IP pinning, and the *same* validation on every redirect hop. While here, add the missing HTTP client **timeout** and **`io.LimitReader`** body cap (the two SSRF Mediums).
3. **Enforce a non-default JWT secret at startup (High).** Fail closed if empty, a known placeholder, or `<32` bytes; source from a secret manager; verify the env override binds.
4. **Lock down `/uploads` (High stored-XSS + Medium public access).** Validate uploads by magic-byte/content-type (allowlist raster images), normalize the stored extension, send `X-Content-Type-Options: nosniff` + `Content-Disposition: attachment`, and serve through an authenticated handler / separate sandboxed origin.

### Short-term (next iteration)
5. **Make multi-step writes atomic (Medium `RunTx`).** Migrate callers to `WithTypedTransaction`, fix inner repos to use the caller's `tx`, delete `RunTx`, and wrap `shoppingListService.Create`. *(Foundational and cheap — kept high in the roadmap despite Medium severity.)*
6. **Encrypt API keys/secrets at rest (Medium).** App-layer AES-GCM/KMS for `user_ai_configs.api_key`; move db/smtp/jwt/AI secrets out of YAML into a secret manager; add secret-scanning pre-commit hooks.
7. **Close the `AddRecipeToList` IDOR (Medium).** Add the `IsPrivate && UserID != userID` guard.
8. **Stop leaking `err.Error()` to clients (Medium).** Add a shared `respondError` helper that maps `AppError.Code` → status (also fixes the Medium error-translation quality issue) and returns generic messages while logging detail.
9. **Fix the logout button (Medium).** Wire it through `authService.logout()`/`useAuth` so the JWT is cleared.
10. **Add auth rate-limiting + token revocation (Mediums).** IP/account-scoped throttling on login/forgot/reset; shorten token lifetime and/or add `tokenVersion`/`jti` revocation; invalidate sessions on password reset.
11. **Add upload/body size limits (Medium).** `MaxMultipartMemory` + `MaxBytesReader`; cap/stream PDF reads with `io.ReadFull` (also fixes the partial-read quality Medium) and add the missing `userID` guard.
12. **Harden the AI surface (Mediums).** Cap input length before prompt-build; add delimiters + system/user separation against prompt injection; validate parsed output (enum whitelist for `Category`).

### Longer-term (hardening & hygiene)
13. **Plan the vite 8 / ESLint 9 toolchain upgrades; remove the dead `axios` dependency** (Medium + Lows). Run `npm audit`/`govulncheck` in CI.
14. **Add security-headers middleware** (`nosniff`/CSP/`X-Frame-Options`/HSTS) and `referrerPolicy="no-referrer"` on remote images (Info + Lows).
15. **Default Gin to release mode; restrict `GET /users/list`; add `iss`/`aud` claims; tighten repo-layer tenant scoping; mass-assignment nested DTOs; CORS wildcard+credentials guard** (Mediums/Lows).
16. **Frontend quality/a11y debt:** decompose `AddRecipeModal`, memoize `RecipeGraph`, dedupe `HomePage`/auth pages, add keyboard support + dialog semantics, gate debug tooling behind `import.meta.env.DEV` (Mediums/Lows).

---

## 5. Positive Observations

The review surfaced substantial, genuinely-good security and engineering practice:

**Backend security**
- **No SQL/GORM injection** — every query uses bound `?` placeholders; `Order`/column names are hard-coded literals; user-driven sort is done in-memory over a whitelist (`be-dataaccess.md`).
- **bcrypt cost 14**, the hash is `json:"-"` (never serialized), no plaintext/weak hashing (`be-auth.md`).
- **JWT algorithm pinned to HMAC** at both verification sites — `alg:none` and RS↔HS confusion are blocked; signature and `exp` are actually validated (`be-auth.md`).
- **Strong password-reset tokens** — 256-bit `crypto/rand`, 1-hour expiry, single-use within a transaction (`be-auth.md`).
- **No repository mass-assignment** — explicit `Select(...)` column whitelists; tenant IDs come from the JWT context, never the request body (`be-dataaccess.md`).
- **AI provider errors are not leaked**, API keys are hidden from JSON responses, and model output reaches **no dangerous sink** (parameterized GORM + auto-escaping React) (`be-ai.md`).
- **Path traversal investigated and mitigated** (UUID rename + `filepath.Base`/`Ext`, non-bindable `ImageURL`); **CORS is not a wildcard** (explicit origin allowlist) (`be-storage.md`, `be-handlers-config.md`).

**Frontend security**
- **No `dangerouslySetInnerHTML`/`innerHTML`/`eval`/`document.write` anywhere** — all untrusted AI/imported recipe content renders through React auto-escaping; no markdown/HTML renderer present (`fe-xss-apiclient.md`).
- **No secrets in the client bundle**, no `console.*` logging, no AI keys on the client, no extra PII in storage, no source maps in prod, no tabnabbing surface (`fe-data-exposure.md`).
- **Bearer-header (not cookie) auth** structurally mitigates classic CSRF; the token is never logged, DOM-injected, or placed in a URL; relative `/api/v1` paths enforce same-origin by construction; 401 handling is centralized (`fe-auth.md`, `fe-xss-apiclient.md`).

**Code quality / architecture**
- **Clean layering** (handler → service → repository) with **consumer-defined interfaces** (textbook Go), structured `zap` logging, constructor DI, solid `context.Context` propagation, and **no concurrency defects** (`be-quality.md`).
- **Strict TypeScript genuinely enforced** — `strict` + `noUnusedLocals`/`noUnusedParameters`; zero `any`/`@ts-ignore`; the five `!` assertions are all guard-justified; clean `type-check` + `lint --max-warnings 0`; consistent service-layer try-catch; **BEM CSS** and snake_case API contract adherence throughout (`fe-quality.md`).

---

## 6. Appendix: Document Index

| # | Source doc | Phase / Area | Headline finding |
|---|------------|--------------|------------------|
| 1 | `00-route-inventory.md` | Phase 0 / Recon — route & auth map | Inventories 38 routes; flagged SSRF, public `/uploads`, IDOR candidates, `/users/list` exposure (downstream-confirmed or refuted) |
| 2 | `00-dependency-audit.md` | Phase 0 / Recon — deps | 7 high / 6 moderate npm advisories; axios flagged High **(later corrected to dead-dep Low)**; vite 8 / ESLint 9 modernization; run `govulncheck` |
| 3 | `00-threat-model.md` | Phase 0 / Recon — trust boundaries | B3 URL-fetch egress is the most dangerous boundary (SSRF); maps 7 boundaries + 6 untrusted-input data flows |
| 4 | `be-auth.md` | Backend / Auth & session | **[High]** weak/default JWT secret; + no rate-limiting, non-revocable tokens, user enumeration |
| 5 | `be-ssrf.md` | Backend / URL fetcher | **[Critical]** no SSRF egress filtering; **[High]** redirect SSRF; timeout & body-cap DoS |
| 6 | `be-storage.md` | Backend / file storage | **[High]** stored-XSS via unrestricted upload type on public `/uploads`; upload DoS; public file access |
| 7 | `be-ai.md` | Backend / AI integration | **[Medium]** plaintext storage of user LLM API keys; AI cost/DoS; prompt injection |
| 8 | `be-dataaccess.md` | Backend / data-access (IDOR/injection) | **[Medium]** `AddRecipeToList` IDOR (private-recipe disclosure); SQLi clean; item-IDOR **refuted** |
| 9 | `be-handlers-config.md` | Backend / handlers, CORS, config | **[High]** weak JWT secret (dup); `err.Error()` leakage; plaintext on-disk secrets; Gin debug mode |
| 10 | `be-quality.md` | Backend / code quality | Non-atomic `RunTx` (dead `WithTypedTransaction`); `IsNotFound` contract break; ad-hoc error translation |
| 11 | `fe-auth.md` | Frontend / auth & session | **[Medium]** logout button never clears the JWT; localStorage-JWT tradeoff |
| 12 | `fe-xss-apiclient.md` | Frontend / XSS sinks & API client | No XSS sinks (positive); native `fetch` (no axios); **axios is a dead dependency** |
| 13 | `fe-data-exposure.md` | Frontend / client data exposure | No secrets in bundle/logs/storage; remote `<img>` without `referrerPolicy` (Low) |
| 14 | `fe-quality.md` | Frontend / code quality | Strict TS enforced (positive); 690-line `AddRecipeModal`; dead auth pages; a11y gaps |

---

*End of consolidated report. Findings are synthesized from the 14 source documents listed above; no new analysis was performed. Severity counts: Critical 1 · High 3 · Medium 27 · Low 27 · Info 8 (total 66).*
