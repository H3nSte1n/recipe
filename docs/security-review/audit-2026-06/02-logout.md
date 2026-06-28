# Audit 02 — Logout clears JWT / session-end correctness (PR #38)

Date: 2026-06-28
Scope: Verify that logout clears the JWT before switching screens, so a page
refresh after logout does not silently re-authenticate from a stored token.

## Check 1 — Code read (frontend session/auth path)

Files reviewed under `services/frontend/src`:

### Logout handler — `services/frontend/src/App.tsx:11-16`
```ts
const handleLogout = (): void => {
  // Clear the JWT from localStorage before switching screens, otherwise a page
  // refresh would silently re-authenticate from the still-stored token.
  clearSession();        // -> authService.logout()
  setScreen('landing');
};
```
- `clearSession` is the imported alias for `authService.logout` (`App.tsx:2`:
  `import { isAuthenticated as checkAuth, logout as clearSession } from './services/authService';`).
- Token clearing (`clearSession()`) runs BEFORE the screen switch (`setScreen('landing')`).
  Order is correct.

### `authService.logout()` — `services/frontend/src/services/authService.ts:51-53`
```ts
export function logout(): void {
  localStorage.removeItem(TOKEN_KEY);   // TOKEN_KEY = 'token' (line 3)
}
```
- Confirmed: performs `localStorage.removeItem('token')`.

### `isAuthenticated()` — `services/frontend/src/services/authService.ts:75-77`
```ts
export function isAuthenticated(): boolean {
  return !isTokenExpired();   // isTokenExpired() returns true when getToken() is null
}
```
- `isTokenExpired()` (lines 59-73) returns `true` when no token is present, so after
  logout removes the token, `isAuthenticated()` returns `false`.

### Initial screen / refresh behavior — `services/frontend/src/App.tsx:9`
```ts
const [screen, setScreen] = useState<Screen>(checkAuth() ? 'home' : 'landing');
```
- On mount (i.e. a page refresh), the initial screen is derived from `checkAuth()`
  (= `isAuthenticated()`). Since the token was removed at logout, a refresh evaluates
  to `landing`, so it will NOT silently re-authenticate. This closes the documented gap.

### Supporting wiring (not strictly required, consistent)
- `services/frontend/src/pages/HomePage.tsx:13,16,65` — `HomePage` receives `onLogout`
  and wires it to its logout control; `App.tsx:19` passes `handleLogout`.
- `services/frontend/src/hooks/useAuth.ts:20-23` — alternate `useAuth().logout()` also
  calls `authService.logout()` first, then `setIsAuthenticated(false)` (same correct order).
  Note: `App.tsx` does not use this hook; it uses the direct `handleLogout` path above.

Code-read conclusion: the logout flow routes through the token-clearing path
(`localStorage.removeItem('token')`) BEFORE switching screens, and the mount-time
screen selection depends on token presence, so a post-logout refresh lands on landing.

## Check 2 — Live browser (Playwright)

- Throwaway user registered via API (POST http://localhost:18080/api/v1/auth/register):
  - email: `audit-33700144-8546-4593-bd43-db9930211ed7@example.com`
  - password: `Audit-Passw0rd!2026`
  - Response: HTTP 200/201 with user id `fa049860-be4b-490f-8d88-6b5f8aafc79f`
    (registration endpoint requires `first_name` / `last_name` fields).
- Browser automation could NOT be executed: `mcp__playwright__browser_navigate` failed with
  `Chromium distribution 'chrome' is not found at /Applications/Google Chrome.app/...`
  (Playwright Chrome channel not installed in this environment).
- Steps 2c-2f (login via UI, assert token present, click logout, assert token null,
  reload and assert no silent re-auth) were therefore NOT performed live.
- Frontend (http://localhost:5173 -> HTTP 200) and backend (http://localhost:18080) are up;
  only the browser binary is missing.

## Verdicts

- Check 1 (code-read: logout clears JWT before screen switch; refresh does not re-auth):
  PASS — `App.tsx:11-16` calls `clearSession()` (`authService.logout()`,
  `authService.ts:51-53` -> `localStorage.removeItem('token')`) BEFORE `setScreen('landing')`;
  mount screen derived from `checkAuth()`/`isAuthenticated()` (`App.tsx:9`,
  `authService.ts:75-77`), so a post-logout refresh lands on landing.
- Check 2 (live browser end-to-end):
  INCONCLUSIVE — Playwright Chrome binary unavailable
  ("Chromium distribution 'chrome' is not found"); UI login/logout/reload not exercised.
  Throwaway creds were created successfully; verification relies on the code-read above.
