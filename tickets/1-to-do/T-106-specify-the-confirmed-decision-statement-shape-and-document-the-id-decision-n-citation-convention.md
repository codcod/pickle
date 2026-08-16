---
id: T-106
title: specify the confirmed-decision statement shape and document the <ID> decision <N> citation convention
project: pickle
depends-on: []
spawned-by: [T-105]
impact: low-medium
complexity: low
cost: S
---

# T-106 — specify the confirmed-decision statement shape and document the <ID> decision <N> citation convention

## Outcome

A ticket author writing confirmed design decisions is told the shape to write them in, and anyone
citing a decision from another ticket is told the form to cite it in. Both conventions already
exist and are already relied on across projects; neither is written down anywhere, so today they
are learned only by imitation.

## Description

Two conventions carry real weight in the brine flow and are documented in none of the three payload
resources (`tickets-README.md`, `TEMPLATE.md`, `review-protocol.md` — verified: zero matches):

1. **The decision shape.** A confirmed design decision is written as a numbered item whose leading
   **bold run is the decision statement** and whose remainder is the rationale:
   `1. **The hook performs no network I/O.** No `git fetch`. A stale remote-tracking ref can only …`
2. **The citation form.** A decision is cited from another ticket as `<TICKET-ID> decision <N>` —
   for example, one contract in this repo is inherited across three tickets, each citing the last
   by that form. `review-protocol.md` §5 already makes "contradicts a locked decision" a *blocking*
   severity, so citing a decision precisely is load-bearing at review time, yet the form is
   nowhere specified.

**These are emergent conventions to be codified, not new rules to impose.** Measured at filing
(2026-08-16; figures will drift, re-measure rather than trusting them): the statement shape holds
for **367 of 397 decisions (92%)** in this repo and — the load-bearing datum — **120 of 138 (86%)**
in an unrelated three-child brine workspace that has never seen this ticket. A convention that
reached ~9 in 10 unprompted, in two corpora independently, is a convention the payload should
state rather than one the payload would be inventing.

**Foreign-workspace test applies.** This edits `TEMPLATE.md`, which ships to projects that are not
pickle, so per `AGENTS.md` the wording must not cite this repo's ticket ids as things to look up,
must not quote counts from a corpus the reader does not have, and must not phrase paths relative
to this source tree. The reference workspace above is evidence *for the ticket*, not text to ship:
the shipped sentence should teach the shape by example, with a syntactic filler id.

Also note the citation form must be written **prefix-agnostically** (`<ID> decision <N>`, not
`T-NNN decision N`): children set their own `ticket_prefix`, and the reference workspace uses
`RICK-` and `SNOW-` alongside a `T-` child.

### Open questions for refinement

- **Whether `board audit` gains a row.** The natural companion is a *warning* when a decision does
  not match the shape. Two constraints: backfilling is refused by precedent (T-025, archaeology
  with no consumer), so enforcement must be prospective and can never reach 100%; and an audit row
  is permanent surface for a convention that already sustains itself at ~90% unaided. Refinement
  should decide deliberately, and **not adding the row is a legitimate outcome** — in which case
  this ticket is documentation only.
- **Where the citation form is documented** — `tickets-README.md` §7 (ticket structure) and/or the
  `TEMPLATE.md` section comment. Pick one home, cross-reference from the other.

### Scope fence

- **No retrofit** of the ~10% non-conforming decisions in either corpus (T-025 precedent).
- **No locked-vs-ticket-local marking.** Parked with a pre-registered trigger in `NOTES.md` §
  *"ADR exploration (2026-08-15) — explored, nothing filed; the convention already works"*.
- **No command.** T-105 is the query surface and does not depend on this ticket: it reads the
  corpus as it stands and reports non-conforming items rather than requiring conformance. The two
  are independently schedulable in either order.

### Soft couplings (no hard `depends-on`)

T-105 (the query command, spawned this split), T-098/T-099 (the payload's foreign-workspace rules
and the lint that enforces the mechanical part of them — the new wording must pass
`payload_lint_test.go`).

### Grading rationale

`impact: low-medium` — higher than T-105's `low` because this closes a *present* documentation gap
in a convention the review protocol already depends on, rather than serving prospective demand.
`complexity: low`, `cost: S`: prose in one or two payload files, plus a lint/test pass, plus an
audit row only if refinement decides to add one.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-16 — created (TO DO). source: review: split out of T-105 during an adversarial pass that
  found it bundled four separable changes. The format specification and any audit enforcement are
  independently schedulable from the query command and are neither a prerequisite nor a consequence
  of it, so they became this ticket rather than staying tasks in that plan (rules §3). Evidence that
  the conventions are emergent rather than imposed — 86% conformance in an unrelated workspace — was
  gathered in the same pass
