# Frontend XSS Sinks & API Client Review

Scope: XSS sinks (`dangerouslySetInnerHTML`, `innerHTML`, `eval`, DOM injection, untrusted URL/HTML rendering) and the HTTP client configuration (base path, token attachment, 401 handling, error surfacing) across `services/frontend/src/`. Read-only review grounded in real code.

---

## Findings

### [Info] No `dangerouslySetInnerHTML` / `innerHTML` / `eval` / `document.write` anywhere — React auto-escaping is the primary XSS defense (positive)

**Location:** Entire `services/frontend/src/` tree (24 `.ts`/`.tsx` files).

**Description:** A full grep for `dangerouslySetInnerHTML`, `innerHTML`, `outerHTML`, `insertAdjacentHTML`, `eval(`, and `document.write` returns **zero matches**. No HTML-string rendering, no markdown/HTML rendering library (no `react-markdown`, `marked`, `dompurify` in `package.json`). All recipe text fields (`title`, `description`, `notes`, `instruction`, ingredient `name`/`notes`) — which originate from untrusted sources (AI parsing, URL/PDF import) — are rendered as plain JSX children (e.g. `RecipeModal.tsx`, `RecipeCard.tsx`), so React's built-in contextual escaping applies. This is the correct posture and neutralizes the stored-XSS risk from AI/URL-parsed content.

**Impact:** None — this is a positive control.

**Recommendation:** Keep it this way. If a markdown/rich-text renderer is ever introduced for recipe content, route it through DOMPurify and add an ESLint rule (`react/no-danger`) to block `dangerouslySetInnerHTML` regressions.

---

### [Low] Untrusted `image_url` rendered directly into `<img src>` with no scheme validation

**Location:**
- `components/RecipeCard.tsx:28` — `<img className="recipe-card__image" src={recipe.image_url} alt={recipe.title} />`
- `components/RecipeModal.tsx:207` — `<img src={recipe.image_url} alt={recipe.title} />`
- `components/RecipeGraph.tsx:200` — `<img className="recipe-graph__node-img" src={n.recipe.image_url} alt="" />`
- `components/AddRecipeModal.tsx:556` — `<img src={imagePreview} alt="preview" />` (where `imagePreview` is seeded from `initialRecipe?.image_url` at `:257`)

**Description:** `recipe.image_url` (`types/recipe.ts:85`) is server data that can be populated from recipe import / URL parsing, i.e. partially attacker-influenced. It is bound directly to `<img src>` with no allowlist or scheme check. React does **not** sanitize `src` attribute values. A `javascript:` URI in an `<img src>` does **not** execute script in modern browsers (img is not a navigable context), so this is **not** a direct XSS vector, and no `onError`/`onLoad` handler is attached that could be abused. The residual risk is that an arbitrary attacker-chosen URL causes the victim's browser to issue an outbound request on render (referrer leakage, IP/tracking-pixel disclosure, loading of attacker-controlled content).

**Impact:** Low — no script execution; limited to forced outbound resource loads / privacy leakage from rendered recipe images.

**Recommendation:** Validate `image_url` against an `http:`/`https:` (or relative `/`) scheme allowlist before binding to `src`; reject/blank otherwise. Consider a Content-Security-Policy `img-src` directive to bound where images may load from.

---

### [Medium] JWT stored in `localStorage` — fully exfiltratable by any future XSS

**Location:** `services/authService.ts:20,42` (`localStorage.setItem('token', …)`), `:56` (`getToken`), `:84` (`Authorization: Bearer ${token}`).

**Description:** The session JWT is persisted in `localStorage` and read back to build the `Authorization` header. `localStorage` is readable by any JavaScript running on the origin. Today there is no XSS sink (see first finding), so this is latent — but it raises the blast radius of any future XSS: a single injected script could read the token and exfiltrate a full bearer credential. There is no httpOnly-cookie alternative in use.

**Impact:** Medium (latent) — converts any future DOM/stored XSS into full account-takeover via token theft, with no server-side revocation path visible on the frontend.

**Recommendation:** Prefer an httpOnly, `Secure`, `SameSite` cookie for the session token so it is not script-readable; if `localStorage` must stay, treat XSS prevention as critical (CSP, the ESLint `no-danger` rule above) and keep token lifetimes short.

---

### [Info] API client base path adheres to the relative `/api/v1` convention — no hardcoded backend URL (positive)

**Location:** `api/apiClient.ts:3-11`, `services/authService.ts:9,31`, `services/recipeService.ts:12,18,47,53,77,101,119`.

**Description:** There is no axios instance and no `baseURL`/`withCredentials` configuration — the app uses the native `fetch` API directly. Every request targets a **relative** path (`/api/v1/...`); no absolute backend host (`http://app:8080`, `http://localhost:8080`, etc.) is hardcoded anywhere in `src/`. This matches the project convention (relative paths proxied by Vite) and, importantly, means same-origin is enforced by construction.

**Impact:** None — positive control.

**Recommendation:** Maintain the relative-path convention; add a lint/grep guard against absolute backend URLs if desired.

---

### [Low] Bearer token is attached per-call, not by a central interceptor — safe today only because all URLs are relative

**Location:** `services/authService.ts:79-85` (`getAuthHeaders`), spread into each request in `services/recipeService.ts` (e.g. `:14,21-23,49,55-57,79-82,103-106,121-124`). `api/apiClient.ts` (`apiFetch`) does **not** attach the token itself.

