# Backend Security Review — File Storage & Upload Handling

This document reviews the file storage backend (`pkg/storage/`), the multipart upload handlers
(`internal/handler/recipe_handler.go`), the storage service wiring
(`internal/service/recipe_service.go`), and the unauthenticated static file mount at `/uploads`
(`internal/router/router.go`). It is grounded in the real code; every finding cites `file:line`.

Live storage backend is **local only** — the S3 backend is an unimplemented stub and is unreachable
(see Info findings). Relevant config (`env.development.yaml.sample`):
`storage.type: local`, `local_path: ./uploads`, `base_url: http://localhost:8080/uploads`.

---

### [High] Unrestricted upload file type + MIME sniffing → stored XSS via `/uploads`

- **Location:** `pkg/storage/local_storage.go:29-52` (no type validation), `internal/handler/recipe_handler.go:40-42,77-79` (no validation), `internal/router/router.go:34-36` (public static serve).
- **Description:** `UploadFile` accepts any uploaded file and writes it to disk with the
  **attacker-controlled extension preserved**: `ext := filepath.Ext(file.Filename)` then
  `filename := uuid + ext` (`local_storage.go:30-31`). There is no Content-Type check, no
  magic-byte/MIME validation, and no extension allow-list anywhere in the handler
  (`recipe_handler.go:40-42`) or service (`recipe_service.go:103-113`). The file is then served
  publicly by Gin's `engine.Static("/uploads", ...)` (`router.go:35`), which is backed by
  `http.FileServer`. That server sets `Content-Type` from the file extension via
  `mime.TypeByExtension` (so `.html` → `text/html`, `.svg` → `image/svg+xml`), and when the
  extension is unknown/absent it falls back to `http.DetectContentType` content sniffing — which
  classifies a body beginning with `<html`/`<!DOCTYPE` as `text/html`. No
  `X-Content-Type-Options: nosniff` and no `Content-Disposition: attachment` headers are set. An
  attacker can therefore upload an `.html`/`.svg` (or extension-less HTML) "image" and have it
  served as active content.
- **Impact:** Stored cross-site scripting. Exploit path (state honestly): an `<img src>` will *not*
  execute the payload — the victim must navigate directly to the upload URL `/uploads/<uuid>.html`,
  and the attacker must deliver that (unguessable) URL. Impact is origin-dependent: in the sample
  config uploads are served from the backend origin (`:8080`), which is separate from the SPA origin
  (`:5173`) where the JWT lives in `localStorage`, so this is **not** automatic token theft. It
  escalates to full account takeover (Critical) if `/uploads` is served from the **same origin** as
  the SPA — the typical single-origin reverse-proxy production deployment — where injected script
  can read the JWT and act as the victim. Independent of origin, script runs on a trusted app domain
  (phishing, CSRF pivot).
- **Recommendation:** Validate uploads against an allow-list of image content types by sniffing the
  actual bytes (`http.DetectContentType` / decode the image), not the client extension; reject
  anything that is not a known raster image. Normalize the stored extension from the detected type.
  On the serving side, always send `X-Content-Type-Options: nosniff` and
  `Content-Disposition: attachment` (or serve uploads from a separate, cookie-less, sandboxed
  origin/CDN). Consider re-encoding images server-side to strip embedded scripts/metadata.

---

### [Medium] No upload size limit → disk-fill (images) and memory-exhaustion (PDF) DoS

- **Location:** `pkg/storage/local_storage.go:47` (`io.Copy`, no limit); `internal/handler/recipe_handler.go:220-224` (PDF read into memory); no `MaxMultipartMemory`/`MaxBytesReader` configured (`internal/router/router.go:17-28`, `cmd/api/main.go`).
- **Description:** Image uploads are streamed to disk with an unbounded `io.Copy(dst, src)`
  (`local_storage.go:47`) — there is no per-file or per-request size cap, and the router never sets
  `engine.MaxMultipartMemory` or wraps the body in `http.MaxBytesReader`. For PDF import, the handler
  reads the **entire** uploaded file into a heap buffer: `fileBytes := make([]byte, file.Size)`
  followed by `f.Read(fileBytes)` (`recipe_handler.go:220-221`). `file.Size` is the size the
  multipart parser actually spooled (no client amplification), but with no size cap an attacker can
  upload an arbitrarily large file that is then pulled fully into memory on top of the multipart temp
  spool.
