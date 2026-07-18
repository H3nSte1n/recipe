# Recipe App — Go-Live Blocker Remediation Status (2026-07-16)

Follow-up to `AUDIT-2026-06.md` Section 2 (the four go-live-behind-VPN blockers). This file
tracks what got fixed in the `phase-7-recipe-post-remediation-audit` branch on 2026-07-16, and
what's still open across all severities found by the audit.

---

## ✅ Implemented (2026-07-16) — all four original blockers cleared

| # | Item | What changed |
|---|---|---|
| 1 | DB port exposure | `docker-compose.yml`: removed public `5432:5432`; password now sourced from gitignored `.env` via `DB_PASSWORD` (see `.env.sample`) |
| 2 | Broken registration | `LandingPage.tsx` + `authService.ts` now send `first_name`/`last_name`; dead `RegisterPage.tsx`/`LoginPage.tsx` deleted; verified live (`201 Created`) |
| 3 | Retired Claude model IDs | `pkg/ai/model.go` + `recipe_service.go` → `claude-sonnet-5` / `claude-opus-4-8` / `claude-haiku-4-5`; migration `000015_update_retired_claude_models` updates seeded `ai_models` rows; **live-verified with the rotated key**: `claude-haiku-4-5` and `claude-sonnet-5` both returned real `200`s from the Anthropic API |
| 4 | Leaked Anthropic key | Owner rotated the key in the Anthropic console; old key redacted from `doc/recipe-api/`, `doc/recipe-api/` gitignored; old key purged from local git objects (verified via full repo blob scan, not just `git log`) |

All 5 PRs (#42–#46) merged/closed; `main` now reflects the full state through Phase 8. Backend/frontend build, Go test suite, `tsc`, `eslint` all green on `main`.

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
| No automated test asserting the registration request shape (frontend/backend field-name drift) | Medium | This exact class of bug has no regression guard now that it's fixed |

~~GitHub PR base-chain gap (#43→#44→#45 stacked on the wrong branch)~~ — **resolved**: retargeted
PR #46 to `main` and merged (it already contained phase-3 via an internal merge), merged #43
separately for one doc-only fix that had drifted off the stack, closed #42/#44/#45 as superseded.
All 5 PRs are now merged/closed, `main` reflects everything.

---

**Bottom line:** the app is clear to launch behind the VPN — all four original blockers are fixed,
merged, and live-verified. Everything else above is real but deliberately scoped out of this pass
— see `AUDIT-2026-06.md` and the two remediation-plan backlogs (`recipe-remediation-post-vpn.md`,
`recipe-remediation-code-quality.md`) for full detail.

---

## Recommended priority order for what's next

1. **Confirm deployment topology.** Several "not blocking" calls above (TLS, security headers,
   rate-limiting) assume a real deployment either matches `docker-compose.yml` as-is or fronts it
   with a reverse proxy — neither is confirmed. This is a 5-minute conversation that changes how
   urgent items 2–4 actually are.
2. **Rate-limiting/lockout + email verification on auth endpoints.** Highest-severity item still
   open; becomes a live abuse vector the moment the VPN comes off or the app is shared beyond a
   trusted-few roster.
3. **TLS termination (HTTP → HTTPS) + Postgres `sslmode=require`.** Depends on #1's answer — if a
   proxy already terminates TLS in prod, this is partly done; if not, it's the next real gap.
4. **Frontend: build for production instead of shipping the Vite dev server.** Cheap, contained
   fix (multi-stage Dockerfile), and incidentally resolves the moderate `esbuild` advisory too —
   good ROI even though its exploit path is two-actor.
5. **JWT revocation on password reset/account deletion.** Flagged in the audit as arguably
   blocker-adjacent; worth doing early even though it's technically deferred.
6. Everything else in the `BEFORE-VPN-REMOVAL` table — batch these together (`gin` trust-proxy
   config, security headers, `iss`/`aud`/`iat`/`nbf` claims, pagination/body-size validation,
   error-response leaks) since they're mostly small, related hardening changes.
7. **Add a CI check for model-ID staleness** so the Claude-retirement class of bug (today's
   Blocker #4) can't silently recur — this is the actual root-cause fix, not just today's patch.
8. Verify the three OpenAI model IDs the same way the Claude ones were checked — never
   independently confirmed, same failure mode is possible.
9. Nice-to-haves (debug UI on the landing page, unused `axios` dependency, the 10 code-quality
   items) — no urgency, do opportunistically.
