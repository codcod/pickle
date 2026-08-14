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

## Outcome

After this ships, `pickle board state --json` and `pickle ticket show <id> --json` give any external tool (an agent extension, a CI step, a git hook) a versioned, machine-readable view of board and ticket state, instead of it having to scrape prose or re-implement brine's own grading/WIP/audit rules.

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

### Open scope question: the `## Review` findings table (added 2026-08-07)

`ticket show --json` as scoped above projects frontmatter, status, slug/path and **parsed
History** — but nothing from the `## Review` section. That section holds the repo's largest
structured dataset (**347 dispositioned findings across 53 done tickets**), in a table whose
columns are fixed by a canonical, pasteable skeleton in `skill/resources/review-protocol.md` §5
(T-085): id, severity, **class**, disposition, description, evidence, suggestion. Note that
`TEMPLATE.md` no longer states the column list — it points at that skeleton, so the projection
has exactly one shape to parse against rather than the 13 header variants the corpus drifted
into before T-085. Every cross-ticket question anyone has actually asked of this corpus — the T-045
spawn rate, the rework rate, the review yield — is a group-by over that table, and each was
answered by hand-grepping markdown.

**Refinement must decide, explicitly, whether the projection includes it.** Both answers are
defensible and the ticket should not drift into one by omission:

- **Include** — the findings table is the only part of a ticket that is *already* tabular, so it
  is the cheapest high-value addition to the wire format, and it is what makes the projection a
  measurement substrate rather than a status mirror.
- **Exclude** — it is free-prose-heavy (`description`, `evidence` and `suggestion` are
  sentences, not values), parsing it means a markdown-table reader that nothing else needs, and
  malformed or absent tables are common (a `Disposition summary` line is present in only **40 of
  53** done tickets).

A middle option worth costing: project the **counts and the closed-vocabulary columns only**
(id, severity, disposition, and the `class` column T-085 proposes), skipping the three prose
columns. That answers every measurement question at a fraction of the parsing risk.

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
frontmatter re-render path is proposed** — i.e. as a prerequisite inside the ticket field
writer, which is what creates the hazard. That writer is now **T-102** (T-056 was dropped and
split on 2026-08-14), and T-102's Description carries the pre-registration forward.

### Soft couplings (not `depends-on`)

- **T-056, work area 1 — resolved 2026-08-14: this ticket owns the seam now, and there is
  nothing to sequence against.** T-056 was dropped and split; its areas 1 and 3 (`internal/api`,
  typed errors, compare-and-swap) were **deliberately not filed**, on area 1's own stated
  condition — the extraction only pays for itself if a single audited *write* chokepoint is the
  goal, and it is not while no second writer exists. So the duplication hazard is gone: lifting
  the `serve` view structs into something a CLI command can also use is this ticket's work
  alone, and refinement option (b) (*"fold into T-056 area 1 and drop this"*) is off the table —
  the honest choices are now (a) refine as scoped or (c) drop until a consumer is real.
- **T-043** — CLI test harness: **landed 2026-08-06**. New subcommands are exactly what it
  covers, so a `board json`/`ticket json` verb arrives with a harness already in place
  (`capture(t, …)` for stdout/stderr, `newProject(t)` for a throwaway install, and the
  `runProject*`/`runTicketNew`/`runBoardAudit` cli-level tests as the pattern to copy).
- **T-052** — **done**, and it already resolved the vocabulary this bullet used to flag as
  open: `board audit`'s old single "stale **or** hand-edited" conflation is now `board.Drift`
  (`DriftNone`/`DriftLayout`/`DriftRows`), surfaced as a warning (layout-only) or an error (rows
  differ). If audit health is exposed as structured data here, reuse that vocabulary (or its
  two-tier shape) as the field's values rather than inventing a third naming.
- **T-085** (per-ticket record aggregable) — **a second prospective consumer**, and the one that
  motivates the findings-table scope question above. T-085 adds a `class` column to the findings
  table so recurring defect kinds can be counted; counting them without this projection means
  grepping markdown, which is exactly how the T-045 spawn rate was measured. Neither blocks the
  other: T-085's fields are greppable by a human on day one, and this projection is useful
  without them. Sequencing preference is T-085 first — it costs a column and settles whether the
  data is worth a wire format at all.

### Honest scope of the benefit

**There is no consumer today, and the motivating one was withdrawn the same day this was
filed.** The candidate was an agent-harness extension enforcing flow gates; it was assessed
against the field record hours later and **not filed** — its one evidenced rule already belongs
to T-057, which had itself already concluded that a harness extension is the wrong primary
mechanism (*"a pi extension only guards a pi session"*). See `tickets/NOTES.md`, "Second
postscript (2026-08-04)".

What survives without it: pickle has **no machine-readable output at all**. (The second half of
this argument — *"and T-056 work area 1 would have to build this projection regardless"* —
**expired on 2026-08-14**: area 1 was not filed when T-056 was split, so no other ticket will
build it. Weigh option (c) accordingly.) That is real but weaker than the case at filing time. Graded `low-medium` — enabling infrastructure with deferred, now less certain,
payoff; it should not outrank tickets that fix measured field defects.

**Second prospective consumer, and deliberately not a re-grade (2026-08-07).** T-085 wants this
projection as a measurement substrate. It is *prospective* demand from a ticket that is itself
unrefined, and the 2026-08-04 precedent — set when T-056 was downgraded, and applied again the
same day to decline a T-081 bump — **refuses to credit prospective demand when grading**. The
rule cuts both ways, so `low-medium` stands. What changes is the refinement question, not the
grade: option (c) *"drop until a consumer is real"* is now weaker than it was, because a second
independent use has appeared for the same projection.

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
- **That the projection should be writable, or served over HTTP.** Read-only, CLI-only. Locking
  is **T-101**, the ticket field writer is **T-102**, and `serve`'s first write route is
  **T-079** (all three inherited from T-056, dropped 2026-08-14).

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-04 — created (TO DO). source: chat — the Pi-as-best-tier exploration recorded in
  tickets/NOTES.md (2026-08-04); scope corrected before filing after reading the code (the two
  commands it was to flag did not exist, and the paired `schema_version` guard was found to
  have no reachable hazard, so it was pre-registered here instead of filed)
- 2026-08-06 — patched by T-043's review impact sweep: T-043 landed, so the cli-test harness this
  ticket's acceptance test would have needed already exists — the note now says what to reuse
  instead of what to expect
- 2026-08-07 — scope question added (the `## Review` findings table) and T-085 recorded as a
  second prospective consumer; grade deliberately unchanged per the 2026-08-04 precedent against
  crediting prospective demand
- 2026-08-07 — patched by T-052's review impact sweep: T-052 landed, resolving the vocabulary
  question this ticket's T-052 soft-coupling note had left open (`board.Drift` —
  `DriftNone`/`DriftLayout`/`DriftRows` — replaces the old single "stale or hand-edited"
  conflation); the note now says what to reuse instead of what to agree on
- 2026-08-13 — patched by T-085's review impact sweep (step 8): T-085 shipped the `class` column
  and, more importantly for this ticket, a **single canonical table skeleton** — so the open
  scope question's claim that "`TEMPLATE.md` already fixes" the columns is now false in both
  halves (TEMPLATE.md points rather than states, and the list is seven columns led by `class`).
  The assumption is *strengthened*, not invalidated: the middle option this ticket already
  costed — project the closed-vocabulary columns only — now has a fixed shape to parse against
  instead of 13 header variants. Wording corrected; nothing re-graded
