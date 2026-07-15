# Phase 3 — Implicit Tailscale / Perimeter Dependencies

**Subtask:** Scan for controls that silently rely on the Tailscale VPN and break when it is
removed (app launches behind a small trusted-few tailnet; a compromised/malicious *member* is in
scope; the VPN is removed later for public-internet exposure).

**Scope/method:** Static review only — no production code modified. Grepped for source-IP trust
(`100.64`, `X-Forwarded`, `RemoteAddr`, `ClientIP`, `SetTrustedProxies`), auth/abuse controls
(`rate`, `limiter`, `lockout`, `captcha`, `verify`/email-confirm), bind/exposure
(`docker-compose.yml`, Dockerfiles, `vite.config.ts`, server `Run`/listen addr, CORS), and
management/debug endpoints (`pprof`, `expvar`, `/debug`, `admin`, `metrics`). Severities below are
rated **in the post-VPN-removal (public-internet) context** as instructed.

**Classification labels (from the plan):** `GO-LIVE-BEHIND-VPN-BLOCKER` (broken even today, with
the VPN still present) vs `BEFORE-VPN-REMOVAL` (only a hole once publicly reachable).

---

## Finding 1 — Database port published on all host interfaces with a static weak password and TLS disabled

- **Severity (post-VPN context):** CRITICAL
- **Classification:** `GO-LIVE-BEHIND-VPN-BLOCKER` (member-reachable today)
- **Evidence:**
  - `docker-compose.yml:35-36` — `ports: ["5432:5432"]` (publishes Postgres on `0.0.0.0:5432` on the host, i.e. every host interface including the tailnet IP).
  - `docker-compose.yml:38-39` — `POSTGRES_USER=postgres` / `POSTGRES_PASSWORD=your_password` (static, well-known placeholder password).
  - `docker-compose.yml:28` and `env.development.yaml.sample` (`ssl_mode: disable`) — DB connections are plaintext.
- **Why the VPN masks it / why it is a BEHIND-VPN blocker:** The application container talks to Postgres over the internal Docker network (`DB_HOST=db`, `db:5432`), so the published host port `5432:5432` is gratuitous for the app to function. With the port published on `0.0.0.0`, it is reachable on the host's tailnet address. The threat model puts a *compromised/malicious tailnet member* in scope — such a member can connect to `host:5432` and authenticate with the known `your_password`, dumping/altering the entire database, **today, with the VPN still in place.** Removing the VPN escalates this to the entire internet. This is the rare item that is already broken behind the perimeter.
- **Recommended control:** Remove the `5432:5432` mapping entirely (the app reaches the DB over the compose network). If a host port is genuinely needed, bind it to loopback (`127.0.0.1:5432:5432`), set a strong unique password from a secret store, and enable TLS (`ssl_mode=require`/`verify-full`).

## Finding 2 — No rate-limiting / lockout / CAPTCHA / brute-force protection on auth endpoints

- **Severity (post-VPN context):** HIGH
- **Classification:** `BEFORE-VPN-REMOVAL`
- **Evidence:**
  - `internal/router/router.go:64-71` — public `/auth/register`, `/auth/login`, `/auth/forgot-password`, `/auth/reset-password` with no throttling middleware.
  - `internal/middleware/` contains only `auth.go`, `context.go`, `cors.go` — no limiter; grep for `rate|limiter|lockout|throttle|captcha|brute` found no production code.
- **Why the VPN masks it:** The tailnet's small, authenticated, trusted-few membership is the de-facto control bounding who can even reach these endpoints, so online password brute-force / credential-stuffing / reset-token guessing / SMTP-flooding are all rate-limited by "only members can connect." Once public, bcrypt cost-14 raises per-attempt cost but does not bound attempt *volume*; the network was the abuse control.
- **Recommended control:** Add IP- and account-scoped rate limiting with exponential backoff / temporary lockout on login/forgot/reset, and per-target-email throttling on forgot-password (already tracked in `recipe-remediation-post-vpn.md` Phase 1). See Finding 6 — the IP key must not be spoofable.

