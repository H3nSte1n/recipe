# Phase 5 — Triage of `recipe-remediation-post-vpn.md` Against Current Code

**Subtask:** For each item in `~/.claude/plans/recipe-remediation-post-vpn.md`, check current code,
mark Already-done / Partially-done / Still-open, and classify
`GO-LIVE-BEHIND-VPN-BLOCKER` vs `BEFORE-VPN-REMOVAL`.

**Method:** Read every bullet in the plan, then re-checked the cited file/line against current
code (not the plan's citations, which may have drifted). Cross-referenced this audit's own Phase 3
findings (`03-*.md`) where they independently rediscovered the same gap — a strong consistency
signal that neither review missed something the other caught.

---

## Phase 1 (plan): Auth abuse resistance

| Item | Status | Classification | Evidence |
|---|---|---|---|
| Rate limiting / lockout on login, forgot-password, reset-password | **Still open** | `BEFORE-VPN-REMOVAL` | Confirmed in `03-vpn-deps.md` Finding 2 — no limiter middleware exists anywhere in `internal/middleware/`. |
| Token revocation (short-lived+refresh or `tokenVersion`/`jti` denylist; invalidate on reset/delete) | **Still open** | `BEFORE-VPN-REMOVAL` (see note) | Confirmed in `03-auth.md` Finding 1 — `ResetPassword`/`Delete` never touch anything JWT-related; a stolen token survives a password reset for its full 24h lifetime. **Note:** the plan itself flags this as the "priority slice" worth doing even alone — a compromised-tailnet-member threat model (the stated VPN threat model) makes this arguably closer to a blocker than a pure post-VPN item, since the compromised-member scenario is explicitly in-scope *today*. Recommend the Phase 7 consolidation weigh upgrading this specific sub-item's classification. |
| Close user-enumeration oracles (generic register response; dummy-hash login timing; async always-200 forgot-password) | **Still open** | `BEFORE-VPN-REMOVAL` | Re-verified against current code: `user_service.go:66-68` (`Register`) still returns a distinguishable `"email already registered"` error when the email exists — a direct enumeration oracle, not yet made generic. `Login` (`:108-116`) already returns the *same message* ("invalid credentials") for both "no such user" and "wrong password" — but only the not-found path skips the bcrypt comparison entirely, so a **timing** side-channel remains (bcrypt cost ~100ms+ vs. an instant DB miss). `ForgotPassword` (`:129-155`) already returns `nil` (success) uniformly regardless of whether the user exists — but the found-user branch synchronously creates a token, generates randomness, and calls `s.emailService.SendPasswordResetEmail` (a network call) inline, while the not-found branch returns almost immediately — the same class of timing oracle. None of the three sub-items are fully closed; the message-uniformity half of register/forgot-password is partially done, but timing remains open on all three. |

## Phase 2 (plan): Error & information-leak hardening

| Item | Status | Classification | Evidence |
|---|---|---|---|
| Shared `respondError` helper; replace every raw `err.Error()` 500 | **Still open** | `BEFORE-VPN-REMOVAL` | Independently rediscovered in `03-config.md` Finding 1 and `03-idor.md` Finding 1 — `ai_config_handler.go`, `profile_handler.go`, `user_handler.go` still echo `err.Error()` into 500s (live-reproduced a raw Postgres error); `recipe_handler.go` maps everything to a flat 500 with the wrong status code instead. Neither half of the plan's fix has landed. |
| Gin release-mode-by-default + fail-closed on unrecognized `APP_ENV`; confirm Recovery never leaks stack traces; gate/remove `fmt.Print` of LLM response | **Partially done** | `BEFORE-VPN-REMOVAL` | The `fmt.Print` of the full LLM response cited at `claude_model.go:41` in the plan **no longer exists** — grepped `pkg/ai/claude_model.go` and all of `internal/`/`pkg/` for `fmt.Print`/`log.Print`: zero matches (`03-config.md` "Checked and clean" section). This sub-item is **done**. The release-mode default is **not** — `cmd/api/main.go:39-44` still only sets `gin.ReleaseMode` conditionally on `APP_ENV == "production"` rather than defaulting to release and requiring explicit opt-in to debug, and there is no fail-closed check on an unrecognized `APP_ENV` value (`03-config.md` Finding 3). `gin.Recovery()` (bundled in `gin.Default()`) does return a generic 500 without a stack trace in the HTTP response regardless of mode — confirmed clean, not re-flagged as a separate item. |

## Phase 3 (plan): Transport & browser-facing hardening

| Item | Status | Classification | Evidence |
|---|---|---|---|
| Security-headers middleware (HSTS, CSP, X-Frame-Options, nosniff, Referrer-Policy) | **Still open** | `BEFORE-VPN-REMOVAL` | Confirmed in `03-vpn-deps.md` Finding 7 — `router.go:27` wires only CORS; no global security-headers middleware exists. (The per-upload `nosniff` on `/uploads/:filename`, `uploads_handler.go:44`, is unrelated and already correct — it's the *global* header this item is about.) |
| CORS wildcard+credentials guard; explicit nested-write DTOs so client `id`/`recipe_id` are ignored | **Still open** | `BEFORE-VPN-REMOVAL` (CORS) / `LOW`, defense-in-depth (mass-assignment) | `internal/middleware/cors.go:9-18` still passes `allowedOrigins` straight from config into `cors.Config{AllowOrigins: ..., AllowCredentials: true}` with no guard rejecting a literal `"*"` entry. The nested-DTO fix (ignoring client-supplied `id`/`recipe_id` on `RecipeIngredient`/`RecipeInstruction`) is also still open — independently rediscovered in `03-injection.md` Finding 1, which notes it is **not currently exploitable** (GORM's auto-save-association overwrites the FK regardless) but is a fragile shape that should still get the dedicated-DTO fix as defense-in-depth. |
| Frontend: `referrerPolicy="no-referrer"` + scheme allowlist on recipe `<img src>`; self-host/no-referrer the hardcoded Unsplash landing images | **Partially open** | `BEFORE-VPN-REMOVAL` (LOW) | Recipe `<img src>` (`RecipeCard.tsx`, `RecipeModal.tsx`, `RecipeGraph.tsx`) currently only ever points at the app's own signed-URL uploads (`04-data.md` — no external hotlinking occurs for recipe images today, since AI-imported recipes don't populate an image field), so the original risk this targeted is lower than the plan assumed — but no `referrerPolicy`/scheme-allowlist has been added regardless, so if a future feature re-introduces external recipe images the gap is still there structurally. The **Unsplash landing-page images are still hotlinked** and still open: `ScatteredBackground.tsx:6-17` hardcodes 10+ `images.unsplash.com` URLs rendered via CSS `backgroundImage` (`:90`), with no `referrerpolicy`/meta-referrer mitigation — every landing-page view leaks the visitor's presence (via the `Referer` header) to Unsplash's CDN. |

## Phase 4 (plan): Token-storage & defense-in-depth

| Item | Status | Classification | Evidence |
|---|---|---|---|
| Token storage hardening: HttpOnly+Secure+SameSite cookie+CSRF, **or** keep localStorage but add strict CSP + shorter token lifetime + `apiFetch` as sole entry point + auth-gated initial render | **Partially done** | `BEFORE-VPN-REMOVAL` | Two of the four "keep localStorage" sub-requirements are **already done**, independently confirmed in `04-auth.md`: `apiFetch` **is** the sole entry point for every authenticated call (`recipeService.ts`, all five functions), and initial render **is** gated by `checkAuth()` in `App.tsx:9`. The other two are **not**: there is still no CSP (ties to the Phase 3 security-headers item above) and the token lifetime remains `24h` (`env.development.yaml.sample:22`), not shortened. The cookie+CSRF alternative was not pursued (the codebase committed to the localStorage path, per the two done sub-items) — this is a valid design choice, but the remaining two hardening steps for that path are still open. |
| Backend: validate `iss`/`aud`(+`nbf`/`iat`); reject empty/missing `user_id` (401); repo-layer `Where(id=? AND user_id=?)` tenant-scoping fail-safe | **Still open** | `LOW`, defense-in-depth | All three independently rediscovered in `03-auth.md` (Findings 2-4) and `03-idor.md` — none implemented. As `03-idor.md` documents, the *service-layer* ownership checks are correct everywhere probed, so the repo-layer fail-safe is genuinely defense-in-depth today, not covering a live gap — but it remains unimplemented. |

## Summary

Of the plan's 11 numbered sub-items (Phases 1–4), **1 is fully done** (the `fmt.Print` LLM-response
log removal), **3 are partially done** (Gin release-mode default, token-storage hardening,
recipe-image referrer/scheme allowlist), and **7 remain fully open**. Nothing in this plan should
be removed as already-satisfied in bulk — only the one fully-done sub-item warrants deletion from
the live plan; the rest should stay, with the partial items' remaining halves called out explicitly
so a future execution pass doesn't redo already-shipped work.

No item in this plan was found to be a `GO-LIVE-BEHIND-VPN-BLOCKER` under the stated threat model
(small trusted-few tailnet, compromised member in scope) — with one flagged exception worth Phase 7
attention: **JWT non-revocability on password reset** is arguably closer to a blocker than a pure
post-VPN item, since a compromised member is explicitly in-scope *today*, not just after VPN
removal (see the Phase 1 table row above).

## Checks performed

1. Read every bullet of `~/.claude/plans/recipe-remediation-post-vpn.md` end-to-end.
2. Re-read each cited file/line from current code (not trusting the plan's citations to still be
   accurate) — `user_service.go`, `internal/middleware/cors.go`, `cmd/api/main.go`,
   `pkg/ai/claude_model.go`, `env.development.yaml.sample`, `App.tsx`, `recipeService.ts`,
   `ScatteredBackground.tsx`.
3. Cross-referenced every item against this audit's own Phase 3/4 findings (`03-auth.md`,
   `03-idor.md`, `03-config.md`, `03-vpn-deps.md`, `03-injection.md`, `04-auth.md`, `04-data.md`)
   to confirm consistency between the two independent review passes.
4. Grepped for `fmt.Print`/`log.Print` in `pkg/ai/` to confirm the one fully-done sub-item.

---

*No production code was modified. This file is the only artifact written.*
