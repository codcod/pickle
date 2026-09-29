---
id: T-140
title: board audit finds the merge of a DONE ticket in git history and prints the line to record
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: medium
cost: M
---

# T-140 — board audit finds the merge of a DONE ticket in git history and prints the line to record

## Outcome

After this ships, `pickle board audit` looks for the merge of a DONE ticket that has no merge
line in its child's base-branch history and prints the exact `merged to <base>` History line to
add, so recording a merge no longer depends on the human reporting it.

## Description

Evidence, 2026-09-21..29: about 25 "pr N merged" / "record it" prompts, and 17 "DONE but has no
MERGED" audit warnings. T-092 detects the missing line and T-133 scoped its noise; neither finds
the merge.

Detection is local git only: commits reachable from `origin/<base>` (local `<base>` when the
child has no remote) in the child's repository whose subject carries the ticket id in the
Conventional-Commit trailing form `(<ID>)`. Squash subjects and kept-history commits both carry
it; a GitHub squash suffix `(#95)` or a "Merge pull request #95" subject supplies the MR ref. The
audit prints the suggested History line with MR ref and short SHA, and stays read-only and
offline — it never fetches, so a stale remote ref simply means no suggestion and today's warning.

No forge API: NOTES.md § Field-finding triage (2026-09-20) rejected teaching WIP about PR state
because it needed pickle's first GitHub/API dependency; this reads only git. Reuse `changelog
check`'s id scan (T-093, T-097) rather than a second parser. Whether a writer follows
(`audit --fix` or a `ticket` subcommand) is for refinement; the default is the printed
suggestion. Under `umbrella` the child's repository comes from `pickle.toml`.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-29 — created (TO DO). source: self-host: session review 2026-09-21..29: merges were recorded only when the human reported them, ~25 times in a week
