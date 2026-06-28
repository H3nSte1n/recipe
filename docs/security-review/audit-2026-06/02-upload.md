# Audit 02 — Upload & File-Serving Lockdown (PR #34)

Date: 2026-06-28
Scope: Verify PR #34 hardened image upload validation and `/uploads/:filename` serving.
Method: code read + live multipart probes against `http://localhost:18080`.
Stack: storage type = `local` (`env.development.yaml`), so the signature-checked uploads handler is active.

Throwaway users (fresh JWTs): `audit-upl-A-A16545EE-...@example.com`, `audit-upl-B-08066A34-...@example.com`.

Upload endpoint: there is **no** dedicated `/upload` route — images are uploaded via
`POST /api/v1/recipes` (and `PUT /:id`) as multipart (`recipe` JSON part + `image` file part),
handled by `parseRecipeMultipart` (`internal/handler/recipe_handler.go:30-59`).

---

## 1. Serving route is a hardened handler, not a static mount

- `internal/router/router.go:42-49`: when `Storage.Type == "local"`, the app mounts
  `r.engine.GET("/uploads/:filename", uploads.Serve)` using a `signedurl.Signer` —
  **not** `engine.Static`/`StaticFS`.
- `internal/handler/uploads_handler.go:28-50` (`UploadsHandler.Serve`):
  - path-traversal guard (`filename != filepath.Base(filename)`, rejects `..`) → 400 (`:33-37`),
  - HMAC signature verification via `signer.Verify(filename, exp, sig)` → 403 (`:39-44`),
  - sets `X-Content-Type-Options: nosniff` and `Content-Disposition: attachment` (`:46-47`),
  - serves with `c.File(...)` (Content-Type from normalized stored extension).

VERDICT: **PASS** — serving is a signature-gated handler, not a public static mount
(`router.go:48`, `uploads_handler.go:28-50`).

## 2. Upload validation (magic-byte sniff allowlist)

Validation is by sniffed content, not client extension/MIME:
`pkg/storage/local_storage.go:36-43` calls `DetectImageType` before storing and names the file
with the detected extension. `pkg/storage/imagevalidation.go:16-56` allowlists only
JPEG/PNG/GIF/WebP (`:16-21`), explicitly rejecting SVG/HTML which sniff as text (`:23-27`, `:55`).
Body size cap `maxImageUploadBytes = 10 MiB` enforced via `http.MaxBytesReader` +
`ParseMultipartForm` (`recipe_handler.go:22, 31-40`).

Live probes (`POST /api/v1/recipes`, valid token, user A):

| File | Bytes | HTTP | Result |
|------|-------|------|--------|
| `good.png` (real 1x1 PNG magic) | 69 | **201** | accepted; signed `image_url` returned |
| `evil.html` (`<html><script>`) | 38 | **500** | rejected — no recipe/file created |
| `evil.svg` (`<svg><script>`) | 71 | **500** | rejected — no recipe/file created |
| `polyglot.gif` (`GIF89a`+`<script>`) | 38 | **201** | accepted as image/gif (valid GIF magic) — see §3 |
| `big.png` (valid PNG + pad, ~11 MiB) | 11.5 MiB | **413** | `{"error":"upload too large"}` |

Notes:
- HTML/SVG rejection surfaces as HTTP **500** `{"error":"failed to create recipe"}` because
  `DetectImageType`'s error propagates through `service.Create` into the generic handler error
  path (`recipe_handler.go:96-99`). The security outcome is correct (file **not** stored, recipe
  **not** created, no `image_url`); the 500-instead-of-415 is a code-quality caveat, not a vuln.
- The polyglot has a genuine `GIF89a` header so `http.DetectContentType` classifies it `image/gif`
  and it is stored as `.gif`. This is acceptable (task allows "reject OR safe handling") and is
  rendered inert at serve time — confirmed in §3.

VERDICT (HTML reject): **PASS** — not stored (HTTP 500, no image_url; `imagevalidation.go:55`).
VERDICT (SVG reject): **PASS** — not stored (HTTP 500, no image_url; `imagevalidation.go:23-27,55`).
VERDICT (polyglot): **PASS** — accepted but served safely (HTTP 201 → nosniff+attachment, §3).
VERDICT (oversized 413): **PASS** — HTTP 413 `upload too large` (`recipe_handler.go:31-36`).
VERDICT (valid image allowed): **PASS** — HTTP 201 with signed image_url (`local_storage.go:36-56`).

## 3. Serving hardening — headers + signature gate

Live fetch of the signed `good.png` URL (host rewritten to `:18080`):

