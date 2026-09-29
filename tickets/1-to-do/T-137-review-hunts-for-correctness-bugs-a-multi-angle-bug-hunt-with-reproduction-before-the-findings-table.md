---
id: T-137
title: review hunts for correctness bugs: a multi-angle bug hunt with reproduction before the findings table
project: pickle
depends-on: []
spawned-by: []
impact: high
complexity: medium
cost: M
---

# T-137 — review hunts for correctness bugs: a multi-angle bug hunt with reproduction before the findings table

## Outcome

After this ships, a brine review hunts for the correctness bugs that a separate code-review
pass has been finding after it — through the host's own code-review tool when there is one, and
a named-angle hunt when there is not — with each confirmed bug backed by a reproduction.

## Description

Evidence, sessions of 2026-09-21..29 across smppai, porth, messgr, unity and pickle: after a brine
validate the user ran the host's code-review command on the PR **31 times**. Runs typically
returned 7–10 findings; only 2 explicitly reported no correctness bug, and the user acted on
nearly all findings. Not every finding was a correctness bug — the count of those was not
taken — but the confirmed ones include smppai PR 42 acking `data_sm` then dropping it on three
paths, PR 46 accepting a connection after the handler disconnected it, PR 40 never answering a
bind, porth PR 5 growing an unbounded store, porth PR 7 racing on DLRs. The brine round-1 review
of the ticket behind PR 42 was one agent, ~12 tool calls, one blocking finding; its scoped
re-review read only the fix diff and passed. `review-protocol.md` step 3 ("quality audit") is
five bullets; most of the protocol governs how findings are recorded, not how they are found.

Scope, cheapest first:

1. **A host code-review tool, when present, runs on the branch diff** and its output is triaged
   like step 4b's delegated findings — each re-verified before it enters the table. This is what
   the user did by hand 31 times. Worded generically: no host-specific command name (the payload
   is read by projects on any agent).
2. **Without one, step 3 becomes a correctness hunt** over named angles: caller contracts and
   cross-module effects; error, cancellation and shutdown paths; ordering and concurrency;
   boundary inputs; resource lifetimes; breaking changes to API, wire format or config; security
   at trust boundaries; conformance to the ticket's confirmed decisions. Scaled by
   `complexity`: one pass for `low`; for `medium`/`high` the angles may fan out to parallel
   sub-agents, run by the top-level session (a delegated reviewer may be unable to spawn its own —
   refinement confirms per host). A token budget is named.
3. **A `correctness` finding needs a reproduction** (a failing test, a script, a command
   transcript) before it is recorded as confirmed.

The scoped re-review (T-124) runs the same hunt over the fix diff — refinement decides how much.

**Acceptance is a replay**, not a reading: run the new review on a pre-fix commit with known bugs
(smppai PR 42's first review round, porth PR 7) and require it to find them. Without that,
"hunts bugs" cannot be checked.

**T-085's pre-registered class count** is disturbed: a hunt shifts findings toward
`correctness`. Reviews after this ships are counted separately from those before it.

CI parity (a child's `lint`/`test` should be its CI entry point) was considered for this ticket
and not included — one project, already fixed there by its own review addendum.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-29 — created (TO DO). source: self-host: session review 2026-09-21..29: a separate code-review pass was run after brine validate on 31 PRs, typically returning 7–10 findings, only 2 with no correctness bug
