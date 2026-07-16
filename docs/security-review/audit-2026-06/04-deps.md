# Phase 4 — Fresh Frontend Dependency / Build Review

**Subtask:** Run `npm audit`, confirm the unused `axios` dep + vite/eslint advisory state, and
confirm debug tooling (`ThemeExplorer`/`TunnelControls`) does not ship in a production build.

**Scope/method:** Ran `npm audit` fresh against the current lockfile. Confirmed whether `axios` is
actually imported anywhere in `src/` and whether it appears in the real production bundle (built
in `04-data.md`). Read `LandingPage.tsx` for how `ThemeExplorer`/`TunnelControls` are mounted and
whether that's gated by a dev-only check.

---

## `npm audit`: 14 advisories (7 high, 6 moderate, 1 low, 0 critical) — same shape as prior review, all dev/build-time

```
7 high, 6 moderate, 1 low (0 critical)
```

| Package | Severity | Advisory theme | Runtime-exposed? |
|---|---|---|---|
| `axios` 1.0.0–1.15.2 | HIGH ×7 (SSRF, prototype pollution, header injection, etc.) | see below | **No — confirmed unused, not in bundle** |
| `flatted` ≤3.4.1 | HIGH ×2 (unbounded recursion DoS, prototype pollution) | transitive (build tooling) | No — build-time only |
| `form-data` 4.0.0–4.0.5 | HIGH (CRLF injection) | transitive (build tooling) | No — build-time only |
| `esbuild` ≤0.24.2 | MODERATE | dev-server can be probed by any website (dev only) | No — dev-server only, not prod |
| `ajv` <6.14.0 | MODERATE (ReDoS) | transitive (eslint config) | No — lint-time only |
| `brace-expansion` (various) | MODERATE (ReDoS) | transitive (eslint/glob) | No — build/lint-time only |
| `follow-redirects` ≤1.15.11 | MODERATE (header leak on redirect) | transitive of `axios` | No — `axios` unused |
| `@babel/core` ≤7.29.0 | LOW (arbitrary file read via sourcemap comment, build-time only) | transitive (build tooling) | No — build-time only |

- The only offered fix for the `esbuild`/`vite` chain is `npm audit fix --force`, which installs
  `vite@8` — a semver-major breaking change (same as the prior review's characterization; not
  applied here since this audit makes no code changes).

## Finding 1 (re-confirmed) — `axios` is a listed production dependency but is completely unused; not present in the shipped bundle

- **Severity:** LOW (supply-chain hygiene, not a runtime vulnerability)
- **Evidence:**
  - `package.json:14` — `"axios": "^1.7.5"` is listed under `"dependencies"` (not
    `devDependencies`).
  - Grep for `from 'axios'`/`require('axios')`/`import axios` across `src/` returns **zero**
    matches — nothing in the app imports it.
  - **Confirmed via the real production build** (`04-data.md`'s `npm run build` output): grepping
    the emitted `dist/assets/*.js` for the literal string `axios` returns zero matches — Vite/
    Rollup's tree-shaking excludes it entirely since nothing references it. All 7 HIGH `axios`
    advisories above (SSRF via `NO_PROXY` bypass, prototype pollution, header/credential leaks,
    etc.) are therefore **not reachable in the shipped application** — they would only matter if a
    build script, test, or future code change actually imports and calls the library.
- **Recommended control:** Remove `axios` from `package.json` entirely (the app already has its own
  `fetch`-based `apiClient.ts`) — this eliminates 7 of the 14 `npm audit` findings outright with no
  functional impact, and removes the risk of someone importing a known-vulnerable version later
  without noticing.

## Finding 2 — Debug/exploration UI ships unconditionally to the public, unauthenticated landing page

- **Severity:** LOW (no data or auth exposure — cosmetic/UX debug tooling, not a security hole)
- **Evidence:**
  - `src/pages/LandingPage.tsx:5-6` —
    `import TunnelControls from '../components/TunnelControls'` and
    `import ThemeExplorer from '../components/ThemeExplorer'; // TEMPORARY`, both rendered
    unconditionally at `:148-149` (`<TunnelControls .../>`, `<ThemeExplorer />`) with **no**
    `import.meta.env.DEV` (or any other) gate.
  - `ThemeExplorer.tsx:1-2` carries its own comment: `"TEMPORARY — color exploration tool for the
    landing page. To remove: delete this file, ..."` — the author's own intent was for this not to
    persist to production.
  - Both components are **purely cosmetic**: `ThemeExplorer` is a floating panel that lets any
    visitor tweak CSS custom properties (text/surface/background colors) for the current page
    load only (`document.documentElement.style.setProperty`, never persisted or sent anywhere);
    `TunnelControls` exposes sliders that tune the landing page's background animation
    parameters (speed, radius, parallax, etc.) via a local ref, also never persisted/transmitted.
    Neither reads or writes any user data, makes any network call, or exposes any credential/
    config — confirmed by reading both components in full.
  - Verified this ships in the real production build (`dist/assets/index-*.js` from `04-data.md`'s
    build includes both components — Vite/Rollup only tree-shakes genuinely unreferenced code, and
    these are actively rendered from `LandingPage.tsx`).
- **Why it matters:** Not a security vulnerability — there is no data or authentication surface
  here — but it is unpolished/unintended debug tooling visible to every anonymous visitor on the
  public landing page of a production deployment, which the author's own "TEMPORARY" comment flags
  as something that should have been removed already.
- **Recommended control:** Gate both behind `import.meta.env.DEV` (or delete `ThemeExplorer`
  per its own removal note, once the desired theme is finalized) so neither renders in a
  production build.

## Checks performed

1. Ran `npm audit` fresh and categorized each advisory by whether the affected package reaches the
   shipped runtime bundle or is build/lint/dev-server-only.
2. Grepped `src/` for any `axios` import; cross-checked against the real `dist/` build output from
   `04-data.md` to confirm it's tree-shaken out.
3. Read `LandingPage.tsx`'s imports/render tree for `TunnelControls`/`ThemeExplorer` mounting and
   checked for any `import.meta.env.DEV`/build-mode gate.
4. Read both components in full to confirm they have no data/network/auth surface (cosmetic-only).
5. Cross-checked the built bundle from `04-data.md` to confirm both components are present in the
   actual production output (not accidentally already excluded).

---

*No production code was modified. This file is the only artifact written.*
