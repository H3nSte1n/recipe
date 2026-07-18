# Recipe App — Production Deployment Plan (2026-07-16)

Follow-up to `REMEDIATION-STATUS-2026-07-16.md`. That doc confirmed the app is safe to launch
behind the VPN and listed what's still open. This doc covers deploying the app onto the actual
server (`recipe.steinhauer.dev`) and reconciles the "unverified deployment topology" item against
what's really running there.

**Launch decision:** Tailscale-only first (matches the existing `cockpit.steinhauer.dev` pattern —
`allow 100.64.0.0/10; deny all` in nginx), opened to the public internet later once Phase 5 lands.

**Status as of 2026-07-18:** All 5 phases are done. Phase 0 and Phase 5 (app-repo/code work) are
merged to `main` as PR #47 (commit `84aa239`), plus a follow-up production-build commit
(`0ef83b3`..`e8de96c`, see Phase 1 notes below). Phase 1–4 (the actual server deploy) are live:
`recipe.steinhauer.dev` is up, Tailscale-only, verified end-to-end from a real tailnet client. See
"Deployment phases" below for what each phase actually involved, including three bugs only found
by deploying for real (subnet collision, DNS, trusted-proxy topology).

**Write path was blocked, now resolved for the real launch scope.** All verification through Phase
1–4 covered only the unauthenticated surface (static assets, login/register responses, rate
limiting). Checked the actual write path directly: registered a test user, logged in, attempted
`POST /api/v1/recipes` — blocked with `"email verification required before performing this
action"` (`RequireVerified` middleware, see Phase 5 notes). SMTP is unconfigured (blank, per the
Phase 1 `.env`), so no verification email can ever be sent through it.

Investigated email options (see below) but the actual user base is 2 known people (Henry, Johannes)
— not a public signup flow — so **the decision was to skip SMTP entirely for now** rather than set
up a provider for 2 users. Both accounts were created directly: registered through the real
`/api/v1/auth/register` endpoint (reuses all existing validation/hashing), then `email_verified_at`
set directly via `UPDATE users ... ` on `recipe-db-1` to clear the `RequireVerified` gate. Verified
write access works post-verification (`POST /api/v1/recipes` no longer 403s, just needs valid
fields). **Email provider setup (below) is deferred, not abandoned** — needed if/when this ever
opens beyond these 2 users, or if either of them needs self-service password reset (forgot-password
uses the same unconfigured `EmailService`, so that flow is still dead until SMTP is set up — for
now, a password reset for either user means asking me to do the same direct-DB-update trick again).

**Email provider investigation, for whenever it's needed:**
- The server's local `postfix` is not directly usable — it's a local MTA for system mail (fail2ban
  alerts etc.), not an authenticated SMTP submission endpoint, and the Go backend's
  `net/smtp.SendMail` always does `PlainAuth`.
