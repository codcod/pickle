---
id: T-122
title: Verify the docs-readability reviewer's quoted text before presenting a suggestion
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: medium
cost: M
---

# T-122 — Verify the docs-readability reviewer's quoted text before presenting a suggestion

## Outcome

The review protocol's step 4b tells a reviewer to check what the docs-readability reviewer sends
back before acting on it. Today it does not, and a reviewer that invents passages is indis-
tinguishable from one that read the file.

## Description

Step 4b defines the readability pass, names the reviewer, and gives the invocation syntax for
three hosts. It says **nothing about verifying the output.** Grep the protocol for
`fabricat|verbatim|quote|hallucin` and the only hits are about re-running an acceptance test and
copying a cost estimate — nothing about the reviewer's own claims.

That matters because this reviewer is an LLM being asked to quote source text, which is a
failure mode with a known shape. In one recorded run in a downstream project, **all 8
suggestions quoted text that existed nowhere in the tree.** A reviewer following step 4b as
written would have presented all 8 for approval, and applying them means editing prose to match
a passage that does not exist.

Every project using pickle and running 4b is exposed to this; it is a property of the tool the
protocol ships instructions for, not of any one project. It has been worked around in at least
one downstream workspace by a hand-written addendum, which protects that project only.

Proposed addition to step 4b, wording for refinement to settle:

> **Verify every suggestion's quoted "current text" against the file before presenting it.**
> Grep each quoted string; drop any suggestion whose quote does not match verbatim. If most of a
> run fails this way, discard the run and re-invoke rather than salvaging it. Record
> discarded-as-fabricated counts in the ticket's `## Review`.

A second rule from the same source is worth considering in the same ticket, though it is a
quality bar rather than a correctness one: **reject suggestions that change content rather than
wording** — deleting emphasis that carries meaning, dropping a documented path, or swapping a
precise term are content edits, whatever the reviewer labels them.

Open questions for refinement:

- Does the checklist line for 4b change, or does the count live only in the `## Review` prose?
  4b is optional and never blocks, so a fabricated-quote count is not a finding — but silence
  about it has the same auditability problem step 0 already names for reviewer independence.
- Does this belong in `review-protocol.md` alone, or also in whatever guidance ships alongside
  the `docs-readability` subagent/tool definitions?

Provenance: raised from the `unity` workspace, which currently carries this rule in its own
overarching addendum and would retire it if this ships.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-25 — created (TO DO). source: pickle ticket new
