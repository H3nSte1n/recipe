# Phase 6 — AI Model ID Verification

**Subtask:** Verify the configured Claude/OpenAI model IDs in `pkg/ai/model.go` against current
provider model lists (using the `claude-api` skill for Claude IDs) and run a live AI-import
round-trip to confirm or refute the known 404/live-bug.

**Scope/method:** Read the configured model ID constants in
`services/backend/pkg/ai/model.go:14-23`. Invoked the `claude-api` skill's authoritative,
date-stamped model/retirement tables to check each Claude ID's current status. Attempted a live
round-trip against the running stack (`POST /api/v1/recipes/parser/instructions`) to observe the
actual failure mode.

---

## Result: all four configured Claude model IDs are retired as of today (2026-07-15) — confirms and elevates the prior finding

| Configured ID (`pkg/ai/model.go`) | Constant | Retirement date | Status as of 2026-07-15 |
|---|---|---|---|
| `claude-3-5-sonnet-20241022` | `ModelClaude35Sonnet` | 2025-10-28 | **Retired** (~8.5 months ago) |
| `claude-3-opus-20240229` | `ModelClaude3Opus` | 2026-01-05 | **Retired** (~6 months ago) |
| `claude-3-sonnet-20240229` | `ModelClaude3Sonnet` | 2025-07-21 | **Retired** (~1 year ago) |
| `claude-3-haiku-20240307` | `ModelClaude3Haiku` — **also the app's `ModelDefault`** (`model.go:23`) | 2026-04-19 | **Retired** (~3 months ago) |

- **Severity:** HIGH (live functional outage, not a security vulnerability)
- **Evidence:** `pkg/ai/model.go:14-23` hardcodes exactly these four IDs as the app's entire
  Claude model catalog, and `ModelDefault` — the fallback used whenever a user has no AI config or
  their configured model can't be resolved (`recipe_service.go:434-445`, `getUserAIPreferences`
  falling back to `ai.UserAIPreferences{ModelType: ai.ModelDefault}` on any `GetDefaultConfig`
  error) — is **itself** one of the retired IDs (`claude-3-haiku-20240307`).
- **Why this matters more than "some model IDs are stale":** every single Claude ID the app can
  select is now past its retirement date. Anthropic returns `404 Not Found` for a retired model
  ID (per `shared/error-codes.md`'s error-code reference: "404 Not Found... Using deprecated model
  ID"). This means **every AI-dependent feature is currently broken for any deployment using a
  live Anthropic API key**: URL import, PDF import, plain-text instruction parsing, and shopping-list
  item categorization (`ai.AIModel.CategorizeItems`) all resolve to a model ID that will 404 on
  the real API, regardless of which of the four the user's `UserAIConfig` selects.
- **OpenAI IDs not independently re-verified**: `gpt-4`, `gpt-4-turbo-preview`, `gpt-3.5-turbo`
  (`model.go:14-16`) are out of scope for the `claude-api` skill's authoritative tables; flagging
  as unverified rather than asserting a status. `gpt-3.5-turbo` and `gpt-4-turbo-preview` are
  commonly-cited OpenAI deprecation targets, so the same class of staleness is plausible there too
  — recommend a follow-up check against OpenAI's model list before go-live.

## Live round-trip: attempted, INCONCLUSIVE on the specific 404 (blocked by placeholder credentials, not by anything in this audit's control)

- **Attempted:** `POST /api/v1/recipes/parser/instructions` against the live stack with a
  plain-text instruction payload, as the throwaway audit user (whose default AI config uses
  `claude-3-5-sonnet-20241022` per this audit's Phase 3 IDOR probes).
- **Result:** `500 {"error":"failed to parse instructions"}` — confirmed via `docker compose logs
  app` that the underlying cause is `401 Unauthorized ... "invalid x-api-key"` from
  `https://api.anthropic.com/v1/messages`, **not** a 404. `env.development.yaml`'s
  `ai.anthropic_api_key` is the shipped placeholder value (`your_anthropic_api_key`, 22 chars),
  not a real credential.
- **Why this is INCONCLUSIVE rather than a refutation:** the 401 only proves the placeholder key
  is invalid — it says nothing about whether the model ID itself is valid, because Anthropic's API
  checks authentication before resolving the model. A live 404-vs-401 distinction requires a real
  API key, which this audit environment does not have and should not be provisioned with one
  (introducing a live paid credential into an audit session is out of scope and a secret-handling
  risk in its own right). Recorded here as the honest limitation per this audit's execution
  semantics ("a probe that cannot be run is recorded as INCONCLUSIVE, never papered into a green
  pass") rather than either asserting the live 404 was reproduced or claiming the bug is refuted.
- **Confidence in the underlying finding despite the inconclusive live probe:** the retirement
  dates above come from the `claude-api` skill's own authoritative, actively-maintained
  model/retirement tables (cross-checked against today's date, 2026-07-15) — this is a
  significantly stronger source than re-deriving retirement status from first principles, and all
  four dates are unambiguously in the past. The 404-on-retired-model behavior itself is documented
  in the same skill's error-codes reference. Recommend treating the "all four configured IDs are
  retired" finding as confirmed on the strength of that source, with only the *live reproduction*
  (not the underlying fact) marked inconclusive.

## Recommended control

Migrate to current Claude model IDs per the `claude-api` skill's migration guide (retired-model
replacement table): `claude-3-5-sonnet-20241022` / `claude-3-sonnet-20240229` →
`claude-sonnet-5`; `claude-3-opus-20240229` → `claude-opus-4-8`; `claude-3-haiku-20240307` →
`claude-haiku-4-5`. This requires updating `pkg/ai/model.go`'s constants, the seed/migration data
behind `GET /ai-configs/models` (the `domain.AIModel` rows returned during this audit's probes
carry the same retired `model_version` strings), and `mapAIModelToModelType` in
`recipe_service.go:469-490` (the `provider-modelVersion` switch statement keyed on these exact
strings). Also verify the OpenAI IDs against OpenAI's current model list before go-live, and add
an automated check (e.g. `govulncheck`-style CI job, or Phase 6's own `06-vulns.md` scope) that
periodically re-verifies configured model IDs are still active — this exact bug (a model ID silently
sailing past its retirement date with no test catching it) is likely to recur otherwise.

## Checks performed

1. Read `pkg/ai/model.go:14-23` for every configured `ModelType` constant and confirmed
   `ModelDefault`'s value.
2. Invoked the `claude-api` skill and cross-referenced its "Retired Models" and "Deprecated
   Models" tables (in `shared/models.md`, reproduced in `shared/model-migration.md`) against each
   configured ID and against today's date (2026-07-15).
3. Attempted a live AI round-trip (`POST /recipes/parser/instructions`) against the running stack;
   inspected `docker compose logs app` to distinguish the actual failure (401 invalid-key) from the
   hypothesis under test (404 retired-model), and confirmed `env.development.yaml`'s Anthropic key
   is the shipped placeholder, not a real credential.
4. Cross-referenced `recipe_service.go`'s `mapAIModelToModelType` and `getUserAIPreferences`
   fallback path to confirm `ModelDefault` (itself retired) is reachable whenever a user has no AI
   config configured — not just an edge case limited to explicit selection of an old model.

---

*No production code was modified. This file is the only artifact written.*
