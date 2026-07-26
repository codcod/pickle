---
id: T-034
title: board audit: flag table rows with the wrong cell count; harden AddTODORow insert point
project: pickle
depends-on: []
spawned-by: [T-030]
impact: medium
complexity: low
cost: S
---

# T-034 — board audit: flag table rows with the wrong cell count; harden AddTODORow insert point

## Description

> **ABSORBED into T-039 (board triage, 2026-07-26) — this ticket is closed, its work is not.**
> Everything below stands as the record: the analysis, measurements and line references
> are still the authoritative detail for this part of T-039's scope. Do not re-file it;
> do not implement from here. T-039 is the refinable, reviewable unit.

`pickle board audit` validates a board row by finding its **id**, and nothing else about the row's
shape. So a row with the wrong number of cells — or a row split across two physical lines — passes
as clean, and `board.AddTODORow` then inserts *subsequent* rows at the wrong place. Surfaced by the
T-030 applicability gate (2026-07-25), which measured both halves against a freshly built binary in
a throwaway install.

### 1. A malformed row is audit-clean (the detection gap)

Two shapes, both reported `0 error(s)`:

```
| T-003 | pipe | in title | medium | medium | M | [] |     ← 8 cells in a 6-column table
| T-002 | evil                                             ← row split across two physical
project: nope | medium | medium | M | [] |                    lines by an injected newline
```

The parse only needs the id on the first physical line, so nothing notices. This is the check
**T-014 already guessed at** — its item 2 note muses "consider whether the audit should also reject
a row whose cell count is wrong — that is what would have caught both". This ticket is that
consideration, split out so T-014 stays a rendering ticket.

### 2. A split row silently reorders the board (the insert-point bug)

Worse than a cosmetic malformation. Once a continuation line is orphaned, `board.AddTODORow`
inserts new rows **before** it, so ordering scrambles. Measured in the gate's sandbox after three
consecutive creations:

```
T-002, T-003, T-001        ← insertion order was T-001, T-002, T-003
```

The row-insert point is found by scanning for the section/sub-group and its table; an orphaned
continuation line is indistinguishable from a table row to that scan. Harden the scan (and/or make
it refuse to operate on a table it cannot parse cleanly, which is where item 1's cell-count
knowledge pays for itself twice).

**A second, simpler trigger — measured on this repo's real board while filing T-035.** The
insert-point scan reads each row's impact as `cells[3]` of `strings.Split(lines[i], "|")`
(`internal/board/board.go:282-286`) and inserts before the first row satisfying
`impactRank[rowImpact] < newRank` (`:287`). A row with an **extra pipe** shifts that index, so the
impact reads as arbitrary title text with `impactRank` **0** — below every legal impact (`low` is 1,
`:17-20`). The malformed row becomes a wall that **every** newly added row is inserted in front of,
regardless of impact. Reproduced live: creating T-034 and T-035 both landed immediately above the
corrupt `T-021` row, putting low-impact T-035 inside the medium block (hand-corrected). So the bug
needs no injected newline at all — one unescaped `|` in a title is enough, and this repo has been
carrying it. Note that escaping as `\|` does **not** fix this parse (`strings.Split` is not
escape-aware); either replace the character or make the row split escape-aware here.

**Confirmed empirically on 2026-07-26, and this ticket now owns the whole residual.** The 2026-07-26
triage repaired the row inline as `| T-021 | project add\|remove leave the AGENTS.md marker block
stale | medium | low | S | [] |` and dropped **T-035**. Re-measured against that *repaired* row:

```
n fields : 9      (a 6-column row still splits into 8)
cells[3] : "remove leave the AGENTS.md marker block stale"
impactRank -> 0   ← the wall is intact
```

So the prediction above held: the escape fixed the rendered table and nothing else. With T-035
dropped, **this ticket is the only remaining owner** of the insert-point bug, and `tickets/BOARD.md`
is a live reproduction — no fixture needed to write the failing test. Two consequences for the
plan when this is refined:

- The regression gate can no longer be "0 errors against the real `tickets/`" as originally scoped,
  because the real board still contains the offending row. Either the cell-count check must treat
  `\|` as escaped (making the row legal, 6 cells) *and* the row split must be made escape-aware
  together, or the row must be re-repaired by replacement. Decide those two as one question — they
  are the same decision seen from the audit side and the insert side.
- **T-014** item 2 must be told which way this goes: if the answer is "replace, don't escape", its
  `\|` escaping proposal is wrong and the render boundary should substitute instead.

### Scope

- **Audit:** a check that every board row in every section has exactly the column count its
  section's header declares, and that no orphaned continuation line sits inside a table. Error, not
  warning — a malformed row is a broken invariant, not a judgement call.
- **`board.AddTODORow` / `board.MoveRow`:** make the insert-point scan robust against a table it
  cannot parse, rather than silently picking a wrong line.
- **Regression gate:** the new check must report **0 errors against this repo's real `tickets/`** —
  except for the one known-corrupt row, which is **T-035**'s repair. Sequencing matters: land T-035
  first, or this ticket's own acceptance test starts red on real data. Neither hard-depends on the
  other.

Explicitly **out of scope**: escaping cells at the render boundary (**T-014** item 2, `|` and
newline collapsing) and rejecting bad input at the CLI boundary (**T-030**, merged/landing). This
ticket is the third leg — **detecting** a malformation whatever its source, including a hand-edit or
a bad merge that neither of the other two can intercept.

### Couplings

`spawned-by: [T-030]` — found by T-030's applicability gate, outside its input-boundary scope.

Soft couplings (no `depends-on`, no ordering enforced):

- **T-035** — repairs the single corrupt row in this repo's own board. Should land first so this
  ticket's regression gate is meaningful; see Scope.
- **T-014** — owns render-boundary escaping and explicitly hands this check off; remove that musing
  from T-014 item 2 when this lands, or leave it as a cross-reference.
- **T-030** — closed the `ticket new` route that produced the split row. After it, item 1's split-row
  case must be reproduced by hand-editing a board or via `ticket move --reason`, not by `ticket new`.
- **T-033** — the same "the validator endorses a malformed artifact" class, one layer down
  (duplicate frontmatter keys inside a ticket file). Independent, but worth reading together; a
  shared verdict on error-vs-warning would be good.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-25 — created (TO DO). source: pickle ticket new
- 2026-07-26 — TO DO → DROPPED: absorbed into T-039 (board triage merge); content preserved here as the record
