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

### Checklist

- [x] Reviewer independence settled (step 0): **delegated** — the reviewing agent authored the branch in this same session, so audits 2–4a ran in a freshly spawned, adversarially briefed sub-agent; classification, severity, dispositions and the move stayed with the orchestrating reviewer, and every delegated finding was re-verified by hand before entering the table below
- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass on the ticket's changed `.adoc`/`.md` files — run over all three; **15 suggestions, 15 quotes verified verbatim, 0 discarded as fabricated** (step 4b)
- [x] Findings recorded with severity, class and disposition (step 5)
- [x] Ticket moved to `tickets/5-rework/`; `## History` appended (step 6a)
- [ ] Other references / governing documents — deferred to the rework pass's own review (step 7)
- [x] Remaining-tickets impact sweep done (step 8)
- [ ] Summary + commit message presented for approval (step 9) — not applicable on a rework verdict; nothing is published

### Commands

`just build` · `just test` · `just lint` · `just docs-check` — **all four green** (verified independently by the delegated reviewer at commit `3e9b9be`). `just test` includes `payload_lint_test.go` over the embedded payload.

### Acceptance test — the five specific checks

1. No step renumbered — **met**; number set identical to `main` (`0,1,2,3,4,4a,4b,5,6,7,8,9`), only offsets moved; step 7 shows the widened title.
2. Class vocabulary byte-identical — **met**; table extracted from both sides and diffed, 8 rows + header identical.
3. Checklist gained no line — **met**; `grep -c '^- \[ \]'` = 11 on both sides, step-7 line extended in place.
4. Rule ships in an installed payload — **met**; throwaway install as `pickle-test` per the self-modify policy, `grep -c governing` = 7 (≥ 3 required).
5. Rule applied to pickle's own governing documents — **met on substance, was unrecorded** (F8). Re-performed during this review: `DESIGN.md:190-192` ("projects layer extra checks … keyed to its step numbers") stays true *precisely because* no step was renumbered; `DESIGN.md:53,126` and `AGENTS.md:73-74` reference the protocol generically; `tickets/NOTES.md` carries no claim this branch falsified. **Nothing this branch made false — no reconciliation edit needed**, and this line is the record Finish item 3 required.

### Findings

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | blocking | `spec-unclear` | — | Step 7 mints a **blocking** finding at a point where the blocking machinery has already been passed, and its three clauses can be discharged in mutually exclusive ways. Step 6b has already moved the ticket to `6-done/` on the finding that no blocking finding exists, and nothing in steps 7–9 routes back to §5/6a. Step 5 also says blocking findings are **not** fixed inline, while step 7's bullet says reconcile now. | `review-protocol.md:316-320` vs `:298-300` (6b) and `:259-263` (no inline fix); contrast step 8 at `:326-328`, which avoids this by framing everything as a disposition | Reconciling *within reach* is done in the review and recorded as a `fixed inline` non-blocking finding; it becomes blocking, re-entering §5/6a, only when the reconciliation cannot be made in this review. No renumbering, so decision 1 stays safe. |
| F2 | blocking | `spec-unclear` | — | The reach test is undecidable in an umbrella layout — the ticket's *own* decision 3 rests the conditional on umbrella followability, and then ships a condition ambiguous exactly there. "The same repository" has no antecedent when the code sits in the child and the design of record in the overarching repo; "can commit to" does not say *commit locally* vs *publish*, a distinction this same file treats as decisive. | `review-protocol.md:317-318` vs `:333-341` (publish-gated policy); decision 3 in the plan | Name the antecedent: "the repository holding the branch under review, or any repository this review is already authorised to commit to under the project's commit policy". |
| F3 | blocking | `spec-unclear` | — | Decision 6 locks "*or record why not* is a first-class outcome, **not an escape hatch**" and names the two legitimate grounds. **None of that bound reached the payload**: as shipped, "wrote a note" always satisfies step 7. A shipped rule contradicting a confirmed decision of its own plan is blocking by §5's definition. | `review-protocol.md:311` and `:379` (bare alternative) vs decision 6 in the plan | One clause naming what a legitimate "why not" looks like — the document is right and the ticket's prose was loose; the reconciliation belongs to a ticket that owns the document; or the reach limit of F2. |
| F4 | blocking | `spec-unclear` | — | The identification back-reference is empty: step 7 defers to "whatever else step 1 named", but step 1 names no documents — it says only "read any project-wide decisions … from the project's `AGENTS.md`". Decision 2 rests the entire no-config-key argument on this pointer, so the mechanism is weaker than the decision claims and the four example kinds carry all the weight. | `review-protocol.md:310` vs `:122-123`; decision 2 in the plan | Point at the documents **`AGENTS.md` itself** names, rather than at step 1. |
| F5 | blocking | `spec-unclear` | — | Terminology collision inside one file: `:309` uses "locked decisions" as a *kind of governing document*, while `:257` uses "contradicts a locked decision" to mean a **ticket's** confirmed design decision — the flow's own first-class concept. A foreign reader can read step 7's item as "the tickets' locked decisions", which §5 already covers. The CHANGELOG got this right and the payload did not. | `review-protocol.md:309` vs `:257`; `tickets-README.md:507`; cf. `CHANGELOG.md:23` "locked-decisions **guide**" | Adopt the CHANGELOG's noun phrase in the payload and in `lifecycle.adoc:120`. |
| F6 | non-blocking | `docs-gap` | folded | `lifecycle.adoc:121` writes "**one** a branch made false is blocking…" — a bare pronoun reaching back across an em-dash list to "governing documents". Independently flagged by the step-4b readability reviewer (quote verified verbatim). | `lifecycle.adoc:119-122` | "A governing document the branch made false is blocking within the review's reach and non-blocking otherwise." Folded into the rework pass, which must re-sync this paragraph with the rewritten payload anyway. |
| F7 | non-blocking | `design` | folded | The new manual paragraph is inserted **between** the class-vocabulary paragraph and "A review closes with two one-line summaries under the table", splitting one continuous narrative about the findings table with an unrelated duty. | `lifecycle.adoc:119-122` sits between `:109-117` and `:124` | Move it to the end of the section, after the `cost:` paragraph. Folded into the rework pass for the same reason as F6. |
| F8 | non-blocking | `docs-gap` | fixed inline | Finish item 3 required the summary to state the result of applying the new rule to `DESIGN.md`/`AGENTS.md` (acceptance check 5); it existed only in a hand-back chat message, not in the archive, so the branch's own proof that its rule is followable was unverifiable. | `## Review` was `<!-- empty until IN REVIEW -->`; History carried only "acceptance green" | Fixed by this review: check 5 above now records it. This is also this branch's first real exercise of its own step 7. |

