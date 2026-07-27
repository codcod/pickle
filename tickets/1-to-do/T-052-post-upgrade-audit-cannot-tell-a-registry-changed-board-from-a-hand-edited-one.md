---
id: T-052
title: post-upgrade audit cannot tell a registry-changed board from a hand-edited one
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: low
cost: S
---

# T-052 — post-upgrade audit cannot tell a registry-changed board from a hand-edited one

## Description

Registering a child-project and then upgrading — the documented onboarding sequence — ends in
an error and a non-zero exit, for a workspace where nothing is wrong:

```
$ pickle project add snowball snowball --build "just build" --test "just test" --lint "just fmt-check"
registered child-project "snowball" at snowball
$ pickle upgrade
  + .agents/skills/ticket-flow/
  + AGENTS.md (marker updated)
  …
ERROR: BOARD.md is stale or hand-edited — run pickle board sync
pickle: post-upgrade audit found 1 error(s)          # exit 1
$ pickle board sync
  reformat only (ordering / WIP counts / spacing / preamble)
$ pickle doctor && pickle board audit
pickle doctor: 0 error(s), 0 warning(s)
board audit: 88 tickets, 0 error(s), 0 warning(s)
```

The board *is* stale, and legitimately so: registering a child changes the board's **generated
shape** — it gains a per-child section under every status heading and a per-child WIP line — so
a board that was in sync a second earlier no longer matches what the tickets now generate. The
post-upgrade self-check (`internal/cli/install.go:139-149`) audits with no sync step and
classifies staleness as an error.

The verdict's *advice* is right (`board sync` is exactly the next command) but its
**classification is not**: "stale **or** hand-edited" collapses a routine consequence of a
registry change into the same bucket as a rule violation. Two costs, both about trust rather
than data: a scripted or agent-driven onboarding sees a failed command in a healthy sequence,
and a human learns to expect one error mid-flow — the habit that hides a real one. Observed
during the `snowball` onboarding of the `unity` workspace (pickle 0.1.0, 2026-07-27).

What must **not** be assumed at refinement: that the fix is "make `upgrade` run
`board sync`". `upgrade` staying out of the board's business is a property worth keeping, and
silently rewriting a board that *might* be hand-edited is the very loss that made `board sync`
an explicit, separate command (see `tickets/NOTES.md` on T-044: the board is a pure generated
artifact precisely so nothing hand-written is at risk). The smaller and more defensible change
is a verdict that **separates the two causes** — registry/ticket state changed since the board
was generated → "run `pickle board sync`" as a *warning* (exit 0); rendered content diverges
from what the tickets generate in a way sync would not explain → error. Whether that
distinction is even computable from the current audit inputs is the first question refinement
has to answer, and a negative answer (with the message merely re-worded) is a legitimate
outcome.

Soft coupling: **T-051** (the same `project add` → `upgrade` sequence leaves five workspace-side
edits unmentioned) — both came from one onboarding session and both concern what that command
pair tells the operator; they ship independently.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-27 — created (TO DO). source: idea — field finding from adding a second child-project to the `unity` workspace with pickle 0.1.0
