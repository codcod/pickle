---
id: T-029
title: regression-test the non-gating guarantee at the move.go pickup gate
project: pickle
depends-on: []
spawned-by: [T-024]
impact: medium
complexity: low
cost: S
---

# T-029 — regression-test the non-gating guarantee at the move.go pickup gate

## Description

T-024's defining property is that `spawned-by:` **never gates a pickup**. That guarantee is
enforced in **two** places — the audit's in-development loop (`internal/audit/audit.go:140-156`)
and `move.Move`'s own, stricter pickup gate (`internal/move/move.go:98-113`) — but only the
first is guarded by a test.

The T-024 review proved the gap by mutation: adding `SpawnedBy` to move.go's gate loop
(`range append(append([]string{}, t.DependsOn...), t.SpawnedBy...)`) left the **entire suite
green, 9/9 packages**. The same mutation applied to the audit loop correctly fails
`TestAudit/in-dev_spawned-by_parent_not_done`. So the gate a human actually hits when running
`pickle ticket move T-NNN in-development` — the one that produces the user-visible error — has
no regression guard for the decision it must honour.

### Scope

One case in `internal/move/move_test.go`: create a ticket whose `spawned-by` names a parent
still in `1-to-do/` (the `depends-on` string-rewrite trick at `move_test.go:39` is the pattern
to copy), move it to `in-development`, and assert the move **succeeds**. Mirror the naming of
the audit's `in-dev spawned-by parent not done` case so the pair is greppable. Verify the test
is load-bearing by re-running the mutation above and confirming it now fails.

Two smaller test-hardening items from the same review, worth folding in while in this code:

- `internal/cli/cli_test.go:111-113` asserts only that `board audit` exits non-zero for a
  dangling `spawned-by` id, so it would also pass on an unrelated audit error. Assert the
  message, or record the coarseness deliberately.
- `internal/cli/cli_test.go:55-65`'s `newProject` helper makes CWD safety **opt-in**: any future
  test in package `cli` that calls a config-resolving command without it will walk up and write
  into the live board (the original T-024 blocking finding). A package-level `TestMain` that
  chdirs to a temp dir would make safety the default.

### Couplings

`spawned-by: [T-024]` — born from T-024's review (non-blocking findings N1, N5, N6). No hard
dependency: T-024's code is what makes the test meaningful, but the test can be written
whenever.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-25 — created (TO DO). source: pickle ticket new
