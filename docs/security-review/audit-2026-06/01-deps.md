# 01 — Dependency State Snapshot & Drift (Phase 1 Audit, P1.3)

_Date: 2026-06-28. Audit branch: `phase-1-recipe-post-remediation-audit`._
_Scope: snapshot current backend (Go) + frontend (npm) dependency state and diff against
the prior baseline `docs/security-review/00-dependency-audit.md` (dated 2026-06-27)._

**Method / limitations**
- Backend: read `services/backend/go.mod` + spot-checked `go.sum` for versions. **No
  `govulncheck`/OSV scan was run** (would require network and/or could mutate state) — Go
  assessment is version-based only, same limitation as the baseline. **INCONCLUSIVE** on
  live Go CVEs; defer to Phase 6.
- Frontend: read `package.json` + `package-lock.json`; resolved installed versions from the
  existing `node_modules`. `npm audit` and `npm audit --omit=dev` **ran successfully against
  the existing lockfile** (read-only, no install, no lockfile mutation) — counts below are
  authoritative for the current tree.

---

## Backend (Go) dependencies

**Go toolchain: `go 1.25.0`** (no separate `toolchain` directive). Current major line, not EOL.
Unchanged vs baseline.

### Direct dependencies (`go.mod` require block #1)

| Module | Version | Note |
|---|---|---|
| github.com/PuerkitoBio/goquery | v1.10.2 | HTML parse of imported URLs — SSRF/parse surface. |
| github.com/anthropics/anthropic-sdk-go | v0.2.0-alpha.13 | **Alpha / pre-1.0** — maintenance/stability risk (not a known CVE). |
| github.com/aws/aws-sdk-go-v2/service/s3 | v1.96.0 | Modern v2 SDK (good). |
| github.com/gin-contrib/cors | v1.7.6 | CORS config correctness matters more than version. |
| github.com/gin-gonic/gin | v1.10.1 | Recent. |
| github.com/glebarez/sqlite | v1.11.0 | **Pure-Go SQLite used for tests** (avoids cgo). Test-only path; not request-exposed. |
| github.com/golang-jwt/jwt/v5 | v5.3.1 | Maintained successor to deprecated `dgrijalva/jwt-go` — correct choice. |
| github.com/golang-migrate/migrate/v4 | v4.18.2 | Migration tooling, not request-path. |
| github.com/google/uuid | v1.6.0 | Stable. |
| github.com/ledongthuc/pdf | v0.0.0-20240201131950-… | **Untagged pseudo-version**, low-activity PDF parser; parses untrusted uploads. Manual review surface. |
| github.com/sashabaranov/go-openai | v1.38.0 | Recent. |
| github.com/spf13/viper | v1.19.0 | Config loading. |
| github.com/stretchr/testify | v1.10.0 | Test dep. |
| go.uber.org/zap | v1.27.0 | Logging. |
| golang.org/x/crypto | v0.47.0 | Used for **AES-GCM at-rest encryption** (remediation). Frequent advisory source — verify with govulncheck. |
| gorm.io/driver/postgres | v1.5.11 | Recent. |
| gorm.io/gorm | v1.25.12 | Recent 1.25.x. |

Security-relevant indirect (spot-check): `golang.org/x/net v0.49.0`, `github.com/jackc/pgx/v5
v5.7.2`, `google.golang.org/protobuf v1.36.6` — all on recent lines.

