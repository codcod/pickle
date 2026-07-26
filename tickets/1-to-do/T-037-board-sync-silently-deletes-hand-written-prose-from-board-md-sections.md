---
id: T-037
title: board sync silently deletes hand-written prose from BOARD.md sections
project: pickle
depends-on: []
spawned-by: []
impact: high
complexity: medium
cost: M
---

# T-037 — board sync silently deletes hand-written prose from BOARD.md sections

## Description

`pickle board sync` rebuilds each status section from the ticket files and **discards any
hand-written prose inside that section**, without warning, without a diff, and without a
`--dry-run` signal that distinguishes "reformat" from "delete your paragraphs".

**Reproduced 2026-07-26** during the board triage. Two explanatory paragraphs were added between
the `### pickle` sub-heading and the TO DO table (a "Parked: T-009, T-010, T-016" note and a
merge-candidates list). `./pickle board sync` reported:

```
  reformat only (ordering / WIP counts / spacing / Last-updated)
board sync: rebuilt tickets/BOARD.md (1 change(s))
```

— and both paragraphs were gone. The summary line says *reformat only*, and the change count
says *1*, while nine lines of authored content were removed. The content was recovered only
because the triage had taken a manual backup first.

**This is T-018's bug class, in a command T-018 did not touch.** T-018 ("upgrade must not
silently discard user content") was graded **high** and shipped a surgical-edit + parse-back-gate
design so that `upgrade` could never destroy hand-written `pickle.toml` comments or the
`AGENTS.md` marker body. `board sync` has the identical failure mode against `BOARD.md`, which is
the one file in the flow the rules explicitly call **hand-maintained** (`tickets/README.md`, the
board rule; `BOARD.md`'s own header says "The maintained index…"). The flow tells users to write
in this file and ships a command that eats what they write.

The blast radius is every installed project, not just this repo: `BOARD.md` is where a team would
naturally record parked sets, triage decisions, dependency narrative, or per-child conventions.
The skeleton (`skill/resources/BOARD.md`) even ships with prose sections of its own — the
"Dependency chain" and "Known soft couplings" blocks below the tables survive today only because
they sit *outside* any status section, which is an accident of layout rather than a guarantee.

### Scope

1. **Preserve prose.** `sync` must rewrite only the table rows it owns and leave every other line
   in a section byte-identical. The natural shape mirrors T-018: locate the table span precisely,
   edit in place, and leave the rest alone — rather than re-rendering the section.
2. **Gate the write.** Adopt T-018's parse-back gate: after building the new text, re-parse it and
   refuse the write unless every non-row line is unchanged and the ticket set round-trips.
3. **Make `--dry-run` honest.** "reformat only (ordering / WIP counts / spacing / Last-updated)"
   was actively misleading here — it must name content deletion when content would be deleted, and
   the change count must reflect lines, not sections. Consider printing a real diff.
4. **Decide the contract and document it.** If some region genuinely cannot be preserved, say so
   in `BOARD.md`'s header and in the README, so users know where it is safe to write.

### Relationships (soft couplings, no hard `depends-on:`)

- **T-018** (done) is the reference implementation for both the surgical edit and the parse-back
  gate — reuse its approach and its fuzz-style test corpus rather than inventing a second one.
- **T-014·2** and **T-023** are the *other* two ways `sync` corrupts `BOARD.md`, both re-confirmed
  in the same triage run: `sync.rowFor` (`internal/sync/sync.go:281`) renders `Title` unescaped, so
  it re-corrupts a repaired `|` cell; and it overwrites a correct `branch` cell with the
  slug-derived guess. All three are "sync asserts its model over the file's truth" and may be
  cheaper to fix as one pass over `internal/sync`.
- **T-015** touches `internal/sync` test gaps (decision D3) but not this; no overlap in scope.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-26 — created (TO DO). source: board triage — reproduced directly when `board sync`
  deleted two authored paragraphs from the TO DO section while reporting "reformat only"
