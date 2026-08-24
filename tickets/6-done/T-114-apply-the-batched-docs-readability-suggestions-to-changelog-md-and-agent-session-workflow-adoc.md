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

None. No `depends-on:`. All ten target anchors were re-verified present at pickup (2026-08-24,
see History) — not merely at refinement, since the tree moved twice more since then. T-111 landed
but touches neither file. **T-113 landed** (`pickle scaffold release`, merged to `main`
2026-08-24) and added a fourth `[Unreleased]/Added` bullet to `CHANGELOG.md`; it does not alter
any of this ticket's six target sentences or the section boundaries they sit in, so it changes
line numbers only, confirmed below — no further re-verification is needed before editing.

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
   **Superseded in part at pickup (2026-08-24, see History):** the 0.11.0 release folded Task 1's
   three anchors into a released section too, so "Task 1 = unreleased/mandatory vs. Task 2 =
   released/droppable" no longer partitions the file. Resolved with user sign-off: **Task 1 stays
   unconditionally in scope regardless of which heading it now sits under** — the edits are the
   same low-risk mechanical sentence splits decision 3 already governs, and only their section
   changed, not their riskiness. Task 2 remains independently droppable on its own terms.
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

#### Task 1 — `CHANGELOG.md`, reviewer-independence / scaffold-docs / cli-reference entries (3 edits)
**Re-anchored at pickup (2026-08-24):** the 0.11.0 release moved all three into `## [0.11.0]`
(no longer `[Unreleased]`; see decision 2's superseding note) at lines ~40, ~49, ~65, and the
"chore: cut v0.11.0" commit (`09fcd69`) independently reworded two of the three while cutting the
release. Anchor on the *current* wording below, not the paraphrase quoted at refinement.
- **~:40-47** `Added` / reviewer independence (the T-112 entry) — the release-cut commit already
  split the original one-sentence version into two ("...independently spawned reviewer." / "It
  must then hand-verify..."). One split remains: break the second sentence's semicolon into a
  full stop, so "hand-verify and record" and "what stays with the reviewer" land as separate
  sentences — completing the three-way split the suggestion asked for.
- **~:49-53** `Added` / `pickle scaffold docs` — current text still carries the trailing
  "— entirely optional and separate from the ticket flow: …" clause undisturbed by the release-cut
  commit; promote it into its own sentence ("This is entirely optional…"), per the original
  suggestion.
- **~:65-72** `Fixed` / `cli-reference.adoc` — the release-cut commit already split the sentence
  about the `renumber` command; the list-heavy flag sentence right before it is untouched and is
  the actual target: recast "`pickle ticket new --family`, and `pickle project add`'s
  `--build`/…/`--wip-review`, existed but were never mentioned in the manual; `pickle flow
  show|list` had no section at all" as two sentences led by "The manual mentioned neither …".

#### Task 2 — `CHANGELOG.md`, released sections (3 edits) — executed, not dropped (see History)
- **~:99-109** `[0.10.0]` / `pickle board decisions` — split "The same answer previously needed a
  hand-written `awk` that re-solved two parsing traps… and got the child filter wrong…" into the
  statement plus its two flaws.
- **~:159-170** `[0.9.0]` / atomic writes — split the temp-file-and-rename sentence, opening the
  second with "As a result,".
- **~:254-266** `[0.7.0]` / installed skill audience — split the three-part "The examples are
  now…, the skeleton's warrant…, and the two classes are defined as…" into parallel sentences.

#### Task 3 — `docs/user-manual/concepts/agent-session-workflow.adoc` (4 edits)
- **:4** opening paragraph — break after "…no model or agent-tier configuration." so the core
  claim lands before the judgment/mechanics elaboration.
- **:18** § *Session boundaries are also cost boundaries* — split at "Switching models mid-session
  doesn't erase…"; render the first half as "…, which is reprocessed by whichever model handles
  the next turn."
- **:56** § *A pattern mapped to the procedures*, *Refine a ticket* row — recast the READY-gate
  sentence as an explicit if/then, and "a fresh session reading just the ticket" → "a fresh session
  that reads just the ticket". **Table cell — observe decision 6.**
- **:102** § *Notes*, first bullet — replace the dash before "it runs a scoped, bounded re-check"
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

Reviewed 2026-08-24 on `feat/T-114-docs-readability-batch` (3 commits at review start:
`cc30911`, `d33f16f`, `15ebe29`). **Reviewer independence:** the reviewing agent authored this
branch in the same session, so per protocol step 0 the implementation/quality/consistency/docs
audits (steps 2–4a) were **delegated** to an independently spawned sub-agent (fresh context, no
memory of writing the code, briefed adversarially, run in an isolated worktree). Every delegated
finding below was re-verified by hand against the tree before being recorded; classification,
severity, disposition and the move stayed with the orchestrating reviewer throughout.

- [x] Reviewer independence settled (step 0): **delegated** to an independent sub-agent for
      steps 2–4a; each finding hand-verified before recording
- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass on the changed `.adoc`/`.md` files (step 4b) — ran (the `docs_readability`
      tool); 8 suggestions, 0 applied — see F4
- [x] Findings recorded with severity, **class** and disposition; summary line present (step 5)
- [x] Ticket moved to `6-done/`; `## History` appended (step 6)
- [x] Other references updated; board regenerated by the move (step 7)
- [x] Remaining-tickets impact sweep done (step 8)
- [x] Summary + bookkeeping commit presented for approval (step 9)

**Acceptance test — re-run verbatim on the branch, all green.** `just build`, `just test`,
`just lint` (`go vet`), `just docs-check` all clean. Fact-invariance `inv()` check against
`main`: `INVARIANT OK` for both `CHANGELOG.md` and `agent-session-workflow.adoc`. `git
diff --word-diff` confirmed by hand: every token changed is a sentence-boundary marker (full
stops, capitalisation, "As a result", "This is", "rather than", a comma for a dash) — no
substantive word swapped for a different claim. Table check: `awk … | grep -c '^| '` → **33**,
exactly as decision 6 requires. `./pickle changelog check` reports 5 candidates (`T-114` itself
— expected, decision 5 — plus `T-103`/`T-042`/`T-111`/`T-071`), confirmed identical on `main`
before this branch: pre-existing, not caused by this ticket — see F3.

**All ten suggestions applied; none declined.** Task 1 (3 edits, `CHANGELOG.md` `[0.11.0]`) and
Task 2 (3 edits, `CHANGELOG.md` `[0.10.0]`/`[0.9.0]`/`[0.7.0]`) **both executed — Task 2 was not
dropped**. Task 3 (4 edits, `agent-session-workflow.adoc`) executed as specified. Two of Task 1's
three targets had already been independently reworded by the `chore: cut v0.11.0` commit
(`09fcd69`) before this ticket could touch them (recorded in `## History` at pickup); those two
were re-anchored on current wording and completed the same class of split the suggestion asked
for, rather than being applied against the now-stale refinement-time quote.

### Findings

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | non-blocking | other | fixed inline | The plan's mandatory Finish step 2 ("write a summary: which suggestions were applied/declined, whether Task 2 was executed or dropped") was never persisted — `## Review` was still the empty placeholder and no commit carried a body. Judged non-blocking rather than a decision-4 violation needing rework: the substance was fully reconstructable and verifiable from the diff itself, and writing it now is prose this ticket's own process required, with no behaviour change. | `## Review` (this section, pre-edit) was `<!-- empty until IN REVIEW -->`; `git log --format=%B main..feat/T-114-docs-readability-batch` showed one-line subjects only | Recorded above: all ten applied, Task 2 executed. |
| F2 | non-blocking | design | noted | Task 2 anchor 4 (`pickle board decisions`, `[0.10.0]`) was specified as "the statement plus its two flaws" (implying three parts) but landed as statement + one sentence joining both flaws with "and". Readable and fact-preserving; a defensible reading, not worth a rework round over. | `CHANGELOG.md:103-104` | None — left as is. |
| F3 | non-blocking | spec-unclear | noted | The acceptance test's literal bar (`./pickle changelog check` must report "no candidates") is already false on `main` before this branch, from four unrelated, pre-existing tickets (T-042, T-071, T-103, T-111) that shipped without changelog entries. Confirmed the candidate set (minus T-114 itself) is identical before and after this branch — not caused by, or in scope for, this ticket. | `./pickle changelog check` output, run on both `main` and the branch | None — pre-existing gap, no owning ticket found; left noted rather than promoted (fails the promotion test for this ticket's theme). |
| F4 | non-blocking | design | noted | Step 4b's docs-readability pass returned 8 further suggestions over the two changed files: 3 propose splitting sentences this ticket just split further (opening paragraph, the cost-boundary sentence, the Notes cache-note bullet); 5 target `CHANGELOG.md` prose entirely outside this ticket's 10 authorized anchors (Unicode line terminators, over-long title, merge-line continuation, base-branch bookkeeping rule, Outcome-presence check). Exactly the situation the ticket's own Docs-update section warned about. | `docs_readability` tool output against `feat/T-114-docs-readability-batch` | Declined all 8 as a conscious stop, per the ticket's explicit instruction ("apply any that are clearly right, and otherwise record a conscious stop … do not file a follow-up ticket for another round"); none were clearly right enough to justify reopening scope. |

**Disposition summary:** 4 findings, **0 blocking** — 1 `fixed inline` (F1), 3 `noted` (F2, F3,
F4). No new tickets: nothing here passes the promotion test.

```
cost: estimated S, actual S
```

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
- 2026-08-23 — patched by **T-111's review impact sweep**: T-111 is `6-done/` (branch unmerged),
  so the "two soft couplings are unstarted" note is stale for that half. It changes nothing here:
  T-111 touched neither file this ticket edits — its diff is `user-manual.adoc`,
  `cli-reference.adoc`, `concepts/project-structure.adoc`, `concepts/releasing.adoc` and
  `installation.adoc`, and **not** `CHANGELOG.md` or `concepts/agent-session-workflow.adoc`. This
  ticket's ten target anchors are therefore untouched by it; the re-verify caveat now applies to
  T-113 alone. Scope and grade unchanged.
- 2026-08-23 — patched by **T-038's review impact sweep**, and it touches this ticket twice, one of
  which is not T-038's doing. (1) **T-038 adds a `### Fixed` block to `[Unreleased]`, shifting all
  three of Task 1's CHANGELOG anchors down by exactly 17 lines** once its branch merges (`:13`/`:22`/
  `:38` → the T-112, `scaffold docs` and `cli-reference` entries, measured on the branch). Anchor on
  the quoted opening phrases, not the numbers. (2) **Pre-existing, and larger: the 0.11.0 release
  already invalidated Task 1's premise.** Task 1 is headed "`[Unreleased]` section" but all three of
  its entries now sit under `## [0.11.0] - 2026-08-22` — a *released* section, which decision 2
  assigns to Task 2 and marks droppable. So the Task 1/Task 2 scope split no longer partitions what
  it was written to partition. Not fixed here (this sweep did not cause it, and re-deciding scope is
  refinement's call): re-refine the split before pickup. Grade unchanged.
- 2026-08-24 — re-refined at pickup (applicability gate, per a fresh sub-agent audit; ticket stayed
  in `2-ready/`, no move). Two blocking findings, both resolved with user sign-off before
  implementation: **(1)** the T-038 sweep's premise was confirmed and closed — decision 2 amended
  with a superseding note: Task 1 stays **unconditionally in scope** even though all three of its
  anchors now sit under `## [0.11.0]`, because the edits are the same low-risk mechanical splits
  decision 3 already governs and only their heading changed, not their riskiness; Task 2 remains
  independently droppable, and both tasks are executed (see Task 2 heading and Finish). **(2)** T-113
  merged 2026-08-24, adding a fourth `[Unreleased]/Added` bullet; confirmed it does not touch any
  of the six CHANGELOG target sentences or move a section boundary, so only line numbers shifted
  (Prerequisite gate updated). A third, non-blocking fact found during re-verification: the
  "chore: cut v0.11.0" commit (`09fcd69`, 2026-08-22) **independently reworded** the reviewer-
  independence and cli-reference entries while cutting the release, before this ticket could touch
  them — the refinement-time quotes for those two are now paraphrases, not verbatim anchors. Task 1
  re-anchored on current wording accordingly; the underlying edit (complete the sentence split /
  recast the list-heavy sentence) is unchanged in kind. Task 3's four `.adoc` anchors were
  re-verified untouched (line 101 → 102 only). All ten target sentences confirmed present
  verbatim-or-current-wording; none dropped a fact. Grade unchanged (low/low/S).
- 2026-08-24 — READY → IN DEVELOPMENT: picked up
- 2026-08-24 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-08-24 — IN REVIEW → DONE: review clean; 4 non-blocking, all dispositioned
- 2026-08-24 — merged to main (PR #69, 1a59501)
