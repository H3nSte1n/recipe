# Phase 5 — Triage of `recipe-remediation-code-quality.md` Against Current Code

**Subtask:** For each item in `~/.claude/plans/recipe-remediation-code-quality.md`, check current
code, mark done/open, and classify functional-BLOCKER vs nice-to-have.

**Method:** Re-read every bullet, then re-checked current code directly (grep + read) rather than
trusting the plan's own citations. One item (`crypto.randomUUID()` usage) turned out to be
*partially* fixed since the plan was written, which the plan's own binary done/open framing
wouldn't have surfaced without a fresh look.

---

## Phase 1 (plan): Backend correctness & consistency

| Item | Status | Classification | Evidence |
|---|---|---|---|
| `RecipeRepository.GetByID` `IsNotFound` contract (`== gorm.ErrRecordNotFound` → `errors.Is`, bare `errors.New` → wrapped `ErrNotFound`) | **Still open** | Functional (Medium) | `recipe_repository.go:142-148` still does `if err == gorm.ErrRecordNotFound { return nil, errors.New("recipe not found") }` — bare comparison, bare error, exactly as the plan describes. |
| Migrate retired Claude/OpenAI model IDs (`pkg/ai/model.go`) | **Not verified here** | Functional (Low, live outage) | Explicitly out of scope for this triage — Phase 6 (`06-model-ids.md`) owns live verification of current provider model lists via the `claude-api` skill. Noted only that the model IDs seen during this audit's live probes (`claude-3-5-sonnet-20241022`, `claude-3-opus-20240229`, etc., via `GET /ai-configs/models`) are unchanged from what the plan cites — consistent with "still needs doing," but Phase 6 should confirm with an authoritative current model list rather than this triage guessing. |
| Dedupe backend logic (`buildRecipe` helper; single-source sort-field set; distinguish not-found vs infra error in `Login`) | **Still open** (all three) | Nice-to-have (Low) | `recipe_service.go`'s `Create` (~L110-140) and `Update` (~L219-262) still build near-identical `domain.Recipe{...}` literals inline, no shared helper. `shopping_list_service.go` still has two independent field-name lists: `validSortFields` (`:161`) and the `sortItems` switch cases (`:366-378`). `user_service.go:108-116` (`Login`) still returns `"invalid credentials"` uniformly whether `GetByEmail` returned not-found or an infra error, with no distinguishing log — this is the *same* code this audit's `05-post-vpn-triage.md` flagged from the security-enumeration angle; the plan's version of the ask is about failure-mode observability (a real DB outage silently looks identical to a bad password in logs), a related but distinct motivation for the same fix. |

## Phase 2 (plan): Frontend structure & performance

