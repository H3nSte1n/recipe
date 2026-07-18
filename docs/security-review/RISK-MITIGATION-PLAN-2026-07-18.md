# Recipe App — Risk 1 & Risk 5 Mitigation Plan (2026-07-18)

**Status as of 2026-07-18 23:14:** Risk 1 (Parts A, B, C) fully done. Risk 5
(script + Kuma monitor + cron) is deployed and manually test-verified; first
live cron-triggered heartbeat still pending confirmation, and the
failure-injection test hasn't been run. `scripts/check-no-db-port.sh` and the
`backend-ci.yml` change are uncommitted in the app repo. See per-section
status below.

Follow-up to `DEPLOYMENT-PLAN-2026-07-16.md`, which shipped the app to
`recipe.steinhauer.dev` / `recipe.johanneszimmer.com` and left two standing
risk-mitigation tasks open (see that doc's "Security risks" section):

- **Risk 1** — Docker port-publishing bypasses ufw; Postgres's `sslmode=disable`
  decision depends entirely on the `db` service never publishing a port, with
  no firewall backstop if that regresses.
- **Risk 5** — Tailscale-only access is a network ACL, not authentication;
  Funnel/Serve enabling on either recipe vhost would bypass both the nginx ACL
  and ufw. Checked clear on 2026-07-18, but that was a one-time check.

Both are server-side (devops-skill) changes. This plan is written up front so
the actual `/devops` session executes against agreed steps rather than making
judgment calls live.

---

## Risk 1 — Postgres port-publishing vs ufw

### Part A: Live-test the assumption (one-time, non-destructive) — ✅ done 2026-07-18

Ran exactly as planned: `nginx:alpine` on `-p 19999:80` while ufw was active
with default-deny and no allow rule for 19999. External curl to the server's
public IP on that port returned HTTP 200 (nginx welcome page) — confirmed
Docker's `DOCKER` iptables chain bypasses ufw's `INPUT` filtering. Container
stopped immediately after; port re-verified closed (`curl` timeout, exit 28).
Recorded in `~/.agents/state/devops/server.md` under `notes:`. This confirms
Risk 1 Part B is a real, load-bearing safeguard, not theoretical.

**Goal:** confirm empirically that Docker's own iptables rules (`DOCKER` chain)
bypass ufw's `INPUT` chain filtering, rather than relying on the general
Docker/ufw interaction being "well known."

**Steps:**

1. On the server, confirm ufw is actually denying inbound by default:
   `ufw status verbose` (expect `Status: active`, default incoming `deny`).
2. Start a disposable, non-privileged throwaway container publishing a port
   that nothing else uses, e.g.:
   `docker run --rm -d --name ufw-probe -p 19999:80 nginx:alpine`
3. From a machine **outside** the server's LAN/tailnet (a different network —
   not another container, not localhost, since the point is to test the
   external firewall path), attempt: `curl -m 5 http://<server-public-ip>:19999/`
4. **Expected (and the reason this task exists):** the request succeeds
   (nginx welcome page), despite ufw's default-deny — demonstrating the
   `DOCKER` chain inserts ACCEPT rules ahead of ufw's `INPUT` filtering.
5. Immediately tear down: `docker stop ufw-probe` (auto-removes via `--rm`).
   Confirm the port is closed with a second external `curl` (should time out
   /connection-refused).
6. Record the result (pass/fail either way) in
   `~/.agents/state/devops/server.md`, so this isn't re-litigated later —
   note the date and that it was empirically confirmed, not assumed.

**Risk of this step:** briefly exposes port 19999 externally on an unrelated,
non-sensitive test container for the duration of one curl round-trip. Low
blast radius, but still a real external exposure — will confirm before
running this against the live server rather than doing it unannounced.

### Part B: Standing guard against Postgres ever publishing a port — 🟡 app-repo half done, deploy-time wiring open

**Goal:** turn "Postgres has no `ports:` mapping" from a one-time deploy-day
fact into something that fails loudly if it ever regresses (e.g. someone adds
`ports: ["5432:5432"]` to `docker-compose.prod.yml` for a debugging session
and forgets to revert it).

**Approach — checked-in script, not just a manual review note:**

1. ✅ Done, 2026-07-18. `scripts/check-no-db-port.sh` added at the **repo
   root** (deviation from the path originally sketched here,
   `services/backend/scripts/` — moved because the compose files it checks
   live at the repo root, not inside `services/backend/`). Runs `docker
   compose -f docker-compose.yml -f docker-compose.prod.yml config --format
   json`, parses the resolved `db` service's `ports` via `python3 -c` (no
   `jq` dependency — confirmed the target server doesn't have `jq`
   installed, so this needed to work without it), fails with a clear error
   if non-empty. Required secrets with no compose defaults (`DB_PASSWORD`,
   `JWT_SECRET`, `SECURITY_ENCRYPTION_KEY`) are given dummy values purely so
   `docker compose config` can resolve — real secrets aren't needed just to
   inspect port mappings. Tested both directions: passes against the current
   (safe) compose files, and correctly fails when a `db.ports` mapping is
   injected via a throwaway override file (not by editing tracked files).
