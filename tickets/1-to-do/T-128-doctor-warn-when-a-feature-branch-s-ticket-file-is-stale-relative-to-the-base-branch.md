---
id: T-128
title: doctor: warn when a feature branch's ticket file is stale relative to the base branch
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: low-medium
cost: S-M
---

# T-128 — doctor: warn when a feature branch's ticket file is stale relative to the base branch

## Outcome

After this ships, `pickle doctor` on an in-tree project checks out on a `feat/T-NNN-*` branch
reports a warning when that ticket's own file disagrees with its base-branch copy (a different
status directory, or a shorter `## History`) — the same "mirror-image hazard" `pickle serve`
already warns about (T-108), now reachable from the command an agent actually runs mid-workflow,
without needing a running `serve` process.

## Description

T-108 documented and partly fixed the in-tree "mirror-image hazard": bookkeeping only lands on
the base branch, so a feature branch cut before a later status move (or review-disposition
commit) carries a stale copy of its own ticket, and every command that reads `tickets/` reports
that stale copy without saying so. T-108 shipped the warning on `pickle serve` only, and its
Implementation Plan (decision 9) explicitly deferred the rest: "Extending the signal to other
readers is a follow-up, not part of this ticket." `tickets-README.md` §0 still states the fix as
prose discipline for readers ("read the ticket and the board from the base branch... not the
branch under review"), with nothing mechanical catching the drift outside `serve`.

**This is not theoretical — it reproduced in a real installed project.** In `messgr` (an in-tree
brine install, unrelated to pickle's own repo), a ticket's board-bookkeeping move (`IN
DEVELOPMENT` → `IN REVIEW`, "acceptance green") was committed correctly on `main` after its
`feat/T-NNN-*` branch had already been cut. The feature branch's own worktree still showed the
pre-move status and a `## History` one line short. An agent resuming work on that branch, reading
the ticket from the base branch per the documented discipline, correctly reported the ticket as
not eligible for the `implement` trigger — technically correct, but surprising and blocking,
because nothing had told it *why* its own branch disagreed with base, and the fix (rebase onto
base to pick up the bookkeeping commit) is mechanical and could have been suggested on the spot.
`doctor` is the natural place for it: it is what an agent runs to sanity-check state before
acting, it is already layout-aware (T-108's `checkLayoutInvariant`), and T-046 already made it
self-host-aware for a related in-tree quirk.

**Scope.** Detection only, on `doctor`, for the in-tree layout. `doctor` already knows the
resolved layout and the checked-out branch; it needs the base branch name (recorded or resolved
the same way `serve`'s `staleBoardBranch` gets it — no new `main`/`master` guessing per T-108
decision 8) and a way to diff one ticket file between `HEAD` and that base. The check applies
only when `HEAD` matches a registered child's `branch_prefix` (mirrors `serve`'s condition), and
only to the ticket(s) whose id the branch name carries (a `feat/T-NNN-*` branch names its own
ticket) — not a full-board diff, which is a different and heavier check `board audit` might
reasonably own some day but is out of scope here.

**Soft couplings (not `depends-on`):**

- **T-108** (done) — supplies `checkLayoutInvariant`, the recorded `layout` key, and
  `staleBoardBranch`'s base-branch resolution in `internal/serve/serve.go`; this ticket reuses
  both rather than re-deriving them.
- **T-046** (done) — established `doctor` as self-host-aware; this ticket's warning must not fire
  on pickle's own `feat/T-NNN-*` branches in ways that misfire against the self-modify policy's
  throwaway-install testing (it only reads the branch's own ticket file, not any installed
  payload, so this is expected to be a non-issue, worth confirming in refinement).

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-17 — created (TO DO). source: field-use: observed in `messgr` (an in-tree brine
  install) — a ticket's board-bookkeeping move landed on `main` after its feature branch was cut,
  leaving the branch's own worktree copy stale; an agent resuming work read the ticket from base
  per the documented "mirror-image hazard" discipline (`tickets-README.md` §0) and correctly, but
  confusingly, reported the ticket as not ready for its trigger, with no mechanical signal
  pointing at the fix (rebase onto base).
