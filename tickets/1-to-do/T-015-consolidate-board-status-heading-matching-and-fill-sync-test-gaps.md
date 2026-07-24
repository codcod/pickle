---
id: T-015
title: consolidate board status-heading matching and fill sync test gaps
project: pickle
depends-on: []
impact: low
complexity: low
cost: S
---

# T-015 — consolidate board status-heading matching and fill sync test gaps

## Description

Internal-quality polish surfaced by the T-008 (board sync) review — no user-facing behaviour
change. Two items, both in the `internal/board` + `internal/sync` layer:

1. **De-duplicate status-heading matching.** The "match a `## ` heading line to its status
   display name, longest-name-first so a prefix can't shadow a longer one" logic now exists in
   **four** places: `board.Parse`, `board.ParseCells`, `sync.matchStatus`, and a variant in
   `board.sectionSpan` (`board.go:~329`). Extract a single exported helper (e.g.
   `board.MatchStatusHeading(line string) string`, returning `""` for a non-status heading)
   and route all four callers through it. This also removes the per-call rebuild+sort of the
   status-name slice (negligible perf, but the shared helper can memoise it once).

2. **Fill the sync D3 test gap.** `internal/sync.TestSyncTerminalMembership` covers only the
   first half of decision D3 ("a DONE ticket **not** on the board is not re-added"). The plan
   also specified the second half — "a DONE ticket listed under the **wrong** section is
   relocated (to DONE) with its `merged` cell." Add that case. While doing so, document the
   observed behaviour that a `merged` cell does **not** survive relocation *from a wrong
   section* (because `board.ParseCells` keys carry-over cells by the columns of the section the
   row was found in, so a row misfiled under, say, TO DO has no `merged` cell to carry) — decide
   whether that is acceptable (likely yes: a misfiled row's cells were never DONE-shaped) or
   whether carry-over should fall back to an id-keyed scan, and encode the decision in the test.

Soft coupling: builds on T-008 (board sync) and T-002 (board audit engine), both merged; no
hard `depends-on:` — the target code is already on `main`.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-24 — created (TO DO). source: T-008 review (non-blocking findings N1 duplication + N2 test gap)
