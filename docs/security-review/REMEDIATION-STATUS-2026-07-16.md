# Recipe App — Go-Live Blocker Remediation Status (2026-07-16)

Follow-up to `AUDIT-2026-06.md` Section 2 (the four go-live-behind-VPN blockers). This file
tracks what got fixed in the `phase-7-recipe-post-remediation-audit` branch on 2026-07-16, and
what's still open across all severities found by the audit.

---

## ✅ Implemented (2026-07-16)

| # | Item | What changed |
|---|---|---|
| 1 | DB port exposure | `docker-compose.yml`: removed public `5432:5432`; password now sourced from gitignored `.env` via `DB_PASSWORD` (see `.env.sample`) |
| 2 | Broken registration | `LandingPage.tsx` + `authService.ts` now send `first_name`/`last_name`; dead `RegisterPage.tsx`/`LoginPage.tsx` deleted; verified live (`201 Created`) |
| 3 | Retired Claude model IDs | `pkg/ai/model.go` + `recipe_service.go` → `claude-sonnet-5` / `claude-opus-4-8` / `claude-haiku-4-5`; migration `000015_update_retired_claude_models` updates seeded `ai_models` rows; verified live via `GET /ai-configs/models` |
| 4 | Leaked key — *scoping only* | Confirmed exposure is confined to local `stash@{0}`, never pushed to any branch — but rotation and stash cleanup are **not done** (see below) |

---

## 🔴 Still open — blocks go-live behind the VPN

| Item | Importance | Go-live impact | Why still open |
|---|---|---|---|
| Rotate the exposed Anthropic API key | Critical | **Blocks go-live** — Blocker #2 from `AUDIT-2026-06.md`, unresolved | Requires action in the Anthropic console; cannot be done by an agent |
| Resolve `stash@{0}` and scrub the leaked key from it | Critical | **Blocks go-live** (same blocker) | The stash also holds real unrelated WIP (`ai_config_service_test.go` conflicts with a different version already at HEAD) — a merge decision for a human, not an agent |

Once those two are done, **all four original go-live-behind-VPN blockers are cleared.**

---

## 🟡 Open — `BEFORE-VPN-REMOVAL` (does not block today's launch, needed before the VPN comes off)

| Item | Importance | Go-live impact |
|---|---|---|
| No rate-limiting/lockout/CAPTCHA on auth endpoints | High | Not blocking now; a compromised tailnet member (or public internet) can brute-force |
| Open registration, no email verification | High | Not blocking now; becomes a real abuse vector once public |
| Plaintext HTTP, binds all interfaces, no TLS termination | High | Not blocking now *if* a reverse proxy fronts it in the real deployment (unverified — see below) |
| Postgres `sslmode=disable` (TLS still off) | High | Not blocking now — DB is internal-network-only after the port fix — but must be set up before a non-Docker-network deployment (owner's call: to be handled at production publish time) |
| No server-side JWT revocation (password reset / account deletion) | Medium | Not blocking now — requires a token already stolen; recommend prioritizing early in the backlog regardless |
| `gin.Default()` trusts all proxies | Medium (latent) | Not blocking now; will make any future IP-based rate-limiter spoofable via `X-Forwarded-For` |
| No security-transport headers (HSTS/CSP/X-Frame-Options/Referrer-Policy) | Low/Medium | Not blocking |
| No global JSON request-body-size limit | Low/Medium | Not blocking |
| No `iss`/`aud` claims on JWTs | Low | Not blocking |
| No `iat`/`nbf` claims (blocks cheap revocation later) | Low | Not blocking |
| Auth middleware degrades missing `user_id` to `""` instead of rejecting | Low | Not blocking |
| Denied cross-tenant requests return 500 (not 403/404); raw Postgres error leak in `ai_config_handler.go` | Low/Medium | Not blocking |
| Nested recipe-write DTOs double as GORM models (mass-assignment shape) | Low | Not blocking; FK always overwritten today |
| No cap on nested ingredient/instruction array size/length | Low | Not blocking |
| Public-recipe pagination unvalidated (negative offset 500s, unbounded `LIMIT`) | Low | Not blocking |
| Gin release-mode not fail-closed by default | Low | Not blocking |
| `GET /users/list` enumerates all members' name+ID | Low | Not blocking |
| Frontend Docker image ships Vite **dev server**, not a production build | Medium | Not blocking (requires a two-actor scenario: member's browser lured to a malicious site) — cheap to fix, also resolves a moderate esbuild advisory |

---

## ⚪ Open — nice-to-have (no security/functional urgency)

| Item | Importance | Go-live impact |
|---|---|---|
| Debug UI (`ThemeExplorer`/`TunnelControls`) renders on public landing page | Low | None |
| `axios` unused dependency, 7 HIGH advisories | Low (hygiene) | None — cheap win, zero imports |
| 10 code-quality items (`IsNotFound` contract, `RecipeGraph` memoization, component decomposition, a11y, vite/ESLint modernization) | Low | None |

---

## ⚠️ Also flagged, not yet acted on

| Item | Importance | Go-live impact |
|---|---|---|
| OpenAI model IDs (`gpt-4`, `gpt-4-turbo-preview`, `gpt-3.5-turbo`) never independently verified — commonly-deprecated IDs | Medium | Could break OpenAI-provider AI features the same way the Claude blocker did; out of scope for this pass |
| No CI check for model-ID staleness or dependency vulnerabilities | Medium (process) | Root cause of the Claude model-ID blocker recurring — nothing catches the next retirement |
| Deployment topology unverified — no production manifest/reverse-proxy/TLS terminator in the repo | Medium | The "not blocking" column above assumes `docker-compose.yml` reflects prod; if it doesn't, some items may already be more/less urgent than stated |
| GitHub PR base-chain gap (#43→#44→#45 stacked on the wrong branch) | Low (process only) | Not an app defect, doesn't block |
| No automated test asserting the registration request shape (frontend/backend field-name drift) | Medium | This exact class of bug has no regression guard now that it's fixed |

---

**Bottom line:** with the Anthropic key rotated and the stash resolved, the app is clear to launch
behind the VPN. Everything else above is real but deliberately scoped out of this pass — see
`AUDIT-2026-06.md` and the two remediation-plan backlogs (`recipe-remediation-post-vpn.md`,
`recipe-remediation-code-quality.md`) for full detail.