## Finding 3 — Open registration with no email verification

- **Severity (post-VPN context):** HIGH
- **Classification:** `BEFORE-VPN-REMOVAL`
- **Evidence:**
  - `internal/service/user_service.go:62-90` — `Register` creates the user + profile immediately; no verification token, no pending/unverified state.
  - Grep for `verif|confirm.*email|activat` in `internal/` found only unrelated ownership/signed-URL `Verify` calls — no email-confirmation flow exists.
  - `internal/router/router.go:66` — `/auth/register` is fully public.
- **Why the VPN masks it:** Behind the tailnet only vetted members can register at all, so unverified/throwaway accounts are not a realistic abuse vector. Account creation is implicitly gated by network membership. On the public internet, anyone can mint unlimited accounts (each of which is an authenticated actor that can drive the SSRF/AI/upload surfaces and consume paid LLM calls), and there is no proof the email belongs to the registrant.
- **Recommended control:** Require email verification before the account is usable (or at minimum before privileged actions), combined with registration rate-limiting (Finding 2). Track as a BEFORE-VPN-REMOVAL item.

## Finding 4 — Backend binds all interfaces and is served as plaintext HTTP with no external TLS

- **Severity (post-VPN context):** HIGH
- **Classification:** `BEFORE-VPN-REMOVAL`
- **Evidence:**
  - `cmd/api/main.go:76` — `r.Run(":" + cfg.App.Port)` (Gin listens on `0.0.0.0:8080`, all interfaces, plain HTTP — no `RunTLS`, no cert config anywhere in the repo).
  - `docker-compose.yml:17-18` — `ports: ["8080:8080"]` publishes the API directly on the host.
  - `env.development.yaml.sample` — `storage.base_url: http://localhost:8080/uploads`, `frontend.url: localhost:5173` (HTTP throughout; no HTTPS scheme).
- **Why the VPN masks it:** Tailscale provides WireGuard transport encryption between members, so the lack of application/edge TLS is invisible behind the tailnet — Bearer JWTs and login credentials traverse an already-encrypted tunnel. Remove the VPN and the same plaintext HTTP carries credentials and the localStorage JWT in the clear across the public internet (sniffable / MITM-able).
- **Recommended control:** Terminate TLS at a reverse proxy / load balancer in front of the app before public exposure; serve the SPA and API over HTTPS; set `base_url`/origins to `https://`. Add HSTS (see Finding 7).
- **INCONCLUSIVE caveat:** This is conditional on the deployment topology — see "Inconclusive items" below. The repo ships no reverse-proxy/TLS manifest, so if production fronts the app with an HTTPS terminator this is already mitigated at the edge.

## Finding 5 — `GET /users/list` lets any authenticated caller enumerate all members' names/IDs (email leak already remediated)

- **Severity (post-VPN context):** LOW *(corrected during this audit's Phase 3 — see below;
  originally logged here as MEDIUM/email-enumeration, which is now stale)*
