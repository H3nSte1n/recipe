# Phase 3 — Fresh SSRF / AI / Storage Bypass Review

**Subtask:** Find any URL-fetch or file path NOT routed through the hardened `pkg/urlparser`/
`pkg/storage`, and any second LLM call site bypassing the Phase-7 (PR #37) prompt-injection
delimiters.

**Scope/method:** Grepped the entire backend for any HTTP client construction, raw socket dial,
file-path join/open, or LLM prompt construction outside the known-hardened packages, then traced
every call site back to its source.

---

## Result: no bypass found in any of the three surfaces

### 1. Outbound HTTP / SSRF surface

- Grep for `http.Client{`, `http.Get`, `net.Dial`, and `.Do(req)` across `internal/` and `pkg/`
  returns exactly **one** client construction and **one** dial site, both inside
  `pkg/urlparser/` (`service.go:38`, `fetcher.go:39`, `ssrf.go:148`). There is no second HTTP
  client anywhere in the codebase that could fetch a user-supplied URL outside the SSRF-guarded
  path (`safeDialContext`, resolved-IP checks, redirect re-validation — already verified live in
  Phase 2's `02-ssrf.md`).
- `pkg/ai/*.go` (Claude/GPT model clients) use the official Anthropic/OpenAI SDKs, which talk only
  to the fixed provider API host — not user-controlled URLs — so they are not part of the SSRF
  surface.

### 2. File-path / storage surface

- Grep for `os.Open`, `os.Create`, `os.WriteFile`, `filepath.Join` across `internal/` and `pkg/`
  returns exactly two files: `pkg/storage/local_storage.go` (the hardened store) and
  `internal/handler/uploads_handler.go` (the signed-URL serving handler). No other code path
  reads or writes to the upload directory.
- **Write path** (`local_storage.go:29-53`, `UploadFile`): the filename is always server-generated
  (`uuid.New().String() + ext`, where `ext` comes from `DetectImageType`'s magic-byte sniff, not
  the client-supplied filename/extension) — the client-controlled filename never reaches
  `filepath.Join`.
- **Delete path** (`local_storage.go:56-66`, `DeleteFile`): takes `filepath.Base(fileURL)` before
  joining, stripping any `../` component even if `fileURL` were attacker-influenced.
- **Serve path** (`uploads_handler.go:29-49`): rejects any `filename` containing `..` or not equal
  to its own `filepath.Base`, then requires a valid HMAC signature (`signer.Verify`) before
  `c.File(...)`, and sets `nosniff` + `Content-Disposition: attachment`. Matches the already-verified
  PR #34 hardening (`02-upload.md`) — re-confirmed by reading rather than re-probing live to avoid
  duplicate destructive state on the shared stack.
- PDF import (`recipe_handler.go:246,270-272`) reads the uploaded file through
  `http.MaxBytesReader` + `io.LimitReader(f, maxPDFUploadBytes+1)` into an in-memory `[]byte`
  passed to `pdfParser.Parse` — no path/filename ever touches disk for PDF import, so there is no
  separate file-path surface to bypass here.

### 3. LLM prompt-injection surface

- Three LLM-facing operations exist: recipe parsing (`Parse`), instruction parsing
  (`ParseInstructions`), and item categorization (`CategorizeItems`). All three are implemented
  once each in `pkg/ai/gpt_model.go` and `pkg/ai/claude_model.go`, and **both** models call the
  same shared prompt builders in `pkg/ai/prompts.go`:
  `buildRecipePrompt`/`buildInstructionsPrompt`/`buildCategorizePrompt`.
- Every builder uses the same nonce-fenced pattern: untrusted content
  (URL-fetched page, PDF text, plain-text instructions, ingredient names) is wrapped via
  `fencedContent(nonce, content)` in the **user** message, while the **system** message carries
  only the fixed instructions plus `dataDirective(nonce)` telling the model to treat the fenced
  block as inert data. There is no second code path that sends untrusted content directly as a
  system message or unwrapped user message.
- Output-side allowlisting is consistent across all three: `parseAIResponse` clamps numeric fields
  (`clampNonNegative`) and requires a non-empty title; `parseCategorizeItemsResponse` normalizes
  every category through the `validCategories` allowlist (`normalizeCategory`); `parseInstructions`
  requires the content to already be a JSON array before parsing. No call site trusts raw LLM
  output past these validators.
- `urlparser`'s AI usage (`pkg/urlparser/service.go`) and `pdfparser`'s AI usage both call into
  `recipeService`'s `modelFactory.CreateModel(...).Parse(...)`, i.e. the same `Parse` method and
  therefore the same `buildRecipePrompt` — confirmed no separate/ad-hoc prompt construction exists
  for the URL-import or PDF-import flows specifically.

## Checks performed

1. Grepped for `http.Client{`, `http.Get`, `net.Dial`, `.Do(req)` across `internal/` and `pkg/`.
2. Grepped for `os.Open`, `os.Create`, `os.WriteFile`, `ioutil.WriteFile`, `filepath.Join` across
   `internal/` and `pkg/`; read both matching files end-to-end.
3. Grepped for every call site of `ai.ModelFactory`/`CreateModel`/`Parse`/`ParseInstructions`/
   `CategorizeItems` (`internal/service/recipe_service.go`, `shopping_list_service.go`,
   `pkg/ai/gpt_model.go`, `claude_model.go`) and confirmed both model implementations route through
   the shared `pkg/ai/prompts.go` builders with no bypass.
4. Read `pkg/ai/parser.go` in full for output-side validation/allowlisting on all three LLM
   response types.
5. Re-read `internal/handler/uploads_handler.go` and `pkg/storage/local_storage.go` for
   path-traversal defenses (cross-referenced against Phase 2's `02-upload.md`, not re-probed live).

---

*No production code was modified. This file is the only artifact written.*
