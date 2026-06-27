# Backend Security Review — URL Parser / Fetcher (SSRF)

> Phase review of boundary **B3** (outbound network egress) from `00-threat-model.md`.
> Scope: `pkg/urlparser/*` and the `RecipeHandler.ImportFromURL` → `recipeService.ImportFromURL`
> → `urlparser.Service.Parse` → `contentFetcher.Fetch` call chain. Every finding is grounded in
> the real code at the cited `file:line`.

## Data-flow recap

1. `POST /api/v1/recipes/import/url` → `RecipeHandler.ImportFromURL`
   (`internal/handler/recipe_handler.go:183`). Behind `AuthRequired()`, but registration is open,
   so the auth bar is low.
2. Binds `domain.ImportURLRequest{ URL string binding:"required,url" }`
   (`internal/domain/recipe.go:149-152`) — only Gin's **syntactic** `url` validator.
3. `recipeService.ImportFromURL` (`internal/service/recipe_service.go:343`) passes `req.URL`
   verbatim to `s.urlParser.Parse(ctx, req.URL, aiModel)` (line 355).
4. `urlparser.service.Parse` (`pkg/urlparser/service.go:35`) calls `fetcher.Fetch(ctx, urlStr)`.
5. `contentFetcher.Fetch` (`pkg/urlparser/fetcher.go:27`) issues `http.NewRequestWithContext(ctx,
   "GET", urlStr, nil)` then `client.Do(req)` against the **raw, unvalidated** user URL.
6. The shared `http.Client` (`pkg/urlparser/service.go:22-33`) sets only
   `CheckRedirect: defaultRedirectPolicy` and **no other safety controls**.

---

### [Critical] No SSRF egress filtering — user-supplied URL fetched against any host (cloud metadata / internal services)

- **Location:** `pkg/urlparser/fetcher.go:27-35` (request build + `client.Do`);
  `pkg/urlparser/service.go:35-37` (raw URL passed in);
  `internal/handler/recipe_handler.go:190-196` (binding + dispatch);
  `internal/domain/recipe.go:150` (validator).
- **Description:** The only validation applied to the import URL is Gin's `binding:"required,url"`
  tag, which is a purely *syntactic* check that the string parses as a URL. There is **no
  allowlist**, **no destination-host validation**, and **no private/loopback/link-local IP
  filtering** anywhere in the fetch path. The URL string flows unchanged from the HTTP body into
  `http.NewRequestWithContext` / `client.Do`. An authenticated user (open registration) can therefore
  make the server issue arbitrary outbound GET requests to:
  - **Cloud metadata** — `http://169.254.169.254/latest/meta-data/...` (AWS IMDSv1),
    `http://metadata.google.internal/...` → IAM credentials / instance secrets.
  - **Loopback** — `http://127.0.0.1:<port>/` and `http://[::1]:<port>/`.
  - **RFC1918 / internal ranges** — `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`.
  - **Docker-internal service names** — `http://app:8080/...` (the API itself) and
    `http://db:5432/` (Postgres), both resolvable inside the compose network (service name `app`
    is documented in `CLAUDE.md`).
- **Impact:** Server-Side Request Forgery. From inside the Docker network this is directly
  exploitable to reach the cloud metadata endpoint (credential theft) and to port-scan / reach
  internal services that are not exposed externally. This is the highest-ranked risk in the threat
  model (B3) and requires no LLM cooperation — the network request fires before any AI parsing.
  Exfiltration of the *response body* is partially constrained (see the Info finding on the
  blind/oracle behavior), but the request-forgery primitive itself — internal reachability,
  port/host enumeration via success-vs-error, and hitting state-changing internal GET endpoints —
  is fully available.
- **Recommendation:** Validate the destination before and during the fetch:
  1. Restrict the scheme to `http`/`https` explicitly.
  2. Resolve the hostname and reject any request whose resolved IP(s) fall in
     loopback / private (RFC1918) / link-local (`169.254.0.0/16`, incl. `169.254.169.254`) /
     unique-local (`fc00::/7`) / `::1` / multicast / unspecified ranges. Reject by *resolved IP*,
     not by hostname string, to also block `app`/`db` and DNS names that resolve internally.
  3. Pin the validated IP for the actual connection via a custom `Transport.DialContext` (or
     re-validate the IP in `DialContext`) so the address checked is the address dialed — this
     closes the TOCTOU / DNS-rebinding gap.
  4. Prefer an explicit egress allowlist if the product can tolerate it.

---

### [High] Redirects are followed without per-hop destination validation (redirect-based SSRF / rebinding)

- **Location:** `pkg/urlparser/helpers.go:10-15` (`defaultRedirectPolicy`);
  `pkg/urlparser/service.go:24` (wired as `CheckRedirect`).
- **Description:** `defaultRedirectPolicy` only caps the redirect *count* at 10; it performs **no
  validation of the redirect target**. The Go default client follows up to 10 redirects to whatever
  `Location` the remote server returns. Even if a host/IP allowlist were added on the *initial* URL,
  an attacker-controlled public server (e.g. `https://attacker.example/recipe`) could `302`-redirect
  the server to `http://169.254.169.254/...` or `http://app:8080/...` and bypass it. This is also
  the classic DNS-rebinding / redirect SSRF bypass vector.
- **Impact:** Defeats any front-door URL check; the server can be pivoted to internal/metadata hosts
  via a redirect from an allowlisted or innocuous-looking origin.
- **Recommendation:** In `CheckRedirect`, apply the *same* IP/host validation (from the Critical
  finding) to every hop's `req.URL`, rejecting redirects to disallowed addresses. Combine with
  per-connection IP pinning so the dialed address is always the validated one.

