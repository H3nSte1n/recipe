# Frontend Code-Quality Review — Conventions, TypeScript, State & Structure

Scope: **code quality, not security** (XSS / auth / data-exposure are covered by `fe-xss-apiclient.md`, `fe-auth.md`, `fe-data-exposure.md`). Reviews the React 18 + TypeScript-strict frontend under `services/frontend/src/` against the project's stated standards in root `CLAUDE.md`, `services/frontend/CLAUDE.md`, and `copilot-instructions.md`. Read-only; grounded in real `file:line` references. `npm run type-check` and `npm run lint` were run (both pass clean).

---

## Findings

### [Medium] `RecipeGraph` recomputes the entire layout and edge set on every render — including every drag/pan frame

**Location:** `components/RecipeGraph.tsx:94-107` (inside the component body, not memoized).

**Description:** `computeLayout(recipes)` (a recursive longest-path column assignment, `:18-84`), `nodeMap`, `canvasW`/`canvasH`, and the `edges` array (`:100-107`) are all computed unconditionally in the render body. Panning calls `setTransform` on **every** `onMouseMove` (`:141-148`) and `handleWheel` (`:109-126`), each of which triggers a re-render that re-runs the full layout — recursion over all recipes, `recipes.find`/`recipes.some` inner loops (`:19-23`, O(n²) in the worst case) — even though the layout depends only on `recipes` and never changes while dragging.

**Impact:** Avoidable CPU work on a hot interaction path; on larger recipe sets the graph drag will visibly jank.

**Recommendation:** Wrap the layout-dependent values in `useMemo(() => …, [recipes])` (and derive `nodeMap`/`edges`/`canvasW`/`canvasH` from that memo). Only `transform` should change per mouse move.

---

### [Medium] `AddRecipeModal` is a ~690-line component mixing parsing, data-fetching, sub-recipe orchestration, and presentation

**Location:** `components/AddRecipeModal.tsx` (entire file; `AddRecipeModal` itself spans `:253-689`).

**Description:** A single file holds: free-text ingredient/instruction parsers (`parseIngredients`, `parseInstructions`, `formatRecipe`, `formatRecipeIngredients`), an `AutoResizeTextarea` subcomponent, a `SubRecipeCard` subcomponent, ~20 `useState` hooks (`:254-287`), the create/update/delete API calls, and the full JSX. `handleSave` (`:450-514`) both creates child recipes in a loop **and** builds/sends the parent payload. This is well past the "components too large / doing too much" line and mixes data-fetching with presentation.

**Impact:** Hard to test, hard to read, high regression surface — counter to the project's stated "readability over cleverness".

**Recommendation:** Extract the pure parse/format helpers into `utils/` (next to `formatters.ts`), lift sub-recipe state into a `useSubSections` hook, and move the create/update orchestration into `recipeService`. `AutoResizeTextarea`/`SubRecipeCard` can become their own files.

---

### [Medium] The "open recipe by id, then set serves" async dance is duplicated four times in `HomePage`

**Location:** `pages/HomePage.tsx:46-55` (`handleGraphNodeClick`), `:92-103` (grid card `onClick`), `:119-127` (`onSubRecipeClick`), `:128-136` (`onParentRecipeClick`), and partially `:157-165` (`onSaved`).

**Description:** The same pattern — `const full = await getRecipeById(id); setSelectedRecipe(full); setServes(full.servings ?? 2)` wrapped in `void (async () => { … })()` with a `catch` fallback — is copy-pasted at least four times with minor variations (some fall back to the partial recipe, some swallow silently).

**Impact:** DRY violation; the inconsistent `catch` behavior (fallback vs. silent ignore) is the kind of drift that copy-paste produces.

**Recommendation:** Extract one `openRecipe(recipe: Recipe)` helper (or a small hook) that does the fetch + state set + fallback once, and call it from every handler.

---

