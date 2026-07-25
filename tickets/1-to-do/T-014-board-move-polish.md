---
id: T-014
title: board-row and move polish (WIP counts, cell escaping, subgroup spacing, atomicity)
project: pickle
depends-on: [T-007]
spawned-by: []
impact: low
complexity: low
cost: S
---

# T-014 — board-row and move polish (WIP counts, cell escaping, subgroup spacing, atomicity)

## Description

Non-blocking follow-ups from the T-007 review. `pickle ticket move` (T-007) works and is
board-audit-clean; these are cohesive quality/robustness items on the board-rendering + move
surface, none of which change the happy path:

1. **WIP-count headings not refreshed.** `board.MoveRow` moves rows correctly but never updates
   the `### <child> (n/limit)` count in the IN DEVELOPMENT / IN REVIEW headings, so the count
   goes stale after a move (had to be hand-corrected during T-007). `board audit` ignores the
   count (`Parse` strips `" ("`), hence cosmetic. Recompute and rewrite the `(n/limit)` for the
   affected child + section on every move (and consider auditing it as a warning).
2. **Board cells are not escaped.** A `|` (or a stray newline) in a rendered cell malforms the
   row — reproduced with `ticket move T-003 dropped --reason "has | pipe"` producing
   `| T-003 | Gamma | has | pipe |` (a 4-cell row in a 3-column section). Same class for
   `title` (via `ticket new`/`AddTODORow`) and the child name substituted into `BOARD.md`
   (this **supersedes T-013 item 5**). Escape/replace `|` and collapse newlines in every cell
   `board.renderRow` emits (one choke point), and add a table test.
3. **create-sub-group spacing.** When `MoveRow` creates a *new* `### <child>` sub-group at the
   end of a section, the inserted row is left with no blank line before the following `## `
   heading (cosmetic; the empty-existing-subgroup case was already fixed in T-007). Emit a
   trailing blank in the created block.
4. **Move atomicity.** `move.Move` writes the History line to the old path *before* the
   `os.Rename` + board rewrite, so a failure mid-apply can leave a partial move (file moved but
   board stale, or history written but file un-renamed). Acceptable for a local dev tool, but
   tighten if cheap: write history into the destination after a successful rename, and/or add a
   note to the CLI error telling the user to re-run `board audit`. (`move` already runs a
   post-move audit self-check, which surfaces a partial move loudly.)

All items are rendering/robustness polish on a working command, hence non-blocking.

> **Process note carried from the T-007 review (already resolved, no action here):** `main`
> briefly carried a *duplicate* `T-007` tracked file because a refine-commit used `mv` instead
> of `git mv` and never staged the `1-to-do/` deletion. `board audit` reads the **working
> tree**, so it passed while the **committed** tree was dirty. The T-007 branch removes the
> stale file, so its merge repairs `main`. Lesson for the flow: use `git mv` for ticket moves;
> a git-aware audit mode (check `git ls-tree` too) would have caught it — capture separately if
> wanted.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-24 — created (TO DO). source: T-007 review (non-blocking findings); via pickle ticket new