---

### [Medium] No request timeout on the HTTP client (DoS / slow-loris / goroutine hang)

- **Location:** `pkg/urlparser/service.go:22-25` (`&http.Client{ CheckRedirect: ... }` — no
  `Timeout`); `pkg/urlparser/fetcher.go:35` (`client.Do`).
- **Description:** The shared `http.Client` sets **no `Timeout`**, and the `Transport` is the default
  (no `ResponseHeaderTimeout`, `DialTimeout`, etc.). The only deadline is whatever is on the inbound
  `ctx` (`c.Request.Context()`), which Gin does not bound by default. A malicious or slow target —
  including an internal host reached via SSRF that accepts the connection and never responds (e.g.
  dialing `db:5432`) — can hold the request open indefinitely.
- **Impact:** Resource exhaustion / denial of service: each import can tie up a goroutine and
  connection for an unbounded time; combined with the SSRF primitive this makes internal-port probing
  and hang-based DoS cheap.
- **Recommendation:** Set an explicit `http.Client.Timeout` (e.g. 10–15s) and transport-level
  dial/response-header timeouts, and/or wrap the fetch in a `context.WithTimeout`.

---

### [Medium] Unbounded response body read (memory exhaustion)

- **Location:** `pkg/urlparser/fetcher.go:50` (`io.ReadAll(resp.Body)`).
- **Description:** The full response body is read into memory with `io.ReadAll` and no size cap. A
  malicious target (or an internal endpoint reached via SSRF) can return an arbitrarily large or
  endless body, which is buffered entirely in memory before being handed to goquery.
- **Impact:** Memory-exhaustion DoS; a single import request can consume large amounts of RAM.
- **Recommendation:** Wrap the body with `io.LimitReader(resp.Body, maxBytes)` (e.g. a few MB) before
  `io.ReadAll`, and reject/truncate oversized responses. Consider also checking `Content-Type`.

---

### [Low] Scheme not explicitly restricted — relies on incidental `http.Transport` behavior

- **Location:** `internal/domain/recipe.go:150` (`binding:"required,url"`);
  `pkg/urlparser/fetcher.go:28` (`http.NewRequestWithContext(... "GET", urlStr ...)`).
- **Description:** There is no explicit allowlist of URL schemes. Gin's `url` validator accepts
  non-HTTP schemes (e.g. `ftp://`), and dangerous schemes like `file://` / `gopher://` are *not*
  rejected by application code. In practice the default `http.Transport` only implements `http`/`https`
  and returns an error for other schemes, so `file://`/`gopher://` requests fail rather than execute —
  the protection is **incidental**, not intentional. This is defense-in-depth, not a live exploit.
- **Impact:** Low on its own (transport rejects unsupported schemes), but relying on library
  internals is brittle; a future change to the client/transport could reintroduce scheme abuse.
- **Recommendation:** Explicitly parse the URL and require `scheme == "http" || scheme == "https"`
  before fetching, as part of the validation added for the Critical finding.

---

### [Info] SSRF is partially blind — fetch errors are masked, but success surfaces LLM-derived content

- **Location:** `internal/handler/recipe_handler.go:197-201` (generic error response);
  `pkg/urlparser/fetcher.go:46-47` (non-200 → error); `pkg/urlparser/service.go:55-66` (content →
  LLM → returned recipe); `pkg/urlparser/parser.go:38-40` (`"no content found"`).
- **Description:** On any fetch/parse failure the handler returns a fixed
  `{"error":"failed to import recipe"}` (HTTP 500) and logs the real error server-side only, so the
  raw upstream error/body is **not** reflected verbatim to the caller. This makes raw-body
  exfiltration harder than a classic full-read SSRF. However, the attacker still gets: (a) a
  **boolean success/failure oracle** (200 + parsed recipe vs. 500) usable for internal host/port
  enumeration, (b) **timing** differences (amplified by the missing timeout, Medium finding), and
  (c) **partial content leakage** — on success the fetched body is text-extracted by goquery
  (`parser.go`) and passed to the LLM, whose output (recipe title/description/ingredients) is
  returned to the user and can echo fragments of an internal response.
- **Impact:** Reduces but does not eliminate data exfiltration. The request-forgery and
  enumeration capabilities from the Critical finding remain fully exploitable; this finding documents
  why severity for *data theft specifically* is bounded rather than maximal.
- **Recommendation:** No change needed for this behavior itself (masking errors is good); resolve the
  underlying SSRF (Critical/High) so the oracle/leak is moot.

---

## Summary

- **Critical:** 1 — no SSRF egress filtering (cloud metadata + internal Docker services reachable).
- **High:** 1 — redirects followed without per-hop destination validation (bypass / rebinding).
- **Medium:** 2 — no HTTP client timeout (DoS); unbounded response body read (memory DoS).
- **Low:** 1 — URL scheme not explicitly restricted (incidentally mitigated by `http.Transport`).
- **Info:** 1 — SSRF is partially blind (errors masked) but exposes a success/timing oracle and
  partial LLM-echoed content.

**Total: 6 findings (1 Critical, 1 High, 2 Medium, 1 Low, 1 Info).**

**Exploitability:** SSRF to cloud metadata (`169.254.169.254`) and internal hosts (`app:8080`,
`db:5432`, loopback, RFC1918) is **directly exploitable** by any registered user — there is no
allowlist, no private-IP filtering, and no scheme/redirect-target validation in the fetch path.
Full raw-body exfiltration is constrained because fetch errors are masked and success content is
filtered through the LLM, but the request-forgery, internal enumeration, and credential-endpoint
reach are real and unmitigated.
