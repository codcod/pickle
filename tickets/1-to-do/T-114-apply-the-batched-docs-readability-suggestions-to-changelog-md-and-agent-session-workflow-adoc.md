---
id: T-114
title: apply the batched docs-readability suggestions to CHANGELOG.md and agent-session-workflow.adoc
project: pickle
depends-on: []
spawned-by: [T-112]
impact: low
complexity: low
cost: S
---

# T-114 — apply the batched docs-readability suggestions to CHANGELOG.md and agent-session-workflow.adoc

## Outcome

After this ships, two of the project's densest documentation surfaces — the changelog and the
agent-session-workflow manual page — read in shorter, more scannable sentences. No documented
fact changes; only the sentence structure carrying it. A reader skimming either file spends less
effort per paragraph.

## Description

A review's optional docs-readability pass (review protocol step 4b) returned eleven suggestions
over the two prose files that review had changed. One targeted a sentence the review itself had
just written and was applied during the review; the remaining **ten target prose that predates it**
and were therefore out of that review's scope. Step 4b's own rule is that readability suggestions
"never enter the findings table" — so absent a ticket these would simply have been discarded, which
is why they are collected here.

Every suggestion below is a **sentence-structure change with no change of meaning**: almost all of
them split one long sentence into two or three, or replace a dash-joined clause with a colon. None
asserts a new fact, corrects an error, or touches a code path. That uniformity is what makes them
batchable into a single ticket rather than judged one at a time.

### The suggestions, as returned

**`CHANGELOG.md`** (6):

1. `[Unreleased]` / Added / reviewer independence — split the dense three-clause sentence
   ("...delegate the ... audits to an independently spawned reviewer, hand-verify every delegated
   finding, and record which happened ... in the checklist; classification, dispositions, and the
   ticket move stay with the reviewer") into three sentences, one per action.
2. `[Unreleased]` / Added / `pickle scaffold docs` — turn the trailing "— entirely optional and
   unrelated to the ticket flow: ..." afterthought into its own sentence.
3. `[Unreleased]` / Fixed / `cli-reference.adoc` — recast the list-heavy "X, and Y's
   `--a`/`--b`/`--c`, existed but were never mentioned; Z had no section at all" as two sentences
   led by "The manual mentioned neither ...".
4. `[0.10.0]` / Added / `pickle board decisions` — split "The same answer previously needed a
   hand-written `awk` that re-solved two parsing traps every time ... and got the child filter
   wrong ..." into a statement plus its two flaws.
5. `[0.9.0]` / Added / atomic writes — split the temp-file-and-rename sentence at "As a result,".
6. `[0.7.0]` / Fixed / installed skill audience — split the three-part "The examples are now ...,
   the skeleton's warrant ..., and the two classes are defined as ..." into parallel sentences.

**`docs/user-manual/concepts/agent-session-workflow.adoc`** (4):

7. Opening paragraph — break after "{product} deliberately ships no model or agent-tier
   configuration." so the core claim lands before the judgment/mechanics elaboration.
8. § *Session boundaries are also cost boundaries* — split the accumulated-transcript sentence at
   "Switching models mid-session doesn't erase ...".
9. § *A pattern mapped to the procedures*, *Refine a ticket* row — recast "The READY gate's
   *completeness* is now mechanically checked: a required `### ` heading missing from the plan
   refuses the move" into an explicit if/then, and "a fresh session reading just the ticket" →
   "a fresh session that reads just the ticket".
10. § *Notes*, first bullet — replace the dash before "it runs a scoped, bounded re-check" with a
    colon, and "not open-ended reasoning" → "rather than open-ended reasoning".

### Scope notes

- **Neither file is skill payload.** `CHANGELOG.md` and `docs/user-manual/` are this project's own
  documentation, not `skill/`, so nothing here is subject to the payload lint or the
  foreign-workspace constraints that govern shipped skill text. No `skill/` file is touched.
- **Suggestion 1 is a loose end from the originating review**, not a fresh observation: that
  review's own record claims both in-scope suggestions were applied inline, but only the manual
  sentence was — the changelog entry it had just written was left as returned. Corrected in that
  ticket's History; the suggestion itself is carried here.
- **Judgement is still required per suggestion.** These are a reviewer's proposals, not findings;
  the implementer should accept the ones that genuinely read better and record which were declined
  and why, rather than applying all ten mechanically. A suggestion that flattens a deliberate
  emphasis should be dropped.
- **`[0.7.0]`, `[0.9.0]` and `[0.10.0]` are released sections.** Editing the prose of a shipped
  changelog entry is safe (it restates history, it does not alter it), but the edits must not
  change any stated fact, ticket id, version or date — only sentence boundaries.

### Couplings

Soft only, no `depends-on:`. T-111 and T-113 both touch `CHANGELOG.md`'s role in the release
convention; if either lands first this ticket should re-verify the entries it edits still exist in
the form quoted above. Neither blocks it — the edits are independent of any release mechanism.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-22 — created (TO DO). source: review: the docs-readability pass (step 4b) during T-112's
  review returned 11 suggestions over the two prose files that ticket changed; 1 applied inline
  there, 10 targeted pre-existing prose and fell outside its scope. Step 4b discards suggestions
  that are not applied, so they were batched here at the user's request rather than lost. Graded
  low/low/S against the backlog: no fact or behaviour changes, the edits are mechanical sentence
  splits, and it sits with the other low/low/S polish items (T-038, T-042, T-103) rather than with
  T-067, whose dead cross-references are a correctness gap in the docs pipeline.