- **Impact:** A single authenticated user (registration is open) can exhaust disk by uploading large
  "images", or drive backend memory pressure / OOM by importing a very large PDF. Denial of service.
- **Recommendation:** Enforce a maximum upload size at the edge (`http.MaxBytesReader` on
  `c.Request.Body` and a sane `engine.MaxMultipartMemory`), reject oversized files with HTTP 413, and
  cap the PDF parser input (stream/limit instead of `make([]byte, file.Size)`).

---

### [Medium] Unauthenticated public access to all uploaded files (`/uploads/*`)

- **Location:** `internal/router/router.go:34-36`; URL minted in `pkg/storage/local_storage.go:51`.
- **Description:** The static mount `engine.Static("/uploads", r.config.Storage.LocalPath)` is
  registered on the bare engine, **outside** the JWT-protected group (`router.go:42-44`). Every file
  written by `UploadFile` is reachable by anyone with the URL, with no `Authorization` check and no
  tie between the requester and the recipe/owner. Recipe images for **private** recipes
  (`IsPrivate`, `recipe.go:128`) are stored in exactly the same public directory, so a leaked or
  shared image URL exposes that asset to unauthenticated third parties regardless of recipe privacy.
- **Impact:** Broken access control for uploaded media. Confidentiality of private-recipe images
  depends solely on URL secrecy, not on authentication or ownership.
- **Recommendation:** Serve uploads through an authenticated handler that verifies the caller may
  access the owning recipe (or issues short-lived signed URLs), rather than a blanket public static
  mount. At minimum, document that all uploads are world-readable-by-URL and keep filenames
  unguessable (see filename-predictability note below).

---

### [Low] Path traversal in stored/deleted filenames — mitigated (defense-in-depth verified)

- **Location:** `pkg/storage/local_storage.go:30-33` (upload), `pkg/storage/local_storage.go:54-56` (delete).
- **Description:** The task asks whether user-controlled filenames reach a filesystem path without
  sanitization. They do not, and the reasons are worth recording explicitly:
  - **Upload:** the user-supplied `file.Filename` base name is **discarded** — the stored name is
    `uuid.New().String() + filepath.Ext(file.Filename)` (`local_storage.go:30-31`). Only the
    extension survives, and `filepath.Ext` returns the suffix after the final dot and cannot contain
    a path separator. Additionally, Go's `mime/multipart` `Part.FileName()` already applies
    `filepath.Base`, stripping any directory components before the handler ever sees the name. So a
    filename like `../../etc/cron.d/x` cannot escape `uploadDir` via `filepath.Join` here.
  - **Delete:** `DeleteFile` resolves the target with `filepath.Base(fileURL)` (`local_storage.go:55`)
    and the input (`existingRecipe.ImageURL`) is a server-generated value, not user-supplied — and
    `ImageURL` is not a bindable request field (`CreateRecipeRequest` only exposes
    `Image *multipart.FileHeader json:"-"`, `recipe.go:137`), so a client cannot inject an arbitrary
    path into the delete sink.
- **Impact:** No path-traversal write/delete primitive was found. Residual risk is only the
  attacker-influenced extension, addressed by the stored-XSS finding above.
- **Recommendation:** Keep discarding the client base name; additionally normalize/allow-list the
  extension when the type-validation fix (High finding) is implemented.

---

### [Info] Filename predictability — UUIDv4 (unguessable), the load-bearing mitigation

- **Location:** `pkg/storage/local_storage.go:31`.
- **Description:** Stored filenames are `uuid.New()` (random UUIDv4, 122 bits of entropy) plus an
  extension. Because `/uploads/*` has no authentication (Medium finding above), this randomness is
  the *only* thing preventing enumeration of other users' files. UUIDv4 is not guessable or
  sequentially enumerable, so user A cannot discover user B's uploads by iterating IDs.
