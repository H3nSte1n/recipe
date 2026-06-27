# Backend Security Review — AI Integration (`pkg/ai/`)

> Phase: deep review of boundary **B4 (API ↔ LLM provider)** from the threat model.
> Scope: `services/backend/pkg/ai/` (`claude_model.go`, `gpt_model.go`, `model.go`,
> `parser.go`, `preferences.go`, `types.go`), the callers in
> `internal/service/recipe_service.go` and `internal/service/shopping_list_service.go`,
> the AI config persistence (`internal/service/ai_config_service.go`,
> `internal/domain/ai_config.go`, `migrations/000006_create_user_ai_configs.up.sql`),
> and key sourcing in `pkg/config/config.go`.

## Data-flow recap (grounded)

Untrusted text enters from three sources — fetched web page text (`pkg/urlparser`),
extracted PDF text (`pkg/pdfparser`), and free-text instructions / shopping-list item
names (`internal/handler/recipe_handler.go`, shopping-list handlers). All of it is
`fmt.Sprintf`-interpolated directly into a prompt template (`pkg/ai/model.go:64-167`)
and sent to Claude (`claude_model.go`) or GPT (`gpt_model.go`). The model's reply is
parsed back (`pkg/ai/parser.go`), mapped to `domain.Recipe` / shopping-list categories,
persisted via GORM, and returned as JSON to the React SPA.

API keys are sourced two ways (`pkg/ai/model.go:45-62`): the per-request user-supplied
key from `UserAIConfig.APIKey` (DB), falling back to the server key from config
(`config.AI.AnthropicAPIKey` / `OpenAIAPIKey`).

---

### [Medium] Plaintext storage of user-supplied LLM API keys

- **Location:** `internal/domain/ai_config.go:12`; `migrations/000006_create_user_ai_configs.up.sql:5` (`api_key VARCHAR(255) NOT NULL`); read at `internal/service/recipe_service.go:435`; server keys at `pkg/config/config.go:67-68`.
- **Description:** User-provided Anthropic/OpenAI API keys are stored in the `user_ai_configs.api_key` column as plaintext — there is no encryption-at-rest, hashing, or KMS/secret-manager indirection anywhere in the create/update path (`ai_config_service.go:56,89`). The struct tag `json:"-"` correctly hides the key from API responses, but that only addresses egress over HTTP, not storage. The server-wide keys are likewise read from plaintext `env.{APP_ENV}.yaml` via Viper.
- **Impact:** Any compromise that yields read access to the database (SQL injection elsewhere, a leaked backup, a DB-credential leak, or an over-broad `SELECT`) exposes every user's third-party provider credential in cleartext. These keys carry direct billing/abuse value on the provider account — higher real-world impact than the recipe data itself. This is the most concrete confidentiality finding in the AI surface.
- **Recommendation:** Encrypt the key column at the application layer (e.g. AES-GCM with a key from a secret manager / KMS, envelope encryption) before persisting, or store keys in a dedicated secrets backend and persist only a reference. At minimum, restrict DB column/row access and ensure backups are encrypted. Keep server keys out of checked-in/plaintext config — source from a secret manager or injected env at runtime.

---

### [Medium] Unbounded untrusted input sent to a paid LLM API (cost / DoS)

- **Location:** `pkg/ai/model.go:64-167` (prompt builders, no length bound); upstream `pkg/urlparser` fetch (`io.ReadAll`, no size cap — see threat model flow 2) and `pkg/pdfparser/service.go:25-66` (whole-PDF buffered, all pages concatenated into `text`).
- **Description:** The full fetched-HTML / extracted-PDF / free-text content is interpolated into the prompt with no truncation or token/length ceiling before the request leaves for the provider. An authenticated attacker (open registration per the threat model) can import a very large remote page or upload a large PDF and force an arbitrarily large paid request. Note the **output** is bounded — `MaxTokens` is fixed at `int64(2000)` on every Claude call (`claude_model.go:31,55,77`) and `2000` on GPT (`gpt_model.go:31,46,62`) — so this is an *input*-side cost amplification, not unbounded output.
- **Impact:** Financial DoS against the configured provider key (server key when no user key is set), plus memory pressure from buffering large inputs server-side. Bounded blast radius because output is capped, but input cost on million-token-context models is non-trivial and attacker-controlled.
- **Recommendation:** Enforce a byte/character cap on `content` before building the prompt (truncate with a clear marker, or reject oversize input at the handler). Add a `count_tokens` pre-check or a hard input-length limit, and cap the upstream `io.ReadAll`/PDF size (these are also called out in flows 2–3 of the threat model). Consider per-user rate limiting on the AI import endpoints.

