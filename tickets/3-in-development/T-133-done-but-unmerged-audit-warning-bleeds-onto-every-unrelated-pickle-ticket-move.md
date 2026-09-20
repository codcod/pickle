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

### 0. Feature branch (mandatory)

```
cd .
git checkout main
git checkout -b feat/T-133-unfinalized-merge-warning-noise
```

### Prerequisite gate (hard)

None.

### Confirmed design decisions (do not deviate without asking)

1. **Blanket removal, not self/other filtering.** Every DONE-but-unmerged warning is dropped
   from what a `ticket move` call surfaces, including one about the very ticket just moved to
   DONE (which, immediately after that move, always lacks a merge line yet — the human hasn't
   merged it) — matching this ticket's own Description ("stop copying the T-092 unfinalized-merge
   warning specifically"), not a narrower "only suppress warnings about tickets other than the
   one being moved" rule. `pickle board audit`, run explicitly, still reports it unconditionally
   — including right after a move, if a mover wants confirmation.
2. **Errors stay untouched.** `move.Move`'s post-condition self-check against `a.Errors`
   (`internal/move/move.go` ~line 205-207) is unchanged — only `a.Warnings` is filtered before
   being copied into `res.Warnings`. The audit-clean error guarantee `move.go`'s own doc comment
   describes still gates every move.
3. **New `audit.Result.UnfinalizedMerges []string` field**, populated in parallel with the
   existing `Warnings` entry at the same call site (`internal/audit/audit.go` ~line 264), rather
   than string-matching the warning text inside `move.go` or adding a `Kind`/category enum to
   every warning. This is additive: every other consumer of `audit.Result` (`board audit`'s CLI
   printer, `board state --json` via `state.Build`, the serve dashboard's `HealthView`) keeps
   reading `.Warnings` exactly as before and needs no change — this warning stays discoverable
   there without a filter step of its own.

### Tasks

#### Task 1 — `internal/audit/audit.go`

- Add `UnfinalizedMerges []string` to the `Result` struct (~line 22-26):

  ```go
  // Result is the outcome of an audit.
  type Result struct {
  	NumTickets int
  	Errors     []string
  	Warnings   []string
  	// UnfinalizedMerges holds exactly the DONE-but-unmerged warnings from the
  	// unconditional whole-tree scan below (T-092) — a parallel, filterable
  	// copy of the subset of Warnings that move.go excludes from what a
  	// `ticket move` call surfaces (T-133). Every other Warnings consumer
  	// (board audit, board state --json, the serve dashboard) ignores this
  	// field and keeps seeing the warning in Warnings as before.
  	UnfinalizedMerges []string
  }
  ```

- At the unfinalized-merge detection loop (~line 246-264), replace the lone `r.warnf(...)` call
  with:

  ```go
  msg := fmt.Sprintf("%s: DONE but has no 'MERGED' History line — not merged yet, or the merge line was forgotten (rules §4: append it and run pickle board sync)", ref)
  r.Warnings = append(r.Warnings, msg)
  r.UnfinalizedMerges = append(r.UnfinalizedMerges, msg)
  ```

  (identical message text to today — no golden-output test changes needed for `board audit`
  itself).

- Sort the new field alongside the existing `sort.Strings(r.Warnings)` (~line 268):
  `sort.Strings(r.UnfinalizedMerges)`.

#### Task 2 — `internal/move/move.go`

- Change the post-move self-check (~line 202-206) so only the filtered warnings are copied:

  ```go
  a := audit.Audit(root, cfg)
  res.Warnings = withoutUnfinalizedMerge(a.Warnings, a.UnfinalizedMerges)
  ```

- Add the helper near `checkWIP`/`appendHistory`:

  ```go
  // withoutUnfinalizedMerge drops audit's own-ticket DONE-but-unmerged warnings
  // (T-092) from what a single `ticket move` call surfaces (T-133): that scan
  // covers the whole tree unconditionally, so left in, a stale merge on any one
  // ticket reprints on every unrelated move for as long as it stays unmerged.
  // `pickle board audit`, run explicitly, still reports it in full.
  func withoutUnfinalizedMerge(warnings, unfinalized []string) []string {
  	if len(unfinalized) == 0 {
  		return warnings
  	}
  	drop := make(map[string]bool, len(unfinalized))
  	for _, w := range unfinalized {
  		drop[w] = true
  	}
  	kept := make([]string, 0, len(warnings))
  	for _, w := range warnings {
  		if !drop[w] {
  			kept = append(kept, w)
  		}
  	}
  	return kept
  }
  ```

### Acceptance test

```
just build
just test
just lint
```

Add to `internal/move/move_test.go`:

- `TestMoveDoesNotSurfaceUnfinalizedMergeWarningForOtherTicket` — fixture: one ticket A in
  `6-done/` with no `merged to <base>` History line, one unrelated ticket B in `1-to-do/`. Move B
  to `2-ready/`; assert `res.Warnings` contains no "DONE but has no 'MERGED' History line" entry
  for A. Separately assert `audit.Audit(root, cfg).Warnings` **does** contain it — the regression
  guard that `pickle board audit` keeps surfacing it unconditionally (Outcome's explicit clause).
- `TestMoveDoesNotSurfaceUnfinalizedMergeWarningForSelf` — move a ticket straight into DONE (no
  merge line yet, the normal state immediately post-move); assert its own self-referential
  warning is also absent from that move's `res.Warnings` (decision 1), while `board audit` still
  reports it.

Manual smoke (mirrors the dogfooding repro in the Description):

```
D=$(mktemp -d) && cp pickle "$D/pickle-test" && cd "$D" && ./pickle-test install --in-tree
# ... file two tickets: one moved to DONE with no merge line, one unrelated TO DO ticket ...
./pickle-test ticket move <unrelated-id> ready
# before fix: prints "warning: .../<done-id>...: DONE but has no 'MERGED'..."
# after fix: that line is gone
./pickle-test board audit
# still reports the DONE-but-unmerged warning
```

### Docs update (mandatory when user-facing)

No README/docs page describes `ticket move`'s warning surface in enough detail to need updating
— `pickle board audit`'s own (unchanged) output continues to be where this warning is documented
as discoverable. No doc changes needed.

### Finish (mandatory)

1. Acceptance test green; `just build`/`just test`/`just lint` clean.
2. No docs to update (see above).
3. Write a summary (files touched: `internal/audit/audit.go`, `internal/move/move.go`,
   `internal/move/move_test.go`; decision: blanket removal including self-warnings, `board audit`
   unaffected).
4. Suggested commit message:
   `fix(cli): stop echoing unrelated DONE-but-unmerged warnings on ticket move (T-133)`.
5. Root-path child (`path = "."`) — tidy WIP commits into atomic ones before presenting.
6. Commit locally on the ticket branch; do not push or open an MR without user approval.
   `pickle ticket move T-133 in-review --reason "acceptance green"` and hand back.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-20 — created (TO DO). source: review: T-092's unfinalized-merge warning, surfaced as
  recurring noise while reviewing a separate project's (messgr) Claude Code session transcripts
  for pickle dogfooding friction; root cause confirmed by reading `internal/move/move.go` and
  `internal/audit/audit.go`.
- 2026-09-20 — TO DO → READY: plan complete
- 2026-09-20 — READY → IN DEVELOPMENT: picked up
