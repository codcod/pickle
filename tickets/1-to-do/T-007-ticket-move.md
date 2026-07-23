---
id: T-007
title: ticket move (state machine + per-child WIP + cross-child merge gate)
project: pickle
depends-on: []
impact: high
complexity: medium-high
cost: M-L
---

# T-007 — ticket move (state machine + per-child WIP + cross-child merge gate)

## Description

Implement `pickle ticket move T-NNN <status> --reason "<why>"` as one atomic operation:
relocate the ticket file between status directories, append the dated `## History` transition
line, and update the board row (correct section **and** child sub-group) — the three edits the
board rule requires, done together.

Enforce:

- the **state machine** — forward transitions plus the allowed backward/abort moves
  (`in-development → ready`, `ready → to-do`, `→ dropped`), with backward/abort **sign-off**
  rules;
- **per-child WIP limits** (counts only the ticket's own `project`);
- the **dependency gate** on pickup (`→ in-development`): every `depends-on:` target is in
  `6-done/` **and merged to the base of its own child-project's repo** (cross-child aware).

Reject illegal moves with a clear message. Needs T-001 and T-002. Phase P3.

## Implementation Plan

<!-- empty until refined -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-23 — created (TO DO). source: step-3 board bootstrap (phased plan P3)
