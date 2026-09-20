---
id: T-133
title: DONE-but-unmerged audit warning bleeds onto every unrelated pickle ticket move
project: pickle
depends-on: []
spawned-by: [T-092]
impact: medium
complexity: low
cost: S
---

# T-133 — DONE-but-unmerged audit warning bleeds onto every unrelated pickle ticket move

## Outcome

`pickle ticket move` no longer reprints another ticket's "DONE but has no 'MERGED' History line"
warning on every unrelated move made while that ticket's PR is pending. The warning still fires
on an explicit `pickle board audit`, so a stale merge stays discoverable — it just stops riding
along on commands that have nothing to do with it.

## Description

`internal/move/move.go` (~line 202-206) runs a full-tree `audit.Audit(root, cfg)` as
`ticket move`'s post-condition self-check and copies **all** of its `Warnings` into the move
`Result`, which the CLI prints unconditionally. One of those warnings, added by T-092
(`internal/audit/audit.go` ~line 246-264, the "Unfinalized-merge detection" scan), fires for
*every* ticket sitting in `6-done/` with no `merged to <base>` History line — deliberately, so a
done ticket nobody depends on still gets flagged (T-092's own rationale for scanning done tickets
unconditionally rather than only via the dependency-scoped check a few lines above it).

The side effect: review-to-merge lag is normal (the human merges, and may lag — rules §3), so
while any ticket sits DONE-but-unmerged, its warning reprints on **every** `pickle ticket move`
call for the rest of the board, not just moves of that ticket. Dogfooding brine in a separate
project (messgr) this rode along on 5+ moves in a single session for one stale ticket, and
recurred across at least 6 different sessions over a week (different stale tickets each time —
T-031, T-033, T-036, T-038, T-042, T-048). It's not wrong — the audit is accurate — but it's pure
noise on commands that don't touch the flagged ticket, which is exactly what T-092 did *not* set
out to create (it wanted the warning discoverable, not omnipresent).

Proposed fix: keep the full-tree audit as `move.Move`'s error-level post-condition check
unchanged (errors must still gate/surface — that's the "audit-clean" guarantee move.go's own
doc comment describes), but stop copying the T-092 unfinalized-merge *warning* specifically into
what a move surfaces, since `pickle board audit` run explicitly already covers discovering it.
Soft coupling: spawned from reading `internal/audit/audit.go`'s "Unfinalized-merge detection"
comment (T-092) and `internal/move/move.go`'s post-move self-check comment while confirming this
finding — no code changed there yet, just read.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-20 — created (TO DO). source: review: T-092's unfinalized-merge warning, surfaced as
  recurring noise while reviewing a separate project's (messgr) Claude Code session transcripts
  for pickle dogfooding friction; root cause confirmed by reading `internal/move/move.go` and
  `internal/audit/audit.go`.
