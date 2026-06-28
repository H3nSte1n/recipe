# Audit 02 — Secret Hygiene & At-Rest AI Key Encryption (PR #31)

**Date:** 2026-06-28
**Scope:** Verify PR #31 — `env.*.yaml` gitignored, no live secret in git history, AES-256-GCM encryption of `user_ai_configs.api_key` keyed from an env var.
**Method:** Read-only code review + local read-only git/db/test probes. No production code under `services/` modified. No git mutations performed.

---

## Check 1 — `env.*.yaml` gitignored, `.sample` tracked

- Ignore rule: `services/backend/.gitignore:10` → `env.*.yaml` with negation `!env.*.yaml.sample` (line 11).
- `git check-ignore -v services/backend/env.development.yaml` →
  `services/backend/.gitignore:10:env.*.yaml	services/backend/env.development.yaml` (confirmed ignored).
- `git ls-files` shows the real env file is **NOT tracked**; only the template is tracked:
  `services/backend/env.development.yaml.sample`.
- The real `env.development.yaml` exists on disk (untracked) — correct posture.
- History scan: `git log --all --diff-filter=A --name-only | grep env.*.yaml` → no non-sample env yaml was **ever** added to git history.

**VERDICT — Check 1: PASS.** Real env yaml is gitignored and was never tracked; only `env.development.yaml.sample` is committed. Evidence: `services/backend/.gitignore:10-11`, `git check-ignore` output above.

---

## Check 2 — No live secret recoverable from git history

The previously-leaked secret location cited in the prior review (`env.development.yaml:39`) refers to an **untracked** on-disk file that was never committed — branch/HEAD history is clean of it. However, the task's prescribed probe (`git log --all -p`) surfaces a **real-format Anthropic API key plus JWT bearer tokens** that ARE recoverable from the repository's git data via the local stash.

- `git log --all -p -S "Snuq…REDACTED"` and `git rev-list --all | … git grep` locate the key in Bruno API-client files inside **`refs/stash@{0}`** (commits `4ecc373` "WIP on main", `1b99175` "index on main"):
  - `doc/recipe-api/ai/ai create config.yml`
  - `doc/recipe-api/ai/ai update config.yml`
- `git stash show -p stash@{0}` evidence:
  - line 1937: `+ "api_key": "sk-ant-api03-Snuq…REDACTED…MDAAA",` (full key intentionally redacted here — it is a real, unrotated `sk-ant-api03` Anthropic key; GitHub push-protection independently flagged the verbatim value, corroborating it is live-format)
  - lines 1908/1946/1969: three `token: eyJhbG…` JWT bearer tokens.
- The Bruno doc files are **NOT gitignored** (`git check-ignore "doc/recipe-api/ai/ai create config.yml"` → NOT IGNORED), so re-staging/re-stashing them will re-leak.

**Blast-radius scoping (does not change the verdict, but calibrates severity):**
- The key is recoverable **only** from local `stash@{0}` (Bruno files), **NOT** from branch/HEAD ancestry: `git log -p HEAD -S "Snuq…REDACTED"` returns nothing; no full `sk-ant-api03-…` key exists in HEAD history.
- Git stashes do **not** push to the remote (`git ls-remote` shows only `refs/heads/*`, no `refs/stash`), so a fresh clone from `origin` (github.com/H3nSte1n/recipe.git) does **not** contain the key.
- The JWT bearer tokens are **expired** (`exp 1775850307` = 2026-04-10; today 2026-06-28). The Anthropic API key has **no expiry**.
- Committed security-review docs (`docs/security-review/be-auth.md:128`, `be-handlers-config.md:81`) reference the key only in **redacted** form (`sk-ant-api03-...`); the full value is not in those tracked files. Test fixtures use placeholders only (`sk-ant-EXAMPLE`, `sk-ant-api03-EXAMPLE-not-a-real-key`).

**Why FAIL, not PASS:** "Recoverable from history" is literally true via the task's own `git log --all` probe. More decisively, the exposed Anthropic key is **real and unrotated** — it was exposed in plaintext (per the prior review's env.yaml finding and now the stash) and must be rotated regardless of which ref holds it. Grading PASS would tell the orchestrator "nothing to do" and suppress the one action that matters.

**Required remediation:**
1. **Rotate the leaked Anthropic API key** (`sk-ant-api03-Snuq…MDAAA`) at the provider — treat as compromised.
2. `git stash drop stash@{0}` to remove the key + tokens from local git data.
3. Gitignore/scrub `doc/recipe-api/` Bruno collection (it embeds keys and tokens) so the next stash/commit doesn't re-leak.

