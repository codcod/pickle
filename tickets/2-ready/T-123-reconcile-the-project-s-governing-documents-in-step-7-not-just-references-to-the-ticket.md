---
id: T-123
title: Reconcile the project's governing documents in step 7, not just references to the ticket
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: low
cost: S
---

# T-123 — Reconcile the project's governing documents in step 7, not just references to the ticket

## Outcome

After this ships, a review that changed behaviour the project's own governing documents assert —
its design of record, conventions, locked decisions, decisions log — either updates them or records
why not, and the review checklist says which. A review can no longer pass while leaving a governing
document asserting something the code no longer does.

## Description

Step 7 of `review-protocol.md`, "Update other references", covers exactly two things:

> - **`BOARD.md` needs no hand edit** — it is generated …
> - Any ticket or doc that referenced this ticket **by id**.

Both are references **to the ticket**. Nothing in the protocol asks whether the documents that
*govern* the work — a design of record, a conventions file, a locked-decisions guide, a decisions
log — still describe what the code now does. Step 4a covers the **shipped** docs tree (the
product's user-facing documentation); the governing documents are usually not in it, and under the
default `umbrella` layout they are not even in the same repository as the code.

The gap is not theoretical. Three independent instances from one downstream workspace: a design of
record asserting a CLI returned "the API's bytes" for a field, when that ticket's **own review had
retracted "byte-identical" twice**; a design listing a ticket as an outstanding follow-up after it
was implemented, reviewed and merged; and four governing documents drifting from the code for a
month **at an unchanged child HEAD**, so no external change caused it.

The common shape: a review *does* notice the truth, records it in the ticket, and the ticket is
archived. The governing document — the thing the next ticket is cut from — keeps the falsehood.
That is worse than an undocumented change, because it is confidently wrong and it propagates:
later tickets are written from it.

### What refinement confirmed

- **Step 7 really is the whole surface.** No other step asks the question. 4a's sweep is scoped to
  "the docs tree" the project *ships*; step 8's impact sweep is scoped to *tickets* that depend on
  this one. A governing document is neither, so it falls between them.
- **The class vocabulary already has the right word.** `stale-xref` is defined as "a reference this
  branch made false: line anchors, cross-references, comments, or plan prose describing behaviour
  that changed" — a governing document made false by the branch is precisely that. No new class is
  needed, and `docs-gap` is wrong here because it is scoped to *user-facing* docs.
- **The protocol can name the documents without pickle knowing them.** Step 1 already tells the
  reviewer to read "any project-wide decisions … from the project's `AGENTS.md`", so step 7 can
  refer to the documents step 1 named plus the kinds listed above. That is what keeps this a
  prose-only change (decision 2 below).
- **Evidence stays out of the payload.** The three instances above argue for the decision; they are
  not shipped text. A reader of an installed skill has no access to that workspace, and
  `payload_lint_test.go` rejects counts and claims of that shape.

### Decisions taken at refinement (user-confirmed)

The three open questions this ticket carried are settled and written as numbered decisions in the
plan: **no config-schema change** — the protocol names the kinds of document and defers
identification to what step 1 told the reviewer to read (2); **severity is conditional** — blocking
when the branch made a reachable governing document false, non-blocking with a recorded reason when
the document is out of the review's publishing reach (3); the finding is classed **`stale-xref`**
(4); step 7's **existing checklist line is extended** rather than a new one added (5); and the step
**keeps its number 7** while its title widens, because addenda key on numbers (1).

### Explicitly out of scope

- **A `governing_docs` config key.** Considered and deferred: it would add schema, install/upgrade
  handling, `doctor` validation and manual coverage to what is otherwise a prose rule. If the rule
  proves invisible in practice — reviewers not knowing which documents are meant — a later ticket
  adds the key, informed by that usage.
- **A machine check.** Nothing in `internal/audit` learns about governing documents: the flow does
  not know which files they are (above), so there is nothing to check against.
- **A `review_addendum` for pickle itself.** pickle's own governing documents are `DESIGN.md` and
  `AGENTS.md`, and the generic rule already reaches them; creating pickle's first overarching
  addendum file is a separate decision affecting every future review here, not a task of this
  ticket. This ticket's own review is instead the first place to apply the shipped rule.

### Couplings

T-122 edits the same file (`skill/resources/review-protocol.md`) in a different section (step 4b).
They are independent and can land in either order; whichever lands second rebases.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .
git checkout main
git checkout -b feat/T-123-reconcile-governing-documents
```

Root-path child (`project: pickle`), so tidy WIP commits into atomic ones before presenting
(Finish, below) and default to keeping that history rather than squashing.

### Prerequisite gate (hard)

None. No `depends-on:`. The one soft coupling (T-122, same file, different section) imposes no
ordering — if it lands first, rebase before starting.

### Confirmed design decisions (do not deviate without asking)

1. **Step 7 keeps its number; only its title widens** — to cover reconciling governing documents as
   well as updating references. The protocol's intro states that project review addenda are "keyed
   to this protocol's step numbers", so renumbering silently invalidates every addendum an
   installed project has written. Titles carry no such contract, and "Update other references" is a
   title the new duty outgrows.
2. **No config-schema change: the protocol names the *kinds* of governing document and defers
   identification to the project.** It lists the design of record, conventions, locked decisions and
   decisions log as examples, and points at the documents step 1 already told the reviewer to read
   (via the project's `AGENTS.md`). Adding a `governing_docs` key would turn a prose rule into a
   schema, install/upgrade, `doctor` and manual change, and the rule has no usage behind it yet.
3. **Severity is conditional on reach, and the condition is written into the rule.** A governing
   document this branch made false is a **blocking** finding when it is within the review's reach —
   the same repository, or one this review can commit to — because a false governing document
   propagates into every ticket cut from it, which is the same reasoning that makes missing docs
   coverage blocking in 4a.1. When the document is out of reach (another repository, another commit
   policy, another team), the finding is **non-blocking** and takes a disposition per the rules §5,
   with the reason for the reach limit recorded. Unconditional blocking would make the protocol
   unfollowable in an umbrella layout across repository boundaries, and an unfollowable rule is
   ignored wholesale.
4. **The finding is classed `stale-xref`; no new class value is added.** The existing definition
   ("a reference this branch made false … or plan prose describing behaviour that changed") already
   covers it. `docs-gap` is explicitly the *user-facing* docs class and must not absorb this. The
   closed vocabulary is not extended, re-ordered or renamed — its value is comparability.
5. **The checklist gains no new line; step 7's existing line is extended.** Step 7 already has one
   line, and the reconciliation is part of the same step's answer.
6. **"Or record why not" is a first-class outcome, not an escape hatch.** A reviewer may
   legitimately conclude the governing document is right and the ticket's own prose was loose, or
   that the reconciliation belongs to a ticket that owns the document. Both are acceptable; silence
   is not. This mirrors the recording requirement step 0 and step 4b already use.
7. **Nothing is said about *how* a governing document is edited.** No commit-policy change, no new
   pathspec rule: the rule states the duty, and the project's configured commit policy already
   governs where and how the edit lands.
8. **The payload states the rule on its own reasoning — no counts, no workspace names, no ticket
   ids.** The Description's three instances are evidence for the decision, not shipped text;
   `payload_lint_test.go` enforces the mechanical part of this.

### Tasks

> **The line numbers in the tasks below are stale by 21 lines; T-122 has landed.** Verified against
> `main` after the merge: `## 7.` is at `:301` (tasks say `:280`), the step-7 checklist line at
> `:365` (tasks say `:344`), and 4a's "Follow any project-specific…" line is unmoved at `:176`,
> being above T-122's insertion. Locate every target by its heading or line text rather than the
> number — the numbers are recorded here only to show the offset is understood.

#### Task 1 — widen step 7 and add the reconciliation rule
In `skill/resources/review-protocol.md`, at `## 7. Update other references` (`:280`):

- change the title to cover both duties (decision 1), e.g.
  `## 7. Update other references — and reconcile the project's governing documents`, leaving the
  number `7` untouched;
- keep the two existing bullets (`:282-285`) as they are;
- add a third bullet plus a short paragraph stating the rule, in payload-safe prose (decision 8):
  **if the ticket changed behaviour the project's governing documents assert — its design,
  conventions, locked decisions, decisions log, and whatever else step 1 named — update them in the
  same review, or record explicitly why not**; a review that leaves a governing document asserting
  something the code no longer does has not finished;
- state the boundary against 4a in one clause: 4a audits the docs the project *ships*; these are
  the documents the *next ticket is written from*, which is why they need naming separately
  (decision 2);
- state the severity rule (decision 3) and the class (decision 4) in one sentence each, citing §5
  for the disposition vocabulary rather than restating it.

#### Task 2 — extend the step-7 checklist line
In the same file's `### Checklist (paste into the ticket's `## Review` section)` block, extend the
existing step-7 line (`:344`) so the reconciliation is part of the same tick — no new line
(decision 5), e.g.:

`- [ ] Other references updated if needed; governing documents reconciled, or an explicit note why
not; board regenerated by the move (step 7)`

#### Task 3 — cross-reference from the docs audit
In the same file's `## 4a.` section, after its "Follow any project-specific documentation rules"
line (`:176`), add one sentence noting that 4a's sweep is the *shipped* docs tree and that the
project's governing documents are step 7's business, pointing there. Rationale: 4a is where a
reviewer currently forms the belief that documentation is already covered, and that belief is
exactly what let the gap persist. One clause, no restatement of the rule.

