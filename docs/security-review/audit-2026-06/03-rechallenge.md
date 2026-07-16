# Phase 3 — Re-Challenge of Four Prior Verdicts

**Subtask:** Actively re-test shopping-list-item IDOR (was "refuted"), upload path-traversal (was
"mitigated"), public-recipes-by-design, and the S3-stub fatal-on-select, rather than trusting the
prior report's checkboxes.

**Scope/method:** For each item, re-read the current code from scratch (not the prior write-up)
and, where practical, re-ran a live probe against the shared stack, to independently confirm or
overturn the original `docs/security-review/REPORT.md` verdict.

---

## 1. Shopping-list-item IDOR (`UpdateItem`/`DeleteItem`/`ToggleItem`) — prior verdict: REFUTED

- **Re-tested:** YES — live, in this audit's `03-idor.md` (user B against user A's item ID for
  `PUT`/`DELETE`/`PATCH .../toggle`). All three were denied.
- **Verdict: CONFIRMED (still refuted — no IDOR).** `shopping_list_service.go:226-239`
  (`verifyItemOwnership`) loads the item, resolves its **actual** parent list via
  `item.ListID` (not the URL's `:id` param), and checks `list.UserID != userID` before every
  mutation. A user cannot modify another user's item by guessing/supplying its UUID, regardless of
  what `:id` is in the URL.
- **Note — the original "optional hardening" was not applied and still isn't:** the handlers
  (`internal/handler/shopping_list_handler.go:166-224`, `UpdateItem`/`DeleteItem`/`ToggleItem`)
  still read only `c.Param("itemId")` and never compare it against the URL's `:id` (list ID) —
  the nested route shape (`/shopping-lists/:id/items/:itemId`) remains **decorative**: a request to
  `/shopping-lists/<any-list-id-including-someone-elses>/items/<your-own-item-id>` would succeed
  identically to using the item's real list ID, since `:id` is never read. This was true at the
  time of the original report and remains true today. It is not a vulnerability (ownership is still
  correctly enforced against the item's real parent list), but it is a still-open code-quality/API
  honesty item — recommend re-flagging it as a nice-to-have in `05-quality-triage.md` rather than
  silently dropping it.

## 2. Path traversal in stored/deleted filenames — prior verdict: investigated, MITIGATED

- **Re-tested:** Static re-review in this audit's `03-bypass.md` (not re-run as a live traversal
  probe, since the code-level mechanism fully accounts for the mitigation and a live probe would
  only re-confirm what's structurally guaranteed).
- **Verdict: CONFIRMED (still mitigated).** Independently re-derived the same three-layer
  reasoning as the original review, from the *current* code rather than trusting the prior
  citation:
  1. **Write path** (`pkg/storage/local_storage.go:29-53`) — the on-disk filename is always
     `uuid.New().String() + ext`, where `ext` comes from magic-byte sniffing
     (`DetectImageType`), never from the client's filename or its extension. The client-supplied
     name never reaches `filepath.Join`.
  2. **Delete path** (`local_storage.go:56-66`) — takes `filepath.Base(fileURL)` before joining,
     stripping any `../` even if `fileURL` were attacker-influenced (it isn't, in practice — it
     comes from the stored `ImageURL`, itself server-generated).
  3. **Serve path** (`internal/handler/uploads_handler.go:29-37`, added since the original
     report as part of PR #34) — explicitly rejects any `filename` containing `..` or not equal to
     its own `filepath.Base`, in addition to requiring a valid HMAC signature. This is a *stronger*
     mitigation than existed at the time of the original review (which predates the signed-URL
     handler entirely — uploads were reportedly served differently before PR #34).
- No new traversal vector found in this pass.

## 3. `recipe GetByID` returns other users' public recipes by ID — prior verdict: BY DESIGN, not a vulnerability

- **Re-tested:** Static re-review (`recipe_service.go:336-351`) plus route-level re-check
  (`internal/router/router.go:98-118`).
- **Verdict: CONFIRMED (still by design).** `GetByID` only rejects access when
  `recipe.IsPrivate && recipe.UserID != userID` — a non-private recipe is intentionally readable by
  any authenticated caller who has (or guesses) its UUID, matching the product's "public recipe"
  concept. One clarification worth recording: `GET /recipes/:id` and `GET /recipes/public` both
  sit inside the **JWT-protected** route group (`router.go:55-57` wraps the whole `recipes` group in
  `protected.Use(r.auth.AuthRequired())`) — so "public recipe" means "public to any authenticated
  member of the app," not "public to the unauthenticated internet." This is consistent with the
  original finding and doesn't change the verdict, but is worth stating explicitly since the word
  "public" could otherwise be misread as "no-auth-required" when triaging `03-vpn-deps.md`
  (it is not — the VPN's implicit gate on *registration*, not this route's auth check, is what's
  covered there).

## 4. S3 storage stub — prior verdict: fails fatal-on-select (by design / INFO)

- **Re-tested:** Static re-review of `pkg/storage/factory.go`, `pkg/storage/s3_storage.go`, and the
  boot sequence in `cmd/api/main.go`.
- **Verdict: CONFIRMED.** `pkg/storage/factory.go:15-16` — selecting `storage.type: s3` returns
  `fmt.Errorf("s3 storage not implemented")` from the factory (the `s3FileStore.UploadFile`/
  `DeleteFile` methods in `s3_storage.go:23-30` are unimplemented stubs). `cmd/api/main.go:53-55`
  calls `storage.NewFileStore(&cfg)` and does `logger.Fatal(...)` on any error — so choosing `s3`
  in config causes the server to **refuse to boot** rather than silently accepting uploads into a
  no-op storage backend. This is the correct fail-closed behavior for an incomplete feature: it is
  a feature-completeness gap (S3 storage doesn't work), not a security hole (it cannot be
  half-configured into a silently-broken, data-losing state at runtime).

## Summary

| # | Item | Prior verdict | Re-challenge verdict |
|---|---|---|---|
| 1 | Shopping-list-item IDOR | Refuted | **Confirmed refuted** (decorative nested-route param still unaddressed — code-quality nit, not a vuln) |
| 2 | Upload path traversal | Mitigated | **Confirmed mitigated** (three independent layers, one added since) |
| 3 | Public-recipe-by-ID | By design | **Confirmed by design** (clarified: "public" = public-to-authenticated-members, not internet-public) |
| 4 | S3 stub | Fatal-on-select | **Confirmed** (fails closed at boot, not a runtime data-loss risk) |

No prior verdict was overturned. All four hold up under independent re-derivation from current
code (and, for #1, a fresh live probe).

## Checks performed

1. Live probe (shared with `03-idor.md`): user B against user A's shopping-list item for
   `PUT`/`DELETE`/`PATCH .../toggle`.
2. Read `shopping_list_handler.go`'s three item-mutation handlers to confirm `:id` is still
   unused/decorative.
3. Read `local_storage.go` (write + delete) and `uploads_handler.go` (serve) end-to-end for
   traversal defenses, independent of the prior write-up's citations.
4. Read `recipe_service.go:GetByID` and the router's auth-group wiring to confirm the "public"
   recipe path still sits behind `AuthRequired()`.
5. Read `pkg/storage/factory.go`, `s3_storage.go`, and `cmd/api/main.go`'s boot sequence for the
   S3-selection failure mode.

---

*No production code was modified. This file is the only artifact written.*
