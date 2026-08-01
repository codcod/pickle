---
id: T-063
title: order TO DO/READY by value per cost, not impact alone: use the cost axis, de-rank blocked tickets, behind a configurable order policy
project: pickle
depends-on: []
spawned-by: [T-056]
impact: medium-high
complexity: medium
cost: M
---

# T-063 — order TO DO/READY by value per cost, not impact alone: use the cost axis, de-rank blocked tickets, behind a configurable order policy

## Description

The board orders each child's TO DO/READY group by **impact alone** (`impactRank`,
`internal/board/board.go:22-25`; `Sort`, `:248-276`). Tickets also carry a `cost` grade —
documented as implementation effort in sessions (`tickets-README.md:133`), required by the
audit (`internal/audit/audit.go:24`), validated (`internal/ticket/ticket.go:368`), collected
at creation (`internal/cli/ticket.go:83`) and rendered as a board column (`board.go:105`) —
and **nothing reads it**. "What should I pick up next?" is the flow's central question and
the board answers it with half the data it already has.

This ticket adds the other half, as a **pure function of the ticket files**: order by value
per cost, and sink tickets that cannot legally be started. It introduces no new state, so it
keeps decision **D1 — deterministic, no hand-curated order** (`board.go:239-241`) and the
audit's "BOARD.md equals a fresh render" assertion (`audit.go:86`) intact.

### The two ordering changes

1. **Use `cost`.** A `costRank` mirroring `impactRank`'s seven levels (`S:1, S-M:2, M:3,
   M-L:4, L:5, L-XL:6, XL:7` — the adjacent-pair ranges are legal values, `ticket.go:368`),
   combined with impact into a score. Cheap high-impact work should outrank expensive
   high-impact work; today they tie and fall through to id order.
2. **De-rank blocked tickets.** A ticket whose `depends-on:` is not done-and-merged **cannot
   be picked up** — `move.Move` refuses it (`internal/move/move.go:101-115`). Sorting an
   unstartable ticket to the top of the pickup list is a worse defect than ignoring cost, and
   readiness is equally derivable from the tree. `spawned-by:` must not participate: it gates
   nothing (`move.go:100`, `tickets-README.md:154-156`).

Both flow into `pickle serve` for free — the dashboard sorts through the exported `board.Sort`
rather than a copy (`internal/serve/view.go:88,101`).

### Configurable as a named policy, not weights

Free-form weights are the wrong shape: nobody can tune them without outcome data, so they
would be invented numbers, and they would make two workspaces' boards incomparable. A small
enum can be validated, documented and golden-tested:

```toml
[board]
order = "impact"           # default — today's behaviour, byte-identical boards on upgrade
# order = "value-per-cost"
```

The default **must** be today's behaviour, or every existing workspace's board churns on
upgrade.

### What was considered and rejected

- **A stored score (frontmatter attribute or sidecar `.md`), recomputed on a trigger.** This
  was the original request's shape: reassess on `ticket new` and on move-to-done. It is
  unnecessary — a derived order is always current, and `board.Regenerate` already runs on
  exactly those events (`internal/cli/ticket.go:144`, `move.go:137`, `internal/sync`). Storing
  it costs a second source of truth the audit must police for staleness (a failure mode that
  cannot exist today), plus either N rewritten ticket files per reassessment polluting every
  diff, or one hot merge-conflict sidecar that stops tickets being self-describing — the
  objection T-056 work area 5 already raises.
- **Ordering `3-in-development/` and `4-in-review/` too.** Also in the original request. Those
  are WIP-capped (default 1, `pickle.toml:33-34`, enforced `move.go:151-170`) and render no
  grade columns (`board.go:106-109`). More importantly they are *already committed* work —
  their order does not change what anyone does next. If a workspace raises the caps this can
  be revisited on evidence.

### Design questions a plan must answer

- **`complexity` vs `cost` as the divisor.** There are two effort-ish axes. `cost` is effort;
  `complexity` is design uncertainty (`tickets-README.md:131-133`). Value-per-cost should
  divide by `cost`; `complexity` is better as a tiebreak or a risk marker. Decide explicitly
  and record it — this is the main reason to ship a fixed policy rather than a knob.
- **Ranges project false precision.** `low-medium`, `S-M`, `M-L` exist to encode *honest
  uncertainty* on unrefined tickets (`tickets-README.md:135-137`). Rendering a decimal score
  against one is a lie dressed as data. Prefer rendering an *ordering* (or at most a coarse
  bucket) over a number; if a score column is added, justify it.
- **Families must stay contiguous.** `famRank` sinks a family to its umbrella's impact
  (`board.go:298-308`, T-059). Under a new policy it must sink to the umbrella's *score*, or
  families fragment.
- **Tiebreaks.** Small integer ranks tie widely. The existing chain (family key → umbrella
  first → own impact → `Num`) must be preserved beneath the new comparator, and
  `sort.SliceStable` kept.
- **Section headings are hardcoded** — `"TO DO (impact order, per child)"`, `board.go:38-39`
  — and become policy-derived. *Verified non-hazard:* `board.Parse` matches a status heading
  by prefix (`board.go:79`), so a changed parenthetical still parses.
- **Config plumbing is ~six stations and two are duplicated** (`internal/config/config.go`:
  default, struct, `applyDefaults`, `AddProject`'s bypass, `Validate`, `Render`) — the same
  trap catalogued in T-045. A new `[board]` table is a first for the schema; confirm
  `Render` (`config.go:264-299`) emits it and the in-place upgrade path
  (`verifyOnlyPayloadVersion`, `:388-409`) tolerates it.
- **Scope: overarching or per-child?** Recommended overarching — the board is one artifact.
- **Is it worth it at this backlog size?** 17 TO DO tickets in a single child. The honest
  counter-position, from T-045 and `NOTES.md:36-40`, is *recalibrate `impact`* rather than
  add ordering machinery. `family:` (T-059) shipped on the argument that impact ties widely
  at this size, which is precedent — but refinement must argue this rather than assume it,
  and dropping the ticket is a legitimate outcome.

### Soft couplings

- **T-056** (`spawned-by`) — its work area 5 asks whether to rank manually or "don't rank at
  all and add axes instead", and says the cheap option "deserves a real hearing before
  drag-and-drop is built". This ticket **is** that hearing, resolved in favour of derived
  ordering. Landing it should shrink or retire T-056 work area 5; refinement of either must
  reconcile them.
- **T-045** — adjacent (a `user-visible:` axis, a backlog cap) but a different mechanism, and
  measurement-gated. This ticket adds no axis, so the two do not collide.
- **T-052** — changing `order` in `pickle.toml` makes BOARD.md differ from a fresh render
  until `board sync`, and the audit reports that as a hand-edit. Exactly the class of false
  diagnosis T-052 describes; this ticket adds one more trigger for it.
- **T-042** — touches `internal/board`; `NOTES.md:47-48` warns it must not run concurrently
  with other board work. Sequence, do not parallelise.
- **T-059** (done) — the family sort this comparator must not break.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-01 — created (TO DO). source: chat — exploration of a proposed value/impact ordering
  feature; the proposal's stored-score-plus-trigger and in-review-ordering halves were rejected
  during that exploration and the reasons are recorded in the Description