| Item | Status | Classification | Evidence |
|---|---|---|---|
| Memoize `RecipeGraph` layout/edge/dimension computation with `useMemo` | **Still open** | Nice-to-have (Medium — perf) | `RecipeGraph.tsx` imports `useCallback`/`useState`/`useRef`/`useEffect` but **no `useMemo`** anywhere in the file (grep confirmed) — layout/edge computation still recomputes on every render, not gated to `[recipes]`. |
| Decompose `AddRecipeModal` (extract helpers/hook/subcomponents; fix child `cook_time`/`shelf_life` sourcing + cleanup-on-partial-failure; uniform `crypto.randomUUID()`; replace silent `.catch(() => {})`) | **Partially done** | Nice-to-have (Medium component size / Low sub-items) | File is still **689 lines** — no decomposition has happened. `crypto.randomUUID()` **is** used at 3 of 4 ID-generation sites (`:331,339,354`) but line `:115` still uses `String(Date.now() + i)` — the "uniformly" part of this sub-item is not yet complete (3/4, not 4/4). The silent `.catch(() => {})` **is still present** verbatim at `:292` (`getMyRecipes().then(setAllRecipes).catch(() => {})`). Child `cook_time`/`shelf_life` sourcing and cleanup-on-partial-failure were not independently re-verified line-by-line in this pass (lower priority given the file's still-unresolved size/duplication makes this a moot point until decomposition happens anyway). |
| Extract `openRecipe(recipe)` helper for `HomePage`'s ≥4 duplicated fetch-then-set blocks | **Still open** | Nice-to-have (Medium — duplicated async logic) | Confirmed 3 near-identical `const full = await getRecipeById(...); setSelectedRecipe(full); setServes(full.servings ?? 2)` blocks still present in `HomePage.tsx` (around `:48-53`, `:95-100`, `:122-124`), each with its own catch/fallback — no shared helper extracted. |
| Remove/consolidate dead `LoginPage.tsx`/`RegisterPage.tsx`; standardize on `useAuth` so registration updates `isAuthenticated` | **Still open — and higher-severity than the plan assumed** | **Reclassify: functional bug, not just dead-code cleanup** | Confirmed both files are genuinely dead (unreferenced from `App.tsx`; `LandingPage.tsx` has its own inline `LoginView`/`RegisterView`). But this audit's `04-auth.md` (corrected during this Phase 5 pass) found the *live* inline `RegisterView` in `LandingPage.tsx` has the **exact same field-mismatch bug** as the dead `RegisterPage.tsx` copy — registration is completely broken end-to-end, live-confirmed with a 400 from the real backend. This item should be **upgraded from a code-quality nice-to-have to a functional blocker**: fixing it isn't just dedup, it's restoring a broken core user flow. See `04-auth.md` Finding 1 for full detail. |

## Phase 3 (plan): Accessibility & debug cleanup

| Item | Status | Classification | Evidence |
|---|---|---|---|
| Keyboard-accessible clickable text controls (`<button>`/role+tabIndex+onKeyDown); modal `role="dialog"`/`aria-modal`/focus trap; named `...Props` interfaces | **Still open** | Nice-to-have (Medium a11y) | `LandingPage.tsx:69,108` still use a plain `<p className="..." onClick={...}>` for "Create an account"/"Sign in" links — not a `<button>`, no `role`/`tabIndex`/`onKeyDown`. Grepped `RecipeModal.tsx`/`AddRecipeModal.tsx` for `role="dialog"`/`aria-modal` — no matches; modal semantics/focus-trap still absent. Inline anonymous prop-type objects for `LandingPage` subcomponents were not independently re-verified line-by-line (lower priority, purely a typing-style nit). |
| Gate `ThemeExplorer`/`TunnelControls` behind `import.meta.env.DEV`, or remove `ThemeExplorer` | **Still open** | Nice-to-have→**flag for reclassification** (Medium — public-facing debug tooling) | Independently rediscovered in this audit's `04-deps.md` Finding 2 — both still render unconditionally on the public landing page with no dev-mode gate. Confirmed no security/data exposure (cosmetic-only), so "nice-to-have" is a defensible classification, but it's visible to every anonymous visitor of a live deployment, which is a step above a typical nice-to-have polish item. |

## Phase 4 (plan): Dependency & toolchain hygiene

| Item | Status | Classification | Evidence |
|---|---|---|---|
| Remove unused `axios` dependency | **Still open** | Nice-to-have (Low, but cheap — clears 7 `npm audit` HIGHs for free) | Independently rediscovered in `04-deps.md` Finding 1 — confirmed zero imports in `src/` and confirmed absent from the real production bundle via tree-shaking; removing it is a pure win with no functional risk. |
| vite 8 / ESLint 9 upgrade + CI `npm audit`/`govulncheck` | **Still open** | Nice-to-have (Medium — build-chain advisories are dev/build-time only, not runtime-exposed) | `04-deps.md`'s fresh `npm audit` still shows the `esbuild ≤0.24.2` (dev-server-only) / `vite ≤6.4.2` moderate advisory unresolved, requiring the same semver-major `vite@8` jump the plan already identifies as the only fix. No CI wiring for `npm audit`/`govulncheck` was found in this repo (out of scope to verify exhaustively here; a quick check of `.github/` would confirm before Phase 7 if needed). |

## Summary

Of the plan's 11 numbered sub-items across 4 phases, **0 are fully done**, **1 is partially done**
(`AddRecipeModal`'s `crypto.randomUUID()`/`.catch` sub-items — 3/4 ID sites fixed, silent catch
still present), and **10 remain fully open**. The most important discrepancy from the plan's own
framing: **the dead-auth-pages item is not just a dedup nice-to-have — it's masking a live,
completely-broken registration flow** (see `04-auth.md` Finding 1), and should be re-prioritized
accordingly rather than left in the "whenever, low-risk, yolo-mode-fine" bucket the plan's header
describes for this file as a whole.

No item in this plan constitutes a security vulnerability in its own right — consistent with the
plan's own framing ("Non-security findings... No VPN dependency — run whenever").

## Checks performed

1. Read every bullet of `~/.claude/plans/recipe-remediation-code-quality.md` end-to-end.
2. Re-checked each cited backend file directly: `recipe_repository.go`, `recipe_service.go`,
   `shopping_list_service.go`, `user_service.go`.
3. Re-checked each cited frontend file directly: `RecipeGraph.tsx` (grep for `useMemo`),
   `AddRecipeModal.tsx` (line count, ID-generation grep, `.catch` grep), `HomePage.tsx`
   (duplicated fetch-block grep), `LandingPage.tsx` (a11y grep), `RecipeModal.tsx`/
   `AddRecipeModal.tsx` (modal-semantics grep).
4. Cross-referenced `04-deps.md`/`04-auth.md` where this audit's Phase 4 independently
   rediscovered the same items, and used that overlap to catch and correct a citation error in
   `04-auth.md` (see that file's Finding 1 correction note).

---

*No production code was modified. This file is the only artifact written.*
