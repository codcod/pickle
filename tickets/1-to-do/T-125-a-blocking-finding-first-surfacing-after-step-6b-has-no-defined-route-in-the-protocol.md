---
id: T-125
title: a blocking finding first surfacing after step 6b has no defined route in the protocol
project: pickle
depends-on: []
spawned-by: [T-123]
impact: medium
complexity: low
cost: S
---

# T-125 — a blocking finding first surfacing after step 6b has no defined route in the protocol

## Outcome

After this ships, a reviewer who discovers a blocking defect *after* the ticket has already been
moved to `6-done/` has a defined, legal next step instead of a dead end. The protocol either names
the route or states plainly that the finding becomes a new ticket, so the reviewer stops having to
invent an answer at the point where the state machine offers none.

## Description

`resources/review-protocol.md` runs steps 0-9 in order. Step 6 moves the ticket: 6a sends it to
`5-rework/` when a blocking finding exists, 6b to `6-done/` when none does. Steps 7, 8 and 9 then
run *after* that move.

`6-done/` is a terminal status. The flow's transition table declares no outbound transition from
it, so once 6b has run there is no legal move back to `5-rework/`. But nothing stops a reviewer
finding a blocking defect at step 7 (reconciling governing documents means reading them closely),
at step 8 (the impact sweep re-reads dependent tickets), or at step 9 (writing the summary is when
the whole change is seen at once). The protocol gives that reviewer no route.

Today each late step quietly assumes the problem away rather than addressing it:

- **step 8** frames everything it turns up as taking a disposition - and dispositions are defined
  only for *non-blocking* findings, so a blocking one has no slot;
- **step 7** states a reconciliation duty and deliberately says nothing about routing, which is
  correct for its own scope but leaves the general question open;
- **step 9** presents results for approval and never contemplates a finding at all.

The gap is generic: it belongs to the protocol's step ordering, not to any one finding type.

**Why this is worth fixing rather than tolerating.** The failure mode is silent. A reviewer who
finds something blocking at step 7 has three bad options - ignore it, invent a move the tooling
will refuse, or write a fix record asserting a route that does not exist. All three have happened
in practice, which is what surfaced this.

**Likely shape of the answer** (for refinement to settle, not decided here): a late blocking
finding almost certainly becomes a **new ticket** rather than a rework of the concluded one, since
the reviewed ticket has already shipped its verdict and the flow's own principle is that severity
governs whether *this* ticket ships. If so, the fix is one clause in the protocol saying exactly
that, plus a matching line in the rules' severity section so the two do not drift. Whether the
concluded ticket also earns a `## History` note pointing at the follow-up is the open question.

**Explicitly out of scope.** Adding a `6-done -> 5-rework` transition to the flow. Terminal
statuses are terminal by design; re-opening an archived ticket would undermine the archive's
meaning and the board's `merged` accounting. The point of this ticket is to define the route, not
to widen the state machine.

**Couplings.** Sibling of T-124 (which widens what a *scoped re-review* must check). Both come
from the same review series and both touch `resources/review-protocol.md`, but they are
independent: T-124 is about re-review scope, this one about step ordering. Either can land first;
whichever lands second rebases.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-26 — created (TO DO). source: pickle ticket new
