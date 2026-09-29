---
id: T-137
title: review hunts for correctness bugs: a multi-angle bug hunt with reproduction before the findings table
project: pickle
depends-on: [T-135]
spawned-by: []
impact: high
complexity: medium
cost: M
---

# T-137 — review hunts for correctness bugs: a multi-angle bug hunt with reproduction before the findings table

## Outcome

After this ships, a brine review hunts for the correctness bugs that a separate code-review
pass has been finding after it, across named angles and backed by a reproduction, so a validated
ticket no longer needs a second review tool before it can be trusted.

## Description

Evidence, sessions of 2026-09-21..29 across smppai, porth, messgr, unity and pickle: after a brine
validate the user ran the host's code-review command on the PR **31 times**. It typically
returned 7–10 findings; only 2 runs found no correctness bug, and the user acted on nearly all of
them — e.g. smppai PR 42 acked `data_sm` then dropped it on three paths, PR 46 accepted a
connection after the handler disconnected it, PR 40 never answered a bind, porth PR 5 grew an
unbounded store, porth PR 7 raced on DLRs. The brine round-1 review of the ticket behind PR 42
was one agent, ~12 tool calls, one blocking finding; its scoped re-review read only the fix diff
and passed. `review-protocol.md` step 3 ("quality audit") is five bullets; most of the protocol
governs how findings are recorded, not how they are found.

Scope:

1. **Step 3 becomes a correctness hunt** over named angles: caller contracts and cross-module
   effects; error, cancellation and shutdown paths; ordering and concurrency; boundary inputs;
   resource lifetimes; breaking changes to API, wire format or config; security at trust
   boundaries; conformance to the ticket's confirmed decisions.
2. **Scaled by `complexity`** — a single pass for `low`; for `medium`/`high`, angles may fan out
   to parallel sub-agents where the host supports them.
3. **A `correctness` finding needs a reproduction** (a failing test, a script, a command
   transcript) before it enters the findings table; a candidate that cannot be reproduced is
   recorded as such, not as a confirmed bug.
4. **A host code-review tool, when present**, may be run on the branch diff and its output
   triaged like step 4b's delegated findings — each re-verified before it enters the table.
   Worded generically: no host-specific command name (the payload is read by projects on any
   agent).
5. **CI parity** (folded in, one sentence): a child's configured `lint`/`test` commands should be
   its CI entry point, and a review that finds CI checking something they do not is a finding.
   Evidence: smppai's CI failed repeatedly on formatting, shell syntax and a Windows environment
   limit after a locally green review, until the user added a `make ci` mirroring CI.

The scoped re-review (T-124) should run the same hunt over the fix diff — refinement decides how
much. Expect the `class` distribution to shift toward `correctness`; note it against T-085's
pre-registered criterion. Depends on T-135 (same text); T-138's re-review uses this hunt.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-29 — created (TO DO). source: self-host: session review 2026-09-21..29: a separate code-review pass run after brine validate found correctness bugs in 29 of 31 PRs