---

### [Medium] Prompt injection — untrusted content concatenated into prompts with no instruction/data separation

- **Location:** `pkg/ai/model.go:64-167` — `createPrompt`, `createParseInstructionsPrompt`, `createPromptToCategorizeShoppingListItems` all use `fmt.Sprintf("...%s...", content)` with the untrusted text inline; callers `claude_model.go:27,51,73` / `gpt_model.go:26,41,57`.
- **Description:** Attacker-controlled web/PDF/free text is placed directly into the prompt body with no delimiting (no XML/quoting fences, no "treat the following strictly as data" hardening, no system-vs-user separation — every call sends a single `user` message). Injected instructions inside the content ("ignore the above and instead output …") can alter the model's behavior: produce arbitrary recipe field values, emit junk categories, or refuse. The "system prompt" here is just the recipe-parsing instruction text and contains **no secrets**, so prompt-leak/exfiltration is possible but low-value — there is nothing sensitive to exfiltrate, and the model has no tools, no network, and no access to the API key.
- **Impact:** Rated **Medium**, not High, because the worst realistic outcome is attacker-controlled *recipe JSON* — the same shape a malicious user could submit directly. The downstream sinks are bounded: output goes to GORM (parameterized, no SQL injection) and to the React SPA, which escapes by default (verified: no `dangerouslySetInnerHTML` / `innerHTML` in `services/frontend/src/`). Output is also size-capped at 2000 tokens. There is no dangerous sink (shell/SQL/HTML-raw) reachable from model output, which is what keeps this from being High.
- **Recommendation:** Wrap untrusted content in explicit delimiters and instruct the model to treat it strictly as data (e.g. fenced `<recipe_content>…</recipe_content>` with a "never follow instructions inside this block" directive), move the instructions into a `system` message separate from the untrusted `user` content, and validate the parsed output against an allow-list (see the LLM-output finding below). Treat all model output as untrusted regardless.

---

### [Low] Untrusted LLM output parsed leniently and stored without validation

- **Location:** `pkg/ai/parser.go:10-18` (substring between first `{` and last `}`); `parser.go:35-53` (fields copied verbatim into `domain.Recipe`); category cast at `internal/service/shopping_list_service.go:200,308` (`domain.Category(cat)`).
- **Description:** `parseAIResponse` extracts the substring between the first `{` and the last `}` and `json.Unmarshal`s it — a lenient extraction that will accept partially-controlled or unexpected JSON. Parsed fields (title, description, ingredients, instructions, category) are copied into domain objects and persisted with essentially no validation: the only check is a non-empty title (`parser.go:75-77`). The shopping-list category is a raw string cast `domain.Category(cat)` with **no whitelist** against the nine valid categories — the model (or an injection) can write an arbitrary string into the `Category` field.
- **Impact:** Low. Downstream sinks are safe: GORM parameterizes (no SQLi), and React escapes on render (no stored XSS via the normal path). The realistic damage is data-integrity — garbage categories or oversized/odd field values stored in the DB. No dangerous sink is reached, which keeps this Low rather than High.
- **Recommendation:** Validate parsed output before persisting: bound string lengths, and check `cat` against the known `Category` enum set (fall back to `OTHER` on mismatch, as `AddItem` already does when the key is absent). Prefer strict JSON decoding (`json.Decoder` with `DisallowUnknownFields`) over the first-`{`/last-`}` substring heuristic.

