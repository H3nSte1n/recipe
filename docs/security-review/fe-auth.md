# Frontend Auth Review — JWT Storage, Auth Hook, Route Protection

Scope: client-side JWT handling and session lifecycle of the Recipe React frontend — storage/read/clear in `localStorage`, the `useAuth` hook, the `apiFetch` 401 handler, and the client-side screen gating that stands in for route protection. Grounded in `services/frontend/src/services/authService.ts`, `hooks/useAuth.ts`, `api/apiClient.ts`, `App.tsx`, `pages/LandingPage.tsx`, `pages/LoginPage.tsx`, `components/HomeHeader.tsx`, `pages/HomePage.tsx`, and `services/recipeService.ts`. Read-only review. Cross-references the backend token model in `docs/security-review/be-auth.md`.

---

## Findings

### [Medium] Logout button never clears the JWT — token persists in localStorage and stays usable

**Location:**
- `services/frontend/src/App.tsx:12` (`onLogout={() => setScreen('landing')}`)
- `services/frontend/src/pages/HomePage.tsx:16,65` (`onLogout` prop forwarded)
- `services/frontend/src/components/HomeHeader.tsx:52-57` (the sign-out button: `onClick={onLogout}`)
- Unused-but-correct paths: `services/frontend/src/hooks/useAuth.ts:20-23` and `services/frontend/src/services/authService.ts:51-53`

**Description:** A correct sign-out path exists — `authService.logout()` (authService.ts:51-53) calls `localStorage.removeItem('token')`, and `useAuth.logout()` (useAuth.ts:20-23) wraps it and resets state. But the actual sign-out button in the header is wired through `HomeHeader.onLogout → HomePage.onLogout → App`'s handler, which is just `() => setScreen('landing')` (App.tsx:12). That handler only flips the React screen state; it never calls `authService.logout()` and never touches `localStorage`. The working logout code is bypassed by the UI.

**Impact:** After a user clicks "Sign out", the JWT remains in `localStorage`. The app *appears* logged out (landing screen shown), but:
- A page refresh re-runs `App`'s initializer `checkAuth() ? 'home' : 'landing'` (App.tsx:9), `isAuthenticated()` returns true, and the user is silently signed back in.
- The token is still a valid Bearer credential and works against the API.

This compounds with the backend's **[Medium] "stateless tokens are not revocable / 24h lifetime"** (`be-auth.md`): the client never clears the token *and* the server cannot revoke it, so a token a user believes they signed out of stays fully usable for up to 24 hours. On a shared or public device this is effectively a session that cannot be ended from the UI, pushing the practical risk toward High in that threat model.

**Recommendation:** Wire the logout button to the real logout. Have `App` use the `useAuth` hook (or call `authService.logout()`) in its `onLogout` handler so the token is removed from `localStorage` before switching screens. Mirror the pattern already used correctly by the 401 path in `apiClient.ts` (see positive below).

---

### [Medium] JWT stored in localStorage (readable by any JS / XSS) — conditional on the app's XSS surface

**Location:** `services/frontend/src/services/authService.ts:3` (`TOKEN_KEY = 'token'`), `:20` and `:42` (`localStorage.setItem` after login/register), `:56` (`localStorage.getItem`).

**Description:** The access token is persisted in `localStorage` under the key `token`. `localStorage` is readable by any JavaScript executing in the origin, so any successful XSS (or a malicious/compromised third-party script) can exfiltrate the token verbatim. This is the well-known, industry-standard tradeoff of localStorage-vs-cookie token storage — chosen here for the simple Bearer-header flow (CLAUDE.md documents `localStorage` → `Authorization: Bearer` as the intended design).

Grounding the severity in the actual XSS surface (a sibling review covers XSS in depth): a grep of `services/frontend/src/` found **no** `dangerouslySetInnerHTML`, `innerHTML`, `eval`, or `document.write` sinks. Recipe content is rendered through React's default JSX escaping. So the *currently observed* in-app XSS surface is low, which keeps this at Medium rather than High. The rating is conditional: if the sibling XSS review finds an injection sink (e.g. rendering imported/AI-parsed recipe HTML unescaped), the impact of localStorage storage rises accordingly.

**Impact:** If any XSS is introduced, the token is trivially stolen; combined with the non-revocable 24h backend token (`be-auth.md`), a stolen token grants full account access for up to 24h with no way to invalidate it. With no observed XSS sink today, this is a latent design risk rather than an active exploit.

**Recommendation:** Treat this as an accepted-but-documented tradeoff. The higher-assurance option is to move the token to an `HttpOnly`, `Secure`, `SameSite` cookie issued by the backend (removes JS read access, but then requires CSRF protection — see positive). At minimum: keep the XSS surface at zero (no `dangerouslySetInnerHTML`, sanitize any future HTML rendering), add a strict Content-Security-Policy to limit script injection and exfiltration, and shorten token lifetime / add revocation server-side. Defer final severity to the XSS sibling review.

---

### [Low] Auth gating is client-side only and based on an unverified `exp` claim; a forged/garbage token briefly renders protected chrome

**Location:**
- `services/frontend/src/App.tsx:9` (`useState<Screen>(checkAuth() ? 'home' : 'landing')`)
- `services/frontend/src/services/authService.ts:59-77` (`isTokenExpired` / `isAuthenticated` decode the JWT payload with `atob` + `JSON.parse`, read only `exp`)

