---
id: T-079
title: amend rick artifacts in pickle serve: digest-CAS, atomic write, and a lifecycle-field guard
project: pickle
depends-on: [T-077]
spawned-by: []
family: T-075
impact: medium
complexity: high
cost: L
---

# T-079 — amend rick artifacts in pickle serve: digest-CAS, atomic write, and a lifecycle-field guard

## Outcome

After this ships, a human can edit a rick artifact directly in `pickle serve` before approving it — the edit content-addressed and lifecycle-guarded, so an edited artifact can never be mistaken for one the agent itself produced and validated.

## Description

The ambitious member of T-075, and the one that has to be careful: let a human **edit** a
rick artifact in the browser before approving it in the agent TUI.

### Why this is not just "add a textarea"

rick's gate has three verbs and **none of them is "I'll fix it myself"**. `[R]evise` means
*ask the agent to redo it* — which is the wrong tool for a typo, a wrong constant, or a
sentence that needs one clause cut. Human amendment is a fourth verb rick does not have
(call it **Amend**), and introducing it from outside collides with two guarantees:

1. **Validated-before-gate.** The artifact was validated by `s-artifact-validator` *before*
   the gate was presented; an edit after that point means `[A]pprove` blesses unvalidated
   content, which the next phase then consumes as a contract.
2. **Single-writer on `docs/specs/**`.** The session holds that write scope and is parked at
   the gate. It can overwrite the path at any moment (`[R]evise`) or **delete the file**
   outright (`[D]iscard` is explicitly permitted to do so).

### Scope

- **Editor:** split-pane `<textarea>` with live preview via an htmx `POST /fragments/preview`
  against the existing goldmark renderer. **No JS editor dependency** — CodeMirror/EasyMDE
  would mean hundreds of KB plus a licence file, and `serve` has a no-build-step discipline
  with htmx as its only vendored asset (`internal/serve/static/`).
- **Digest CAS:** capture a content digest when the editor opens; refuse to save if the file
  changed underneath, and render that as a first-class "the session revised this — reload"
  state rather than a 500. Digest, not mtime. This is the same shape as rick's own best idea:
  a verification record bound to HEAD *and* a worktree digest
  (`sdlc-cli/internal/checks/verification.go`).
- **Atomic write:** temp file + `os.Rename`, so a concurrent reader never sees half a
  document.
- **Lifecycle-field guard:** parse the result back and **refuse the write if `status:` or any
  identity field changed**. This is `config.verifyOnlyPayloadVersion`
  (`internal/config/config.go:347`) inverted — pickle already has the exact pattern for
  "surgical single-field edit, reject if anything else moved". It makes T-075's *pickle never
  writes `status: approved`* invariant a testable claim rather than a convention.
- **Handoff, not silence:** the save confirmation hands the human a one-line instruction to
  paste into the session — *"I amended the artifact in place; revalidate it and re-present
  the gate"* — which restores guarantee (1) by re-running rick's validator before approval.

### Deliberately not blocked on rick

pickle **cannot** validate the edit itself today: the artifact schemas live in rick's skills
as prose for an LLM ("Required Structure and section order", "Validation Checklist" in
`framework/skills/ai-sdlc/artifacts/*/SKILL.md`), not as a machine-checkable spec. The clean
fix is an upstream ask — a deterministic `rick check artifact <path>` — recorded in
`tickets/NOTES.md`. This ticket ships the paste-to-revalidate handoff instead, so it is not
hostage to another product's roadmap; local validation becomes a follow-up if that ask ever
lands.

### Folded in from T-056 (2026-08-14): this ticket owns `serve`'s first write route

T-056 ("make the serve dashboard writable") was **dropped and split on 2026-08-14**. Its work
area 6 — everything that turns `serve` from a proven non-writer into a server that accepts a
POST — folds **into this ticket**, because `POST /fragments/preview` plus a save endpoint makes
this the first non-GET route in `serve` whatever else happens. The work was always implicit
here; folding it makes it explicit, so the `L` grade stands rather than rising.

What comes with it:

- **Two structural safety guarantees die on the day this ships, and both need replacing.**
  (a) The **method-qualified mux** (`internal/serve/serve.go:68-80`, moved from `:63-77` by
  T-127) registers every route as
  `GET /…`, so today anything but GET/HEAD is a 405 *before a handler is reached* — the comment
  above the block says exactly that. **T-127 added a second such barrier that this ticket must
  also clear:** in multi-root mode `MultiHandler` mounts each project's whole sub-handler as
  `mux.Handle("GET /p/{slug}/", http.StripPrefix(…))`, so a POST to `/p/{slug}/…` is rejected
  **405 at the top-level mux, before the per-root handler is ever consulted** (verified during
  T-127's review). Registering a write route inside `Handler` is therefore *not sufficient* under
  multi-root — `MultiHandler`'s mount must stop being GET-only too, and the write route must
  resolve *which root* it writes to from the `{slug}` segment. (b) **`TestServeNeverWrites`**
  (`internal/serve/serve_test.go:805`) is a sha256 snapshot of the whole tree taken around a full
  crawl; under multi-root it must snapshot **every** served root, not just one.
  **Replace that test, do not delete it:** the same snapshot, asserting that *only* the file the
  request was allowed to touch changed. That is a strictly stronger invariant than "nothing
  changed", and it is what catches a handler regenerating a board it had no business touching.
- **CSRF tokens and `Origin`/`Sec-Fetch-Site` checks are required, not optional.** A write
  endpoint on `127.0.0.1:8745` is reachable from any page the user has open: form POSTs are not
  preflighted, and DNS rebinding defeats a naive `Host` check. Whatever middleware this lands
  must be the one a later ticket inherits — do not let a second write route invent its own.
- **T-053's contract must be overturned on the record, not left to rot.** Decision **1**
  (*"Read-only, absolutely. No handler writes, moves, renames or regenerates anything … The CLI
  stays the single writer. A test asserts this"*) is overturned outright; decision **9** and the
  non-goal *"`serve` is a human surface, not something an agent is told to run"* is **narrowed**
  (still human-only, but the payload's "the CLI is the only writer" framing must be re-checked);
  the non-goals *"no ticket editing or moving from the browser, no authentication"* are
  overturned in part — this ticket edits **artifacts, not tickets**, and CSRF/Origin is
  authentication-adjacent, so a non-loopback `--addr` now needs a stronger warning than "no
  authentication". T-053's decisions 2–8 and 10 survive untouched; say so explicitly at
  refinement so the surviving ones are not assumed dead too.
- **T-056's "cheaper v1: have the UI shell out to the CLI" does not transfer here.** That option
  preserved "the CLI is the only writer" for `tickets/`, where CLI verbs exist. No pickle
  command writes `docs/specs/**`, so there is nothing to shell out to. Weighed and declined —
  but if a `pickle` verb for artifact amendment is ever proposed, re-weigh it.

Soft coupling: **T-101** (READY as of 2026-08-14) ships the atomic-write helper this ticket's
Scope describes — `internal/atomicfile.WriteFile`, lifted from `config.writePreservingMode` —
plus a flock tree lock over `tickets/`. **Consume the helper; do not write a second temp-file +
`os.Rename`.** T-101 lands first on any plausible ordering (it is READY and unblocked, this
ticket sits three deep behind T-075/T-076/T-077), which answers the "check which lands first"
question this paragraph used to ask. The **lock** does not transfer: this ticket writes a
*different* tree whose other writer is a Claude Code session pickle does not control, which is
why the digest-CAS above exists instead. **T-102** (the ticket-frontmatter field writer) is the
`tickets/`-side sibling of the lifecycle-field guard — same inverted-`verifyOnlyPayloadVersion`
pattern, different file format; compare notes rather than sharing code.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-07 — created (TO DO). source: pickle ticket new
- 2026-08-14 — scope folded in from T-056 work area 6 (dropped and split that day): this ticket
  now owns `serve`'s first write route and everything that implies — replacing the
  method-qualified mux and the `TestServeNeverWrites` snapshot, CSRF/`Origin` checks, and the
  on-the-record renegotiation of T-053's decisions 1 and 9. Grade unchanged (`L`): the work was
  already implicit in `POST /fragments/preview`.
- 2026-09-02 — patched by T-127's review impact sweep: `serve` now serves N roots from one
  process. Two corrections. (1) `MultiHandler` mounts each root's sub-handler GET-only, so a POST
  to `/p/{slug}/…` 405s at the *top-level* mux before the per-root handler runs — a second
  barrier this ticket must clear, and the write route must resolve its target root from `{slug}`;
  `TestServeNeverWrites` must also snapshot every served root. (2) Line references refreshed:
  the mux block moved `:63-77` → `:68-80` (caused by T-127), and `serve_test.go:722` → `:805`
  (drift that predates T-127, corrected while here). Grade unchanged (`L`).