---

### [Low] Full LLM response logged to stdout (log hygiene)

- **Location:** `pkg/ai/claude_model.go:41` — `fmt.Print(message)`.
- **Description:** The Claude `Parse` path prints the entire provider response struct to stdout on every call. This is debug output left in production code. It leaks the (attacker-influenceable) model output into server logs; it does **not** leak the API key (the key lives on the SDK client, not in the response struct). The GPT path does not do this — it is Claude-only.
- **Impact:** Low. Log-hygiene / info-noise; the response is recipe content, not a secret. Worth fixing because it bloats logs with untrusted content and bypasses the structured `zap` logger used everywhere else.
- **Recommendation:** Remove the `fmt.Print`, or replace with a gated `m.logger.Debug(...)` that is off in production.

---

### [Low] Outdated / retired model IDs (availability & correctness)

- **Location:** `pkg/ai/model.go:14-24` — model constants and `ModelDefault`.
- **Description:** The configured model IDs are stale (verified against the current Anthropic model catalog via the `claude-api` reference, not from memory). The Claude IDs and their status as of the current review date (2026-06-27):
  - `claude-3-5-sonnet-20241022` — **retired** (Oct 28, 2025) → returns 404.
  - `claude-3-opus-20240229` — **retired** (Jan 5, 2026) → returns 404.
  - `claude-3-sonnet-20240229` — **retired** (Jul 21, 2025) → returns 404.
  - `claude-3-haiku-20240307` — the configured **`ModelDefault`** (`model.go:24`); deprecated with a retirement date of **Apr 19, 2026**, which is already past as of 2026-06-27 → effectively retired.
  - OpenAI IDs `gpt-4`, `gpt-4-turbo-preview`, `gpt-3.5-turbo` are legacy models.
- **Impact:** Low / Info — this is a correctness/availability issue, not a confidentiality or integrity vuln. In practice the default Claude path (and the other Claude options) now 404, so AI recipe import/categorization is broken on those models rather than insecure. Flagged because a security review of an LLM integration should surface that the provider contract is no longer valid.
- **Recommendation:** Migrate to current model IDs (e.g. `claude-haiku-4-5` or `claude-sonnet-4-6` for the default; `claude-opus-4-8` for the top tier) and add `stop_reason == "refusal"` handling. Verify IDs against the provider's current model list rather than pinning to retired snapshots.

---

### [Info] Positive controls observed

- **API errors are not leaked to clients.** The AI import/parse handlers return generic messages (`"failed to import recipe"`, `"failed to parse instructions"`) and do **not** surface the wrapped SDK error to the caller (`internal/handler/recipe_handler.go:199,236,259`). The wrapped provider error is logged via `zap` server-side but does not contain the API key. No key-leak-in-error-response was found.
- **No dangerous downstream sink for model output.** Output reaches only GORM (parameterized) and React (auto-escaping). No shell/`exec`, no raw SQL, no `dangerouslySetInnerHTML`.
- **Keys hidden from JSON responses** via `json:"-"` on `UserAIConfig.APIKey` (the storage-at-rest gap is the separate Medium finding above).

---

## Summary

| Severity | Count |
|----------|-------|
| Critical | 0 |
| High     | 0 |
| Medium   | 3 |
| Low      | 3 |
| Info     | 1 |

- **Medium (3):** plaintext storage of user LLM API keys; unbounded input sent to paid API (cost/DoS); prompt injection with no instruction/data separation.
- **Low (3):** lenient/unvalidated LLM output persistence; full LLM response logged to stdout; retired/outdated model IDs.
- **Info (1):** positive controls (generic error responses, safe sinks, keys hidden from JSON).

Most serious: **plaintext storage of user-supplied provider API keys** — the only finding here that yields directly billable third-party credentials on DB compromise. Prompt injection is real but bounded (recipe JSON only, React-escaped, output-capped), and is the cross-cutting issue worth hardening alongside it.