**Disposition summary:** 5 blocking (F1–F5, no disposition — they define the rework scope) · 2 folded into the rework pass (F6, F7) · 1 fixed inline (F8) · 0 new tickets. Separately, the step-4b readability pass returned 13 further suggestions against prose this branch never touched (protocol opening, in-tree box, step 0, step 4b, §5 intro; lifecycle's state machine, READY gate, pickup gates, done≠merged; CHANGELOG's T-122, T-113 and doctor entries) — all quoted accurately, all **noted** and closed rather than turned into a prose-refactor of this ticket's diff.

cost: estimated S, actual M — the three-file edit was S as graded; the shipped paragraph needing a second pass on five counts is what makes the ticket M.

**Note on the shape of this review.** Every blocking finding is in the same six lines of payload prose, and none is a coding error — the deliverable *is* the rule, so an under-specified rule is a defective deliverable rather than a cosmetic one. The four commands were green throughout and the five mechanical checks all passed; what the delegated audit caught is that a rule can satisfy every structural check and still be unfollowable by the reader it ships to. That is the argument for step 0's delegation, made against this very branch.

### Rework fix record (commit `6f0f135`)

- **F1 — fixed.** The governing-document check now triggers during the consistency audit
  (step 4, `review-protocol.md:158-161`) and classifies at step 5 like any other finding
  (`:260-261`, a cross-reference mirroring the existing 4a.1 pattern) — both **before** step 6's
  move. Step 7 no longer discovers severity; it states the rule and, for a non-blocking
  in-review disposition, is where the mechanical edit lands. A blocking one is fixed during the
  rework pass it already routed to, consistent with step 5's "do not fix it inline". No step
  renumbered.
- **F2 — fixed.** "the same repository" replaced with an explicit antecedent: "the repository
  holding the branch under review, or any repository this review is already authorised to
  commit to under the project's commit policy" (`review-protocol.md:321-322`).
- **F3 — fixed.** Step 7's bullet now names the two legitimate grounds decision 6 always
  intended (document is right, prose was loose; reconciliation belongs to a ticket that owns
  the document) plus the reach limit, instead of a bare, unbounded alternative
  (`review-protocol.md:313-317`).
- **F4 — fixed.** The identification back-reference now points at what `AGENTS.md` itself
  names, not at step 1, in both step 4 and step 7 (`review-protocol.md:160`, `:314`).
- **F5 — fixed.** "locked decisions" (governing-document sense) renamed to "the
  locked-decisions guide" in both the payload and the manual, adopting the CHANGELOG's already-
  correct noun phrase; "contradicts a locked decision" (a ticket's own confirmed decision, §5's
  sense) is now the only remaining use of the bare phrase (`review-protocol.md:314`, `:260`;
  `lifecycle.adoc:127`).
