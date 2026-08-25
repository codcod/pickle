---
id: T-124
title: scoped re-review only checks listed findings, missing defects the fix's own replacement text introduces
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: low
cost: S
---

# T-124 — scoped re-review only checks listed findings, missing defects the fix's own replacement text introduces

## Outcome

After this ships, a scoped re-review checks the fix's own diff for new defects — not only
whether the listed findings are closed. A rework fix that introduces a fresh false statement,
contradiction, or gap in the text/code it just wrote is caught in the same round instead of
surfacing as a new blocking finding one or more rounds later.

## Description

`review-protocol.md:119-120` (mirrored at `tickets-README.md:400-401`) scopes a scoped re-review
to: *"only the findings listed there need re-verification; do not re-audit the whole feature from
scratch."* Taken literally, that mandate covers whether each **listed** finding is now closed —
it gives the reviewer no instruction to look at the **new** text or code the fix pass just wrote
to close it. In practice, that is exactly where the next round's blocking findings keep turning
up:

- **T-109** — round 2 found R1-R3 in surfaces round 1's fix hadn't reached; round 3 found S1
  explicitly in "the *replacement* prose from round 2 … including two of review 2's own inline
  fixes"; round 4 found T1 in "the replacement for S1". Four rounds, each finding sitting in the
  round before's own fix.
- **T-067** — rework round 2 found "2 new blocking … N1 invariant guards the pattern not the
  scanner, N2 failure message prescribes a fix that breaks the build" on the just-landed fix.
- **T-098** — a re-review found "F2 blocking (N3's fix left the provenance glosses contradicting
  their own tie-break)" — the fix *for* N3 created F2.
- **T-018** — a re-review found "new false statement + missed third copy" plus a "rework record
  overclaim" — new false statements introduced by the rework pass itself.
- **T-013** — "new blocking B4 (advisory payload diff can abort the whole upgrade)" surfaced only
  after B1-B3 were fixed.
- **T-122** — round 2 found "R1/R2 blocking: F2 residual on blockquote/list prefixes, F4 exit not
  admitted by the skip" — the first fix didn't fully close its own claim.

Six of the eight tickets that were ever reworked more than once show this same shape: the
second-or-later round's blocking finding lives in text the *previous round's fix* wrote, not in
behaviour the reviewer was told to re-check. The rework procedure itself only asks the fixer to
"fix only the listed findings" (`resources/tickets-README.md` rework procedure) — it says nothing
about auditing the new lines that fix produces, and the scoped re-review that follows has no
mandate to either.

**Proposed change.** Widen the scoped re-review's stated mandate in both
`resources/review-protocol.md:119-120` and `resources/tickets-README.md:400-401` (they restate
the same rule and must move together) from *"only the findings listed there need
re-verification"* to something like: *"only the findings listed there need re-verification, plus
the diff that fixed them" — read the fix's diff against the pre-rework commit and check the new
text/code it introduces for the same defect classes, blocking or not, before closing.* This stays
bounded (the diff, not "re-audit the whole feature") while closing the blind spot that let a
fix's own replacement text ship unread.

**Not in scope.** `NOTES.md § "T-109 partial merge"` records a distinct, still-unfiled idea —
verifying a branch's pushed tip matches the ticket's own Review section before reporting a push
as done. That is a publish-verification gap, not a review-scope gap; this ticket does not
subsume it. T-112 (review-protocol bias-mitigation for a same-session implementer reviewing their
own ticket) is also a different axis — reviewer independence, not re-review scope — and does not
overlap this one.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-25 — created (TO DO). source: chat: measured the rework-cause distribution across
  `tickets/6-done/`'s and `tickets/5-rework/`'s `## Review` findings tables at the user's
  request (59 blocking, 227 non-blocking rows); the scoped re-review rule's "only the listed
  findings" wording is the recurring shape behind repeat-round rework (T-109 x4, T-067/T-018 x3,
  T-098/T-013/T-122 repeat rounds)