**Remediation-relevant deps present (PRs #31–#38 context):**
- `golang.org/x/crypto v0.47.0` — supports AES-GCM at-rest credential encryption.
- `github.com/glebarez/sqlite v1.11.0` — pure-Go SQLite for tests.
- No new dedicated SSRF/HTTP-client library was added; URL/PDF import still relies on
  `goquery` + `ledongthuc/pdf` + stdlib `net/http`. (SSRF controls, if any, are in app code,
  not a library — flag for the SSRF-focused audit subtask.)

---

## Frontend (npm) dependencies

| Package | Declared (`package.json`) | Locked / installed | Note |
|---|---|---|---|
| axios | ^1.7.5 | **1.13.5** | Direct **runtime** dep. Still in vulnerable range (`<1.16.0`). **Zero imports in `src/` — dead dependency** (confirmed by grep). |
| react | ^18.3.1 | 18.3.1 | OK. |
| react-dom | ^18.3.1 | 18.3.1 | OK. |
| @types/react | ^18.3.3 | (dev) | — |
| @types/react-dom | ^18.3.0 | (dev) | — |
| @typescript-eslint/eslint-plugin | ^7.0.0 | 7.18.0 | v7 line (current is v8). |
| @typescript-eslint/parser | ^7.0.0 | 7.18.0 | v7 line. |
| @vitejs/plugin-react | ^4.2.1 | 4.7.0 | — |
| eslint | ^8.57.0 | **8.57.1** | **ESLint v8 is EOL/unmaintained.** |
| eslint-plugin-react-hooks | ^4.6.0 | (dev) | — |
| eslint-plugin-react-refresh | ^0.4.5 | (dev) | — |
| typescript | ^5.2.2 | 5.9.3 | Current. |
| vite | ^5.1.0 | **5.4.21** | Major behind audit's recommended fix (vite 8). Pulls vulnerable esbuild 0.21.5 / rollup 4.57.1. |

Transitive of note (current install): `esbuild 0.21.5`, `rollup 4.57.1` (in vulnerable
`4.0.0–4.58.0` range), `form-data 4.0.5`, `follow-redirects 1.15.11`.

### `npm audit` summary (authoritative — ran against existing lockfile)

| Scope | info | low | moderate | high | critical | total |
|---|---|---|---|---|---|---|
| `npm audit` (incl. dev) | 0 | 1 | 6 | 7 | 0 | **14** |
| `npm audit --omit=dev` (prod only) | 0 | 0 | 1 | 2 | 0 | **3** |

Prod-only 3 = the axios runtime chain (axios + transitive `form-data` / `follow-redirects`).
The remaining 11 are dev/build-toolchain (vite → esbuild/rollup/postcss/picomatch/etc.).

---

## Drift vs `00-dependency-audit.md` (baseline 2026-06-27)

**Net drift since baseline: ZERO.** The dependency tree is unchanged between the baseline
(2026-06-27) and this audit one day later — nothing added, removed, upgraded, or downgraded
in `package.json`/`package-lock.json` or `go.mod` for the items the baseline tracked.

**Caveat (what this diff does NOT show):** this measures *drift since the 06-27 baseline*,
not the *pre→post-remediation delta*. The baseline appears to already be a post-remediation
snapshot (it lists `x/crypto v0.47.0`, and `glebarez/sqlite` is present in go.mod), so
"no drift" does **not** establish whether PRs #31–#38 themselves changed dependencies.
Per the task premise, glebarez/sqlite and the AES-GCM crypto path were *added by* those PRs;
that addition would have landed before the baseline was written. Confirming the actual
pre→post-remediation dependency delta requires a pre-#31 `go.mod` from git history, which is
out of scope for this read-only subtask → **INCONCLUSIVE** on the remediation delta.
(`docs/STATUS.md` and the existing security-review docs contain no dependency-change record
to confirm it non-git.)

| Item | Baseline | Now | Status |
|---|---|---|---|
| Backend deps (gin, jwt/v5, x/crypto, gorm, pgx, x/net, s3, anthropic alpha, ledongthuc/pdf) | as listed | identical versions | **No change** |
| Go toolchain | go 1.25.0 | go 1.25.0 | **No change** |
| axios (declared / installed) | ^1.7.5 / 1.13.5 | ^1.7.5 / 1.13.5 | **No change** — still vulnerable range, still **dead** (0 `src/` imports). |
| vite | ^5.1.0 / 5.4.21 | ^5.1.0 / 5.4.21 | **No change** — still 3 majors behind fix (vite 8). |
| eslint | ^8.57.0 / 8.57.1 | ^8.57.0 / 8.57.1 | **No change** — still EOL. |
| @typescript-eslint/* | ^7.0.0 / 7.18.0 | same | **No change** — still v7 (v8 current). |
| npm audit totals | 1 low / 6 mod / 7 high / 0 crit = 14 | 1 / 6 / 7 / 0 = 14 | **Identical.** |

**Conclusion:** the dependency-hygiene findings raised in the baseline (axios upgrade to
≥1.16.0, vite major bump, ESLint 9 / typescript-eslint 8 modernization, removal of the dead
axios dep) are **all still open** — every flagged version is unchanged from the baseline.
Whatever #31–#38 changed (the task notes they added the AES-GCM crypto path and
glebarez/sqlite), those PRs did not close any of the baseline's dependency-hygiene flags.

---

## For later phases

1. **Phase 6 (verification):** run `govulncheck ./...` in `services/backend` for an
   authoritative Go CVE result — required to clear `x/crypto v0.47.0`, `x/net v0.49.0`,
   the untagged `ledongthuc/pdf`, and the alpha `anthropic-sdk-go`. Backend remains
   **INCONCLUSIVE** on live CVEs here.
2. **axios:** confirmed still a direct runtime dep with **zero `src/` usage** → recommend
   removing it outright (eliminates 3 prod-scope advisories at once). If kept, bump to
   ≥1.16.0. Worth an explicit finding in the remediation phase.
3. **Frontend toolchain:** vite 5→8 (semver-major; test `@vitejs/plugin-react` compat),
   ESLint 8→9, typescript-eslint 7→8. Dev/build-scope but EOL.
4. **SSRF subtask:** no SSRF-specific library exists; URL/PDF import SSRF controls (if any)
   live in app code over stdlib `net/http` + `goquery` — review there, not in `go.mod`.
