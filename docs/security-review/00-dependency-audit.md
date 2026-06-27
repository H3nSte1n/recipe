# 00 — Dependency Audit

_Security recon: inventory of backend (Go) and frontend (npm) dependencies, flagging
versions with known vulnerabilities or that are EOL/unmaintained._

## Methodology

- **Frontend findings are authoritative**: produced by `npm audit --json` run against the
  installed `node_modules` in `services/frontend/` on 2026-06-27. Severities, advisory
  titles, vulnerable ranges, and fix availability are quoted directly from that report.
- **Backend findings are version-based only**: no network vulnerability scanner (e.g.
  `govulncheck`, OSV) could be run in this environment. The Go assessment below is derived
  from the module versions declared in `services/backend/go.mod`. Where a specific CVE is
  not verifiable from the version alone, the risk is described as a category rather than
  attributed to a fabricated CVE ID. **A `govulncheck ./...` run is required to confirm
  the Go side authoritatively.**

Source files reviewed:
- `services/backend/go.mod`
- `services/frontend/package.json` (declared ranges) + installed versions via `npm ls`

---

## Backend (Go) dependencies

**Go toolchain version: `go 1.25.0`** — current/recent major line; not EOL.

### Direct dependencies

| Module | Version | Status / notes |
|---|---|---|
| github.com/gin-gonic/gin | v1.10.1 | Recent (1.10.x). No obvious known-vuln from version alone. Confirm with govulncheck. |
| github.com/gin-contrib/cors | v1.7.6 | Recent. CORS config correctness matters more than the lib version — review allowed origins separately. |
| **github.com/golang-jwt/jwt/v5** | **v5.3.1** | **Good**: this is the actively maintained successor to the deprecated `dgrijalva/jwt-go`. v5.3.x is current. No version-level concern. |
| github.com/golang-migrate/migrate/v4 | v4.18.2 | Recent. Migration tooling, not request-path exposed. |
| github.com/google/uuid | v1.6.0 | Current, stable. |
| golang.org/x/crypto | v0.47.0 | Recent. This module is a frequent source of advisories (e.g. SSH/terminal) — keep current; verify with govulncheck. |
| gorm.io/gorm | v1.25.12 | Recent 1.25.x line. |
| gorm.io/driver/postgres | v1.5.11 | Recent. |
| github.com/jackc/pgx/v5 | v5.7.2 (indirect) | Recent Postgres driver. |
| github.com/spf13/viper | v1.19.0 | Recent. Config loading, not request-path. |
| github.com/PuerkitoBio/goquery | v1.10.2 | Recent. Used for HTML parsing of imported URLs — SSRF/parsing surface; review usage, not just version. |
| github.com/ledongthuc/pdf | v0.0.0-20240201… | Pseudo-version (untagged commit) from 2024-02. Niche/low-activity PDF parser; parses untrusted PDF uploads. Parser robustness is a real concern even absent a named CVE — treat parsing of user-supplied PDFs as risk surface. |
| github.com/sashabaranov/go-openai | v1.38.0 | Recent. |
| github.com/anthropics/anthropic-sdk-go | v0.2.0-alpha.13 | **Alpha / pre-1.0 SDK.** Not a known-CVE issue, but alpha API stability/maintenance risk; pin and watch for breaking updates. |
| github.com/aws/aws-sdk-go-v2/service/s3 | v1.96.0 | Recent v2 SDK (good — v1 aws-sdk-go is the legacy line). |
| golang.org/x/net | v0.49.0 (indirect) | Recent. Historically a recurring advisory source (HTTP/2, html). Keep current; verify with govulncheck. |
| google.golang.org/protobuf | v1.36.6 (indirect) | Recent. |

### Backend summary

No clearly deprecated or obviously known-vulnerable module was identified from versions
alone. Notably **the project uses `golang-jwt/jwt/v5` (maintained), not the deprecated
`dgrijalva/jwt-go`** — a common red flag that is _absent_ here. The Go toolchain (1.25) and
the major frameworks (gin, gorm, x/crypto, x/net) are all on recent lines.

Caveats requiring a real scanner:
- `golang.org/x/crypto` and `golang.org/x/net` are the most frequent Go CVE sources; only
  `govulncheck` can confirm whether the pinned patch levels are affected.
- `ledongthuc/pdf` (untagged, low-activity) and `goquery` sit on untrusted-input parsing
  paths (PDF import, URL import) — worth manual review beyond version matching.
- `anthropics/anthropic-sdk-go` is an **alpha** release — maintenance/stability risk.

---

## Frontend (npm) dependencies

Declared direct deps (`package.json`): `axios ^1.7.5`, `react ^18.3.1`, `react-dom ^18.3.1`;
devDeps include `vite ^5.1.0`, `eslint ^8.57.0`, `@typescript-eslint/* ^7.0.0`,
`typescript ^5.2.2`, `@vitejs/plugin-react ^4.2.1`.

Installed versions of note: **axios@1.13.5**, **vite@5.4.21**, **eslint@8.57.1**.

### `npm audit` severity counts

| Severity | Count |
|---|---|
| Critical | 0 |
| High | 7 |
| Moderate | 6 |
| Low | 1 |
| **Total** | **14** |

### Notable advisories