### [Medium] Dead, duplicated auth pages — `LoginPage.tsx` / `RegisterPage.tsx` are never imported

**Location:** `pages/LoginPage.tsx`, `pages/RegisterPage.tsx` (plus `styles/LoginPage.css`, `styles/RegisterPage.css`). A grep for `LoginPage`/`RegisterPage` finds only their own definitions — `App.tsx` renders `LandingPage`, whose `LoginView`/`RegisterView` (`pages/LandingPage.tsx:36-111`) re-implement the same forms.

**Description:** Two full page components and their stylesheets are unreachable code. The live login form lives inside `LandingPage`. Worse, the two implementations have drifted: `LoginPage` and `LandingPage.LoginView` call `useAuth().login`, but `RegisterPage` (`:23`) and `LandingPage.RegisterView` (`:86`) call the `register` service **directly** rather than through `useAuth`, so registration never updates the `useAuth` `isAuthenticated` state. The duplication is exactly how that inconsistency goes unnoticed.

**Impact:** Maintenance hazard and confusion about which form is canonical; dead CSS shipped in the bundle.

**Recommendation:** Delete the unused `LoginPage`/`RegisterPage` (+CSS), or consolidate the forms into one reusable `AuthForm` used by `LandingPage`. Standardize on `useAuth` for both login and register.

---

### [Medium] Clickable text elements (`<p>` / `<span>`) are not keyboard-accessible

**Location:**
- `pages/LandingPage.tsx:69` and `:108` — `<p className="landing-page__form-link" onClick={…}>` ("Create an account" / "Sign in").
- `pages/RegisterPage.tsx:73` — `<span onClick={onBack} style={{ cursor: 'pointer' }}>Sign in</span>` (note: in a dead file per the finding above — primary evidence is the live `LandingPage` lines; this just reinforces deleting it).

**Description:** Interactive affordances are built on non-interactive elements with bare `onClick`, no `role="button"`, no `tabIndex`, and no key handler. They cannot be focused or activated via keyboard, and screen readers don't announce them as actionable. (Note the contrast: `RecipeCard.tsx:11-24` and `AddRecipeModal.tsx:547-553` do this correctly with `role`/`tabIndex`/`onKeyDown`.)

**Impact:** Keyboard and assistive-tech users cannot switch between login/register or go back.

**Recommendation:** Use `<button type="button">` styled as a link (the codebase already does this for other text actions), or add `role`/`tabIndex`/`onKeyDown`. Also drop the inline `style={{ cursor: 'pointer' }}` in favor of a CSS class (BEM), per the styling standard.

---

### [Medium] Developer/debug tooling shipped on the public landing page

**Location:** `components/ThemeExplorer.tsx` (file header literally says `// TEMPORARY`, `:1-2`) and `components/TunnelControls.tsx` (animation slider panel), both rendered in `pages/LandingPage.tsx:147-149`.

**Description:** `ThemeExplorer` is a self-described temporary color-picker that mutates `document.documentElement` CSS variables (`:27`), and `TunnelControls` is a 138-line live slider panel for the background animation. Both render on the unauthenticated landing page for every visitor (`LandingPage.tsx:148-149`, with a `{/* TEMPORARY */}` marker).

**Impact:** Debug UI exposed to end users; the "TEMPORARY" component is the kind of thing that quietly ships to production. Not a security issue, but a polish/quality regression.

**Recommendation:** Gate both behind `import.meta.env.DEV` (or remove `ThemeExplorer` per its own removal note) so they never render in production builds.

---

### [Low] API call swallows its error silently

**Location:** `components/AddRecipeModal.tsx:292` — `getMyRecipes().then(setAllRecipes).catch(() => {});`

**Description:** The sub-recipe autocomplete list is loaded with a `.catch(() => {})` that discards the error entirely — no state, no log. The project standard is "try-catch on all API calls with user-friendly error messages." This is a non-critical feature (link suggestions), so a hard error message may be overkill, but a fully empty catch hides failures during debugging.