#### Task 4 — the manual
In `docs/user-manual/concepts/lifecycle.adoc`, in the review treatment (the *Reviews: severity, then
disposition* section, around `:95-125`), add two or three lines recording that a review also
reconciles the project's governing documents, or records why not, with the severity condition
stated once and a pointer to the protocol's step 7. Do not restate the rule's full text — the
payload is the authority and the two must not drift into two policies.

#### Task 5 — CHANGELOG entry
Add an `[Unreleased]` entry in `CHANGELOG.md` in the established style (bold lead sentence, then the
qualification, ticket id in trailing parens), naming `resources/review-protocol.md` as the changed
payload file and noting that every installed project picks the rule up on its next
`pickle upgrade`.

### Acceptance test

```
just build
just test
just lint
just docs-check
```

All clean. `just test` is load-bearing, not a formality: it runs `payload_lint_test.go` over the
embedded payload, which mechanically enforces decision 8 for the `skill/` file this ticket edits;
`just docs-check` runs the xref tests over the edited manual page.

Then, specifically:

1. **No step was renumbered** (decision 1) — `grep -n '^## [0-9]' skill/resources/review-protocol.md`
   shows `## 0.` through `## 9.` with `## 4a.`/`## 4b.` present and every number as on `main`
   (compare with `git show main:skill/resources/review-protocol.md | grep -n '^## [0-9]'`); the
   line for step 7 shows the widened title.
