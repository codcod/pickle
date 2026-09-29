---
id: T-136
title: pickle ticket show: print a ticket as recorded on the base branch, from any checkout
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: low
cost: S
---

# T-136 — pickle ticket show: print a ticket as recorded on the base branch, from any checkout

## Outcome

After this ships, an agent on any branch reads a ticket as recorded on the base branch with
`pickle ticket show T-NNN`, instead of typing the `git ls-tree` / `git show` recipe the payload
currently spells out, and the payload points status questions at `pickle board state --json`.

## Description

Under `in-tree`, a feature branch's copy of `tickets/` is stale (T-128, T-130, T-131), so the
payload tells implement, validate and rework to resolve the ticket from the base branch with a
hand-written `git ls-tree <base> -- tickets/ | grep …` then `git show <base>:<path>`. That recipe
appears three times, and agents still improvise around it: last week's sessions show globbed
paths that did not match (`tickets/4-in-review/RICK-201-*.md`) and `ls` of status directories on
a stale branch.

Shape: `pickle ticket show <ID>` resolves the id across status directories and prints the
ticket's status and path, then the file. Under `in-tree` it reads from the base branch
regardless of the checked-out branch; under `umbrella` it reads the working tree (the board is
not on any child's branch). Unknown id → non-zero exit. Additive CLI surface, so a minor release
under the 1.0 stability promise.

Payload: the three recipes become one `pickle ticket show` line; "what is the status of X"
guidance points at `pickle board state --json` rather than `ls` + `BOARD.md`. Soft coupling:
T-135 trims the same files — sequence after it.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-29 — created (TO DO). source: self-host: session review 2026-09-21..29: the base-branch read recipe is repeated three times in the payload and agents improvised around it with failing globs
