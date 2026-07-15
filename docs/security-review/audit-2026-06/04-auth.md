# Phase 4 — Fresh Frontend Auth / Token-Handling Review

**Subtask:** Re-check `authService`/`apiClient` for token storage, a single mandatory request
entry point, 401 handling, client-side route gating, and whether registration updates
`isAuthenticated`.

**Scope/method:** Read `src/api/apiClient.ts`, `src/services/authService.ts`, `src/hooks/useAuth.ts`,
`src/App.tsx`, `src/pages/{Landing,Login,Register}Page.tsx`, and `src/services/recipeService.ts`
(the only service making authenticated calls). Live-tested the registration flow's actual request
shape against the running backend.

---

## Finding 1 — The registration flow is broken end-to-end: the frontend never sends the fields the backend requires

- **Severity:** HIGH (functional correctness, not directly a security vulnerability, but it means
  the "does registration update `isAuthenticated`" question is moot — registration cannot succeed
  at all)
- **Correction (caught during this audit's Phase 5 cross-check):** the first version of this
  finding cited `src/pages/RegisterPage.tsx` as the live code path. That file is actually **dead
  code** — `App.tsx` never imports it, and `LandingPage.tsx` does not use it either. The real,
  live, user-reachable registration form is the **inline `RegisterView` function defined directly
  inside `LandingPage.tsx`** (`LandingPage.tsx:74-111`). Both implementations independently have
  the *same* bug (they were evidently forked/copy-pasted from one another), so the live-reproduced
  400 below is unaffected — but the evidence citation is corrected here to point at the code path
  that actually runs. The dead-code duplication itself is called out separately below and in
  `05-quality-triage.md` (it maps to an existing item in
  `~/.claude/plans/recipe-remediation-code-quality.md`).
- **Evidence:**
  - `src/pages/LandingPage.tsx:75,100` — the live `RegisterView` form collects a single `"Name"`
    text field (`name`), with no separate first/last name inputs.
  - `src/pages/LandingPage.tsx:86` — `handleSubmit` calls
    `registerUser(name, email, password)` → `src/services/authService.ts:29-31`, which POSTs
    `{ name, email, password }` to `/api/v1/auth/register`.
  - Backend `RegisterRequest` (`services/backend/internal/domain/user.go`, confirmed via
    `00-notes.md`'s documented schema) requires `email`, `password`, `first_name`, `last_name` —
    **all** `binding:"required"`. There is no `name` field on the backend struct at all.
  - **Live-confirmed** against the running stack:
    ```
    POST /api/v1/auth/register {"name":"Test User","email":"...","password":"..."}
    -> 400 {"error":"Key: 'RegisterRequest.FirstName' Error:Field validation for 'FirstName' failed on the 'required' tag\nKey: 'RegisterRequest.LastName' ... 'required' tag"}
    ```
  - `LandingPage.tsx`'s `RegisterView` is wired live: `App.tsx:25`
    (`onRegister={() => setScreen('home')}`) is passed down through `LandingPage`'s own
    `onRegister` prop to `RegisterView`'s `onSuccess` (`LandingPage.tsx:158`) — any real visitor
    clicking "Create an account" on the landing page hits this broken path today.
  - **Separately, dead duplicate code exists**: `src/pages/RegisterPage.tsx` and (per the
    code-quality plan) `src/pages/LoginPage.tsx` are unreferenced from `App.tsx`/`LandingPage.tsx`
    and implement the same forms a second time (with the identical bug in `RegisterPage.tsx`'s
    case) — see the "minor observation" below and `05-quality-triage.md`.
- **Why it matters:** No user can self-register through the UI at all right now — every attempt
  fails with a raw validation error (`LandingPage.tsx:88-89` catches it and shows a generic
  "Registration failed. Please try again.", so the user doesn't even see *why*). This is a
  pre-existing functional bug (independent of the VPN/security threat model) that happens to also
  answer this subtask's specific question: registration cannot update `isAuthenticated`/`screen`
  state because `registerUser()` always throws before `onSuccess()` is called
  (`LandingPage.tsx:86-87`) — the app never reaches the "swap to home screen" step by this path.
  Flagging as HIGH functional severity so it lands in the Phase 5 quality triage rather than being
  lost in a security-only framing.
- **Recommended control:** Either split the form into first/last name inputs and send
  `first_name`/`last_name`, or (simpler) have the frontend split the single `name` field on the
  first space before sending. Fix it in `LandingPage.tsx`'s live `RegisterView` (and either delete
  the dead `RegisterPage.tsx`/`LoginPage.tsx` duplicates or fix them too if they're kept/consolidated
  per the code-quality plan). Add an integration/E2E test for the register flow so this class of
  drift is caught before merge.

## Token storage & 401 handling — as previously documented, no regressions found

- **Storage:** `authService.ts` stores the JWT under `localStorage['token']`
  (`TOKEN_KEY`), consistent with the project's documented convention
  (`CLAUDE.md`: "JWT auth: token from login response → localStorage"). No XSS sink exists to
  exfiltrate it (`04-xss.md` — clean), so the localStorage-vs-httpOnly-cookie tradeoff is not
  actively exploitable today, though it remains the standard CSRF-immune-but-XSS-vulnerable
  tradeoff inherent to `localStorage` JWTs (unchanged from the prior review).
- **Single mandatory entry point:** every authenticated call in `recipeService.ts` (`createRecipe`,
  `updateRecipe`, `getMyRecipes`, `deleteRecipe`, `getRecipeById` — the only service file making
  protected calls) routes through `apiFetch` (`api/apiClient.ts`) and manually attaches
  `getAuthHeaders()`. No protected call bypasses `apiFetch` with a raw `fetch()`. `authService.ts`'s
  own `login`/`register` correctly use raw `fetch` instead (there is no token yet to attach, and no
  401-driven logout is meaningful pre-auth).
- **401 handling:** `apiClient.ts:4-11` — on any `401` response, `apiFetch` calls `logout()`
  (clearing the token) and redirects to `/`, then returns a `Promise` that never resolves so the
  caller's subsequent `.then()`/`await` code never runs with stale-auth assumptions. This is a
  reasonable pattern; not resolving the promise is intentional (the page navigation makes any
  further handling moot) rather than a bug.
- **Client-side route gating:** `App.tsx` is a two-state toggle (`'landing' | 'home'`), initialized
  from `checkAuth()` (`isAuthenticated()`, which just checks the JWT's `exp` claim client-side, as
  expected — the real enforcement is server-side per-request). There is no client-side router with
  deep-linkable protected sub-routes to bypass; the entire authenticated app surface is gated
  behind the single `screen === 'home'` branch, and every actual data call still requires a valid
  token to succeed against the backend regardless of what `screen` the client thinks it's on.

## Minor observation (not a security finding)

- `src/hooks/useAuth.ts` is used by `LandingPage.tsx`/`LoginPage.tsx`, but `App.tsx` itself
  duplicates the same login/logout/`isAuthenticated` logic directly against `authService` rather
  than using the hook, and `LoginPage.tsx` does not appear to be mounted from `App.tsx` at all
  (only `LandingPage`'s inline `LoginView` is). This looks like the "dead auth pages" duplication
  already tracked in `~/.claude/plans/recipe-remediation-code-quality.md` — leaving it for Phase 5's
  triage rather than re-litigating here.

## Checks performed

1. Read `src/api/apiClient.ts`, `src/services/authService.ts` end-to-end.
2. Read `src/hooks/useAuth.ts` and every importer (`LandingPage.tsx`, `LoginPage.tsx`) and
   `src/App.tsx` for route-gating logic.
3. Read `src/services/recipeService.ts` (the only frontend service issuing authenticated calls)
   to confirm every call attaches `getAuthHeaders()` and routes through `apiFetch`.
4. Read `src/pages/RegisterPage.tsx` and traced its field shape against the backend's
   `RegisterRequest` binding tags; then confirmed via `App.tsx`/`LandingPage.tsx` import tracing
   that this file is unreferenced dead code and re-traced the same bug in the actually-live
   `RegisterView` inline in `LandingPage.tsx:74-111`.
5. Live-tested the actual registration request shape against the running backend
   (`http://localhost:18080/api/v1/auth/register`) to confirm the mismatch is a real, reproducible
   400 rather than a hypothetical concern.

---

*No production code was modified. This file is the only artifact written.*
