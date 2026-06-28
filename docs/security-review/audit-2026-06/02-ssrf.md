# Audit 2026-06 — Phase 2: PR #32 (SSRF egress hardening)

Scope: verify the URL-import egress path is protected against SSRF (cloud metadata,
docker-internal services, loopback, IPv6 loopback, CGNAT/Tailscale, and redirect-based
bypass). AUDIT ONLY — no production code modified.

- Endpoint under test: `POST /api/v1/recipes/import/url` (router.go:110), body `{"url":"..."}`.
- Live stack: backend `http://localhost:18080`, container `recipe-app-1` (image `recipe-app`).
- Auth: fresh throwaway user registered (HTTP 201) + login → Bearer JWT (len 264).
- Backend has outbound network; container outbound confirmed OK (used for redirect probe).

---

## 1. Code controls

All controls live in `services/backend/pkg/urlparser/`.

### safeDialContext — pins/validates the RESOLVED IP on every dial
`pkg/urlparser/ssrf.go:128-160`. Splits host/port, re-resolves via
`net.DefaultResolver.LookupIPAddr` (ssrf.go:134), rejects if ANY resolved address is
non-public via `isBlockedIP` (ssrf.go:142-146), then dials the validated IP literal
directly (`net.JoinHostPort(ipAddr.IP.String(), port)`, ssrf.go:153) so the connection
cannot be rebound to a different address (closes TOCTOU / DNS-rebinding window).
Wired as `Transport.DialContext` for the import HTTP client at `pkg/urlparser/service.go:32`.

`isBlockedIP` (ssrf.go:36-63) blocks loopback, private (RFC1918 + ULA fc00::/7),
link-local unicast/multicast (incl. 169.254.0.0/16 metadata and fe80::/10), multicast,
unspecified, and the extra special-use IPv4 CIDRs `0.0.0.0/8`, `100.64.0.0/10` (CGNAT/
Tailscale), `240.0.0.0/4` (ssrf.go:15-19). IPv6 transition forms (6to4, NAT64) are
decoded to embedded IPv4 and re-checked (ssrf.go:41-43, embeddedIPv4 ssrf.go:69-84).
A defense-in-depth pre-check `validatePublicURL` (ssrf.go:90-120) runs first in
`service.Parse` (service.go:56) for a fast rejection before any connection.

### Redirect re-validation (same IP check per hop)
`CheckRedirect: defaultRedirectPolicy` set on the client at `service.go:39`.
`defaultRedirectPolicy` (helpers.go:13-21) caps the chain at 10 hops and rejects
non-http/https schemes. The authoritative per-hop resolved-IP check is enforced because
`safeDialContext` runs on EVERY dial, including each redirect hop (proven live in §2).

### 15s request timeout
`requestTimeout = 15 * time.Second` (service.go:13-15); applied as `http.Client.Timeout`
(service.go:41) AND as an independent `context.WithTimeout` around the fetch
(service.go:64). Transport sub-timeouts also set: TLS handshake 10s, response-header 10s,
expect-continue 1s (service.go:33-35).

### 5 MiB body cap
`maxResponseBytes = 5 << 20 // 5 MiB` (fetcher.go:13). Enforced with
`io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))` and an over-cap rejection
(fetcher.go:56-62). The `+1` distinguishes "at cap" from "over cap". Unit tests
`TestFetch_RejectsOversizedBody` / `TestFetch_AcceptsBodyAtLimit` /
`TestFetch_HonorsClientTimeout` exist (fetcher_test.go:21-59) and were RUN during this
audit — all 3 PASS (`go test ./pkg/urlparser/ -run 'TestFetch_...' -v`).

---

## 2. Live probes

Each import returned HTTP 500 with generic body `{"error":"failed to import recipe"}`.
The handler masks the cause, so block confirmation comes from backend logs
(`urlparser/service.go:57` WARN "rejected URL import destination" for the pre-check, and
`urlparser/service.go:70` for the per-dial `safeDialContext` block). In all cases NO
internal content was fetched — the connection was refused at validation/dial. A 4xx/5xx
block is the correct behaviour; a 200 with fetched internal content would be a FAIL.

