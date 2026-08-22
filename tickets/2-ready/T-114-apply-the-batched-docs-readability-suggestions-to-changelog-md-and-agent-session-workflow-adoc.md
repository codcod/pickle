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

### 0. Feature branch (mandatory)

```
cd .
git checkout main
git checkout -b feat/T-114-docs-readability-batch
```

Root-path child (`project: pickle`), so tidy WIP commits into atomic ones before presenting and
default to keeping that history rather than squashing.

### Prerequisite gate (hard)

None. No `depends-on:`. All ten target anchors were re-verified present at refinement (see
*Confirmed design decisions* 1). The two soft couplings (T-111, T-113) are unstarted; if either
merges first, re-verify the `CHANGELOG.md` anchors before editing, since both touch that file's
role in the release convention.

### Confirmed design decisions (do not deviate without asking)

1. **All ten anchors were verified against the tree at refinement**, with line numbers recorded in
   *Tasks* below. Four appeared missing on a first line-based grep purely because both files
   hard-wrap; a whitespace-normalised search found all four. **Use a wrap-tolerant search when
   locating them** — a naive `grep` for a quoted phrase will produce false negatives and may lead
   an implementer to conclude a target no longer exists.
2. **Released `CHANGELOG.md` sections are in scope, but isolated into their own task** (Task 2:
   `[0.10.0]`, `[0.9.0]`, `[0.7.0]`). Rewording a shipped entry restates history rather than
   altering it, and the file is read top-to-bottom by anyone catching up. The task boundary exists
   so the decision stays reversible: if the preference is to leave released sections untouched,
   **drop Task 2 entirely** — no other task depends on it and the ticket needs no re-refinement.
3. **No fact may change — only sentence boundaries.** No ticket id, version, date, flag name,
   command, link or claim may be added, removed or altered. This is mechanically enforced by the
   invariance check in *Acceptance test*, not left to eyeballing.
4. **The implementer may decline individual suggestions.** These are a reviewer's proposals, not
   findings. A split that flattens deliberate emphasis, or that separates a claim from the
   qualification it depends on, should be rejected. **Record every declined suggestion and why** in
   the finish summary — a silent partial application is indistinguishable from an oversight.
   Applying all ten is not the success condition; applying the ones that read better is.
5. **No `CHANGELOG.md` entry is added for this ticket.** A changelog records changes to the
   project; rewording the changelog's own prose is not one, ships no behaviour, and an entry for it
   would itself become a candidate for rewording later. This is a deliberate exception to the usual
   docs step, stated so review does not read its absence as an omission.
6. **Suggestion 9 edits text inside an AsciiDoc table cell.** The *Refine a ticket* row's `Why`
   column is a cell in a `[cols="2,2,2,4"]` block; its content may span lines but a stray leading
   `|` would start a new cell and silently corrupt the table. The table is written one cell per
   line and has **four columns and eight body rows** (33 cell-start lines: one header line plus
   8×4 body cells) — counted at refinement, not estimated. Verify both numbers still hold after
   editing: `just docs-check` catches a malformed block but not a cell that merely absorbed the
   wrong text.

### Tasks

#### Task 1 — `CHANGELOG.md`, `[Unreleased]` section (3 edits)
- **:13** `Added` / reviewer independence (the T-112 entry) — split the three-clause sentence
  beginning "When the agent about to review a ticket authored its branch…" into three sentences,
  one per action: delegate; hand-verify and record; what stays with the reviewer.
- **:22** `Added` / `pickle scaffold docs` — promote the trailing "— entirely optional and
  unrelated to the ticket flow: …" clause into its own sentence ("This is entirely optional…").
- **:38** `Fixed` / `cli-reference.adoc` — recast the list-heavy sentence as two, led by "The
  manual mentioned neither …", with the `pickle flow show|list` omission as a second sentence.

#### Task 2 — `CHANGELOG.md`, released sections (3 edits) — *droppable per decision 2*
- **:60** `[0.10.0]` / `pickle board decisions` — split "The same answer previously needed a
  hand-written `awk` that re-solved two parsing traps… and got the child filter wrong…" into the
  statement plus its two flaws.
- **:121** `[0.9.0]` / atomic writes — split the temp-file-and-rename sentence, opening the second
  with "As a result,".
- **:223** `[0.7.0]` / installed skill audience — split the three-part "The examples are now…, the
  skeleton's warrant…, and the two classes are defined as…" into parallel sentences.

#### Task 3 — `docs/user-manual/concepts/agent-session-workflow.adoc` (4 edits)
- **:4** opening paragraph — break after "…no model or agent-tier configuration." so the core
  claim lands before the judgment/mechanics elaboration.
- **:18** § *Session boundaries are also cost boundaries* — split at "Switching models mid-session
  doesn't erase…"; render the first half as "…, which is reprocessed by whichever model handles
  the next turn."
- **:56** § *A pattern mapped to the procedures*, *Refine a ticket* row — recast the READY-gate
  sentence as an explicit if/then, and "a fresh session reading just the ticket" → "a fresh session
  that reads just the ticket". **Table cell — observe decision 6.**
