# Phase 4 — Fresh XSS / Sink Review

**Subtask:** Grep for `dangerouslySetInnerHTML`/`innerHTML`, user-data-derived `href`/`src`, and any
unsanitized recipe-content rendering across components.

**Scope/method:** Grepped all of `services/frontend/src/` for every classic React/DOM XSS sink
(`dangerouslySetInnerHTML`, `innerHTML`, `outerHTML`, `eval`, `new Function`, string-based
`setTimeout`), every `<a>`/`href`/`window.location`/`window.open` navigation sink, and every place
recipe-derived (server/AI-originated) content reaches the DOM.

---

## Result: no XSS sink found

- **No `dangerouslySetInnerHTML`, `innerHTML`, or `outerHTML` anywhere** in
  `services/frontend/src/` (grep confirmed, zero matches). There is no markdown-rendering library
  (`react-markdown`, `marked`) or HTML-sanitizer (`DOMPurify`, `sanitize-html`) in `package.json`
  either — consistent with the app never rendering raw HTML from any source (recipe content, AI
  output, or otherwise). All recipe text (`title`, `notes`, `description`, ingredient/instruction
  text) is rendered via plain JSX interpolation (`{recipe.title}`,
  `RecipeModal.tsx:213,278`, `RecipeCard.tsx:34`, `RecipeGraph.tsx:205`), which React escapes by
  default — there is no code path where recipe content (including AI-imported content, which is
  the most plausible "attacker-controlled" text in this app) is inserted as raw markup.
- **No dynamic `href`/navigation sinks.** Grep for `<a `, `window.open`, `window.location`,
  `location.href` found exactly one hit: `src/api/apiClient.ts:7` —
  `window.location.href = '/'` — a **hardcoded** redirect-to-home on a 401 (not derived from any
  user/server data), not a sink.
- **`<img src={...}>` is the only dynamic attribute sink**, used in four places
  (`RecipeCard.tsx:28`, `RecipeModal.tsx:207`, `RecipeGraph.tsx:200`, `AddRecipeModal.tsx:556`),
  all bound to `recipe.image_url` (backend-issued, signed-URL-or-external-source string) or a
  local `imagePreview` blob URL. `<img src>` is not a script-execution sink even for a
  `javascript:`-scheme value (unlike `<a href>`), so this is not exploitable as XSS even if
  `image_url` were fully attacker-controlled; worst case is a broken image or (for a `data:`
  URI) an unexpected inline image — no code execution.
- **No `eval`, `new Function`, or string-form `setTimeout`/`setInterval`** anywhere in `src/`
  (grep confirmed).

## Checks performed

1. Grepped `services/frontend/src/` for `dangerouslySetInnerHTML`, `innerHTML`, `outerHTML`.
2. Grepped `package.json` and `src/` for markdown-rendering or HTML-sanitizer libraries.
3. Grepped for `href=`, `src=`, `<a `, `window.open`, `window.location`, `location.href` and read
   every match's context.
4. Grepped for every place `recipe.title`/`.description`/`.notes`/`.instructions` (the closest
   thing to "attacker-controlled" content, since AI-imported recipes originate from arbitrary
   URLs/PDFs) is rendered, confirming plain JSX text interpolation in all cases.
5. Grepped for `eval(`, `new Function(`, template-string `setTimeout`/`setInterval`.

---

*No production code was modified. This file is the only artifact written.*
