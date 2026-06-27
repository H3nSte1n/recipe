# Frontend Security Review — Client-Side Data Exposure

**Scope:** Client-side secret leakage, sensitive data in logs/storage, and unvalidated/external content in the React 18 + TypeScript + Vite frontend (`services/frontend/`). Covers secrets in the client bundle, `console.*` logging, `localStorage`/`sessionStorage`/cookies (PII beyond the JWT), remote image/link rendering, referrer leakage, and the Vite build/proxy config.

**Coordination with sibling tasks:** JWT storage and its decoding/expiry handling are owned by **fe-auth** and are not re-assessed here. DOM-injection / `dangerouslySetInnerHTML` / API-client XSS are owned by **fe-xss-apiclient**. This document focuses only on secret/data leakage and external-content handling.

---

## Findings

### [Low] Remote `<img>` rendered without `referrerPolicy` / `loading` attributes

**Location:**
- `services/frontend/src/components/RecipeCard.tsx:28` — `<img className="recipe-card__image" src={recipe.image_url} alt={recipe.title} />`
- `services/frontend/src/components/RecipeModal.tsx:207` — `<img src={recipe.image_url} alt={recipe.title} />`
- `services/frontend/src/components/RecipeGraph.tsx:200` — `<img className="recipe-graph__node-img" src={n.recipe.image_url} alt="" />`

**Description:** Recipe images are rendered directly from the backend-supplied `recipe.image_url` string with no `referrerPolicy` and no scheme validation. `image_url` is a free-form `varchar(255)` field on the backend (`services/backend/internal/domain/recipe.go:15`). For images uploaded through the app it points at the project's own file/object store (`UploadFile` returns e.g. `https://storage/...`, see `services/backend/internal/service/recipe_service.go:103-121`), but the field is not constrained to the app's own origin, and recipes can be marked non-private (`is_private`) and surfaced to other users (notably the shared graph view in `RecipeGraph.tsx`). If `image_url` ever holds a third-party/attacker-controlled URL (e.g. via a future URL-import change, or a crafted value), every viewer's browser issues a GET to that host on render.

**Impact:** With the browser default referrer policy (`strict-origin-when-cross-origin`), the app's origin is sent as the `Referer` to whatever host serves the image, and the viewer's IP/User-Agent reach that host. For an attacker-controlled `image_url` on a shared recipe this becomes a passive tracking/IP-logging beacon that fires for anyone who opens or browses the recipe — i.e. it can reveal who viewed a recipe and when. For own-origin/storage images the privacy impact is negligible. Because the external-URL condition is possible but not currently confirmed in code, this is rated Low (defense-in-depth).

**Recommendation:** Add `referrerPolicy="no-referrer"` (and `loading="lazy"`) to these `<img>` elements so no `Referer` is leaked regardless of the host. Note: scheme validation of `src` is **not** needed here — `javascript:` URLs are not executed by browsers for `<img src>` loading, so `referrerPolicy` is the meaningful control. Optionally, normalize imported images to the app's own storage on the backend so `image_url` is always same-infra.

---

### [Low] Landing page preloads hardcoded third-party Unsplash images (referrer + tracking)

**Location:** `services/frontend/src/components/ScatteredBackground.tsx:5-17` (12 hardcoded `https://images.unsplash.com/...` URLs), preloaded at `:135-136` (`const img = new Image(); img.src = url;`) and applied as element backgrounds via `applyCardBackground` (`:170`).

**Description:** The decorative animated background on the (unauthenticated) landing page loads a dozen images directly from `images.unsplash.com`. These are developer-hardcoded, not user content, but they are still third-party requests made from the app origin.

**Impact:** On every landing-page visit the browser contacts Unsplash, sending the app origin as `Referer` and exposing the visitor's IP to a third party before any login. This is a minor privacy/tracking and availability (third-party dependency) concern, not a secret leak.

**Recommendation:** Self-host the background images (or bundle them as static assets under `public/`/`src/assets`). If they must remain remote, add `referrerPolicy="no-referrer"`. Severity is Low because the URLs are static and developer-controlled.

---

### [Info] `VITE_`-prefixed env vars are inlined into the client bundle by design

**Location:** `services/frontend/src/types/vite-env.d.ts:7-8` (`VITE_API_URL`, `VITE_ENV`), `services/frontend/.env.example`.

**Description:** Vite inlines any `import.meta.env.VITE_*` value into the shipped JS bundle. The only two declared vars (`VITE_API_URL`, `VITE_ENV`) are non-sensitive, and `VITE_API_URL` is not even referenced anywhere in `src/` (the app uses relative `/api/v1/...` paths through the Vite proxy). This is a forward-looking note, not a current defect.

**Impact:** None today. The risk is future: if anyone ever moves a secret (e.g. the per-user AI provider keys the backend manages) into a `VITE_`-prefixed variable, it would be baked into the public bundle and trivially extractable.

**Recommendation:** Keep all secrets server-side. Never introduce a `VITE_`-prefixed variable for an API key, token, or password. AI provider keys must continue to live only on the backend (per-user, see `ai_config_service.go`) and never be sent to or held by the client.

---

## Positives

- **No secrets in the client bundle.** Grep of `src/` and config for `import.meta.env`, `process.env`, `VITE_`, and `api_key|secret|password|token|bearer|apikey` found no hardcoded API keys, tokens, or passwords. The only env vars are `VITE_API_URL` / `VITE_ENV` (both non-sensitive), and `VITE_API_URL` isn't referenced in source.
- **No AI provider keys on the client.** The backend stores per-user AI keys; grep of `src/` for `ai[_-]?config|ai[_-]?key|openai|anthropic|gemini|provider` returned nothing. The frontend never requests, holds, or renders AI keys.
- **No `console.*` logging anywhere in `src/`.** No tokens, user data, full API responses, or secrets are logged to the console (grep for `console.` → empty).
- **No extra PII in storage.** The only thing written to `localStorage` is the JWT under key `token` (`authService.ts:20,42,52,56`) — owned by fe-auth. No email, name, or other personal data is persisted; no `sessionStorage` or `document.cookie` usage. Email/password live only in transient component state on the auth pages.
- **No tabnabbing surface.** There are no `target="_blank"` links, external `<a href>` to user/third-party sites, or `window.open` calls anywhere in `src/`, so `rel="noopener noreferrer"` is not applicable — there is simply no external-link vector to exploit.
- **No source maps in production.** `vite.config.ts:18` sets `build.sourcemap: false`, so internal source is not shipped in prod builds.
- **Proxy config is benign.** `vite.config.ts` proxies `/api` to `http://app:8080` (dev/Docker only); no secrets or credentials in the proxy config, and the app correctly uses relative API paths.

---

## Summary

| Severity | Count |
|----------|-------|
| Critical | 0 |
| High     | 0 |
| Medium   | 0 |
| Low      | 2 |
| Info     | 1 |

**No secret leaks to the client** — no API keys, tokens, passwords, or AI provider keys are present in the bundle, logged to console, or persisted in storage. The most serious data-exposure issue is the remote `<img src={recipe.image_url}>` rendering without `referrerPolicy` (Low): if `image_url` ever holds a third-party/attacker-controlled URL on a shared recipe, it can act as a viewer IP/referrer tracking beacon. Both Low findings are remediated by `referrerPolicy="no-referrer"` and/or self-hosting images.
