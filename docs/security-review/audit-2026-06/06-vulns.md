# Phase 6 — `govulncheck` (Backend) + `npm audit` (Frontend), Triaged

**Subtask:** Run `govulncheck` (backend) and `npm audit` (frontend), capture HIGH/CRITICAL
advisories, and map each to a triage bucket.

**Scope/method:** Installed and ran `govulncheck ./...` fresh against the backend module (not
previously run in this audit). `npm audit` was already run fresh in `04-deps.md`'s Phase 4
dependency review — referenced here rather than re-run, with its findings mapped into the same
triage buckets as the backend results for a single Phase 6 view.

---

## Backend: `govulncheck ./...` — 34 reachable vulnerabilities, 3 from third-party modules, 31 from the Go standard library

`govulncheck` only reports vulnerabilities in code paths your program **actually calls**
(reachability analysis) — it additionally found 12 more in imported packages and 18 more in
required modules that are *not* called, correctly excluded from the count below.

### Third-party module vulnerabilities (3 modules, 6 distinct advisories)

| Module | Vulnerable → Fixed | Advisory | Reachability | Bucket |
|---|---|---|---|---|
| `github.com/jackc/pgx/v5` (Postgres driver) | `v5.7.2` → `v5.9.2` | **GO-2026-5004: SQL injection via placeholder confusion with dollar-quoted string literals** | Reachable only via `database.MigrateDB` → `sql.DB.Close` → `sanitize.SanitizeSQL` (an internal driver helper) — **not** via any app-constructed query. Every app-level query in this codebase uses GORM's parameterized `Where("... = ?", arg)` form (confirmed exhaustively in `03-injection.md`); the app never calls `SanitizeSQL` or builds raw SQL with dollar-quoted literals itself. | **MEDIUM** — the CVE is real and the fix is a trivial version bump with no functional risk, but the app's own query-construction pattern does not exercise the vulnerable code path as currently written. Upgrade regardless — cheap insurance, and it removes any risk from a future code change that *does* build SQL with dollar-quoted strings. |
| `golang.org/x/net` (HTML parser, via `goquery`) | `v0.49.0` → `v0.55.0` | **GO-2026-5030** (XSS via duplicate attributes), **GO-2026-5029** (incorrect DOCTYPE character-reference handling), **GO-2026-5028** (DoS parsing arbitrary HTML), **GO-2026-5027** (incorrect handling of foreign-content HTML elements) — all in `golang.org/x/net/html` | **Reachable and security-relevant**: `pkg/urlparser/parser.go:25` (`contentParser.Parse`) calls `goquery.NewDocumentFromReader` → `html.Parse` on the **raw HTML fetched from a user-supplied URL during recipe import** — i.e. attacker-influenced (any URL the requesting user chooses to import from) content reaches this exact vulnerable parser. | **HIGH** — this is the one dependency finding in this audit that sits directly in an attacker-reachable data path (untrusted third-party HTML → parser), not just a build/toolchain concern. The DoS variant (GO-2026-5028) is the most directly actionable given `urlparser` already has SSRF/timeout/size-cap hardening (`02-ssrf.md`) but that hardening bounds the *fetch*, not the *parse* — a small-but-adversarial HTML payload could still trigger parser-side quadratic/DoS behavior post-fetch. Recommend bumping `golang.org/x/net` promptly; low risk to functionality since it's a transitive `goquery` dependency, not directly imported. |
| `github.com/aws/aws-sdk-go-v2/{aws/protocol/eventstream,service/s3}` | `v1.7.4`/`v1.96.0` → `v1.7.8`/`v1.97.3` | **GO-2026-5764: DoS via panic in AWS SDK Go v2 EventStream decoder** | Reachable via `pkg/storage/s3_storage.go`'s package-level `init()` (S3 client construction), **not** via any actual S3 API call — this codebase's `s3FileStore.UploadFile`/`DeleteFile` are unimplemented stubs (confirmed in `03-rechallenge.md` item 4), and `storage.type` is `local` in the current config, so the S3 client is never actually driven with live traffic. | **LOW** — reachability here is an artifact of package initialization, not of the app exercising S3 functionality (which doesn't work at all yet — `03-rechallenge.md`). No live exposure today; still worth bumping alongside the others since it's a no-cost version update. |

### Go standard library (31 advisories — dominated by an outdated Go toolchain, not app code)

All 31 remaining findings are in `crypto/tls`, `crypto/x509`, `net/http`, `net/url`, `net/mail`,
`net/textproto`, `encoding/asn1`, `encoding/pem` — every one traces back to the **installed Go
toolchain being `go1.25.0`**, with fixes landing in later `go1.25.x` point releases (`go1.25.2`
through `go1.25.12` across the different CVEs). These are not application-code defects; they are
inherited entirely from which Go point release built the binary.

- **Bucket: MEDIUM (toolchain hygiene), trivial fix.** Bumping the Go toolchain used by
  `services/backend/Dockerfile`'s build stage to the latest `go1.25.x` patch release closes all 31
  in one move — no code change required. A few of the higher-severity-sounding ones worth naming:
  `GO-2026-5039` (unescaped inputs in `net/textproto` errors — reachable via the password-reset
  handler's error path and PDF-import MIME header parsing), `GO-2026-4012`/`GO-2025-4012`-class
  memory-exhaustion bugs in `net/http` cookie/DER/PEM parsing, and the `crypto/tls` handshake/ECH
  issues. None of these were independently exploited or live-probed in this audit; they're listed
  here as a toolchain-currency gap, consistent with `govulncheck`'s own "reachable but not
  necessarily exploited" reporting model.

## Frontend: `npm audit` — see `04-deps.md` for full detail; summary below

Already run fresh during this audit's Phase 4 (`04-deps.md`) rather than re-run here. Summary for
this phase's triage view:

| Finding | Severity | Bucket |
|---|---|---|
| 7 HIGH advisories in `axios` (SSRF, prototype pollution, header/credential leaks, etc.) | HIGH (as published) | **LOW in practice** — confirmed `axios` is unused in `src/` and absent from the real production bundle (tree-shaken); removing the dependency clears all 7 for free (`04-deps.md` Finding 1). |
| `esbuild`/`vite` moderate advisory (dev-server request/response exposure) | MODERATE | **LOW** — dev-server-only, not present in the production build; fix requires a semver-major `vite@8` bump. |
| `flatted`, `form-data`, `ajv`, `brace-expansion`, `follow-redirects`, `@babel/core` | Mixed HIGH/MODERATE/LOW | **LOW** — all build/lint/dev-time-only transitive dependencies, not present in the shipped runtime bundle (`04-deps.md`). |

## Consolidated triage buckets (backend + frontend)

| Bucket | Items |
|---|---|
| **HIGH — fix before go-live** | `golang.org/x/net` HTML-parser vulnerabilities (attacker-reachable via URL-import HTML parsing) |
| **MEDIUM — fix soon, low functional risk** | `jackc/pgx` SQL-injection CVE (driver-level, not exercised by app's parameterized queries, but a free upgrade); Go toolchain point-release bump (closes 31 stdlib advisories); AWS SDK Go v2 DoS advisory (S3 client unused/stub today) |
| **LOW — cheap wins, no live exposure** | frontend `axios` removal (clears 7 advisories); frontend build-toolchain (`vite@8`) upgrade |

## Checks performed

1. Installed `govulncheck` (`go install golang.org/x/vuln/cmd/govulncheck@latest`) and ran
   `govulncheck ./...` against `services/backend`.
2. Read every reported vulnerability's reachability trace to distinguish "called by app code
   the audit already reviewed" from "reachable only through package init / internal driver
   helpers" — informing the severity/bucket judgment above rather than reporting raw advisory
   counts.
3. Cross-checked the `golang.org/x/net`/`goquery` finding against `pkg/urlparser/parser.go` and
   this audit's `02-ssrf.md`/`03-bypass.md` to confirm the parse step sits downstream of (and is
   not covered by) the existing SSRF/fetch hardening.
4. Cross-checked the AWS SDK finding against `03-rechallenge.md`'s confirmation that S3 storage is
   an unimplemented stub in this codebase and `storage.type: local` in current config.
5. Referenced (not re-run) the fresh `npm audit` results already captured in `04-deps.md`.

---

*No production code was modified. This file is the only artifact written.*
