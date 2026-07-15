# Phase 4 — Fresh Frontend Data-Exposure Review

**Subtask:** Check for secrets/keys in the built bundle, `console.log` of sensitive data, image
referrer leakage, and PII over-fetched into the client.

**Scope/method:** Ran a real production build (`npm run build`) and grepped the emitted `dist/`
bundle for API-key-shaped strings and source maps. Grepped `src/` for `console.*` calls. Traced
`recipe.image_url`'s origin to check for third-party hotlinking / referrer leakage. Read the
frontend's TypeScript response types for over-fetched PII.

---

## Result: clean across all four checks

- **No secrets/keys in the built bundle.** Ran `npm run build` (real production build, not a
  simulation) and grepped the emitted `dist/assets/*.js` and `dist/index.html` for
  API-key-shaped patterns (`sk-...`, `AIza...`, `api_key: "..."`) — zero matches.
  `vite.config.ts:17` sets `sourcemap: false`, confirmed no `.map` files are emitted
  (`dist/assets/` contains exactly one CSS + one JS bundle, no source maps to leak original
  source/comments).
- **No `console.log`/`console.debug`/`console.warn`/`console.error` anywhere in `src/`** (grep
  confirmed, zero matches) — no risk of sensitive request/response data being dumped to the
  browser console.
- **`VITE_`-prefixed env vars remain non-sensitive.** Only `VITE_API_URL` and `VITE_ENV`
  are declared (`src/types/vite-env.d.ts:7-8`, `.env.example`); neither is actually referenced
  anywhere else in `src/` (grep found no runtime usage beyond the type declaration), and neither
  carries a secret. Confirms the prior report's "by design" verdict still holds — no new
  `VITE_`-prefixed secret has been introduced since.
- **No image-based referrer leakage.** `recipe.image_url` (rendered via `<img src>` in
  `RecipeCard.tsx`, `RecipeModal.tsx`, `RecipeGraph.tsx`) is only ever populated by the app's own
  upload flow (backend-issued signed `/uploads/...` URL) — the AI recipe-parsing response
  (`pkg/ai/parser.go` on the backend) does not populate an image field from URL/PDF imports, so
  there is no code path where an *external* third-party image URL is hotlinked from a rendered
  recipe. No `<img>`/`<a>` tag points at an attacker- or third-party-controlled origin, so there is
  no vector for leaking the current page URL (or any token) via the `Referer` header — moot, since
  the JWT is never placed in the URL in the first place (confirmed in `04-auth.md`).
- **No PII over-fetched into the client.** `src/types/recipe.ts` (the only domain type consumed by
  the frontend beyond auth) carries recipe fields only — no embedded user object, email, or other
  member PII. The frontend has no shopping-list, AI-config, or user-list UI at all (only
  login/register/recipes exist in `src/pages/`), so the `/users/list` PII-exposure surface flagged
  in `03-vpn-deps.md` Finding 5 is a backend-only concern with no frontend code path consuming it.

## Checks performed

1. Ran `npm run build` and inspected the real `dist/` output (not `dev` mode) for secrets and
   source maps.
2. Grepped `src/` for `console.log|debug|info|warn|error`.
3. Grepped `src/` and `vite-env.d.ts` for every `VITE_`/`import.meta.env` reference.
4. Traced `image_url`'s origin from backend (`pkg/storage`, `pkg/ai/parser.go`) to frontend
   rendering to rule out external hotlinking.
5. Read `src/types/recipe.ts` for any embedded PII beyond `user_id`.

---

*No production code was modified. This file is the only artifact written.*
