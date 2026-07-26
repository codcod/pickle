---
id: T-039
title: BOARD.md write and validate integrity (escaping, sync preservation, row shape, branch cell)
project: pickle
depends-on: []
spawned-by: [T-014, T-023, T-034, T-037]
impact: high
complexity: high
cost: L
---

# T-039 — BOARD.md write and validate integrity (escaping, sync preservation, row shape, branch cell)

## Description

> **SUPERSEDED by T-044 (refinement decision, 2026-07-26) — this ticket is closed, its
> problem is not.** T-044 removes the design flaw this epic would have hardened: BOARD.md
> becomes a pure generated artifact, so the escape-vs-replace question, the sync-preservation
> class, the row-shape checks and the branch cell all dissolve by construction. The one item
> T-044's design does not solve — move atomicity (T-014·4) — is folded into T-044 (its
> decision D7). The absorbed sources below (T-014, T-023, T-034, T-037, in `7-dropped/`)
> remain the authoritative evidence record, now serving T-044. Do not implement from here.

**Epic — merged from T-014, T-023, T-034 and T-037 by the 2026-07-26 board triage.** Every
absorbed ticket is in `tickets/7-dropped/` with its full evidence, measurements and line
references intact; read them for detail. This ticket is the single refinable, reviewable unit.

`tickets/BOARD.md` is the one file the flow calls **hand-maintained**, and three separate code
paths write it — `board.MoveRow`/`AddTODORow` (moves and new rows), `sync` (full rebuild), and
the `audit` that is supposed to catch the damage. All three assert their own model over the
file's contents, and none of them agrees with the others about what a row is. The result is a
file the rules tell users to edit and the tooling quietly corrupts.

### The one decision at the centre: escape or replace?

Everything below turns on a single choice, and the merge exists mainly so it is made **once**:

- `board.renderRow` emits `|` in a cell verbatim, malforming the row (T-014 item 2).
- `insertIntoBoard` splits with `strings.Split(line, "|")` (`internal/board/board.go:282`),
  which is **not escape-aware** (T-034).
- `sync.rowFor` (`internal/sync/sync.go:281`) assigns `Title: t.Front["title"]` raw, so it
  re-corrupts any hand-repair (T-014 item 2, re-measured 2026-07-26).

The 2026-07-26 triage escaped the live T-021 row to `project add\|remove …`. Re-measured
afterwards, the row still splits into 8 fields, `cells[3]` is still
`"remove leave the AGENTS.md marker block stale"`, and its `impactRank` is still 0 — so **every
new TO DO row is still inserted in front of T-021**. The escape fixed rendering and nothing else.
Either the split becomes escape-aware *and* the cell-count check treats `\|` as one cell, or the
render boundary substitutes the character instead. Decide it once, here, and apply it to all
three writers plus the audit.

`tickets/BOARD.md` is therefore a **live reproduction** — no fixture is needed to write the
failing test, but the regression gate can no longer be "0 errors against the real `tickets/`"
until the row is repaired the chosen way.

### Absorbed scope

| from | item | substance |
|---|---|---|
| T-037 | sync destroys prose | `board sync` deletes hand-written paragraphs inside a status section while reporting `reformat only (1 change(s))`. Reproduced 2026-07-26: nine authored lines removed. This is **T-018's bug class** — adopt its surgical-edit + parse-back-gate design: rewrite only the table rows, refuse the write unless every non-row line is byte-identical. Make `--dry-run` name content deletion, and document which regions are safe to write in. |
| T-014·2 | cell escaping | One choke point for `\|` and newlines across `renderRow`, `AddTODORow`, `MoveRow` **and** `sync.rowFor`. Acceptance must assert `board sync` is idempotent on a title containing a pipe. |
| T-034 | row shape + insert point | Audit check that every row has its section's column count, and that no row is split across physical lines (both currently report 0 errors). Harden the `AddTODORow` insert point so a malformed row cannot misplace later rows. |
| T-014·1 | stale WIP counts | `MoveRow` never updates `### <child> (n/limit)`; counts go stale after every move. Recompute per move; consider auditing as a warning. |
| T-014·3 | sub-group spacing | `MoveRow` creating a new `### <child>` sub-group emits wrong blank-line spacing. |
| T-014·4 | move atomicity | `move.Move` writes the History line to the old path *before* the rename, so a crash between the two leaves a ticket with a transition it did not make. |
| T-023 | branch cell is a guess | The `branch` column is derived from the filename slug, not the real branch. Two confirmed instances (T-018, T-030); `sync` actively overwrites a hand-corrected value, so "wrong cell, sync-clean" and "right cell, permanently sync-dirty" are the only stable states today. Options in the dropped ticket: read the plan, audit against the child's git refs, cap the slug and mark the column advisory, or drop the column. |

### Cross-references (not absorbed)

- **T-012 item 5** ("board-row title sanitization") is the same escaping question seen from
  `ticket new`. It sits in the test/coverage epic (**T-043**), which must **defer** to whatever
  this ticket decides rather than choosing separately.
- **T-030** (done) closed the `ticket new` newline route by rejecting `\n`/`\r` at input, and
  deliberately did **not** touch the pipe character. Render-boundary handling is this ticket's job.
- **T-042** absorbs T-015, whose item 2 touches `internal/sync` test gaps (decision D3) — sequence
  to avoid edit collisions.
- **T-018** (done) is the reference implementation for the surgical edit and the parse-back gate;
  reuse its approach and fuzz corpus rather than inventing a second one.

### Why one ticket

Four reviews would re-litigate the escape-vs-replace decision up to four times and rewrite the
same `internal/board` + `internal/sync` render path twice — once to make it surgical, once to
make it escaping-correct. If refinement finds the plan cannot meet the READY gate as a single
unit, split it back: the goal is fewer review cycles, not a bigger ticket.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-26 — created (TO DO). source: board triage — epic merged from T-014, T-023, T-034 and
  T-037, all four moved to 7-dropped/ as absorbed
- 2026-07-26 — TO DO → DROPPED: superseded by T-044 (generated board); move-atomicity residue folded into T-044