- **Impact:** Acceptable as a mitigation; note that it is "security through unguessability," not
  access control — it does not survive URL leakage (referer, sharing, logs).
- **Recommendation:** Treat the UUID as a defense-in-depth layer, not the primary control; pair with
  the authenticated-serving recommendation in the Medium finding.

---

### [Info] Imported PDFs are not persisted to `/uploads`

- **Location:** `internal/handler/recipe_handler.go:206-241`.
- **Description:** Resolving the recon question "do sensitive PDFs land in the public mount?": they
  do not. `ImportFromPDF` reads the PDF into an in-memory buffer and passes the bytes to the parser
  service (`recipe_handler.go:233`); it never calls `fileStorage.UploadFile`. No PDF is written to
  `local_path`, so PDFs are not exposed via `/uploads`. (See the Medium DoS finding for the in-memory
  read concern.)
- **Impact:** None directly — recorded as a negative result.
- **Recommendation:** None.

---

### [Info] PDF read uses a single `f.Read` that can short-read (correctness)

- **Location:** `internal/handler/recipe_handler.go:220-224`.
- **Description:** `f.Read(fileBytes)` is a single `Read` call; `io.Reader` may return fewer bytes
  than requested without error, so the buffer can be partially populated, handing a truncated PDF to
  the parser. This is a correctness bug rather than a direct security issue, but truncated input can
  cause inconsistent parsing behavior.
- **Impact:** Potential silent truncation of imported PDFs.
- **Recommendation:** Use `io.ReadFull` / `io.ReadAll` (with a size cap, per the Medium finding).

---

### [Info] S3 storage backend is an unreachable stub

- **Location:** `pkg/storage/s3_storage.go:23-31`, `pkg/storage/factory.go:15-16`.
- **Description:** `s3FileStore.UploadFile`/`DeleteFile` are no-ops that return `"", nil` / `nil`
  (`s3_storage.go:24-30`). The constructor is never called: `NewFileStore` returns
  `fmt.Errorf("s3 storage not implemented")` for `type: "s3"` (`factory.go:16`), and `main.go`
  treats that as fatal (`logger.Fatal` at `cmd/api/main.go:47-49`). So selecting S3 cannot silently
  store files in a non-persistent/insecure way — the app refuses to start instead.
- **Impact:** No live exposure today. A future implementation must repeat the type/size/ACL controls
  recommended above (and avoid public-read ACLs).
- **Recommendation:** When implementing S3, apply the same content-type validation and size limits,
  and serve via authenticated/signed URLs with private bucket ACLs.

---

### [Info] Multipart branch bypasses request-body validators

- **Location:** `internal/handler/recipe_handler.go:32-42,70-79`.
- **Description:** On `multipart/form-data`, the `recipe` field is parsed with `json.Unmarshal`
  (`recipe_handler.go:34,72`), which does **not** run the Gin `binding:` validators that the JSON
  branch's `ShouldBindJSON` enforces (`recipe_handler.go:44,81`). Constraints such as `SourceType`
  `oneof`, `Servings` `min=1`, and `Rating` bounds (`recipe.go:126,129,136`) are therefore skipped
  whenever a file is uploaded. Not a storage vulnerability per se, but it is reachable via the upload
  path and weakens input validation on the same requests that accept files.
- **Impact:** Invalid/unbounded recipe fields can be persisted via the upload path.
- **Recommendation:** Run the same struct validation (`validator`) on the unmarshaled struct in the
  multipart branch.

---

## Summary

| Severity | Count |
|----------|-------|
| Critical | 0 |
| High     | 1 |
| Medium   | 2 |
| Low      | 1 |
| Info     | 5 |

**Most serious finding:** stored XSS via unrestricted upload type + MIME sniffing served from the
public `/uploads` mount (High). Path traversal in stored/deleted filenames was investigated and found
**mitigated** (UUID rename + `filepath.Base`/`filepath.Ext`, non-bindable `ImageURL`). The High issue
escalates to account-takeover severity if `/uploads` shares an origin with the SPA in production.
