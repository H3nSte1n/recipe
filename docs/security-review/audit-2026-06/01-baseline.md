# Phase 1 — Test-Gate Baseline (P1.4)

**Audit:** recipe-post-remediation-audit · **Date:** 2026-06-28
**Branch:** `phase-1-recipe-post-remediation-audit`
**Gate (`test_command`):** `cd services/backend && make test && cd ../frontend && npm run type-check && npm run lint`

## Result: GREEN ✅

All three gate stages exit 0 on a clean checkout of `main` + the Phase-1 findings docs (docs-only changes; no production code touched).

| Stage | Command | Exit | Result |
|-------|---------|------|--------|
| Backend tests | `make test` (`go test -v ./...`) | 0 | PASS — 9 packages `ok`, 379 `--- PASS`, 0 FAIL |
| Frontend type-check | `npm run type-check` (`tsc --noEmit`) | 0 | PASS — no type errors |
| Frontend lint | `npm run lint` (`eslint . --max-warnings 0`) | 0 | PASS — 0 warnings |

### Backend packages covered (all `ok`)
```
internal/handler   internal/repository   internal/service
pkg/ai   pkg/config   pkg/crypto   pkg/signedurl   pkg/storage   pkg/urlparser
```
(`pkg/crypto`, `pkg/signedurl` are remediation-era additions — AES-GCM at-rest + signed-URL uploads.)

### Toolchain
- Go `go1.25.0 darwin/arm64`
- Node `v26.3.1`

## Port-collision / test-DB note (required by P1.4)
The plan flagged a possible `:5432` collision between the live audit stack and the test DB. **No action was needed:** backend `make test` is `go test ./...` and uses **glebarez/sqlite** (pure-Go, temp-file DB) — it does **not** connect to PostgreSQL on `:5432`. Frontend `type-check`/`lint` need no DB. The live audit stack (db published on host `:15432`, see `00-notes.md`) was therefore left **UP and healthy** throughout the gate run; the gate and the live stack are fully decoupled.

## Baseline meaning
This green gate is the reference point for later phases. Per audit semantics, a green gate does **not** by itself prove any security verdict — it only confirms the build/tests/lint are clean before live probing begins. A red gate discovered in a later phase is a finding to record, not silently fixed.
