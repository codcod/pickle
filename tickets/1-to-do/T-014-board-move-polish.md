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

   **Note added by T-030's refinement (2026-07-25).** A newline in a *title* was measurably worse
   than "malformed row": it **split the row across two physical lines**
   (`| T-003 | evil` / `project: nope | medium | … |`) and `board audit` still reported 0 errors,
   because the row-presence check finds the id on the first line. T-030 (READY) closes the
   `ticket new` route by **rejecting** newline-bearing titles at the input boundary, so after it
   lands this item's newline case is no longer reproducible through `ticket new` — reproduce it via
   `ticket move --reason` or a hand-edited ticket instead. The `|` case is unaffected and still
   reproduces exactly as written above (re-measured: `ticket new 'pipe | in title'` yields an 8-cell
   row in a 6-column table, audit-clean). Render-boundary escaping remains this ticket's job; T-030
   deliberately did **not** do it. Consider whether the audit should also reject a row whose cell
   count is wrong — that is what would have caught both.

   **Note added by the 2026-07-26 board triage.** `pickle board sync` **actively re-corrupts**
   this, which makes a hand-repair of `BOARD.md` worthless and raises the priority of doing the
   escaping at the render choke point. Measured: T-021's live row (title `project add|remove …`)
   was hand-repaired to `project add\|remove …` — 6 rendered columns, audit-clean — and the very
   next `./pickle board sync` rewrote it straight back to the unescaped 7-cell form, because
   `sync.rowFor` (`internal/sync/sync.go:281`) assigns `Title: t.Front["title"]` verbatim into
   `board.RowData` with no escaping before `board.RenderRow`. So `sync` is a *second* unescaped
   render path alongside `MoveRow`/`AddTODORow`; whatever choke point this item establishes must
   cover it, and the acceptance test should assert that `board sync` is idempotent on a title
   containing `|`. The T-035 hand-repair was dropped for exactly this reason — the data fix does
   not hold until this item lands.
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
