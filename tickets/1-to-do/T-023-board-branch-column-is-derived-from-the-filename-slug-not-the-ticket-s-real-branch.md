---
id: T-023
title: board branch column is derived from the filename slug, not the ticket's real branch
project: pickle
depends-on: []
impact: low
complexity: low
cost: S
---

# T-023 — board branch column is derived from the filename slug, not the ticket's real branch

## Description

Non-blocking finding 9 from the T-018 review.

`tickets/BOARD.md` has a `branch` column whose value is derived mechanically from the ticket's
**filename slug** (the branch-cell construction in `internal/move/move.go`) rather than from the
branch the work actually happens on. The two diverge whenever a ticket's title is long and its
Implementation Plan picks a shorter, human-scale branch name — which the READY gate positively
encourages, since it asks for an explicit `git checkout -b` line.

Observed on T-018: the board row read

```
feat/T-018-upgrade-must-not-silently-discard-user-content-pickle-toml-comments-agents-md-marker-body
```

while the real branch, mandated by that ticket's own Implementation Plan, was
`feat/T-018-upgrade-preserve-user-content`. Nothing caught it — `board audit` validates ids,
sections, frontmatter, dependencies and WIP counts, but never the `branch` cell — so the column
can be silently wrong for every row in the table. A reader looking for the work, or a reviewer
trying to check it out, follows a branch name that does not exist.

**Hand-correcting the cell does not work.** The T-018 review tried: editing the row to the real
branch immediately put `board sync --dry-run` into `OUT OF SYNC`, because `sync` rebuilds the cell
from the same slug derivation. So the wrong value is not merely emitted once, it is actively
re-asserted, and the only stable states are "wrong cell, sync-clean" or "right cell, permanently
sync-dirty". The review chose the former and filed this ticket. That makes a code fix the only
real option, and rules out "document the cell as advisory but fix it by hand when it matters".

Options to weigh during refinement:

- have `ticket move` read the branch from the ticket's Implementation Plan (fragile — free prose);
- have `board audit` warn when the cell names a branch absent from the target child's repo (cheap,
  catches the real failure, but couples the audit to git);
- cap the derived slug at a sane length and document the cell as advisory;
- drop the column.

Soft coupling: **T-014** (board-row and move polish) owns the neighbouring board-cell mechanics —
these may want to land together.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-25 — created (TO DO). source: pickle ticket new