2. Wire it in two places so it's actually enforced, not just available:
   - **CI**: ✅ Done. Added a `no-db-port` job to
     `.github/workflows/backend-ci.yml`, and widened that workflow's
     `paths:` triggers (previously scoped to `services/backend/**` only) to
     include `docker-compose.yml`, `docker-compose.prod.yml`, and the new
     script itself — otherwise the job would never fire on the changes it's
     meant to catch. YAML validated locally.
   - **Deploy-time**: ✅ Done, 2026-07-18. No CI/CD pipeline exists on the
     server — deploys are manual (`git archive HEAD` over SSH). Placed a
     copy of the script at `/home/henry/recipe/scripts/` and ran it live
     against the actual production compose files (confirmed clean, no
     `ports:` mapping). Documented as a **standing process requirement** in
     the state file: run `./scripts/check-no-db-port.sh` from
     `/home/henry/recipe` and abort the deploy if it fails, before every
     future `docker compose up -d --build`. This is a manual pre-flight
     step, not an automated gate — there's nothing on the server to enforce
     it beyond the documented process. The script itself will also arrive
     automatically in future deploys once committed, since `git archive`
     picks up all tracked files.
3. ✅ Done. See `~/.agents/state/devops/server.md`, `recipe-db-1` container
   notes.

Both Part A and Part B are now complete (CI enforcement + documented
deploy-time process). App-repo changes (`scripts/check-no-db-port.sh`,
`.github/workflows/backend-ci.yml`) are still uncommitted — working tree
only as of this update.

### Part C: Revisit `sslmode=disable` if the guard ever trips

Not an action to take now — just a documented tripwire. If Part A's live test
comes back _unexpectedly_ (ufw actually does block it, contradicting the
current understanding), or if Part B's guard is ever tripped by an actual
port-publish in prod, `sslmode=disable` for Postgres needs to be revisited
immediately rather than assumed still-safe. No separate task needed beyond
noting this dependency in the state file.

---

## Risk 5 — Tailscale Funnel/Serve regression check — ✅ deployed, verification pending

**Goal:** periodic, alerting check that neither `tailscale funnel` nor
`tailscale serve` has been enabled on this server (which would bypass the
nginx ACL and ufw for `cockpit.steinhauer.dev` / `recipe.steinhauer.dev` /
`recipe.johanneszimmer.com` alike), riding on the existing uptime-kuma
monitoring rather than a silent, unwatched cron job.

**Why cron + Kuma push, not a native Kuma monitor:** Kuma runs containerized
and has no monitor type that execs a command on the host — it only reaches
things over the network (HTTP/TCP/ping/DNS) or receives a "Push" heartbeat
from an external script. `tailscale funnel status` / `serve status` need to
run where the Tailscale CLI actually has host access, so the check itself has
to live in a host-side script; Kuma's role is just to surface a failure if the
script doesn't report "clear" on schedule.

**Steps:**

1. ✅ Done. `/usr/local/bin/check-tailscale-serve.sh` deployed (root:root, 755,
   no secret in it). Checks `tailscale funnel status` / `serve status` for a
   `"No serve config"` substring match (confirmed this is the server's actual
   current wording — no trailing period). Secret push URL lives separately in
   `/etc/default/check-tailscale-serve` (root:root, 600, `export
KUMA_PUSH_URL=...`) — sourcing without `export` does NOT propagate to the
   child script process, confirmed empirically and noted in the state file.
   Uses `curl -G --data-urlencode` for both `status`/`msg` params (no `jq` on
   this host, so URL-encoding goes through curl itself rather than a JSON
   tool).
2. ✅ Done. Kuma Push monitor "Tailscale Funnel/Serve Check" created, 1800s
   (30 min) heartbeat interval, 1 retry. Push URL:
   `http://100.87.135.126:3001/api/push/f20eWvvTVkUFGFi7mXU890bMvCmi9tPJ`.
   (Side effect: Kuma itself had to be rebound from `127.0.0.1:3001` to the
   Tailscale interface `100.87.135.126:3001` first — matching Portainer's
   existing binding pattern — since it was loopback-only and unreachable over
   the tailnet before. See `containers: uptime-kuma` notes in the state file.)
3. ✅ Done, by the user directly (cron domain is inspection-only for this
   skill). Root crontab (`sudo crontab -e`), `*/30 * * * *`. First attempt
   incorrectly included a leading `root` user-field (valid in `/etc/cron.d/`
   but not in a personal crontab, where there's no user column) — caught and
   corrected. Final line: `*/30 * * * * . /etc/default/check-tailscale-serve
&& /usr/local/bin/check-tailscale-serve.sh`. Confirmed present via `sudo
crontab -l`.
4. ✅ Pending. Manual test-run of the script succeeded and Kuma confirmed
   receipt (`{"ok":true}`), but the actual cron-triggered run hasn't been
   observed yet — waiting for the next `:00`/`:30` mark to confirm cron is
   firing it for real (env sourcing, PATH, etc. all behave the same under
   cron as they did in the manual `sudo bash -c` test, but this hasn't been
   watched end-to-end from cron itself). The deliberate-failure-injection
   test described here (temporarily break the match, confirm Kuma alerts)
   has **not** been done yet either.
5. ✅ Done — see `~/.agents/state/devops/server.md`, `notes:` section and the
   `uptime-kuma` container entry.

**Scope note:** this checks Funnel/Serve state for the whole server (both are
host-wide, not per-vhost), so one script/monitor covers `cockpit`, both
`recipe` vhosts, and anything else Tailscale-ACL-gated on this box — no need
for one per vhost.

---

## Execution

Both parts touch the live server and (Risk 1 Part B) the app repo. Plan is to
run this interactively via `/devops`, confirming each step rather than
batching silently — same standard as the original deployment plan. Risk 1
Part A's external curl test is the one step with real (if brief and low-risk)
external exposure, so that step specifically will be called out for
confirmation before running.