2. **The class vocabulary is untouched** (decision 4) —
   `git diff main -- skill/resources/review-protocol.md` shows no added, removed or re-ordered row
   inside the closed-vocabulary table under the "The class column" sub-heading of §5: its eight
   rows are byte-identical to `main`.
3. **The checklist gained no line** (decision 5) —
   `grep -c '^- \[ \]' skill/resources/review-protocol.md` equals
   `git show main:skill/resources/review-protocol.md | grep -c '^- \[ \]'`, and the step-7 line now
   mentions governing documents.
4. **The rule ships, not just exists locally** — install into a throwaway dir per `AGENTS.md`'s
   self-modify policy and confirm the installed payload carries it:
   ```
   D=$(mktemp -d) && cp pickle "$D/pickle-test" && cd "$D" && ./pickle-test install --in-tree --project demo
   grep -c 'governing' .agents/skills/brine/resources/review-protocol.md   # expect ≥ 3 (title, rule, checklist)
   ```
5. **The rule is executable against this very branch** — apply it to pickle's own governing
   documents (`DESIGN.md`, `AGENTS.md`): confirm neither asserts anything this branch made false,
   or update it, and say which in the summary. This is the acceptance criterion that proves the rule
   can be followed rather than only read.

### Docs update (mandatory when user-facing)

User-facing, in two places, both tasks rather than deferrals:
`docs/user-manual/concepts/lifecycle.adoc` gains the review-duty lines (Task 4) and `CHANGELOG.md`
gains an `[Unreleased]` entry (Task 5). Verified at refinement that no manual page enumerates the
protocol's steps or describes step 7 — the only nearby references are lifecycle's severity/
disposition treatment, `concepts/the-flow.adoc`'s payload inventory, and `configuration.adoc`'s
`review_addendum` mention, and only the first is edited. Run the docs-readability reviewer over both
changed files during review (protocol step 4b).

### Finish (mandatory)

1. Acceptance test green, including all five specific checks above.
2. Docs updated (Tasks 4 and 5) and `just docs-check` clean.
3. Write a summary: the three files touched, the step-7 text as shipped, confirmation that no step
   number moved, no checklist line was added and the `class` vocabulary is unchanged, plus the
   result of applying the new rule to `DESIGN.md` and `AGENTS.md`.
4. Suggested commit message:
   ```
   feat(skill): reconcile the project's governing documents in review step 7 (T-123)
   ```
5. Tidy WIP commits into atomic ones (root-path child) before presenting.
6. Commit locally; do not push or open an MR without explicit user approval. Hand back with
   `pickle ticket move T-123 in-review --reason "acceptance green"`.

**Note for whoever picks this up:** the rule this ticket ships applies to this ticket. pickle's
governing documents are `DESIGN.md` and `AGENTS.md`; reconciling them (or recording that nothing
drifted) during this ticket's own review is the cheapest available proof the step is followable.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-25 — created (TO DO). source: field-use: a downstream workspace found three governing documents asserting behaviour their own reviews had already retracted, and now carries the reconciliation rule in its own overarching addendum
- 2026-08-25 — TO DO → READY: plan complete
- 2026-08-25 — plan annotated by T-122's review impact sweep: the cited line numbers shift by ~15 once T-122 lands; targets to be located by heading text instead
- 2026-08-25 — annotation corrected against T-122's finished branch: the shift is 21 lines, not ~15 (`## 7.` at :301, step-7 checklist line at :365)
- 2026-08-26 — T-122 merged; the predicted anchors verified exact on main, annotation switched from predicted to actual