- **F6 — fixed** (was folded; done inline as part of the rework rather than deferred).
  `lifecycle.adoc`'s paragraph now reads "A governing document the branch made false is blocking
  within the review's reach and non-blocking otherwise."
- **F7 — fixed** (was folded; done inline as part of the rework rather than deferred). The
  paragraph moved to the end of the section, after the `cost:` line, restoring the findings-
  table narrative's continuity.
- **F8** — already recorded in this ticket's own `## Review` above (check 5); unaffected by
  this rework pass.

**Verified after the fix:** all four commands (`just build`, `just test`, `just lint`,
`just docs-check`) green, including `payload_lint_test.go` and `TestPayloadDispositionVocabulary`
(the rework's own prose was tightened to avoid literally restating the `fixed inline`/`folded`
disposition tokens outside `tickets-README.md`, per that test). All five of the ticket's original
specific acceptance checks re-verified and still hold: no step renumbered (only step 7's title
text, by design); the class-vocabulary table still byte-identical to `main`; the checklist line
count still 11=11; the installed throwaway payload (`pickle-test`) still greps `governing` ≥ 3
(now 8); `DESIGN.md`/`AGENTS.md` still assert nothing this branch made false.

cost: estimated S, actual M (unchanged by the rework — recorded once, at the review that found
the cost gap).

---

## Scoped re-review (round 2)

Reviewer independence: **delegated** again — the orchestrating reviewer wrote the rework in the
same session. Scope per protocol step 1: verify F1–F7 only, plus any defect the rework itself
introduced. Every finding below was re-verified by hand against the tree before recording.

**Commands:** `just build` · `just test` · `just lint` · `just docs-check` — all four green at
`6f0f135`, including `payload_lint_test.go` and `TestPayloadDispositionVocabulary`.

**Five original acceptance checks:** all still hold (numbering identical to `main`; class table
byte-identical; checklist 11=11; installed `pickle-test` payload greps `governing` = 8; ≥ 3;
`DESIGN.md`/`AGENTS.md` still assert nothing this branch made false).

