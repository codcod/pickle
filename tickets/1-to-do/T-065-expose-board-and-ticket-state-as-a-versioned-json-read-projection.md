---
id: T-065
title: expose board and ticket state as a versioned JSON read projection
project: pickle
depends-on: []
spawned-by: []
impact: low-medium
complexity: medium
cost: M
---

# T-065 — expose board and ticket state as a versioned JSON read projection

## Description

`pickle` has **no machine-readable output at all** — `rg 'encoding/json' internal/` returns
nothing. Every command prints prose for humans. Any programmatic consumer (an agent harness
extension, a CI step, a git hook, a future web client) must either scrape formatted text or
re-implement the tree walk and the grading/WIP/audit rules that `internal/board` and
`internal/audit` already own.

Add a **read-only, versioned JSON projection** of board and ticket state:

- `pickle board state --json` — the whole board: children, status groups in the board's own
  deterministic order, per-child WIP counts and their caps, and audit health.
- `pickle ticket show <id> --json` — one ticket: frontmatter (including `family`,
  `spawned-by`, `depends-on` and any `DuplicateKeys`), status, slug/path, and parsed History.

Both are **new subcommands**. The current surface is `install`, `upgrade`, `doctor`,
`uninstall`, `project {add,list,remove}`, `ticket {new,move}`, `board {audit,sync}`, `serve`,
`version` (`internal/cli/cli.go:48-66`, `board.go:20-22`, `ticket.go:25-27`) — there is no
`board state` and no `ticket show` to hang a flag on.

**The projection already exists in Go**, built for the `serve` templates, and this ticket is
mostly about lifting it rather than designing it:

- `internal/serve/view.go:77` `buildBoard` — grouped board; its own comment notes it builds
  "the same map the board's own Render builds"
- `internal/serve/view.go:168` `buildTicket` — single-ticket view
- `internal/serve/view.go:280-283` `ChildWIP.AtDevLimit()` / `AtReviewLimit()`
- `internal/serve/view.go:298` `buildHealth` — wraps `audit.Audit`
- `internal/serve/view.go:242` `buildActivity` — History-derived activity

These are template-shaped (`template.HTML` fields, predicate methods), so they are not a wire
format as they stand.

### Envelope and versioning

The payload carries a **top-level envelope** with the emitting binary's version, so a consumer
can refuse a dialect it does not understand instead of mis-parsing it. This deliberately
absorbs the version-handshake need that was originally proposed as a per-ticket
`schema_version` frontmatter key: a handshake belongs in the wire format, not in 64 ticket
files.

**Pre-registered, deliberately not filed:** a `schema_version` key in ticket frontmatter, with
a fail-closed guard. It is unnecessary today because **no write path re-renders frontmatter** —
`internal/move/move.go:123` appends (`newText := appendHistory(t.Text, …)`) and
`internal/ticket/ticket.go:529` `Scaffold()` is only ever called for brand-new files
(`internal/cli/ticket.go:141`), while `parseFrontmatter` carries unknown keys through in
`Front`. An unknown field cannot currently be dropped. File that guard **when, and only when, a
frontmatter re-render path is proposed** — i.e. as a prerequisite inside T-056's ticket field
writer (work area 4), which is what creates the hazard.

### Soft couplings (not `depends-on`)

- **T-056, work area 1** — extracting a shared, audited core (`internal/api`). Lifting the
  `serve` view structs into something both a CLI command and an HTTP handler can use **is that
  extraction seam**. Doing this ticket and T-056 area 1 independently means building the
  projection twice; the same duplication hazard T-056 already records against **T-043**.
  Sequence them or fold one into the other — do not run them concurrently.
- **T-043** — CLI test harness; new subcommands are exactly what it would cover.
- **T-052** — `board audit`'s verdict classification. If audit health is exposed as structured
  data, the "stale **or** hand-edited" conflation becomes a field a consumer reads, so the two
  tickets should agree on the vocabulary rather than inventing two.

### Honest scope of the benefit

**There is no consumer today, and the motivating one was withdrawn the same day this was
filed.** The candidate was an agent-harness extension enforcing flow gates; it was assessed
against the field record hours later and **not filed** — its one evidenced rule already belongs
to T-057, which had itself already concluded that a harness extension is the wrong primary
mechanism (*"a pi extension only guards a pi session"*). See `tickets/NOTES.md`, "Second
postscript (2026-08-04)".

What survives without it: pickle has **no machine-readable output at all**, and T-056 work
area 1 would have to build this projection regardless. That is real but weaker than the case
at filing time. Graded `low-medium` — enabling infrastructure with deferred, now less certain,
payoff; it should not outrank tickets that fix measured field defects.

**Refinement must first decide whether this ticket should exist at all**, and specifically
whether it stands alone or folds into T-056 work area 1 — the honest options are (a) refine as
scoped, (b) fold into T-056 area 1 and drop this, (c) drop until a consumer is real. Do not
default to (a) because the ticket is already on the board.

### What must not be assumed at refinement

- **That the `serve` structs can be marshalled as-is.** They carry `template.HTML` and
  behaviour; a wire type is probably separate, with the view types built from it rather than
  the reverse.
- **That `--json` should be added to the existing read commands** (`doctor`, `board audit`,
  `project list`, `version`) as part of this. That is separable polish, it serves a different
  audience, and it is explicitly out of scope here — file it on its own merits if wanted.
- **That the projection should be writable, or served over HTTP.** Read-only, CLI-only. Writes,
  locking and an HTTP surface are T-056.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-04 — created (TO DO). source: chat — the Pi-as-best-tier exploration recorded in
  tickets/NOTES.md (2026-08-04); scope corrected before filing after reading the code (the two
  commands it was to flag did not exist, and the paired `schema_version` guard was found to
  have no reachable hazard, so it was pre-registered here instead of filed)