| Probe | URL | Resolved → block reason (log) | Result |
|---|---|---|---|
| Cloud metadata | `http://169.254.169.254/latest/meta-data/` | 169.254.169.254 → "non-public address" (link-local) | BLOCKED |
| Docker service | `http://app:8080/` | app → 192.168.147.3 → "non-public address" (RFC1918) | BLOCKED |
| localhost | `http://localhost/` | localhost → ::1 → "non-public address" (loopback) | BLOCKED |
| 127.0.0.1 | `http://127.0.0.1/` | 127.0.0.1 → "non-public address" (loopback) | BLOCKED |
| IPv6 loopback | `http://[::1]/` | ::1 → "non-public address" (loopback) | BLOCKED |
| CGNAT/Tailscale | `http://100.64.0.1/` | 100.64.0.1 → "non-public address" (blockedV4CIDRs) | BLOCKED |
| Public DNS → internal | `http://127.0.0.1.nip.io/` | nip.io → 127.0.0.1 → "non-public address" | BLOCKED |
| HTTP redirect → internal | `https://httpbin.org/redirect-to?url=http://127.0.0.1/&status_code=302` | initial public host accepted; redirect hop blocked at dial | BLOCKED |

Redirect probe is the decisive end-to-end test of per-hop re-validation. Public host
`httpbin.org` passed the initial `validatePublicURL`, then its 302 to `http://127.0.0.1/`
was rejected by `safeDialContext` on the redirect dial. Verbatim log:
`urlparser/service.go:70 ... "error": "failed to fetch URL: Get \"http://127.0.0.1/\":`
`blocked dial to non-public address 127.0.0.1 (host \"127.0.0.1\")"`.

Observation (non-blocking): blocked imports surface to the client as HTTP 500 with a
generic message rather than a 4xx. Security is unaffected (no internal data returned,
error detail not leaked to client), but the status code is semantically a client/input
rejection and could be a 400/422.

---

## Verdicts

Code controls:
- safeDialContext pins/validates RESOLVED IP per dial — PASS (ssrf.go:128-160, 142-146, 153; wired service.go:32)
- Redirect re-validation (same IP check per hop) — PASS (service.go:39, helpers.go:13-21; safeDialContext on every dial, proven live)
- 15s request timeout — PASS (service.go:13-15, 41, 64)
- 5 MiB body cap — PASS (fetcher.go:13, 56-62)
- isBlockedIP coverage incl. CGNAT/metadata/IPv6-transition — PASS (ssrf.go:15-19, 36-84)

Live probes (all REJECTED, no internal content fetched):
- http://169.254.169.254/latest/meta-data/ (cloud metadata) — PASS (HTTP 500; log "non-public address (169.254.169.254)")
- http://app:8080/ (docker-internal) — PASS (HTTP 500; resolved 192.168.147.3, "non-public address")
- http://localhost/ — PASS (HTTP 500; resolved ::1, "non-public address")
- http://127.0.0.1/ — PASS (HTTP 500; "non-public address (127.0.0.1)")
- http://[::1]/ (IPv6 loopback) — PASS (HTTP 500; "non-public address (::1)")
- http://100.64.0.1/ (CGNAT) — PASS (HTTP 500; "non-public address (100.64.0.1)")
- HTTP redirect → 127.0.0.1 (httpbin 302) — PASS (HTTP 500; safeDialContext "blocked dial to non-public address 127.0.0.1" on redirect hop)
- Public DNS → 127.0.0.1 (nip.io, bonus) — PASS (HTTP 500; "non-public address (127.0.0.1)")

Overall: PASS. PR #32 SSRF egress hardening is present in code and effective at runtime,
including the redirect-bypass and CGNAT/metadata/IPv6-loopback vectors.