**Prior findings:** F2, F3, F4, F5, F6, F7 — **resolved**. F1 — **partial**: the ordering half is
genuinely fixed (the check now triggers at step 4 and classifies at step 5, before step 6's move),
but step 7's paragraph still contradicts itself on the disposition half. Step 4b's readability
reviewer independently returned **no** suggestion against the reworked manual paragraph, which is
corroborating evidence for F6/F7.

### Round-2 findings

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| N1 | blocking | `spec-unclear` | — | **Step 7 recommends a disposition its own severity rule makes unreachable.** The rule is: blocking when in reach, non-blocking *otherwise* — so the only non-blocking case is **out of reach**. The next sentence then tells the reviewer that the non-blocking one is "most often the one for prose this branch made false and no behaviour change" — i.e. reconcile it now. You cannot edit a document you are, by the same sentence's definition, not authorised to commit to. The rework grafted the *disposition* half of F1's suggested fix (in-reach → non-blocking + inline edit) onto decision 3's *opposite* severity split (in-reach → blocking). Reads plausibly; cannot be executed. | `review-protocol.md:319-327` vs `tickets-README.md:414` | Delete the disposition-recommendation clause entirely. Step 5 already assigns dispositions before step 7 is ever reached, so step 7 restating which one to pick is redundant — and here, wrong. |
| N2 | blocking | `spec-unclear` | — | **F3's second ground contradicts the blocking rule.** The bullet permits discharging step 7 with a note when "the reconciliation belongs to a ticket that owns the document", offered unconditionally; but §5 and step 7 both make that same document **blocking** when the branch made it false and it is in reach. A reviewer facing an in-reach, branch-falsified, other-ticket-owned governing document has two rules returning opposite verdicts (rework vs. proceed to done). | `review-protocol.md:315-317` vs `:261-262` and `:319-322` | Scope the "why not" grounds to the non-blocking case explicitly — they are reasons a *non-blocking* finding is not reconciled, not an escape from a blocking one. |
| N3 | non-blocking | `stale-xref` | folded | `"get reconciled (**mechanics below**)"` — nothing below is mechanics. What follows is classification, severity, class and the 4a boundary. Decision 7 explicitly forbids stating mechanics, so this pointer can never be satisfied. Introduced by the rework's own smoothing edit. | `review-protocol.md:315` vs `:319-330` | Read "severity below", or drop the parenthetical. |
| N4 | non-blocking | `spec-unclear` | folded | Residual F1: the rework *asserts* the discovery order rather than routing it. Step 4 says "classify … per step 7's rule **now**"; step 7 says it "**was** classified during the consistency audit (step 4)". Mutual pointers, and the past tense presupposes step 4 ran. A falsehood first noticed *at* step 7 — plausible, since step 7 is when you open documents referencing the ticket — still has no route back to §5/6a. Mitigated, not eliminated. | `review-protocol.md:158-161` vs `:319-320` | One clause: if it surfaces here, take it back to §5. |
| N5 | non-blocking | `spec-unclear` | noted | §5 declares severity "defined in **the rules §5**, which is their single source of truth", and the rework adds a third blocking category absent from the rules' own list. Precedented by the pre-existing 4a.1 item, but the drift is now two items wide. | `review-protocol.md:247`, `:261-262` vs `tickets-README.md:398-402` | Out of this ticket's scope — the rules file is not its to edit. Recorded so the drift is visible. |
| N6 | non-blocking | `docs-gap` | folded | `CHANGELOG.md` still says "Shipped in `resources/review-protocol.md` (**step 7**)", but after the rework the load-bearing trigger is step 4 and the severity clause is step 5. The rework touched neither the CHANGELOG nor the plan's Tasks 1–3, which also still scope the change to step 7 alone. | `CHANGELOG.md:26`; plan Tasks 1–3 | Widen the CHANGELOG phrasing to name steps 4, 5 and 7. |
| N7 | non-blocking | `stale-xref` | folded | The Rework fix record cites `:260-261` for F1's step-5 cross-reference; the added clause is at `:261-262` (`:260` is the pre-existing bullet opener). Trivial — and precisely the defect class this ticket ships a rule about. | fix record vs `review-protocol.md:260-262` | Correct to `:261-262`. |
| N8 | non-blocking | `design` | noted | F7's move is correct, but the new duty now lands *after* "A review closes with two one-line summaries" — a terminal sentence — so the paragraph describes work occurring after the review closed. | `lifecycle.adoc:119-124` then `:126-131` | Either accept (the section is a concept overview, not a strict sequence) or move it above the closing sentence. Recorded; the placement is defensible as-is. |

**Disposition summary:** 2 blocking (N1, N2 — they define the round-2 rework scope) · 4 folded into that pass (N3, N4, N6, N7) · 2 noted (N5, N8) · 0 new tickets. Step 4b: 11 suggestions, 11 quotes verified verbatim, 0 discarded; 1 targets reworked text (splitting step 7's long sentence) and is folded into the N1/N2 rewrite, which replaces that sentence anyway; the other 10 target untouched prose and are **noted**.

cost: estimated S, actual L — revised up from M. Two rework cycles on the same six lines of prose,
with the second introducing a fresh contradiction while fixing five, is an L in substance whatever
the diff size says.

**Root cause, for the rework pass to weigh.** N1 is not a slip of the pen. F1's suggested fix and
the plan's **decision 3** point in opposite directions: F1 proposed *in-reach → non-blocking, fix
it right there*; decision 3 (user-confirmed at refinement) says *in-reach → blocking*. The rework
took the severity from one and the disposition from the other, and the seam is N1. The two coherent
resolutions are (a) keep decision 3 and delete step 7's disposition sentence, or (b) revisit
decision 3 so an in-reach reconciliation is a recorded inline fix and blocking is reserved for what
cannot be reconciled. (a) is a prose fix inside the existing decisions; (b) is a design change that
needs user sign-off. The choice belongs to the human, not to the rework pass.

## History

- 2026-08-25 — created (TO DO). source: field-use: a downstream workspace found three governing documents asserting behaviour their own reviews had already retracted, and now carries the reconciliation rule in its own overarching addendum
- 2026-08-25 — TO DO → READY: plan complete
- 2026-08-25 — plan annotated by T-122's review impact sweep: the cited line numbers shift by ~15 once T-122 lands; targets to be located by heading text instead
- 2026-08-25 — annotation corrected against T-122's finished branch: the shift is 21 lines, not ~15 (`## 7.` at :301, step-7 checklist line at :365)
- 2026-08-26 — T-122 merged; the predicted anchors verified exact on main, annotation switched from predicted to actual
- 2026-08-25 — READY → IN DEVELOPMENT: picked up
- 2026-08-25 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-08-25 — IN REVIEW → REWORK: 5 blocking findings (F1-F5): step-7 rule under-specified
- 2026-08-25 — REWORK → IN REVIEW: findings fixed
- 2026-08-25 — IN REVIEW → REWORK: 2 blocking findings (N1, N2): step-7 disposition clause contradicts its own severity rule