**Impact:** If the fetch fails, the link-a-sub-recipe dropdown is silently empty with no signal why.

**Recommendation:** At minimum keep a comment justifying the swallow, or set a small non-blocking state so the feature can degrade visibly.

---

### [Low] Inconsistent ID generation and index-based list keys

**Location:** `components/AddRecipeModal.tsx:115` (`id: String(Date.now() + i)`) vs. `:331`,`:339`,`:354` (`crypto.randomUUID()`); index keys at `RecipeModal.tsx:93` (`key={keyIndex++}`), `:171`/`RecipeGraph.tsx:171` (`key={i}` for edges).

**Description:** Sub-section IDs are generated two different ways; `Date.now() + i` can collide for sub-recipes built in the same millisecond and is semantically odd next to `crypto.randomUUID()`. Several lists key on array index — acceptable for static/derived lists but a known footgun if those lists ever reorder.

**Impact:** Low; mainly consistency and latent reorder bugs.

**Recommendation:** Use `crypto.randomUUID()` uniformly for generated IDs; key edges/instruction fragments on a stable identifier where one exists.

---

### [Low] `LandingPage` subcomponents use anonymous inline prop types instead of named `…Props` interfaces

**Location:** `pages/LandingPage.tsx:17` (`HeroView`), `:36` (`LoginView`), `:74` (`RegisterView`) — props typed as inline object literals (`{ onLogin: () => void; … }`).

**Description:** The standard is "Interface over type for object shapes" and "`ComponentNameProps` for prop interfaces." These three components inline the shape instead. Minor, and contained to one file, but it's a documented convention.

**Recommendation:** Declare `HeroViewProps` / `LoginViewProps` / `RegisterViewProps` interfaces.

---

### [Low] Sub-recipe creation: children inherit the parent's cook time, and a mid-loop failure orphans recipes