```
GET /uploads/6155d836-...png?exp=1782758571&sig=cf258eef... HTTP/1.1
-> HTTP/1.1 200 OK
   Content-Type: image/png
   Content-Disposition: attachment
   X-Content-Type-Options: nosniff
```

- Without signature (query stripped): **HTTP 403** `{"error":"invalid or expired link"}`.
- Tampered `sig=deadbeef`: **HTTP 403**.
- Polyglot `.gif` fetched with valid sig: **HTTP 200**, `Content-Type: image/gif`,
  `Content-Disposition: attachment`, `X-Content-Type-Options: nosniff` — inert in a browser.

VERDICT (nosniff): **PASS** — header present on all served files (`uploads_handler.go:46`).
VERDICT (Content-Disposition attachment): **PASS** — header present (`uploads_handler.go:47`).
VERDICT (signature/auth gate): **PASS** — 200 with valid sig, **403** without and with tampered
sig (`uploads_handler.go:39-44`, live).

## 4. Cross-user image access

Design (per code): the signature is a bearer capability over `filename|exp` signed with the
server JWT secret (`pkg/signedurl/signedurl.go:61-81`); it is not user-bound. Real cross-user
protection is that user B cannot **obtain** A's signed URL.

Live test: A created a **private** recipe (`is_private:true`) with an image.
- B `GET /api/v1/recipes/{A_id}` → **HTTP 500** `{"error":"failed to get recipe"}` — B is denied
  and never receives A's signed `image_url`. Enforced at
  `internal/service/recipe_service.go:345-347` (`recipe.IsPrivate && recipe.UserID != userID ->
  ErrUnauthorized`).
- A `GET /api/v1/recipes/{A_id}` (control) → **HTTP 200** with the image_url.
- Filenames are unguessable UUIDv4 (`local_storage.go:43`).

VERDICT: **PASS** — B cannot read A's private recipe and thus cannot obtain a usable signed URL
(`recipe_service.go:345-347`, live). Caveat: denial surfaces as HTTP 500 instead of 403/404
(same generic error path); outcome is correct, status code is a code-quality nit. A leaked signed
URL would work for anyone until expiry — by design (signature is unguessable + 24h TTL,
`signedurl.go:24`).

## 5. ImportFromPDF — userID guard + size cap

Handler `internal/handler/recipe_handler.go:239-296`:
- userID guard: `:240-244` (`userID := middleware.GetUserID(c); if userID == "" -> 401`).
- size cap (`maxPDFUploadBytes = 20 MiB`, `:23`): `http.MaxBytesReader` at `:246`,
  `file.Size > maxPDFUploadBytes -> 413` at `:258-261`, and
  `io.ReadAll(io.LimitReader(f, maxPDFUploadBytes+1))` with `len > cap -> 413` at `:272-280`.
- userID is passed into the service: `:288` `recipeService.ImportFromPDF(ctx, userID, ...)`.

Note: the service method `recipe_service.go:396-417` itself has neither guard — both the auth
check and the size cap live in the handler (cited above). Acceptable since the route is in the
protected (JWT) group (`router.go:108-112`) and the handler is the only caller.

VERDICT: **PASS** — userID guard at `recipe_handler.go:240-244`, size cap at
`recipe_handler.go:246, 258-261, 272-280`.

---

## Summary verdict lines

- Serving via hardened handler (not static): **PASS** — `router.go:48`, `uploads_handler.go:28-50`
- HTML upload rejected: **PASS** — HTTP 500, not stored — `imagevalidation.go:55`
- SVG upload rejected: **PASS** — HTTP 500, not stored — `imagevalidation.go:23-27,55`
- Polyglot handled safely: **PASS** — HTTP 201 then served image/gif + nosniff + attachment
- Oversized upload: **PASS** — HTTP 413 — `recipe_handler.go:31-36`
- Valid image allowed: **PASS** — HTTP 201 signed image_url — `local_storage.go:36-56`
- `X-Content-Type-Options: nosniff`: **PASS** — `uploads_handler.go:46` (live header)
- `Content-Disposition: attachment`: **PASS** — `uploads_handler.go:47` (live header)
- Serve path signature-gated: **PASS** — 200 with sig / 403 without / 403 tampered — `uploads_handler.go:39-44`
- Cross-user image fetch denied: **PASS** — B gets 500 on A's private recipe, no image_url — `recipe_service.go:345-347`
- ImportFromPDF userID guard + size cap: **PASS** — `recipe_handler.go:240-244, 246, 258-261, 272-280`

Code-quality caveats (non-blocking): validation rejections and cross-user denials return HTTP 500
rather than 415/403/404 due to a shared generic error path (`recipe_handler.go:96-99`).
