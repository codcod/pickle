---
id: T-135
title: state each brine payload rule once: cut repetition, move rationale out, calibrate emphasis
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: medium
cost: L
---

# T-135 — state each brine payload rule once: cut repetition, move rationale out, calibrate emphasis

## Outcome

After this ships, every brine session loads noticeably less skill text: each rule is
stated in one place, the justification that does not change how a rule is applied lives in
`DESIGN.md` instead of the payload, and the hard rails still read as hard while everything else
reads as plain guidance. No rule's meaning changes.

## Description

The payload is about 18k words. A validate session loads `SKILL.md` (~3k), `review-protocol.md`
(~5k) and sections of `tickets-README.md` (~6.4k). Much of that is restatement: the umbrella vs
in-tree layout explanation recurs in about seven places, "Project configuration wins" twice, and
several paragraphs defend a decision rather than state it (why rules §7 defines no severity,
step 4b's quote-matching rationale). Frontier models — every phase has run on Opus 5.5 since
2026-09-23 — follow a rule stated once; repeating it costs tokens on every invocation and blurs
which copy governs when the copies drift. T-129 cut what each procedure *loads*; this cuts what
each file *says*.

Scope:

1. **Each rule stated once**; every other place that needs it points to it by section.
2. **Rationale moves out** of the payload into `DESIGN.md` when it does not change how the rule
   is applied; at most one clause of *why* stays inline.
3. **Emphasis calibrated.** Bold, MUST, "unconditionally", "every time" stay on the real rails —
   READY gate, publish gate, merging is the human's, bookkeeping on the base branch, never
   hand-edit `BOARD.md`. The rest is stated plainly, with its reason. Over-emphasis makes an
   Opus-class model treat every sentence as a hard rail.
4. **`SKILL.md` frontmatter `description`** (~150 words, listed in every session whether or not
   brine is used) cut to the trigger phrases plus one sentence.
5. **`review-protocol.md` step 0's "session and tier" advice** cut to one line.

Behaviour-neutral by design: no rule is added, removed or changed (item 5 aside). The behaviour
changes from the same session review — T-137, T-138, T-139 — land afterwards, on the trimmed
text, so this diff can be reviewed as a pure restructure. The review's evidence that nothing was
dropped is a before/after inventory of normative statements. `payload_lint_test.go`, the docs
xref check and any user-manual citation of payload sections must stay green. T-136 edits the
same files (the base-branch read recipe) — soft coupling, sequence after this.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-29 — created (TO DO). source: self-host: payload review for frontier models (2026-09-29 session review, see NOTES.md § Session review and the 1.x release plan (2026-09-29)): the payload restates rules and defends decisions at length