| Package | Direct? | Severity | Representative advisory title(s) | Vulnerable range | Fix available |
|---|---|---|---|---|---|
| **axios** | **Yes (direct dep)** | **high** | 20+ advisories incl. _SSRF via NO_PROXY bypass_ (GHSA-pjwm-pj3p-43mv, CVSS 8.6), _Full MITM via prototype pollution in `config.proxy`_ (GHSA-35jp-ww65-95wh, 8.7), _Auth bypass via prototype pollution_, _ReDoS via cookie name_, _Proxy-Authorization credential leak on redirect_ | `>=1.0.0 <1.16.0` (installed 1.13.5 is affected) | Yes (upgrade to axios ≥1.16.0) |
| **vite** | **Yes (direct dev dep)** | high | Multiple dev-server advisories (via esbuild/rollup chain) | `<=6.4.2` (installed 5.4.21) | Yes, but **fix is `vite@8.1.0` — semver-major** |
| esbuild | No (via vite) | moderate | Dev server allows any website to send requests & read responses (GHSA-67mh-4wv8-2f99) | `<=0.24.2` | Via vite major upgrade |
| rollup | No (via vite, **build bundler**) | high | Rollup 4 has **Arbitrary File Write via Path Traversal** | `4.0.0 - 4.58.0` | Yes |
| form-data | No (transitive, via axios) | high | CRLF injection via unescaped multipart field names (GHSA-hmw2-7cc7-3qxx) | `4.0.0 - 4.0.5` | Yes |
| flatted | No (transitive) | high | Unbounded-recursion DoS in parse(); prototype pollution | `<=3.4.1` | Yes |
| minimatch | No (transitive) | high | ReDoS via repeated wildcards / `matchOne()` GLOBSTAR backtracking / nested extglobs | `<=3.1.3 \|\| 9.0.0-9.0.6` | Yes |
| picomatch | No (transitive) | high | ReDoS via extglob quantifiers (also method injection in POSIX char classes, moderate) | `<=2.3.1` | Yes |
| follow-redirects | No (via axios) | moderate | Custom auth header leak to cross-domain redirect (GHSA-r4q5-vmmm-2653) | `<=1.15.11` | Yes |
| postcss | No (transitive) | moderate | XSS via unescaped `</style>` in CSS stringify output | `<8.5.10` | Yes |
| ajv | No (transitive) | moderate | ReDoS via `$data` option (GHSA-2g4f-4pwh-qvx6) | `<6.14.0` | Yes |
| brace-expansion | No (transitive) | moderate | Zero-step sequence DoS (GHSA-f886-m6hf-6m8v) | `<1.1.13 \|\| 2.0.0-2.0.3` | Yes |
| js-yaml | No (transitive) | moderate | Quadratic-complexity DoS in merge-key handling via repeated aliases | `<=4.1.1` | Yes |
| @babel/core | No (transitive) | low | Arbitrary file read via sourceMappingURL comment (GHSA-4x5r-pxfx-6jf8) | `<=7.29.0` | Yes |

### Non-CVE maintenance flag (frontend)

- **eslint@8.57.1 — ESLint v8 is End-of-Life / no longer maintained** (the v8 line stopped
  receiving updates after the v9 release). This is not a `npm audit` finding but is an
  EOL/unmaintained-toolchain concern. `@typescript-eslint/*` is on the older **v7** line
  (current is v8). These are dev/lint tooling, lower runtime risk but should be modernized.
- **vite@5.4.21** is a full major version behind the audit's recommended fix (`vite@8`).

---

## Priority flags

Highest-concern items to address first:

1. **axios (direct dependency, installed 1.13.5) — HIGH, runtime-exposed.** This is the
   single highest-priority concern: it carries 20+ advisories including SSRF (CVSS 8.6),
   full MITM and auth-bypass via prototype pollution, ReDoS, and Proxy-Authorization
   credential leaks. axios is the frontend's actual HTTP client to the backend. A simple
   non-breaking upgrade to **axios ≥1.16.0** clears every axios advisory and should also pull
   in fixed `form-data` / `follow-redirects` (verify the resolved transitive versions after
   upgrading).
2. **vite build/dev toolchain — esbuild + rollup (HIGH).** Mixed exposure, not uniformly
   dev-only: the **esbuild** advisory is dev-server-only (a website can read dev-server
   responses), but **rollup** is the **production build bundler** and its advisory is
   _Arbitrary File Write via Path Traversal_ — a build-environment risk (a malicious
   dependency/config could write outside the output dir during `vite build`), not merely a
   dev-server issue. The audit's only offered fix is a **semver-major bump to vite 8** — plan
   it deliberately (test build + `@vitejs/plugin-react` compatibility).
3. **ESLint v8 EOL + @typescript-eslint v7 (maintenance risk, not a live CVE).** Modernize
   the lint toolchain to ESLint 9 / typescript-eslint 8 to stay on supported software.
4. **Backend: run `govulncheck ./...` (verification gap).** No Go module looked clearly
   vulnerable from versions alone, and the JWT library choice is correct (`golang-jwt/jwt/v5`,
   not the deprecated `dgrijalva/jwt-go`). But `x/crypto`, `x/net`, the untagged
   `ledongthuc/pdf` parser, and the **alpha** `anthropics/anthropic-sdk-go` warrant an
   authoritative scan and manual review of the untrusted-input parsing paths (PDF/URL import).
