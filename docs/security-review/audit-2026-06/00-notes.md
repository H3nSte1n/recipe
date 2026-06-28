# Audit 2026-06 — Shared Stack & Notes Sink

This file is the **shared stack/notes sink** for the 2026-06 **post-remediation security audit**
(Phase 1, branch `phase-1-recipe-post-remediation-audit`). Subtask **P1.1** produced it.

All later audit phases should reattach to the **same running stack** described here and reuse the
throwaway audit credentials below for cross-tenant / authz probes. This is an **AUDIT**: read/probe
only, never modify production code under `services/`.

---

## Base URLs (host port REMAP)

| Service  | Container port | Host URL                         | Notes |
|----------|----------------|----------------------------------|-------|
| backend  | `8080`         | http://localhost:18080           | API base: `http://localhost:18080/api/v1` |
| db (pg)  | `5432`         | localhost:15432                  | user=`postgres` password=`your_password` db=`recipe_db` |
| frontend | `5173`         | http://localhost:5173            | — |

### Why ports were remapped

The plan's standard host ports **8080** and **5432** were **occupied by UNRELATED services**
(a native Go binary on 8080; an unrelated `dev-postgres` container + kafka on 5432). Those must
**NOT** be disturbed. The orchestrator therefore remapped the host ports to **18080** (backend)
and **15432** (db) via a compose override file in the session scratchpad.

> **Internal container ports remain 8080/5432**, so the frontend→`app:8080` Vite proxy inside the
> Docker network is unaffected. Only the *host-published* ports changed.

### Exact invocation (reattach to the SAME stack)

```bash
cd /Users/henry/dev/Projects/recipe
docker compose \
  -f docker-compose.yml \
  -f /private/tmp/claude-501/-Users-henry-dev-Projects-recipe/57ec04bb-3f8f-4aa6-b125-53e381bd9aa6/scratchpad/compose.override.yml \
  ps
```

Use the same `-f docker-compose.yml -f <override>` pair for any `ps` / `logs` / `exec` against this stack.

---

## Throwaway audit users (DO NOT use in production — throwaway creds)

Both registered via `POST /api/v1/auth/register` and logged in via `POST /api/v1/auth/login`.
Shared password for both: `Audit-Passw0rd!2026`

| Label  | Email                                  | User ID (UUID)                          | Password              |
|--------|----------------------------------------|-----------------------------------------|-----------------------|
| User A | `audit-user-a+5bb8ea14@example.com`    | `eb1ba0c9-3760-4408-a265-07a2a0ed6668`  | `Audit-Passw0rd!2026` |
| User B | `audit-user-b+62cd7d92@example.com`    | `0bf4c728-22f3-4e7a-a6d3-80ab632b8fe8`  | `Audit-Passw0rd!2026` |

### JWT tokens (HS256; `exp`, `user_id`, `email` claims)

> Tokens expire (~1h based on `exp` claim). If expired, re-login with the creds above to mint fresh ones.

**User A token:**
```
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6ImF1ZGl0LXVzZXItYSs1YmI4ZWExNEBleGFtcGxlLmNvbSIsImV4cCI6MTc4Mjc1NzIxMSwidXNlcl9pZCI6ImViMWJhMGM5LTM3NjAtNDQwOC1hMjY1LTA3YTJhMGVkNjY2OCJ9.zdnQP29nk_v4f9oen92lgw_I-G7riFwbBpzoSQXfm0Q
```

**User B token:**
```
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6ImF1ZGl0LXVzZXItYis2MmNkN2Q5MkBleGFtcGxlLmNvbSIsImV4cCI6MTc4Mjc1NzIxMiwidXNlcl9pZCI6IjBiZjRjNzI4LTIyZjMtNGU3YS1hNmQzLTgwYWI2MzJiOGZlOCJ9.k_hjBjEG94goVAAajvMgirTzptpIUOnvtEwI9hnWdeE
```

### Register request schema (from `services/backend/internal/domain/user.go`)

```jsonc
// RegisterRequest — all required
{ "email": "<valid email>", "password": "<min 8 chars>", "first_name": "...", "last_name": "..." }
// LoginRequest
{ "email": "<valid email>", "password": "..." }
```

---

## Health-check evidence (observed 2026-06-28)

| Probe | Command / target | Observed | Expected | Result |
|-------|------------------|----------|----------|--------|
| backend reachable | `POST /api/v1/auth/login` (bad creds) | `401` | 401 | PASS |
| frontend | `GET http://localhost:5173/` | `200` | 200 | PASS |
| db | `docker compose ... ps` | `recipe-db-1` `Up (healthy)` | healthy | PASS |

`docker compose ps` output:

```
NAME                IMAGE                COMMAND                  SERVICE    STATUS
recipe-app-1        recipe-app           "air"                    app        Up (0.0.0.0:18080->8080/tcp)
recipe-db-1         postgres:18-alpine   "docker-entrypoint.s…"   db         Up (healthy) (0.0.0.0:15432->5432/tcp)
recipe-frontend-1   recipe-frontend      "docker-entrypoint.s…"   frontend   Up (0.0.0.0:5173->5173/tcp)
```

Registration/login evidence: both registers returned `201`-style JSON bodies with `id`; both logins
returned a `token` + `user` object. User IDs in the JWT `user_id` claim match the registration `id`.

---

## Limitations / INCONCLUSIVE notes

- None for P1.1. All probes ran successfully; stack confirmed healthy and two audit users provisioned.
- Tokens are time-limited (`exp` claim). Later phases that find a `401` on a previously-valid token
  should re-login rather than treat it as a defect.
- `recipe-app-1` (backend) and `recipe-frontend-1` have no Docker healthcheck defined, so `ps` shows
  no `(healthy)` marker for them; liveness was instead confirmed via HTTP probes above (401 / 200).