**Description:** There is no router or route guard; "route protection" is a two-value screen state machine in `App`. The decision to show `HomePage` is made purely client-side by `isAuthenticated()`, which base64-decodes the JWT payload and checks `exp`. It does **not** (and cannot) verify the signature. Any string shaped like a JWT with a future `exp` (e.g. an attacker who hand-crafts `header.{"exp":9999999999}.sig` and writes it to `localStorage`, or a corrupted token) passes `isAuthenticated()` and causes the `HomePage` shell to render.

**Impact:** Low and primarily UX/info-disclosure, not a real authorization bypass — the backend independently enforces JWT signature verification on every protected route (`be-auth.md` positives), so no protected *data* is returned for an invalid token. The first API call (`useRecipes` → `getMyRecipes`) goes through `apiFetch`, receives 401, and is redirected to `/` (see positive). The net effect is a brief flash of empty home chrome (header, empty grid) before the redirect. No recipe data leaks because all data is API-gated.

**Recommendation:** Accept as low-risk given backend enforcement. If tightening: gate the initial render behind a lightweight `/api/v1/users/me`-style check, or render a loading state until the first authenticated fetch resolves, so unverified tokens never paint protected chrome.

---

### [Low] 401-driven session expiry handling lives only in `apiFetch`; raw-`fetch` data services would leak protected UI

**Location:** `services/frontend/src/api/apiClient.ts:3-11` (the only place 401 → logout+redirect lives); used only by `services/frontend/src/services/recipeService.ts`. Public auth calls in `authService.ts:9,31` correctly use raw `fetch` (no token, no 401 handling needed).

**Description:** Expired/invalid-token detection is centralized in `apiFetch`, which on HTTP 401 calls `logout()` and redirects to `/`. Today the only authenticated data service (`recipeService`) routes through `apiFetch`, so this works. But the pattern is opt-in: a future authenticated service that calls `fetch` directly (the codebase already shows that mixed pattern — `authService` uses raw `fetch`) would not trigger the 401 redirect, leaving the user on a broken protected screen with a dead token.

**Impact:** Low / forward-looking. No current defect — every protected call uses `apiFetch`. Risk is regression-by-omission as new services are added.

**Recommendation:** Make `apiFetch` the single mandatory entry point for all authenticated requests (lint rule or a shared client), or centralize auth + 401 handling in one wrapper so new services cannot bypass it.

---

### [Info — positive] The 401 path clears the token and redirects — the correct pattern the logout button should reuse

**Location:** `services/frontend/src/api/apiClient.ts:5-7` (`if (response.status === 401) { logout(); window.location.href = '/'; ... }`).

**Description:** On an API 401, `apiFetch` calls `authService.logout()` (which removes the token from `localStorage`) and hard-navigates to `/`, then returns a never-resolving promise so downstream code does not run against an unauthenticated state. This is the correct, complete session-teardown pattern — and it stands in direct contrast to the broken manual logout button (Medium finding above), which should be wired the same way.

---

### [Info — positive] Bearer-token (header) auth, not cookies — classic CSRF is structurally mitigated

**Location:** `services/frontend/src/services/authService.ts:79-85` (`getAuthHeaders` → `Authorization: Bearer <token>`); applied in `recipeService.ts` on every call.

**Description:** The token is attached manually as an `Authorization` header read from `localStorage`; it is never stored in a cookie and never auto-attached by the browser. Cross-site requests therefore cannot ride an ambient credential, so classic cookie-based CSRF does not apply to the authenticated API. (This is the flip side of the localStorage tradeoff: choosing header auth buys CSRF immunity at the cost of XSS exposure.) No `withCredentials`/cookie usage observed.

---

### [Info — positive] Token is never logged, never placed in the DOM, and never put in a URL

**Location:** whole of `services/frontend/src/` — `grep` for `console.*` returned no token logging; the token appears only in `localStorage` (authService.ts) and the `Authorization` header (authService.ts:84). Login/register submit credentials via POST body (`LandingPage.tsx` / `LoginPage.tsx` forms), not query strings.

**Description:** No occurrences of the token being written to `console`, interpolated into markup/JSX, or appended to a URL/query string. This avoids the common leakage vectors (browser history, server access logs, `Referer` header, console/error-reporting capture). Password inputs use `type="password"` with appropriate `autoComplete` values.

---

### [Info — positive] No XSS sinks observed in the frontend (supports the localStorage tradeoff being tolerable today)

**Location:** `services/frontend/src/` — `grep` for `dangerouslySetInnerHTML`, `innerHTML`, `eval(`, `document.write` returned no matches.

**Description:** All rendering goes through React's default JSX escaping; no raw-HTML injection sink was found, including for imported/AI-parsed recipe content. This is what keeps the localStorage-JWT exposure (Medium) from being actively exploitable in the current codebase. Final XSS assessment is deferred to the sibling XSS review; if a sink is later introduced, revisit the localStorage severity.

---

## Summary

Critical: 0 · High: 0 · Medium: 2 · Low: 2 · Info: 4 (all positive)

Most serious frontend-auth finding: **[Medium] Logout button never clears the JWT** — the header sign-out button (`HomeHeader.tsx:52-57` → `App.tsx:12`) only flips screen state and bypasses the working `authService.logout()` / `useAuth.logout()`, so the token persists in `localStorage`, survives a refresh (silent re-login), and remains a valid credential. Because the backend cannot revoke it for up to 24h (`be-auth.md`), a user cannot actually end their session from the UI — a real, easily-fixed defect that tips toward High on shared devices. The localStorage-JWT storage tradeoff (Medium, conditional on the XSS surface — currently no sinks observed) is the secondary concern.