- **:101** § *Notes*, first bullet — replace the dash before "it runs a scoped, bounded re-check"
  with a colon; "not open-ended reasoning" → "rather than open-ended reasoning".

### Acceptance test

```
just build && just test && just lint && just docs-check
./pickle changelog check
```

All clean. `./pickle changelog check` must still report **"no candidates — every shipped ticket is
mentioned"**: it parses `CHANGELOG.md` for ticket ids, so it fails loudly if an edit drops one.

Then the **fact-invariance check** — the mechanical enforcement of decision 3. For each edited
file, these four sets must be byte-identical before and after:

```
inv() {
  git show "$1:$2" > /tmp/before.txt; cp "$2" /tmp/after.txt
  for side in before after; do
    {
      grep -oE '^#+ .*$'            "/tmp/$side.txt"            # headings, versions, dates
      grep -oE 'T-[0-9]+'           "/tmp/$side.txt" | sort | uniq -c
      grep -oE '`[^`]+`'            "/tmp/$side.txt" | sort | uniq -c
      grep -oE 'https?://[^ )]+'    "/tmp/$side.txt" | sort | uniq -c
      grep -oE '<<[^>]+>>'          "/tmp/$side.txt" | sort | uniq -c
    } > "/tmp/inv-$side.txt"
  done
  diff /tmp/inv-before.txt /tmp/inv-after.txt && echo "INVARIANT OK: $2"
}
inv main CHANGELOG.md
inv main docs/user-manual/concepts/agent-session-workflow.adoc
```

Both must print `INVARIANT OK`. A difference means a heading, ticket id, inline code span, URL or
AsciiDoc xref changed — which decision 3 forbids, and which no sentence split requires.

Finally, two by-hand confirmations:

1. `git diff --word-diff` over both files shows **only** added/removed sentence-boundary tokens
   (full stops, capitalisation, connectives like "As a result", "This is") — no substantive word
   swapped for a different claim.
2. The `[cols="2,2,2,4"]` table in the `.adoc` still has four columns and eight body rows
   (decision 6). Mechanically:
   ```
   awk '/^\|===/{f=!f;next} f' docs/user-manual/concepts/agent-session-workflow.adoc \
     | grep -c '^| '        # must print 33 (1 header line + 8 rows × 4 cells)
   ```

### Docs update (mandatory when user-facing)

This ticket **is** the docs change; no further documentation is required, and per decision 5 no
`CHANGELOG.md` entry is added for it.

One caution for the review that follows: **step 4b's docs-readability pass will be run over files
whose readability edits are the deliverable**, so it will very likely propose further splits of
sentences this ticket just split. That is the reviewer's tool doing its job, not evidence the work
is incomplete. Apply any that are clearly right, and otherwise record a conscious stop — prose
polish has no fixed point, and a second batch of suggestions is not a finding. Do **not** file a
follow-up ticket for another round; that loop is what §5's promotion test exists to refuse.

### Finish (mandatory)

1. Acceptance test green: the four `just` commands, `changelog check`, both `INVARIANT OK` lines,
   and the two by-hand confirmations.
2. Write the summary: which of the ten suggestions were applied, and — per decision 4 — **which
   were declined and why**. State explicitly whether Task 2 was executed or dropped.
3. Suggested commit message:
   ```
   docs: apply batched readability suggestions to the changelog and session-workflow page (T-114)
   ```
4. Tidy WIP commits into atomic ones (root-path child) before presenting — one commit per task is
   a natural split here, and keeps a dropped Task 2 from leaving a gap in the history.
5. Commit locally; do not push or open an MR without explicit user approval. Hand back with
   `pickle ticket move T-114 in-review --reason "acceptance green"`.

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
- 2026-08-22 — refined: all ten quoted anchors re-verified against the current tree and given line
  numbers (`CHANGELOG.md` :13/:22/:38/:60/:121/:223; `agent-session-workflow.adoc` :4/:18/:56/:101).
  Four looked missing under a naive line-based grep because both files hard-wrap — recorded as
  decision 1 so the implementer does not mistake a wrapped anchor for a stale one. Two structural
  facts found at refinement and written into the plan: suggestion 9 edits text **inside an AsciiDoc
  table cell**, where a stray `|` would silently corrupt the table (decision 6), and
  `pickle changelog check` already parses this changelog for ticket ids, making it a free
  regression gate against a prose edit dropping one. The table's shape was **counted, not
  estimated** — an initial draft of decision 6 asserted nine body rows; it has eight, and the
  wrong number would have had the implementer assert a false invariant. Added a mechanical **fact-invariance check**
  (headings, ticket ids, inline code spans, URLs, xrefs must be set-identical before/after) so
  decision 3 is enforced rather than eyeballed. Released changelog sections isolated into a
  droppable Task 2 (decision 2) so that scope call stays reversible without re-refinement.
  Confirmed no changelog entry is added for this ticket (decision 5) and warned that the review's
  own step-4b pass will propose further splits, which must not spawn another round (docs step).
  **Grade unchanged at low/low/S** — verification confirmed the edits are mechanical and
  fact-preserving. Kept as one ticket: ten edits, two files, one class of change.
- 2026-08-22 — TO DO → READY: plan complete
