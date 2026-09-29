---
id: T-135
title: calibrate the brine payload's emphasis and cut its decision-defence prose
project: pickle
depends-on: []
spawned-by: []
impact: low
complexity: low
cost: M
---

# T-135 — calibrate the brine payload's emphasis and cut its decision-defence prose

## Outcome

After this ships, the brine payload keeps bold, MUST and "unconditionally" for its real rails
only, drops the paragraphs that defend a decision's history rather than explain how to apply
it, and lists itself in a session's skills with the trigger phrases plus one sentence. No rule's
meaning changes.

## Description

Cleanup, landing after the behaviour changes from the same session review (T-137..T-140), not
before them. Scope:

1. **Emphasis calibrated.** Bold, MUST, "unconditionally", "every time" stay on the real rails —
   READY gate, publish gate, merging is the human's, bookkeeping on the base branch, never
   hand-edit `BOARD.md`. The rest is stated plainly, with its reason. Over-emphasis makes an
   Opus-class model treat every sentence as a hard rail.
2. **Decision-defence prose cut** — paragraphs that argue for a decision, recount how it was
   reached or rebut alternatives (e.g. why rules §7 defines no severity, step 4b's quote-matching
   history). Whatever pickle still wants on record moves to `DESIGN.md`.
3. **`SKILL.md` frontmatter `description`** (~150 words, listed in every session whether or not
   brine is used) cut to the trigger phrases plus one sentence.
4. **`review-protocol.md` step 0's "session and tier" advice** cut to one line.

Out of scope, deliberately (challenged 2026-09-29, see NOTES.md § Session review and the 1.x
release plan (2026-09-29)):

- **Cross-file de-duplication.** T-129 loads procedures one at a time; "state it once, point
  there" would make an agent open a second file to get a rule. Repetition across procedure files
  is partly the price of that split. Repetition *within* one file may still be cut.
- **Moving the *why* out.** A rule's reason is what lets a model apply it to a case the rule did
  not foresee, and a foreign workspace has no `DESIGN.md` — the reason would vanish, not move.
  One clause of why stays with each rule; only the defence of it goes.

No field evidence ties a failure to payload length; the one observed symptom of over-emphasis
(the gate on every pickup) is T-139's. Hence low impact. `payload_lint_test.go`, the docs xref
check and any user-manual citation of payload sections stay green.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-29 — created (TO DO). source: self-host: payload review for frontier models (2026-09-29 session review): over-emphasis and decision-defence prose in the payload