**Description:** There is no shared request interceptor; instead each service call manually spreads `...getAuthHeaders()` into the `fetch` headers. Because every call passes a relative same-origin path, the `Authorization: Bearer` header is **never** sent cross-origin today — so there is no token-leakage-to-third-party issue. The weakness is structural: nothing centrally guarantees the token is only attached to first-party requests. If a future call ever passes an absolute/cross-origin URL (e.g. an image-proxy or external import endpoint) and reuses `getAuthHeaders()`, the bearer token would silently leak to that third party. `getAuthHeaders` performs no origin check.

**Impact:** Low (latent) — no current leakage; risk is a future regression sending the token off-origin.

**Recommendation:** Centralize auth-header attachment in `apiFetch` and have it attach the token **only** for relative / same-origin request URLs (reject or strip `Authorization` for absolute cross-origin targets).

---

### [Info] 401 response handling is centralized and does not surface raw backend bodies (positive, with a minor note)

**Location:** `api/apiClient.ts:4-10`.

**Description:** `apiFetch` inspects `response.status === 401`, calls `logout()` (clears the `localStorage` token) and redirects to `/`, then returns `new Promise<Response>(() => {})` — a deliberately never-resolving promise to halt the caller during navigation. It does not read or expose the response body. Minor note: the never-resolving promise leaves the caller's `await` permanently pending; this is harmless given the immediate `window.location.href` redirect, but it is a non-obvious pattern. Auth endpoints (`authService.ts` login/register) use raw `fetch` and are not routed through `apiFetch`, which is acceptable since they are pre-auth.

**Impact:** None — positive control; minor maintainability note only.

**Recommendation:** Optionally document the never-resolving-promise pattern inline, or reject with a typed "redirecting" error instead.

---

### [Info] UI error messages are hardcoded/derived from status — no raw backend response body reflected to the DOM (positive)

**Location:** `pages/LoginPage.tsx:26`, `pages/LandingPage.tsx:51,89`, `pages/RegisterPage.tsx:26`, `pages/HomePage.tsx:73`, `hooks/useRecipes.ts:32`, error strings constructed in `services/recipeService.ts`/`authService.ts`.

**Description:** Errors shown to users are either fixed friendly strings ("Invalid email or password. Please try again.", "Failed to load recipes.") or `Error.message` values assembled from `response.status`/`response.statusText` only — never the raw JSON/HTML body of the backend response. And even where `err.message` is placed in state (`useRecipes.ts:32`), it is rendered as escaped JSX text (`HomePage.tsx:73` actually shows a hardcoded string). No sensitive backend detail or unescaped markup reaches the DOM.

**Impact:** None — positive control.

**Recommendation:** Continue avoiding reflection of raw response bodies into the UI.

---

### [Low] `axios` is a declared dependency but is never imported — the dependency-audit advisories likely do not ship in the bundle

**Location:** `services/frontend/package.json:14` (`"axios": "^1.7.5"`, installed `1.13.5`); no `import … from 'axios'` exists anywhere in `src/`.

**Description:** The dependency audit (`docs/security-review/00-dependency-audit.md`) flags axios as HIGH and states "axios is the frontend's actual HTTP client to the backend." That premise is **incorrect for the current code** — the app uses the native `fetch` API exclusively (`api/apiClient.ts`, `services/*.ts`), and axios is never imported. Because Vite/Rollup tree-shakes unused modules, the axios code (and its `form-data` / `follow-redirects` transitive advisories) is almost certainly **not** included in the production bundle, so the runtime exposure of those CVEs is effectively nil today. It remains a supply-chain liability in `node_modules`/lockfile (build-time tooling, future accidental import).

**Impact:** Low — overstated as runtime-exposed in the dependency audit; real exposure is dead-dependency/supply-chain only.

**Recommendation:** Remove `axios` from `package.json` (the app does not need it). This eliminates every axios/`form-data`/`follow-redirects` advisory outright and is non-breaking. If axios is intentionally planned, upgrade to ≥1.16.0 per the dependency audit and route it through the relative-path + same-origin-token rules above.

---

## Summary

| Severity | Count |
|----------|-------|
| Critical | 0 |
| High     | 0 |
| Medium   | 1 |
| Low      | 3 |
| Info     | 4 |
| **Total**| **8** |

- **No `dangerouslySetInnerHTML` / `innerHTML` / `eval` / `document.write` / HTML-string rendering exists** — React auto-escaping is correctly relied upon for all untrusted recipe content; no markdown/HTML renderer is present. The only untrusted-data DOM bindings are `image_url` → `<img src>` (Low; not script-executing in modern browsers).
- **API client:** native `fetch` (no axios used at runtime); all requests use the relative `/api/v1` path (convention adhered, no hardcoded backend URL); centralized 401 logout/redirect; UI errors are friendly strings, no raw backend bodies reflected.
- **Top risks:** JWT in `localStorage` (Medium, latent — amplifies any future XSS into token theft); per-call (non-central) Bearer-token attachment that is safe only because URLs are relative (Low); unvalidated `image_url` in `<img src>` (Low); dead `axios` dependency carrying advisories that almost certainly do not reach the bundle (Low, corrects the dependency audit's "actual HTTP client" premise).
