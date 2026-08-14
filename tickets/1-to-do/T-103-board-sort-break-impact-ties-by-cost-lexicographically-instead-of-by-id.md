---
id: T-103
title: board.Sort: break impact ties by cost lexicographically instead of by id
project: pickle
depends-on: []
spawned-by: [T-056]
impact: low
complexity: low
cost: S
---

# T-103 — board.Sort: break impact ties by cost lexicographically instead of by id

## Outcome

After this ships, two tickets with the same impact are ordered cheapest-first on the board
instead of oldest-first, so the TO DO group's largest tie stops being an arbitrary list.

## Description

`board.Sort` (`internal/board/board.go:240`) orders TO DO/READY by impact descending and breaks
ties by **id** — i.e. by filing order, which carries no priority information at all. Impact is a
four-value ordinal over a backlog that is now 21 tickets deep in TO DO, so ties are the common
case, not the exception: **7 tickets share `medium` and 7 share `low-medium`** as of
2026-08-14. Within each of those groups the board's order is the order they happened to be
filed in.

The change is to compare `cost` **lexicographically beneath impact** — a `costRank` map
(`S<M<L<XL`, with the adjacent-pair ranges the rules allow slotting between) and roughly four
lines in the existing comparator, ascending so cheap wins a tie. Nothing else changes: no config
surface, no new frontmatter, no second source of truth, and decision **D1** at `board.go:231`
("deterministic, no hand-curated order") stays intact — this is still a pure function of the
ticket files. T-059's `family:` contiguity rule sits above the tiebreak and is unaffected.

### Provenance, and the honest case against it

This is the **one surviving idea** from T-063 (dropped 2026-08-01, which proposed a
value-per-cost *ratio* ordering). T-063 measured the alternatives: the ratio cut tied pairs
34 → 19, while the lexicographic tiebreak cuts them 34 → **10**, can never invert impact the way
a ratio does, and is invariant under any monotone renumbering of the ordinals — the defect that
killed the ratio (renumbering `cost` on an equally defensible scale moved 11 of 18 rows).

T-056 work area 5 recorded the idea with a trigger: file it only **if an `impact` recalibration
pass leaves the `medium` group ≥5 deep**. Two recalibration passes have been run (`NOTES.md`,
2026-08-01 and 2026-08-03) and the group is 7 deep, so the trigger has fired and this is that
ticket.

**T-063's fatal finding still applies and is why this is graded `low`.** The queue anyone picks
from is READY, not TO DO — and across all **294 revisions** of `tickets/BOARD.md`, READY has
held 0 rows in 205 of them, 1 row in 65, 2 rows in 22 and 3 rows in 2. It has never held more
than three. Re-ordering a 21-row TO DO list improves a *reading* surface, not a *pickup* queue.
That is a real but narrow win, which is exactly what `low` means. If refinement finds the change
costs more than the ~4 lines claimed here, dropping it is the right answer.

### Soft couplings

- **T-056** (dropped 2026-08-14) — its work area 5 carried this idea and its trigger.
- **T-063** (dropped) — the hearing that produced the measurements above; read its DROPPED
  banner before refining, including the errors it marks in its own text.
- **T-042** — also touches `internal/board`; sequence, do not run concurrently.
- **T-059** (done) — `family:` grouping in the same comparator.
- **T-045** (dropped) — the `user-visible:` axis, the other proposed answer to wide impact ties.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-14 — created (TO DO). source: chat: refinement split of T-056 (dropped the same day) —
  the surviving residue of its work area 5, filed because the trigger it recorded (an `impact`
  recalibration leaving the `medium` group ≥5 deep) has fired at 7
