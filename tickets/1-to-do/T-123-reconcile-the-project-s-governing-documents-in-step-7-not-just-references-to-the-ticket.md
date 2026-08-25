---
id: T-123
title: Reconcile the project's governing documents in step 7, not just references to the ticket
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: medium
cost: M
---

# T-123 — Reconcile the project's governing documents in step 7, not just references to the ticket

## Outcome

A review that changes behaviour a project's own governing documents describe updates those
documents, or records why not. Today the protocol can pass a review that leaves the project's
design asserting something the code no longer does.

## Description

Step 7, "Update other references", covers exactly two things:

> - **`BOARD.md` needs no hand edit** — it is generated …
> - Any ticket or doc that referenced this ticket **by id**.

Both are references **to the ticket**. Nothing in the protocol asks whether the documents that
*govern* the work — a design of record, a conventions file, a locked-decisions guide, a decisions
log — still describe what the code now does. Step 4a covers the **shipped** docs tree (the
product's user-facing documentation); the governing documents are usually not in it, and in an
umbrella layout they are not even in the same repository as the code.

The gap is not theoretical. Three independent instances from one downstream workspace:

- A design of record asserted that a CLI returned "the API's bytes" for a field. That ticket's
  **own review had retracted "byte-identical" twice** (two separate findings). The review
  recorded the retraction in the ticket and left the design asserting the original claim, where
  it stayed for weeks and was cited as fact.
- A design listed a ticket as "outstanding follow-ups" after that ticket was implemented,
  reviewed and merged.
- Four governing documents drifted from the code for a month **at an unchanged child HEAD** — so
  no external change caused it. Nobody reconciled, because nothing asked anyone to.

The common shape: a review *does* notice the truth, records it in the ticket, and the ticket is
archived. The governing document — the thing the next ticket is cut from — keeps the falsehood.
That is worse than an undocumented change, because it is confidently wrong and it propagates:
later tickets are written from it.

Proposed addition to step 7, wording for refinement:

> **If the ticket changed behaviour that the project's governing documents assert — its design,
> conventions, locked decisions or decisions log — update them in the same review, or record
> explicitly why not.** A review that leaves a governing document asserting something the code
> no longer does has not finished.

Open questions for refinement:

- **Does pickle know what a project's governing documents are?** It does not today. Options: a
  config key (`governing_docs = […]`, per project and/or overarching), inference from the
  addendum's own step 1 "load context" list, or leaving it to the project's `AGENTS.md` to name
  them and having the protocol simply say "the ones step 1 told you to read". The last is
  cheapest and needs no schema change.
- **Blocking or not?** Missing docs coverage in 4a is blocking. This is arguably the same defect
  one layer up, but governing documents are often in another repo with its own commit policy,
  which complicates a blocking verdict.
- Does it get a checklist line? Without one it is invisible, which is the same auditability
  argument step 0 makes for reviewer independence.

Soft coupling: T-122 is the other rule raised from the same downstream workspace at the same
time; they are independent and can land in either order.

Provenance: raised from the `unity` workspace, which currently carries this rule in its own
overarching addendum and would retire it if this ships.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-25 — created (TO DO). source: pickle ticket new