- **Classification:** `BEFORE-VPN-REMOVAL`
- **Correction:** This finding originally cited `docs/security-review/REPORT.md` (Low #15), which
  described `/users/list` as leaking every member's **email address**. Re-reading the current code
  during Phase 3's fresh auth review (`03-auth.md`) shows this is **already fixed**:
  `internal/service/user_service.go:218-235` (`ListAll`) explicitly projects to
  `domain.UserSummary{ID, FirstName, LastName}` — no email field — with an in-code comment stating
  the intent ("non-PII summary so the list endpoint cannot be used to enumerate every member's
  email address"). The "harvest every user's email address" claim below is **stale/incorrect** and
  should not be carried into the Phase 7 consolidated report as a live finding.
- **Evidence (current, corrected):**
  - `internal/router/router.go:81` — `users.GET("/list", r.handlers.UserHandler.ListAll)` inside
    the JWT-protected group, with no role/admin gate (this part is still accurate).
  - `internal/service/user_service.go:218-235` — response is a non-PII `UserSummary` projection
    (`id`, `first_name`, `last_name` only); no email, password hash, or other PII is returned.
- **Residual issue (downgraded, not eliminated):** any authenticated caller can still enumerate
  every member's **name and internal user ID** with no role/admin gate. This is real but
  lower-impact than the original email-enumeration claim — names/IDs alone are a much weaker
  target for spam/credential-stuffing than emails, though ID enumeration could still assist an
  attacker chaining it with another endpoint that accepts a user ID.
- **Why the VPN masks the residual issue:** "Any authenticated user" today means "a trusted tailnet
  member," so member-to-member name/ID visibility within the small known group is low-impact and
  defensible. On the public internet, any self-registered account (Finding 3) becomes an
  authenticated caller that can enumerate every member's name and ID — the network membership was
  the de-facto authorization for even this reduced exposure.
- **Recommended control:** Restrict to an admin role, or remove the endpoint if it has no current
  product use, now that the higher-severity email leak is already closed. Re-evaluated in
  `03-idor.md` for the authorization angle; here it is flagged specifically as a
  perimeter-dependent exposure.

## Finding 6 — `gin.Default()` trusts all proxies: the future IP-scoped rate limiter (the VPN's replacement control) is spoofable on day one

- **Severity (post-VPN context):** MEDIUM (latent — depends on Finding 2's limiter landing)
- **Classification:** `BEFORE-VPN-REMOVAL`
- **Evidence:**
  - `internal/router/router.go:21` — `engine := gin.Default()` with no `engine.SetTrustedProxies(...)` call anywhere. Gin defaults to trusting all proxies, so `c.ClientIP()` is derived from the client-controlled `X-Forwarded-For` header.
  - Grep confirms `ClientIP`/`RemoteAddr`/`X-Forwarded` are **not** used in any production code path today (only in `pkg/urlparser/ssrf_test.go`).
- **Why this is the cleanest perimeter dependency:** It is harmless *now* because nothing keys decisions on client IP. But the `recipe-remediation-post-vpn.md` Phase 1 rate limiter — the control explicitly intended to *replace* the VPN's abuse protection — will key on `c.ClientIP()`. With trust-all proxies, an attacker sets a fresh `X-Forwarded-For` per request and bypasses IP-scoped limiting/lockout entirely. The replacement control would be defeated the moment it ships unless the trusted-proxy set is configured to match the real edge.
- **Recommended control:** Call `engine.SetTrustedProxies(...)` with only the actual reverse-proxy CIDR (or `SetTrustedProxies(nil)` + `RemoteIPHeaders` discipline) so `ClientIP()` reflects the true edge IP before any IP-based limiter is built on it.

## Finding 7 — No transport/security headers (HSTS, CSP, X-Frame-Options, Referrer-Policy)

- **Severity (post-VPN context):** LOW/MEDIUM
- **Classification:** `BEFORE-VPN-REMOVAL`
- **Evidence:**
  - No security-headers middleware exists — grep for `Strict-Transport|X-Frame|Content-Security|nosniff` in production code returns only the per-file upload-handler `nosniff` (`internal/handler/uploads_handler.go:46`) and image-validation comments, not a global middleware.
  - `internal/router/router.go:27` wires only CORS.
- **Why the VPN masks it:** Browser-facing hardening (HSTS to force HTTPS, CSP to contain XSS, clickjacking/referrer controls) matters once arbitrary public browsers and networks are in play. Behind the tailnet the audience is the trusted few on an encrypted tunnel, so the absence is low-impact. Already tracked in `recipe-remediation-post-vpn.md` Phase 3.
- **Recommended control:** Add a security-headers middleware (`Strict-Transport-Security`, `Content-Security-Policy`, `X-Frame-Options`, `X-Content-Type-Options`, `Referrer-Policy`) before public exposure.

---

## Positives / negatives (categories that are clean)

- **Source-IP trust (no implicit tailnet trust):** No production code trusts `RemoteAddr`,
  `c.ClientIP()`, `X-Forwarded-For`, or any private/CGNAT range for auth/authz/abuse decisions
  (grep confirmed; only `ssrf_test.go` references these). The auth boundary is purely
  JWT-signature based, which does not weaken when the VPN is removed.
- **SSRF filter actively blocks the tailnet rather than trusting it:** `pkg/urlparser/ssrf.go:13-17`
  explicitly denies `100.64.0.0/10` (CGNAT/tailnet) along with loopback/RFC1918/link-local. This
  is the *correct* posture — the perimeter is treated as hostile, not trusted. (Verified in
  `02-ssrf.md`.)
- **No unauthenticated management/debug/admin surface:** Grep for `pprof|expvar|/debug|admin|metrics`
  found no such endpoints. The full router (`router.go`) exposes no admin routes, and `/uploads`
  is now served via a signed-URL handler (`router.go:42-49`, `uploads_handler.go`), not a public
  static mount — so there is no "unauthenticated because you're on the tailnet" endpoint.
- **No secrets endpoint relying on network trust:** Secrets are loaded from config/env at boot
  (`cmd/api/main.go`), not exposed over any route.

## Explicitly NOT a perimeter dependency (so as not to dilute the set)

- **CORS `AllowCredentials: true` + localhost origins** (`internal/middleware/cors.go:15`,
  `env.*.yaml.sample` `cors.allowed_origins`): CORS is **browser-enforced regardless of the network
  perimeter** — it does not depend on the VPN. The localhost origin list is a config-correctness
  item (must be set to the real public origin before launch) and is already tracked in the post-VPN
  remediation plan. Listed here only to record that it was considered and ruled out as a
  perimeter-masked control.

## Inconclusive items

- **Deployment topology is assumed from `docker-compose.yml`.** Findings 1 and 4 (DB host-port
  exposure; plaintext-HTTP/0.0.0.0 bind) are asserted on the assumption that this compose file
  reflects the actual deployment. The repository contains **no production manifest, no
  reverse-proxy / TLS-terminator config, and no Kubernetes/systemd unit.** If production fronts the
  stack with an HTTPS terminator and does not publish `5432`, Findings 1/4 are mitigated at the
  edge. INCONCLUSIVE pending the real deploy artifact — flagged rather than asserted as fact.
- **Frontend is shipped as a Vite *dev* server.** `services/frontend/Dockerfile` `CMD ["npm", "run", "dev"]`
  and `vite.config.ts` (`host: '0.0.0.0'`) run the dev server, not a built/static production bundle.
  From the exposure angle this is an additional public attack surface (dev server HMR/websocket,
  verbose errors) if used as-is in production. Detailed dependency/build-mode analysis is owned by
  the Phase 4 deps subtask (`04-deps.md`); noted here only as it intersects public exposure.

## Checks performed

1. grep `100.64` / `X-Forwarded` / `RemoteAddr` / `ClientIP` / `TrustedProxies` across
   `services/backend` (Go) — only test references + the SSRF deny-list; no auth/authz IP trust.
2. grep `rate|limiter|lockout|throttle|captcha|brute` — no rate-limiting/abuse middleware exists.
3. grep `verif|confirm email|activat` — no email-verification flow; reviewed `Register` in
   `user_service.go`.
4. Reviewed server listen address (`cmd/api/main.go:76`), `docker-compose.yml` published ports,
   both Dockerfiles, and `vite.config.ts` for bind/exposure.
5. Reviewed `internal/middleware/cors.go` + sample `cors.allowed_origins` for origin/credential
   posture.
6. Reviewed full `internal/router/router.go` for admin/debug/unauthenticated management routes;
   grep `pprof|expvar|/debug|admin|metrics` — none.
7. grep `nosniff|X-Frame|Strict-Transport|Content-Security` — no global security-headers middleware.
8. Confirmed `pkg/urlparser/ssrf.go` denies the tailnet CGNAT range (does not trust it).

---

*No production code was modified. This file is the only artifact written.*
