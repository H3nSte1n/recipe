# Threat Model — Trust Boundaries & Untrusted-Input Data Flows

> Phase 0 of the security review. This document **maps** the trust boundaries and
> untrusted-input data flows of the Recipe backend (Go/Gin) so that later phases can
> perform deep vulnerability analysis on each surface. It does **not** itself attempt
> to confirm or exploit vulnerabilities — every flow below is grounded in real code,
> and each entry lists the **threat categories to investigate**, not verified findings.

## Architecture recap

- **Frontend**: React 18 + TypeScript SPA. All API calls go through a Vite proxy to `/api/v1/...`.
- **Backend**: Go, Gin framework, clean architecture (handler → service → repository), GORM + PostgreSQL.
- **Auth**: JWT (HS256), `Authorization: Bearer <token>`. Public routes: `register`, `login`,
  `forgot-password`, `reset-password`. Everything else is behind
  `AuthMiddleware.AuthRequired()` (`internal/middleware/auth.go`).
- **External dependencies the backend talks to**: arbitrary remote websites (recipe URL import),
  LLM providers (Anthropic / OpenAI), an SMTP server, and the local filesystem (uploaded images,
  served statically at `/uploads`).

Routes are defined in `internal/router/router.go`.

---

## Trust boundaries

A *trust boundary* is any point where data moves between zones of differing trust. Each is a
place where input must be validated/encoded and output must be constrained.

| # | Boundary | Direction | What crosses it | Code anchor |
|---|----------|-----------|-----------------|-------------|
| B1 | Client (browser / any HTTP client) ↔ API | inbound | JSON & multipart request bodies, headers, query params, URL path params, JWT bearer tokens, uploaded files (image, PDF), import URLs, free-text instructions | `internal/router/router.go`, `internal/handler/*` |
| B2 | API ↔ PostgreSQL | outbound/inbound | Recipes, users, profiles, shopping lists, password-reset tokens, **LLM-generated recipe content** persisted via GORM | `internal/service/*`, repositories |
| B3 | API ↔ arbitrary external website (URL fetch) | **outbound network egress** | User-supplied URL → HTTP GET to any host the server can reach; remote HTML body flows back in | `pkg/urlparser/fetcher.go`, `pkg/urlparser/service.go` |
| B4 | API ↔ LLM provider (Anthropic / OpenAI) | outbound + inbound | Untrusted web/PDF/free-text content embedded into a prompt is sent to the provider; **untrusted model output** flows back and is parsed into domain objects | `pkg/ai/claude_model.go`, `pkg/ai/gpt_model.go`, `pkg/ai/model.go`, `pkg/ai/parser.go` |
| B5 | API ↔ filesystem (local storage + static serving) | outbound (write) + inbound (read/serve) | Uploaded image bytes written to disk; files served back to browsers at `/uploads/*` | `pkg/storage/local_storage.go`, `internal/router/router.go` (`engine.Static("/uploads", ...)`) |
| B6 | API ↔ SMTP server | outbound | Password-reset email containing a reset link/token, sent to a user-controlled recipient address | `pkg/email/email.go`, `internal/service/user_service.go` |
| B7 | API ↔ user's browser (response rendering) | outbound | JSON responses including LLM-generated and user-supplied recipe fields rendered by the React SPA | handlers return `c.JSON(...)`; frontend renders |