**Location:** `components/AddRecipeModal.tsx:462-482` (loop) and `:472-473` (`cook_time: parseInt(cookTime)`, `shelf_life: parseInt(shelfLife)` taken from the **parent's** fields).

**Description:** When building a new child recipe inside the save loop, the child's `cook_time`/`shelf_life` are copied from the parent form's state rather than from the sub-section — likely unintended. Separately, `handleSave` `await`s `createRecipe` for each unlinked sub-section sequentially (`:466`); if a later one throws, the earlier children are already persisted on the backend with no rollback, leaving orphaned recipes.

**Impact:** Possibly-wrong child metadata; orphaned records on partial failure. Low because it's an edge path.

**Recommendation:** Source child timing from the sub-section (or omit it), and either create children in one batched request or clean up on partial failure.

---

### [Low] Modal dialogs lack dialog semantics / focus management

**Location:** `components/RecipeModal.tsx:186-187`, `components/AddRecipeModal.tsx:517-518`.

**Description:** Both modals are `<div>` overlays with click-outside-to-close and an `Escape` handler (good), but no `role="dialog"`, `aria-modal="true"`, labelling, or focus trap / focus-restore. Body scroll is correctly locked (`overflow: hidden`).

**Impact:** Screen-reader users aren't told a dialog opened; keyboard focus can escape behind the overlay.

**Recommendation:** Add `role="dialog"` + `aria-modal` + `aria-label`/`aria-labelledby`, move focus into the dialog on open, and restore it on close.

---

### [Info] All five non-null assertions are guard-justified

**Location:** `main.tsx:6` (`getElementById('root')!`, the standard root mount), `components/RecipeGraph.tsx:19` (`memo.get(id)!`, guarded by `memo.has(id)` on `:18`), `:58` (`byCol.get(col)!.push`, guarded by `:57`), `components/AddRecipeModal.tsx:281` (`buildInitialSubSections(initialRecipe!)`, guarded by `hasSubRecipes` which is `initialRecipe != null && …`), `components/RecipeModal.tsx:254` (`onSubRecipeClick(sub.child!)`, guarded by `if (!sub.child) return null` on `:246`).

**Description:** Every `!` non-null assertion in the codebase is immediately preceded by an explicit guard, so none is hiding a real nullability issue. Noted only to confirm the assertions are safe rather than papering over a type gap.

---

### [Info] API responses are typed by annotation, not runtime-validated — consistent with the project's own pattern

**Location:** e.g. `services/recipeService.ts:30` (`const data: Recipe = await response.json()`), `:89`, `services/authService.ts:19`.

**Description:** Responses are typed purely by TypeScript annotation on `response.json()` with no runtime schema validation (no `zod`/`io-ts`), so a backend contract drift would be a silent type lie. This is exactly the pattern `copilot-instructions.md` endorses ("Always create interfaces for API responses"), and the snake_case interfaces match the Go structs, so this is called out as **checked and acceptable**, not a finding — recorded for rubric completeness.

---

## Positives — what the frontend does well

- **TypeScript strictness is real and enforced.** `tsconfig.json` enables `strict`, `noUnusedLocals`, `noUnusedParameters`, and `noFallthroughCasesInSwitch`. A full-tree grep found **zero** `any`, `as any`, `@ts-ignore`/`@ts-expect-error`, and the five `!` non-null assertions are each guard-justified (see Info finding). `npm run type-check` passes clean, and `npm run lint` passes with `--max-warnings 0`.
- **Service layer error handling is consistent.** Every function in `services/recipeService.ts` and `services/authService.ts` wraps its API call in try-catch, checks `response.ok`, and rethrows a typed `Error`; the UI layer (`LoginPage`, `RegisterPage`, `LandingPage` views, `AddRecipeModal`, `HomePage`) translates those into user-friendly messages in state — matching the stated standard.
- **Conventions are followed closely.** Functional components only; PascalCase component files; `…Props` interfaces for nearly all components; interfaces (not type aliases) for object shapes; type aliases used only for unions (`Screen`, `AuthView`, `Tab`) — which is idiomatic.
- **CSS is BEM throughout.** Across ~179 selectors in `styles/`, the only unprefixed ones are the block roots (`.recipe-card`, `.home-page`, …) and the global `type-*` typography utilities — i.e. correct BEM.
- **API contract adherence.** All calls use relative `/api/v1/...` paths (no hardcoded host); all `types/recipe.ts` and `types/auth.ts` fields are snake_case and match the Go domain structs (`user_id`, `prep_time`, `step_number`, `serving_factor`, …).
- **Solid hook hygiene in places.** `useRecipes` uses a `cancelled` flag to avoid setting state after unmount (`hooks/useRecipes.ts:20-46`) and memoizes `filterRecipes`/`refresh`; `RecipeModal` and `HomePage` use `useMemo`/`useCallback` appropriately; modals use an `onCloseRef` to keep the Escape handler stable without re-subscribing.
- **Accessibility done right where it counts.** `RecipeCard` and the `AddRecipeModal` image dropzone implement full `role`/`tabIndex`/`onKeyDown` keyboard support; interactive icons carry `aria-label`s.

---

## Summary

Findings by severity: **High 0 · Medium 6 · Low 5 · Info 1** (12 total).

No High-severity quality bugs: there is no missing error handling on a critical path and no stale-closure/missing-dependency defect — the riskiest item is the unmemoized `RecipeGraph` layout recompute on drag (Medium, performance). Overall adherence to the project's stated standards is **strong**: strict TypeScript is genuinely enforced (zero `any`, clean type-check + lint), the service/error-handling pattern and BEM/snake_case/relative-path conventions are followed consistently. The quality gaps are concentrated in component decomposition (the 690-line `AddRecipeModal`), some DRY/dead-code drift (`HomePage` repetition, unused `LoginPage`/`RegisterPage`), keyboard accessibility of text-link controls, and debug tooling leaking into the production landing page.