**VERDICT — Check 2: FAIL (go-live blocker candidate).** A real, unrotated Anthropic API key is recoverable from repo git data (`git stash show -p stash@{0}` → `doc/recipe-api/ai/ai create config.yml:api_key`); requires key rotation + stash drop. Blast radius is limited to the local clone (stash, not branch; absent from remote), and the accompanying JWTs are expired — but the live key mandates rotation.

---

## Check 3 — AES-256-GCM at-rest encryption of `user_ai_configs.api_key`

- Migration: `services/backend/migrations/000014_widen_user_ai_configs_api_key.up.sql` widens `api_key` `VARCHAR(255)` → `TEXT` with comment noting AES-256-GCM base64 storage.
- Crypto implementation: `services/backend/pkg/crypto/crypto.go`
  - `aes.NewCipher` over a 32-byte key (`sha256.Sum256` of the secret) → **AES-256**; `cipher.NewGCM` → **GCM** AEAD (`crypto.go:33-44`).
  - `Encrypt`: random per-message nonce via `crypto/rand`, returns `base64(nonce || ciphertext || tag)` (`crypto.go:48-56`).
  - `Decrypt`: base64-decode, split nonce, `aead.Open` authenticates (`crypto.go:60-78`).
  - `NewCipher("")` → `ErrEmptyKey` → **fails closed** when no key configured.
- Key sourcing (ENV, not committed literal): `services/backend/pkg/config/config.go`
  - `EncryptionKey string mapstructure:"encryption_key"` with doc "Inject via **SECURITY_ENCRYPTION_KEY**" (`config.go:24-26`).
  - `v.BindEnv("security.encryption_key")` (`config.go:108-111`) binds the env var to override the YAML placeholder; `v.AutomaticEnv()` also set.
  - Wired in `services/backend/cmd/api/main.go:61` → `crypto.NewCipher(cfg.Security.EncryptionKey)`.
- Committed sample (`env.development.yaml.sample:52`) holds only a **placeholder** `change-me-to-a-long-random-secret`; no real key in any tracked file (HEAD-history grep for `encryption_key`/`jwt secret` returns only placeholders: `CHANGE_ME`, `your-super-secret-key-here`).
- Service-boundary behavior: `services/backend/internal/service/apikey_crypto.go` encrypts on write / decrypts on read with a legacy-plaintext fallback for un-migrated rows.
- Unit tests (`go test ./pkg/crypto/... ./internal/service/`) — **all pass**:
  - `TestCipher_RoundTrip`, `TestNewCipher_RejectsEmptyKey`, `TestCipher_EncryptUsesRandomNonce`, `TestCipher_DecryptRejectsTampering`, `TestCipher_DecryptRejectsNonCiphertext`, `TestCipher_WrongKeyFails`
  - `TestAIConfigService_CreateEncryptsAndReadDecrypts`, `TestAIConfigService_UpdateWithoutKeyChangeDoesNotDoubleEncrypt`, `TestAIConfigService_LegacyPlaintextReadFallback`
- Live DB: `SELECT id, left(api_key,16), length(api_key) FROM user_ai_configs` returned **0 rows** (no ai_config configured) — ciphertext-at-rest not confirmable against live data, but not relied upon.

**VERDICT — Check 3: PASS.** AES-256-GCM with random nonce + auth tag, fail-closed on empty key, key sourced from env `SECURITY_ENCRYPTION_KEY` (no live key in any tracked file), round-trip/tamper/wrong-key + service encrypt-on-write/decrypt-on-read tests all pass. Evidence: `pkg/crypto/crypto.go:25-78`, `pkg/config/config.go:24-111`, `cmd/api/main.go:61`, `migrations/000014_…up.sql`, green test run.

---

## Summary verdicts

- **Check 1 (env.*.yaml gitignored, .sample tracked): PASS** — `.gitignore:10-11` + `git check-ignore` + `git ls-files`; real env yaml never tracked.
- **Check 2 (no live secret in history): FAIL (go-live blocker candidate)** — real unrotated Anthropic key recoverable via `git log --all` / `git stash show -p stash@{0}` in `doc/recipe-api/ai/ai create config.yml`; rotate key + drop stash + gitignore Bruno docs. Scoped to local clone (stash, not branch; absent from remote); accompanying JWTs expired.
- **Check 3 (AES-256-GCM, env-keyed): PASS** — verified in code + all crypto/service tests green; key from `SECURITY_ENCRYPTION_KEY`, no live key in tracked files.