Notable boundary properties:
- **B3 (URL fetch egress) is the most dangerous boundary** and the top priority for the next phase:
  it has **no egress restrictions in the fetched code** (no allowlist, no private-IP/loopback
  filtering, no scheme restriction beyond Gin's syntactic `url` check), it is directly exploitable
  with no LLM cooperation, and from inside Docker it can reach internal services
  (`db:5432`, `app:8080`) and cloud metadata (`169.254.169.254`) — all behind a low auth bar
  (open registration).
- **B4 (LLM) is the broadest-reach boundary**: the same untrusted text crosses *three* boundaries in
  a row — it enters at B1/B3, is sent to the LLM at B4, comes back, is stored at B2, and is shown to
  the browser at B7. Its worst realistic outcome (recipe JSON, escaped by React on render) is more
  bounded than SSRF's, so it ranks second.

---

## Untrusted-input data flows

### 1. File upload (recipe image)

- **Entry point**: `RecipeHandler.Create` / `RecipeHandler.Update`
  (`internal/handler/recipe_handler.go`). On `multipart/form-data` it reads the `recipe` form
  field as JSON and the `image` form field via `c.FormFile("image")`, stuffing the
  `*multipart.FileHeader` into `CreateRecipeRequest.Image` (`internal/domain/recipe.go`).
- **Processing → sink**: `recipeService.Create/Update` calls
  `fileStorage.UploadFile(ctx, req.Image)` →
  `localFileStore.UploadFile` (`pkg/storage/local_storage.go`). Implementation:
  - Filename is randomized: `uuid.New() + filepath.Ext(file.Filename)` — the **attacker-controlled
    extension is preserved**, the base name is not.
  - Writes via `os.Create(filepath.Join(uploadDir, filename))`, copies with `io.Copy`.
  - Returns a public URL `baseURL/filename`.
  - `DeleteFile` resolves the path with `filepath.Base(fileURL)` before `os.Remove`.
- **Boundary crossed**: B5 (filesystem write) and later B7 (served to browsers via
  `engine.Static("/uploads", ...)` in `router.go` when `Storage.Type == "local"`).
- **Threat categories to investigate**:
  - **Unrestricted file type / stored XSS**: no MIME/content-type or magic-byte validation; an
    uploaded `.html`/`.svg` served from the same origin under `/uploads` could execute script in a
    victim's browser (B7).
  - **No size limit visible** in the upload path → disk-fill / memory DoS.
  - **Path traversal**: attacker controls `file.Filename`; the upload side neutralizes it via
    `uuid + Ext`, and `DeleteFile` uses `filepath.Base` — verify these are the *only* sinks and that
    `Ext` can't smuggle a separator.
  - **Validation bypass via multipart branch**: in the `multipart/form-data` branch the `recipe`
    field is parsed with `json.Unmarshal`, which **skips the Gin `binding:` validators** that the
    JSON branch (`ShouldBindJSON`) enforces — fields like `source_type` `oneof`, `servings` `min=1`,
    and `rating` bounds are not validated on multipart submissions.
  - **Content-type sniffing** of served files (the `Static` handler's headers).
  - S3 backend is **not implemented** (`s3_storage.go` is a stub returning `""`/`nil`;
    `factory.go` returns `"s3 storage not implemented"`), so only the local path is live.

### 2. URL parser / fetcher (recipe import from URL) — SSRF surface

- **Entry point**: `RecipeHandler.ImportFromURL` (`internal/handler/recipe_handler.go`) binds
  `domain.ImportURLRequest{ URL string binding:"required,url" }` (`internal/domain/recipe.go`).
  The only validation is Gin's `url` validator (syntactic), not host/scheme restriction.
- **Processing → sink**: `recipeService.ImportFromURL` → `urlparser.service.Parse`
  (`pkg/urlparser/service.go`) → `contentFetcher.Fetch` (`pkg/urlparser/fetcher.go`):
  - `http.NewRequestWithContext("GET", urlStr, nil)` then `client.Do` against the **raw
    user-supplied URL**.
  - The `http.Client` (`pkg/urlparser/service.go`) sets only `CheckRedirect: defaultRedirectPolicy`
    (`pkg/urlparser/helpers.go`) which caps redirects at 10. **No timeout, no allowlist, no
    private-IP/loopback/link-local filtering, no scheme restriction beyond Gin's syntactic check.**
  - Response body is fully read with `io.ReadAll` (no size cap), parsed by `goquery`
    (`pkg/urlparser/parser.go`), JSON-LD extracted (`extractJsonLD` in `helpers.go`), cleaned, then
    handed to the LLM.
- **Boundary crossed**: B3 (outbound egress to attacker-chosen host) → B4 (fetched content → LLM)
  → B2 (stored) → B7 (rendered).
- **Threat categories to investigate**:
  - **SSRF** (primary): server will fetch internal/private addresses (`169.254.169.254` cloud
    metadata, `127.0.0.1`, RFC1918, Docker service names like `app:8080`/`db:5432`), and follow
    redirects toward them.
  - **Response-size DoS**: unbounded `io.ReadAll`.
  - **Missing request timeout** on the client → slow-loris / hang.
  - **Prompt injection downstream**: the fetched HTML is attacker-controlled and flows verbatim
    into the LLM prompt (see flow 4).

### 3. PDF parser (recipe import from PDF)

- **Entry point**: `RecipeHandler.ImportFromPDF` (`internal/handler/recipe_handler.go`):
  `c.FormFile("file")`, then allocates `make([]byte, file.Size)` and reads the file into memory.
- **Processing → sink**: `recipeService.ImportFromPDF` → `pdfparser.service.Parse`
  (`pkg/pdfparser/service.go`):
  - `pdf.NewReader(bytes.NewReader(pdfData), len(pdfData))` using `github.com/ledongthuc/pdf`.
  - Iterates pages, `page.GetPlainText(nil)`, concatenates text, then sends text to the LLM.
- **Boundary crossed**: B1 (upload) → in-memory parse → B4 (extracted text → LLM) → B2 → B7.
- **Threat categories to investigate**:
  - **Memory exhaustion / DoS**: whole file buffered via `make([]byte, file.Size)`; no size limit
    enforced in the handler or parser.
  - **Decompression / "PDF bomb"** handling in the third-party `ledongthuc/pdf` reader (malformed
    or deeply nested/compressed streams).
  - **Malformed-input robustness** (panics/unbounded loops) in the PDF library.
  - **No content-type validation** that the uploaded `file` is actually a PDF.
  - (Note: the handler uses a single `f.Read(fileBytes)` which may short-read — a correctness bug
    worth flagging, not a direct security issue.)
  - **Prompt injection**: extracted PDF text flows into the LLM prompt (flow 4).

### 4. AI prompt handling (untrusted text → LLM → back) — highest-leverage surface

- **Entry points (untrusted text sources)**:
  - Web page text (flow 2), PDF text (flow 3).
  - `RecipeHandler.ParsePlainTextInstructions` →
    `domain.ParsePlainTextInstructionsRequest.PlainText` (free text, `binding:"required"` only).
  - Shopping-list item names (`CategorizeItems`), reachable from shopping-list handlers.
- **Processing → sink (egress)**: all routes build prompts by **string-interpolating the untrusted
  content directly** into the template:
  - `createPrompt(content, contentType)`, `createParseInstructionsPrompt(content)`,
    `createPromptToCategorizeShoppingListItems(items)` — all `fmt.Sprintf` with no delimiting,
    escaping, or instruction/data separation (`pkg/ai/model.go`).
  - Sent to the provider: `ClaudeModel.Parse/ParseInstructions/CategorizeItems`
    (`pkg/ai/claude_model.go`) or `GPTModel.*` (`pkg/ai/gpt_model.go`). Model + key chosen by
    `ModelFactory.CreateModel` (`pkg/ai/model.go`); key may come from user AI config or server config.
- **Processing → sink (ingress, untrusted model output)**: `pkg/ai/parser.go`:
  - `parseAIResponse` takes the substring between the first `{` and last `}` and
    `json.Unmarshal`s it into `AIRecipeResponse` (`pkg/ai/types.go`), then maps to
    `domain.Recipe`. `parseInstructions` / `parseCategorizeItemsResponse` strip markdown fences and
    unmarshal. The resulting fields are persisted (B2) and returned to the browser (B7).
- **Boundary crossed**: B4 (both directions), then B2 and B7.
- **Threat categories to investigate**:
  - **Prompt injection**: attacker-controlled page/PDF/free text can carry instructions
    ("ignore previous instructions…") into the model with no data/instruction separation.
  - **Untrusted LLM output treated as data**: the model output becomes recipe title/description/
    ingredient/instruction fields that are stored and rendered → **stored-XSS** consideration at B7
    (confirm the React render path escapes; check any `dangerouslySetInnerHTML`).
  - **Output-shape trust**: lenient `{`…`}` substring extraction may accept partially-controlled
    JSON; check for injection of unexpected fields / oversized payloads.
  - **Sensitive-data handling**: `fmt.Print(message)` in `ClaudeModel.Parse` logs the full provider
    response to stdout (info-leak / log hygiene).
  - **API-key handling**: per-user API keys (`UserAIPreferences.APIKey`,
    `getUserAIPreferences`) — confirm storage/exposure of keys.

### 5. Email (password-reset) — outbound SMTP

- **Entry point**: `UserHandler.ForgotPassword` → `domain.ForgotPasswordRequest{ Email
  binding:"required,email" }`. `userService.ForgotPassword` (`internal/service/user_service.go`)
  generates a 32-byte token via `crypto/rand`, stores it (1-hour expiry), and calls
  `emailService.SendPasswordResetEmail`.
- **Processing → sink**: `emailService.SendPasswordResetEmail(to, resetToken)`
  (`pkg/email/email.go`):
  - Builds `resetLink := "http://" + frontendUrl + "/reset-password?token=" + resetToken`.
  - Builds the raw message with `fmt.Sprintf("To: %s\r\n" + "Subject: %s\r\n" ... , to, subject, body)`
    and calls `smtp.SendMail(host:port, PlainAuth(...), from, []string{to}, msg)`.
- **Boundary crossed**: B6 (outbound SMTP), and the reset link returns to the user out-of-band.
- **Threat categories to investigate**:
  - **Email header injection**: `to` is interpolated directly into the `To:` header. The recipient
    is a previously-registered, `email`-validated address (drawn from the DB user), which limits
    this, but verify the validator rejects CR/LF and that `to`/`subject` are never attacker-shaped
    elsewhere.
  - **Cleartext reset link**: link is hard-coded `http://` (not HTTPS) → token exposure in transit.
  - **Token in URL / referer leakage**; token strength is fine (`crypto/rand` 32 bytes).
  - **SMTP credential handling** (`from`, `password` from config) and **open-relay / spoofing**.
  - **User-enumeration**: handler returns a generic "if the email exists…" message and the service
    swallows not-found — good; confirm timing doesn't leak.

### 6. Auth inputs (register / login / forgot / reset)

- **Entry points** (`internal/handler/user_handler.go`, all public per `router.go`):
  - `Register` → `domain.RegisterRequest` (email, password `min=8`, first/last name).
  - `Login` → `domain.LoginRequest` (email, password).
  - `ForgotPassword` → `ForgotPasswordRequest`; `ResetPassword` → `ResetPasswordRequest`
    (token + new password `min=8`).
- **Processing → sink**: `internal/service/user_service.go`:
  - Passwords hashed with bcrypt cost 14 (`domain.HashPassword`).
  - JWT issued HS256 with `user_id`/`email`/`exp` claims, signed with config secret
    (`generateToken`); validated in `middleware/auth.go` (rejects non-HMAC alg).
  - Reset token validated for `Used`/expiry, password updated + token marked used in a tx.
- **Boundary crossed**: B1 → B2 (user/token rows) → B6 (email for forgot) → JWT returned to client.
- **Threat categories to investigate**:
  - **User enumeration on register**: `Register` returns `"email already registered"` (distinct
    from the deliberately-generic forgot-password response).
  - **No rate limiting / lockout** on `Login` or `ForgotPassword` → credential brute-force / reset
    spamming.
  - **JWT secret strength & alg confidence**: HS256 with a config secret; check secret source and
    that `none`/RS↔HS confusion is excluded (middleware checks `SigningMethodHMAC`, good).
  - **Token validation completeness**: confirm `exp` is enforced and claims are required.
  - **Authorization / data exposure**: `GET /users/list` (`UserHandler.ListAll`) returns all users
    to any authenticated caller — check field exposure and whether it should be admin-only.
  - **Password policy**: only `min=8`, no complexity/breach checks.

---

## Data-flow summary table

| Source (entry point) | Untrusted input | Crosses boundary | Primary threat(s) to investigate |
|----------------------|-----------------|------------------|----------------------------------|
| `RecipeHandler.Create/Update` → `localFileStore.UploadFile` | Multipart image bytes + attacker-controlled filename/extension | B5 filesystem write → B7 served `/uploads` | Unrestricted file type / **stored XSS**, no size limit (DoS), content sniffing; path traversal (mitigated by `uuid+Ext`/`filepath.Base`) |
| `RecipeHandler.ImportFromURL` → `contentFetcher.Fetch` | Arbitrary URL string | B3 outbound egress → B4 LLM → B2 → B7 | **SSRF** (metadata/internal hosts), unbounded response (DoS), no timeout, redirect abuse, downstream prompt injection |
| `RecipeHandler.ImportFromPDF` → `pdfparser.Parse` | Uploaded PDF bytes | B1 → in-memory parse → B4 → B2 → B7 | **PDF/decompression bomb & malformed-PDF DoS**, memory exhaustion (no size cap), no type check, downstream prompt injection |
| URL/PDF/free-text/shopping items → `createPrompt*` → `Claude/GPT` → `parseAIResponse` | Untrusted text into prompt; untrusted model output back | B4 (both ways) → B2 → B7 | **Prompt injection**, untrusted LLM output stored & rendered (**stored XSS**), response logging (`fmt.Print`), API-key handling |
| `UserHandler.ForgotPassword` → `SendPasswordResetEmail` | Recipient email + reset link/token | B6 outbound SMTP | Email header injection (CR/LF in `To:`), cleartext `http://` reset link, token-in-URL leakage, SMTP creds/relay |
| `UserHandler.Register/Login/ForgotPassword/ResetPassword` | Email, password, reset token | B1 → B2 → B6 / JWT to client | User enumeration (register), no rate limiting/brute-force protection, JWT secret strength, `GET /users/list` data exposure, weak password policy |

---

## Surfaces from the brief — existence check

- **File upload** — exists. Live backend is **local storage only** (`pkg/storage/local_storage.go`);
  the **S3 backend is an unimplemented stub** (`pkg/storage/s3_storage.go` returns empty/nil,
  `factory.go` errors `"s3 storage not implemented"`).
- **URL parser / fetcher** — exists (`pkg/urlparser/fetcher.go`, `parser.go`, `service.go`,
  `helpers.go`, `selectors.go`). Live SSRF surface.
- **PDF parser** — exists (`pkg/pdfparser/service.go`, single file; uses `ledongthuc/pdf`).
- **AI prompt handling** — exists (`pkg/ai/claude_model.go`, `gpt_model.go`, `model.go`,
  `parser.go`, `types.go`, `preferences.go`). Note: prompt builders and parsers live in
  `model.go` and `parser.go`; there is no separate `preferences` logic beyond the
  `UserAIPreferences` struct.
- **Email** — exists (`pkg/email/email.go`, single file; SMTP password-reset only).
- **Auth inputs** — exist (`internal/handler/user_handler.go`,
  `internal/service/user_service.go`, `internal/domain/auth.go`, `internal/domain/user.go`,
  `internal/middleware/auth.go`).

All six requested surfaces are present in code. No surface from the brief is missing; the only
non-functional item is the **S3 storage backend (stub, not wired)**.