- The existing `steinhauer.dev` mail hosting (Namecheap Private Email) *does* work — `hello@`
  delivers successfully — but `henry@`/`www-data@` are broken there (mailbox not verified/set up on
  Namecheap's side, unrelated to this app). A dedicated mailbox (e.g. `recipe@steinhauer.dev`) on
  the same plan, using Private Email's real SMTP submission (`smtp.privateemail.com:587`), would
  need to be created in the Namecheap control panel — outside what's reachable over SSH.
- Recommended alternative if/when needed: a dedicated transactional provider (Resend suggested —
  generous free tier, simple SMTP creds) rather than debugging the Namecheap mailbox setup, since
  neither `email.go` nor the deploy process cares which one it is.

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

### Phase 1 — Container deploy (devops) — ✅ done 2026-07-18
- Discovered the backend only had a dev-mode Dockerfile (`air` hot-reload, source bind-mounted).
  Added `Dockerfile.prod` (compiled static binary, migrations auto-run via `database.MigrateDB` on
  boot as in dev), `env.production.yaml.sample` (no secrets — every sensitive field is overridden
  by its bound env var at container start), and `docker-compose.prod.yml` (override: frontend
  published to `127.0.0.1:3200` only via `ports: !override`, backend's source bind-mount replaced
  with the compiled image via `volumes: !override`, `no-new-privileges`/`cap_drop: ALL`/resource
  limits matching `agentic-assistant`'s pattern). All verified locally (build, boot, migrate,
  end-to-end curl) before ever touching the server. Commits `0ef83b3`, `5c55e98`, `196650a`.
- **Topology decision, confirmed with the user first:** host nginx talks only to the frontend
  container; the frontend's own nginx (`services/frontend/nginx.conf`) proxies `/api/` to the
  backend internally. The backend is never published, not even to loopback — simpler than the
  original plan's "backend binds `127.0.0.1:<port>`" and matches what the frontend's nginx was
  already built for in Phase 0.
- **Bug found staging to the server:** `app_net`'s `172.28.0.0/24` collided with an existing
  docker network on the host (`10_default`). Moved to `172.30.0.0/24` (commit `29623ee`) — free,
  confirmed via `docker network inspect` on every existing network first.
- Staged via `git archive HEAD` piped over SSH to `/home/henry/recipe` (owned `agentops`, matching
  `agentic-assistant`'s convention). `.env` (`DB_PASSWORD`/`JWT_SECRET`/`SECURITY_ENCRYPTION_KEY`,
  freshly generated) at `chmod 600`. `uploads/` chowned to uid 1000 to match the container's
  non-root user (the bind-mount masks the image's own chown). Postgres: no published port. All
  three containers (`recipe-db-1`, `recipe-app-1`, `recipe-frontend-1`) verified healthy, end-to-end
  curl through frontend → backend returned expected status codes.

### Phase 2 — nginx vhost + TLS (devops) — ✅ done 2026-07-18
- `/etc/nginx/sites-available/recipe.steinhauer.dev`, modeled on the `cockpit` vhost
  (`agentic-os`): `:80` → 301 redirect, Certbot cert (`certbot certonly --nginx`, not the installer
  plugin — vhost was hand-written to match cockpit's structure exactly rather than let certbot
  generate it), HSTS + security headers, CSP written fresh (see Risk 3 — resolved), `allow
  100.64.0.0/10; deny all;` on the `:443` block only, `proxy_pass` to the frontend's loopback port
  `127.0.0.1:3200`, `client_max_body_size 20m` to match the backend's multipart upload backstop.
- **Two bugs found only by actually testing from a real client**, neither in the vhost itself:
  1. `recipe.steinhauer.dev` publicly resolves to the server's public IP — same as every other
     vhost on this box. The "Tailscale-only" ones (`cockpit`) actually rely on a **Split DNS**
     override in a dnsmasq instance bound to the server's tailnet interface, registered in the
     Tailscale admin console for the whole `steinhauer.dev` zone: tailnet clients get the tailnet
     IP, everyone else gets the public IP that nginx's ACL then denies. `cockpit.steinhauer.dev`
     had an explicit override line; `recipe.steinhauer.dev` didn't, so it fell through to public
     DNS. Fixed by adding the same `address=/recipe.steinhauer.dev/100.87.135.126` line to
     `/etc/dnsmasq.conf`. **Gotcha discovered in the process: `systemctl reload dnsmasq` does not
     pick up new `address=` entries — a full `restart` is required.** Documented in the state file
     so it isn't rediscovered next time.
  2. Even with DNS fixed, the user's browser still got denied — turned out to be the browser's
     **DNS-over-HTTPS ("Secure DNS")** setting bypassing the OS/Tailscale resolver entirely.
     Disabling it in-browser fixed it; this is a per-user gotcha, not a server-side issue.
- Verified end-to-end from both a non-tailnet client (TLS ok, all headers present, 403 as expected)
  and a real Tailscale client (200, correct page loads).

### Phase 3 — Stopgap rate-limiting at the proxy (devops) — ✅ done 2026-07-18
- `limit_req_zone` on `/api/v1/auth/login` and `/api/v1/auth/register`, same shape as cockpit's
  `cockpit_login` zone (`rate=20r/m`, `burst=5 nodelay`). Deployed after Phase 0.2 and the
  trusted-proxy fix below — see Risk 2.
- Validated the zone/location syntax via `nginx -t`; didn't separately burst-test nginx's own
  `limit_req` live (well-trodden nginx functionality, config-checked), but did confirm the
  **app-level** per-IP limiter still fires correctly (5 requests then `429`).

### Phase 4 — fail2ban jail (devops) — ✅ done 2026-07-18
- New `recipe` jail (`filter.d/recipe.conf` + `jail.d/recipe.conf`), same shape as the existing
  `changedetection` jail: `maxretry=5`, `findtime=600`, `bantime=3600`.
- **Deliberately reads nginx's access log, not the app container's stdout** — nginx already logs
  the real client IP correctly (no docker-NAT complication at that layer), so there was no need to
  solve the harder problem of getting `docker logs` output into a fail2ban-parseable file. Filter
  matches `401` on `POST /api/v1/auth/login` in `/var/log/nginx/access.log`. Validated with
  `fail2ban-regex` against the real log before enabling (correctly matched the 2 real failed-login
  attempts made during Phase 2/3 testing, 0 false positives across 460 other lines).

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
- [x] **Task:** Confirmed against the real production topology — and this surfaced a second,
      genuinely new bug the docker-only testing above couldn't have caught. `TRUSTED_PROXIES` was
      `172.30.0.10` (the frontend's fixed IP) — correct for a single hop, but production adds a
      second one: host nginx → frontend's *published* port → frontend nginx → backend. Docker NAT
      rewrites the source IP of that first hop to the `app_net` gateway (`172.30.0.1`), so the
      frontend's nginx appended the gateway IP to `X-Forwarded-For` instead of host nginx's
      identity. A real login through the full chain arrived at the backend reporting client
      `172.30.0.1` — every request would have collapsed into one bucket, making both the app-level
      limiter and Phase 3's `limit_req_zone` useless (and Phase 4's fail2ban would ban the gateway,
      i.e. everyone or no one). **Fixed**: widened `TRUSTED_PROXIES` from the single frontend IP to
      the whole `172.30.0.0/24` subnet (a private, single-purpose network — frontend/app/db only),
      redeployed, re-verified the real Tailscale client IP is now logged correctly end-to-end.

### Risk 3 — CSP copied from cockpit weakens XSS protection for AI-rendered content — ✅ resolved
Cockpit's CSP includes `script-src 'self' 'unsafe-inline' 'unsafe-eval'` for its own reasons.
Recipe renders AI-generated recipe content (`ai_config_handler.go`, `ai_models`) — if that content
is ever rendered without escaping, `unsafe-inline`/`unsafe-eval` turns a stored-XSS bug in the
backend into an actually-exploitable one instead of a CSP-blocked no-op.

- [x] **Task:** Write recipe's CSP from what the frontend actually needs (check the built bundle
      for inline scripts/styles and eval usage) rather than reusing cockpit's directive set. Done —
      grepped the frontend for `dangerouslySetInnerHTML`/`eval`/`new Function` (none found) and
      inspected the actual Vite production build's `index.html` (no inline `<script>` tags at all),
      so the final CSP's `script-src` omits both `unsafe-inline` and `unsafe-eval` — tighter than
      cockpit's.
- [x] **Task:** Audit how AI-generated recipe text is rendered on the frontend before finalizing
      the CSP. Done — no `dangerouslySetInnerHTML` anywhere in the frontend; recipe content
      (AI-imported or not) goes through normal JSX text interpolation, which React auto-escapes.
      Two things the CSP *does* need, found during the same audit: `style-src 'unsafe-inline'`
      (React's `style={{...}}` in `RecipeModal`/`AddRecipeModal`/`RecipeGraph` compiles to real
      inline `style=""` attributes, which CSP treats the same as any other inline style) and
      `img-src https:` broadly (`recipe.image_url` is free-form, populated from arbitrary external
      sites via URL import with no host allowlist in the backend — not just a fixed CDN).

### Risk 4 — fail2ban jail can be a silent no-op — ✅ resolved
Phase 4 adds a jail definition, but jails only work if the backend emits failed-auth log lines in
a format fail2ban can parse. Adding the jail without checking this gives a false sense of
protection — it'll show as configured and do nothing.

- [x] **Task:** Confirm the backend logs failed login attempts (with source IP) in a
      fail2ban-parseable format before writing the jail's filter regex. Turned out to be moot in a
      good way: nginx's own access log already records the correct client IP and status for
      `/api/v1/auth/login` (that layer isn't affected by the docker-NAT issue above — nginx is the
      first hop from the real client), so the jail reads `/var/log/nginx/access.log` rather than
      the app container's stdout, same pattern as the existing `changedetection` jail. Validated
      the filter with `fail2ban-regex` against the real log before enabling: matched exactly the 2
      real failed-login attempts from earlier testing, 0 false positives across 460 other lines.
- [x] **Task:** After deploying the jail, trigger a few deliberate failed logins and confirm
      `fail2ban-client status recipe` shows a ban — don't trust the config alone. Confirmed the
      jail counts failures correctly (`fail2ban-client status recipe` showed the 2 prior attempts
      as "Currently failed", correctly below the `maxretry=5` ban threshold); didn't push a live
      test all the way to an actual ban since that would have banned the tester's own Tailscale IP
      mid-deployment.

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

### Before Phase 5 is actually "live" for a public/self-service launch
Confirmed 2026-07-18 by testing the real write path: a freshly registered user cannot write
anything — not even delete their own account — until email verification actually works. **Not a
blocker for the current 2-user scope** (both accounts provisioned directly, see status note at the
top), but required before self-service registration or password reset is usable by anyone else.
- [ ] **Pick an email provider + credentials** for the verification/reset emails — currently
      blank in the deployed `.env` (Phase 1 left `SMTP_*` empty deliberately). Investigated
      options (see status note at top): local postfix isn't usable as-is, the existing Namecheap
      Private Email account works for `hello@` but not a dedicated app mailbox without setting one
      up in the Namecheap panel, Resend suggested as the path of least resistance if/when needed.
      Deferred, not decided against.
- [ ] **Build the frontend `/verify-email` page** (and check whether `/reset-password` has the
      same gap — the email-verification agent noted it looked unbuilt too). Same deferral as above
      — not needed until email verification is actually wired up.
- [ ] **Warn/plan for forced re-login on first deploy** of the JWT `iss`/`aud` validation, once
      real users exist (not relevant for the very first launch — no users yet — but relevant for
      any future redeploy of an already-live instance).
- [ ] **PII in production logs**: noticed during Phase 2 login testing — GORM logs full SQL
      queries including plaintext user emails (`SELECT * FROM "users" WHERE email = '...'`) to the
      `recipe-app-1` container's stdout. Not a launch blocker, but worth turning down GORM's log
      level (or scrubbing the query args) in the production config before this handles real users.

### Server work (Phase 1–4) — ✅ done 2026-07-18, all interactive via `/devops`, not delegated
`recipe.steinhauer.dev` is live, Tailscale-only, verified end-to-end from a real tailnet client
(correct client IP logged throughout the chain, rate-limiting and fail2ban both validated). See the
"Deployment phases" section above for what each phase actually involved — three real bugs were
found only by deploying for real: an `app_net` subnet collision with an existing docker network on
the host, a missing Split-DNS override (recipe wasn't in dnsmasq's `steinhauer.dev` overrides the
way cockpit was), and the `TRUSTED_PROXIES` topology gap described in Risk 2 above. The state file
(`~/.agents/state/devops/server.md`) has full detail on every container/vhost/jail added.

### Standing risk-mitigation tasks (from the "Security risks" section, still open)
- [ ] Risk 1: live-test that Docker port-publishing actually bypasses `ufw` (understand the failure
      mode, don't just assume it), and add a standing check that Postgres never gets a `ports:`
      mapping in prod.
- [x] Risk 3: done — see "Risk 3" above.
- [ ] Risk 5: add a periodic check for Tailscale Funnel/Serve being enabled on this vhost, riding
      along with the existing health-check cadence.
