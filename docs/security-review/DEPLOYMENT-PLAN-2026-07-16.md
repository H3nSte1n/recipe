# Recipe App — Production Deployment Plan (2026-07-16)

Follow-up to `REMEDIATION-STATUS-2026-07-16.md`. That doc confirmed the app is safe to launch
behind the VPN and listed what's still open. This doc covers deploying the app onto the actual
server (`recipe.steinhauer.dev`) and reconciles the "unverified deployment topology" item against
what's really running there.

**Launch decision:** Tailscale-only first (matches the existing `cockpit.steinhauer.dev` pattern —
`allow 100.64.0.0/10; deny all` in nginx), opened to the public internet later once Phase 5 lands.

**Status as of 2026-07-18:** Phase 0 and Phase 5 (all app-repo/code work) are implemented, reviewed,
and **merged to `main`** as PR #47 (commit `84aa239`). Nothing has touched the server yet — Phase
1–4 (the actual server deploy) haven't started. See "Implementation status" below for the full
breakdown and what's still open.

---

## Server reconciliation (checked 2026-07-16)

The recipe app is **not deployed anywhere on the server yet** — no container, no nginx vhost, no
listening port. Everything below is about the infra that's *available* to deploy into.

| REMEDIATION-STATUS item | Server reality |
|---|---|
| Deployment topology unverified | nginx is the standing reverse proxy for every other app on the box (steinhauer.dev, cockpit, nextcloud, tools, prices, huggingface), each TLS-terminated via Certbot. Recipe would follow the same pattern but isn't wired up — so "not blocking *if* a reverse proxy fronts it" is currently **false** until Phase 2 below is done. |
| Plaintext HTTP / no TLS | Infra-ready (nginx + certbot pattern exists), not yet applied to recipe. |
| Postgres `sslmode=disable` | Conditionally fine. `nextcloud-db` runs the same shape (`postgres:15-alpine`, port `5432/tcp` docker-internal only, nothing on the host per `ss -tlnp`). If recipe's Postgres never publishes a host port, DB traffic never leaves the docker bridge and `sslmode=require` buys near-zero benefit. **Caveat: see Risk 1 below** — this is not a one-time check. |
| No security headers | Infra-ready but unapplied. nginx.conf has a global fallback (`X-Frame-Options`, `nosniff`, `Referrer-Policy`, `Permissions-Policy`) but no HSTS at that level — every existing vhost sets HSTS individually and recipe's needs to too. |
| `gin.Default()` trusts all proxies | Real gap, and it's a **blocking prerequisite** for infra-level rate-limiting (Phase 3) to mean anything once nginx sits in front. |
| No rate-limiting/lockout on auth | Backend-scope. Infra can add a stopgap (nginx `limit_req`, same pattern as cockpit's login endpoint) but this doesn't replace app-level lockout. |
| Vite dev server in prod image | Not infra — confirmed no separate build step exists anywhere on the server; whatever image ships is what runs. |
| JWT revocation | Pure backend code, no infra angle. |
| Tailscale Funnel/Serve bypassing the ACL | Checked — `tailscale funnel status` / `serve status` both report "No serve config." Not currently a risk, but see Risk 5 below for why this needs re-checking, not just checking once. |

---

## Deployment phases

### Phase 0 — Prerequisites (app repo, before touching the server) — ✅ merged
1. **Frontend production build** — multi-stage Dockerfile (`vite build` → static serve), replacing
   `npm run dev`. Blocks Phase 2. Also clears the moderate esbuild advisory.
   **✅ Merged.** Multi-stage build (`node` → `nginx:alpine` static serve) with SPA fallback
   routing and an `/api/` proxy to the backend. Verified with an actual `docker build` +
   running-container test.
2. **`gin.Default()` → explicit trusted-proxy list** pinned to the real proxy's address. Blocks
   Phase 3 — rate-limiting is spoofable via `X-Forwarded-For` until this lands.
   **✅ Merged, and the underlying topology bug is now actually fixed** (code review caught that
   the first pass didn't matter — see "Code review findings" below). `docker-compose.yml`'s `app`
   service no longer publishes a host port at all; it's reachable only through the frontend's
   nginx, which sits on a fixed address (`172.28.0.10`) on a dedicated `app_net` network.
   `TRUSTED_PROXIES` is set to that address by default. Production (host nginx on loopback) should
   set it to `127.0.0.1` instead, per the code comment. Live-verified: forged `X-Forwarded-For`
   from an untrusted container is ignored; from the trusted address, each distinct forwarded IP
   gets its own independent rate-limit bucket.
3. Confirm `docker-compose.yml` (prod) does **not** publish Postgres's port to the host — same
   shape as `nextcloud-db`.
   **✅ Confirmed already true** — no code change needed. `docker-compose.yml` has no port mapping
   for `db`, with an explanatory comment already in place.

### Phase 1 — Container deploy (devops)
- Stage `docker-compose.yml` to the server (tracked files only, `.env` preserved, diff before
  staging — same process used for `agentic-assistant`).
- Backend binds `127.0.0.1:<port>` only — never `0.0.0.0`.
- Postgres: no published port, same compose network as backend only.
- `.env` (`DB_PASSWORD` etc.) at `chmod 600`, owner `agentops`.

### Phase 2 — nginx vhost + TLS (devops)
- `/etc/nginx/sites-available/recipe.steinhauer.dev`, modeled on the `cockpit` vhost
  (`agentic-os`): `:80` → 301 redirect, Certbot cert, HSTS + security headers (**CSP written fresh
  for recipe, not copied from cockpit — see Risk 3**), `allow 100.64.0.0/10; deny all;`,
  `proxy_pass` to the backend's loopback port.
- Full Config Edit Workflow: diff shown, explicit confirmation, backup, `nginx -t`, reload, verify.

### Phase 3 — Stopgap rate-limiting at the proxy (devops)
- `limit_req_zone` on `/api/v1/auth/login` and `/api/v1/auth/register`, same shape as cockpit's
  `cockpit_login` zone (`rate=20r/m`, `burst=5 nodelay`).
- **Only deploy after Phase 0.2 lands** — see Risk 2.

### Phase 4 — fail2ban jail (devops)
- Jail for repeated recipe-app auth failures, alongside `sshd`/`nextcloud`/`changedetection`.
- **Requires backend to log failed-auth attempts in a fail2ban-parseable format** — see Risk 4.

### Phase 5 — Backend hardening (out of devops scope) — ✅ merged
1. **Rate-limiting/lockout + email verification on auth** (do before moving off Tailscale-only).
   **✅ Merged.**
   - Rate-limiting/lockout: 5 failed logins → 15-minute lock, atomic counter (see code review
     finding #3 below — the first version had a race letting concurrent attempts bypass this
     entirely), per-IP rate limits on login/register/forgot-password (429). All failure modes
     (wrong password, locked account, nonexistent email) return the same generic 401 — no
     account-existence signal (see finding #6).
   - Email verification: reused the existing SMTP `EmailService` (already used for password
     resets) rather than building a new mailer. Verify/resend endpoints, `RequireVerified`
     middleware gates write routes only (reads stay open to unverified users — confirmed as the
     desired behavior). Resend always returns the same generic response regardless of outcome
     (see finding #4). **Still needs:** a real SMTP provider + credentials (currently placeholder
     Gmail creds in dev config — provider choice is your call, not made yet), and a frontend
     `/verify-email` page (doesn't exist yet, mirrors an existing gap on `/reset-password`).
2. **JWT revocation on password reset/account deletion.**
   **✅ Merged.** New `token_revocations` table (no FK to `users`, deliberately, so a revocation
   outlives account deletion), wired into both password-reset and the pre-existing
   account-deletion flow. **Note:** because this also adds `iss`/`aud` validation (item 3), every
   token issued before this deploys will be rejected on rollout — expected, not a bug, but plan for
   a forced re-login the first time this ships (not relevant for the very first launch, no users
   yet; relevant for any future redeploy of an already-live instance).
3. **Batch: `iss`/`aud`/`iat`/`nbf` claims, request-body-size limit, pagination validation,
   error-response leaks.**
   **✅ Merged.** `iss`/`aud`/`iat`/`nbf` claims added and validated; global JSON body-size limit
   (2 MiB, with a 20 MiB backstop for multipart-labeled requests after finding #2 below);
   negative-offset/unbounded-`LIMIT` pagination now rejected/capped; a raw Postgres error leak in
   `ai_config_handler.go` is sanitized (this fix also caught and corrected a related bug where
   `GetByID` was swallowing `gorm.ErrRecordNotFound` into a generic error). Cross-tenant reads on
   recipes/AI-configs return 404 (not 403 — see finding #5) so they're indistinguishable from a
   genuinely missing resource.

---

## Code review findings (PR #47) — all fixed before merge

The integration branch went through an 8-angle finder pass + independent verification before
merging. 6 confirmed correctness/security bugs, all fixed and re-verified (full test suite +
live smoke tests) prior to merge:

1. **Trusted-proxy default didn't match the actual docker-compose topology.** The backend
   published its port directly to the host, bypassing the frontend proxy entirely — so the
   `TRUSTED_PROXIES=127.0.0.1` default from Phase 0.2 never matched anything, and per-IP rate
   limiting collapsed into one shared bucket for every client (a single attacker could exhaust the
   login/register/forgot-password budget for everyone). **Fixed** by removing the direct port
   publish and giving the frontend a fixed network address to trust instead — see Phase 0.2 above.
2. **Global body-size limit was fully bypassable** by spoofing a `multipart/form-data`
   Content-Type on any JSON route (`ShouldBindJSON` doesn't check Content-Type). **Fixed**: falls
   back to a 20 MiB backstop for multipart-labeled requests instead of no limit at all.
3. **Failed-login counter was a non-atomic read-modify-write**, letting concurrent brute-force
   requests bypass account lockout (all racing requests would read the same stale count and write
   back the same incremented value). **Fixed**: single atomic `UPDATE ... RETURNING` statement;
   added a concurrency test firing 20 parallel failed logins and asserting all 20 are counted.
4. **ResendVerification leaked which emails are registered-and-unverified** by returning 429 +
   raw internal error text only for that case, while every other input got a generic 200.
   **Fixed**: always resolves to the same generic response; failures logged server-side only.
5. **Cross-tenant reads returned 403 vs 404**, letting an attacker enumerate valid AI-config/recipe
   IDs by observing which status came back — a regression of a uniform-404 response the pre-PR
   code used deliberately to prevent exactly this. **Fixed**: both `GetByID` paths return
   `ErrNotFound` for cross-tenant access.
6. **Login returned a distinct 423 for locked accounts** vs 401 for everything else, letting an
   attacker who already has a candidate email confirm it's registered by locking it and checking
   for 423. **Fixed** (by your choice, over keeping the more informative UX): same generic 401 in
   all cases.

---

## Security risks in this approach, and the tasks that remove them

### Risk 1 — Docker port-publishing bypasses ufw
Docker inserts its own iptables rules (`DOCKER` chain) for published container ports, ahead of
ufw's `INPUT` chain filtering. `ufw deny incoming` does **not** protect a container that publishes
`0.0.0.0:5432` — the Postgres `sslmode=disable` decision (and the "internal-network-only" claim in
REMEDIATION-STATUS) depends entirely on the port never being published, with no firewall backstop
if that regresses.

- [ ] **Task:** Live-test the assumption once — briefly publish a throwaway container's port and
      confirm ufw does *not* block it, so the failure mode is understood rather than assumed.
- [ ] **Task:** Add a check (manual review step or a small script) that runs on every
      `docker-compose.yml` change to prod, asserting Postgres has no `ports:` mapping. This is a
      standing check, not a one-time deploy-day verification.
- [ ] **Task:** Revisit `sslmode=disable` immediately if the port-publishing check ever fails, or
      if Postgres moves to a different host/managed instance.

### Risk 2 — nginx rate-limiting is spoofable without the trusted-proxy fix — ✅ resolved (app level)
Phase 3's `limit_req_zone $binary_remote_addr` keys on the client IP Gin reports. Until
`gin.Default()`'s trust-all-proxies default is replaced with an explicit trusted-proxy list
pinned to the real proxy, an attacker can set `X-Forwarded-For` to bypass the per-IP limit
entirely — and the limiter will look active in config while doing nothing. This risk turned out to
be live in the app itself (not just the nginx stopgap): PR #47's code review caught that the
initial trusted-proxy fix didn't match the actual docker-compose topology (see "Code review
findings" #1 above), which has now been fixed and live-verified.

- [x] **Task:** Land the trusted-proxy fix (Phase 0.2) and verify it — done, and re-verified after
      the topology bug was caught and fixed (forged `X-Forwarded-For` from an untrusted source is
      ignored; honored from the trusted proxy address).
- [ ] **Task:** Still applies to Phase 3 specifically — do not deploy nginx's `limit_req` stopgap
      until the production vhost's `TRUSTED_PROXIES` value (should be `127.0.0.1` for host nginx,
      per Phase 0.2's code comment) is confirmed to actually match where nginx runs in prod.

### Risk 3 — CSP copied from cockpit weakens XSS protection for AI-rendered content
Cockpit's CSP includes `script-src 'self' 'unsafe-inline' 'unsafe-eval'` for its own reasons.
Recipe renders AI-generated recipe content (`ai_config_handler.go`, `ai_models`) — if that content
is ever rendered without escaping, `unsafe-inline`/`unsafe-eval` turns a stored-XSS bug in the
backend into an actually-exploitable one instead of a CSP-blocked no-op.

- [ ] **Task:** Write recipe's CSP from what the frontend actually needs (check the built bundle
      for inline scripts/styles and eval usage) rather than reusing cockpit's directive set.
- [ ] **Task:** Audit how AI-generated recipe text is rendered on the frontend (dangerouslySetInnerHTML
      or equivalent) before finalizing the CSP — if it's plain-text rendered, `unsafe-inline` may be
      unnecessary entirely.

### Risk 4 — fail2ban jail can be a silent no-op
Phase 4 adds a jail definition, but jails only work if the backend emits failed-auth log lines in
a format fail2ban can parse. Adding the jail without checking this gives a false sense of
protection — it'll show as configured and do nothing.

- [ ] **Task:** Confirm the backend logs failed login attempts (with source IP) in a
      fail2ban-parseable format before writing the jail's filter regex.
- [ ] **Task:** After deploying the jail, trigger a few deliberate failed logins and confirm
      `fail2ban-client status recipe` shows a ban — don't trust the config alone.

### Risk 5 — Tailscale-only relies on network ACL, not authentication
`allow 100.64.0.0/10; deny all` restricts to *anyone on the tailnet*, not authenticated app users.
A compromised tailnet member's device can still reach and brute-force the app — this is the exact
scenario REMEDIATION-STATUS flags. Phase 3/5 mitigate it but don't eliminate it. Separately,
Tailscale Funnel/Serve — checked clear today — is a standing risk if anyone with tailnet admin
access enables it later, since it would bypass the nginx ACL and ufw entirely.

- [ ] **Task:** Treat Phase 5 (rate-limiting/lockout, email verification) as required — not
      optional hardening — before considering the tailnet a sufficient trust boundary long-term.
- [ ] **Task:** Add a periodic check (could ride along with the existing health-check cadence) for
      `tailscale funnel status` / `tailscale serve status` on this vhost, alongside the other apps
      that rely on the same ACL pattern (cockpit).

---

## Open input needed
- None currently blocking — domain (`recipe.steinhauer.dev`) and launch scope (Tailscale-only)
  are confirmed.

---

## Open TODOs (as of 2026-07-18)

### Integration — ✅ done
All five backend branches + the frontend branch were merged into an integration branch, with the
`NewRouter(...)` signature conflict resolved and the triple `000016` migration collision renumbered
(`token_revocations` kept 16, lockout became 17, email verification became 18 — verified up → down
→ up against a scratch Postgres, in order). Full test suite green on the integrated result. PR #47
opened, code-reviewed (see findings above), fixed, CI green, merged to `main` as `84aa239`.

### Before Phase 5 is actually "live" (not just merged)
- [ ] **Pick an email provider + credentials** for the verification/reset emails — currently
      placeholder Gmail creds only. Your call, not made yet.
- [ ] **Build the frontend `/verify-email` page** (and check whether `/reset-password` has the
      same gap — the email-verification agent noted it looked unbuilt too).
- [ ] **Warn/plan for forced re-login on first deploy** of the JWT `iss`/`aud` validation, once
      real users exist (not relevant for the very first launch — no users yet — but relevant for
      any future redeploy of an already-live instance).

### Server work (Phase 1–4, not started, stays interactive with me via `/devops` — not delegated)
- [ ] Phase 1: container deploy (backend loopback-only bind, Postgres no published port, `.env`
      permissions).
- [ ] Phase 2: nginx vhost + TLS for `recipe.steinhauer.dev` (HSTS, security headers, CSP
      **written fresh, not copied from cockpit** — see Risk 3), Tailscale-only ACL.
- [ ] Phase 3: nginx stopgap rate-limiting on `/api/v1/auth/*` (redundant with the now-implemented
      app-level rate-limiting, but still worth having as defense-in-depth per the original plan).
- [ ] Phase 4: fail2ban jail for repeated recipe-app auth failures (needs the backend's failed-auth
      logging format confirmed first — see Risk 4).

### Standing risk-mitigation tasks (from the "Security risks" section, still open)
- [ ] Risk 1: live-test that Docker port-publishing actually bypasses `ufw` (understand the failure
      mode, don't just assume it), and add a standing check that Postgres never gets a `ports:`
      mapping in prod.
- [ ] Risk 3: audit how AI-generated recipe content is rendered on the frontend before finalizing
      the CSP (may make `unsafe-inline` unnecessary).
- [ ] Risk 5: add a periodic check for Tailscale Funnel/Serve being enabled on this vhost, riding
      along with the existing health-check cadence.
