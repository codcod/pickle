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
3. **Reconciling is ordinary review work; step 7 defines no severity of its own.** *(Amended
   twice, both with user sign-off — see History. The original made an in-reach falsified document
   **blocking**; the first amendment made it non-blocking but kept a blocking carve-out for a
   genuine branch-vs-document disagreement. Three review rounds put every one of their blocking
   findings inside that carve-out — F1, then N1/N2, then R1/R2 — and never in the non-blocking
   default. R1 showed the carve-out's stated warrant was false: §5's "contradicts a locked
   decision" demands a citation `<ID> decision <N>`, and rules §7 defines that as a numbered item
   in a **ticket's** Implementation Plan, an address no design of record has. The carve-out is
   therefore removed rather than re-argued. What follows replaces both earlier versions.)*
   - **Within the review's reach** — the repository holding the branch under review, or any
     repository this review is already authorised to commit to under the project's commit policy
     — the reviewer **reconciles the document during the review** and records it as a
     **non-blocking** finding. This is precisely what the rules §5 disposition for "prose this
     branch authored, or made false — and no behaviour change" is defined for, so the flow needs
     no new machinery to carry it, and the common case (a design doc lagging the code) costs one
     recorded edit instead of a rework round-trip.
   - **Out of reach** (another repository, another commit policy, another team) — **non-blocking**,
     with the reach limit recorded as the reason, dispositioned per the rules §5. Blocking here
     would strand the ticket on an edit the reviewer cannot make.
   - **No severity rule of its own.** §5 already owns severity, and its existing categories settle
     every case without step 7 adding one. A stale governing document breaks no golden path, ships
     no wrong behaviour, contradicts no *ticket* decision and is not missing 4a.1 coverage (that
     row is about the docs the project **ships**) — so §5 makes it non-blocking, which is the
     outcome the two bullets above describe. And if the document turns out to be **right** and the
     branch **wrong**, §5's existing "ships wrong behaviour" makes that blocking by the ordinary
     route, with no citation format to fill and nothing new to define. Both cases fall out of §5
     unaided; the carve-out only ever added a way to contradict it.

   The original decision's worry — that a false governing document propagates into every ticket
   cut from it — is preserved: the document still cannot survive the review asserting something
   false. What changes is that the fix happens *in* the review instead of via a rework round-trip,
   and that step 7 states a **duty and its mechanics**, never a severity.
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
- state the class (decision 4) in one sentence, citing the rules §5 for the disposition
  vocabulary rather than restating it. *(Corrected at round-4 review, F5: this bullet
  originally also said "state the severity rule (decision 3)". Twice-amended decision 3 forbids
  step 7 stating any severity of its own, so the instruction contradicted its own decision.)*

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

### Rework fix record — round 1 (commit `6f0f135`)

> **Superseded in part by round 2.** F2, F4, F5, F6 and F7 below stand as recorded and were
> confirmed resolved by the scoped re-review. F1's and F3's fixes were replaced by the round-2
> pass under amended decision 3 — see the round-2 record beneath the re-review findings. The
> `:260-261` citation in F1's entry is off by one (the clause is at `:261-262`); that clause has
> since been reverted entirely, so the line no longer exists on the branch (N7).

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

### Rework fix record — round 2 (commit `6d3aaee`)

Implements **amended decision 3** (user sign-off, recorded in History). Line references are to
`skill/resources/review-protocol.md` on the branch at `6d3aaee`.

- **N1 — fixed.** The rule is now non-blocking by default. In reach → reconcile during the review
  and record the edit, taking the rules §5 disposition for prose this branch made false with no
  behaviour change (`:316-324`). The recommendation now attaches to the case where the reviewer
  *can* act; previously it attached to the out-of-reach case, the one case where they cannot.
- **N2 — fixed.** With non-blocking as the default, the "why not" grounds no longer contradict a
  blocking verdict. They are scoped in their own paragraph (`:334-337`).
- **N3 — fixed.** "(mechanics below)" removed; the bullet now reads "reconciled against what the
  branch shipped, or an explicit note recording why not" (`:313-315`).
- **N4 — fixed.** Step 4 no longer presupposes its own execution (`:158-161`), and step 7 carries
  an explicit route for a falsehood first noticed there (`:322-323`, "A falsehood first noticed
  here rather than at step 4 takes the same route").
- **N5 — fixed.** Step 5's blocking bullet is reverted **byte-identical to `main`**: blocking is
  carried by §5's existing "contradicts a locked decision", so the payload adds no new blocking
  category and the two-item severity drift is closed. Verified by `diff` against `main`.
- **N6 — fixed.** `CHANGELOG.md` now names the steps that actually carry the rule (4, 5 and 7)
  and states the amended severity.
- **N7 — fixed.** Corrected above, with a note that the cited clause no longer exists.
- **N8 — noted, unchanged.** The manual paragraph stays at the end of its section; the section is
  a concept overview rather than a strict sequence, so the placement is defensible.

**Additionally fixed, found during this pass's own verification and not by either reviewer:** the
first draft of this rework routed a blocking disagreement "back to §5 and step 6a" — which is
**unexecutable** once step 6b has run, because `6-done/` is a terminal status with no outbound
transition (`internal/flow/brine.go`, the brine transition table). The shipped text now routes
that case to a follow-up ticket instead, and only adds it to a rework pass's scope when step 6a
actually sent the ticket there. This is the same class of defect as N1 — a rule that reads
plausibly and cannot be executed — caught this time before it shipped.

**Verified after the fix:** all four commands green; the five original acceptance checks all
re-verified (no step renumbered; class table byte-identical to `main`; checklist 11=11; installed
`pickle-test` payload greps `governing` = 8 ≥ 3; `DESIGN.md`/`AGENTS.md` assert nothing this branch
made false — `DESIGN.md:53` "blocking vs non-blocking" and `:191-192` "keyed to its step numbers"
both remain true under the amended rule); `fixed inline`/`folded` still absent from the payload per
`TestPayloadDispositionVocabulary`.

> **Round 3 correction.** The claim above that N5's drift was closed is **false as stated**, and
> so is amended decision 3's third bullet. Deleting §5's blocking category while keeping the
> blocking case in step 7 did not close the drift — it hid it, on a warrant ("§5 already carries
> this as contradicting a locked decision") that R1 below shows does not exist. The N5 verdict
> stands only in its narrow, mechanical sense: the §5 bullet is byte-identical to `main`.

---

## Scoped re-review (round 3)

Reviewer independence: **delegated** again — the orchestrating reviewer wrote the round-2 pass in
the same session. Scope: verify N1–N8, and — per this branch's established pattern and the thesis
of T-124 — audit round 2's **own replacement text** (`git diff 6f0f135 6d3aaee`) as fresh,
unreviewed prose. Every finding re-verified by hand against the rules before recording.

**Commands:** all four green at `6d3aaee`. **Five acceptance checks:** all met (numbering, class
table byte-identical, checklist 11=11, installed payload `governing` = 8, `DESIGN.md`/`AGENTS.md`
unfalsified). **Foreign-workspace test on new payload prose:** passes. `fixed inline`/`folded`
still absent. **Step 4b:** 11 suggestions, 11 quotes verified verbatim, 0 discarded; 2 target
reworked text and are folded into the fix below, which replaces those sentences.

**Prior findings:** N1, N3, N4, N5 (mechanically), N6, N7 — **resolved**. N8 — accepted as-is.
N2 — **partial** (see R3). The self-caught terminal-status fix — **resolved as to the route, not
as to the severity label** (see R2).

### Round-3 findings

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| R1 | blocking | `spec-unclear` | — | **The stated ground for the blocking case is false, leaving that case unbacked.** Step 7 says a genuine branch-vs-document disagreement is blocking "which §5 already carries as contradicting a locked decision". But §5's category demands a citation `<ID> decision <N>`, and rules §7 defines a locked decision as **one numbered item in a ticket's Implementation Plan**. A design of record or conventions file is not a ticket and has no such address — so §5 does not carry the case, and the reviewer cannot fill the citation the bullet demands. §5 is meanwhile declared severity's "single source of truth", so step 7 now declares a blocking category appearing nowhere in that source. **N5's drift was not closed; it was made invisible.** Amended decision 3's third bullet rests on the same false claim. | `review-protocol.md:325-327` vs `:260-261`, `:234-235`; `tickets-README.md` §7 *Confirmed design decisions* and `:503` | Either state the ground on its own terms and drop the §5 attribution, or — preferred — remove the blocking carve-out entirely (see the root-cause note). |
| R2 | blocking | `spec-unclear` | — | **A finding labelled blocking is routed to a follow-up ticket, which the rules say never happens.** Step 7 fixes the severity as blocking, then says that if the ticket already concluded in `6-done/` the disagreement "becomes a follow-up ticket". But a blocking finding is never dispositioned (leave the cell `—`), must move to `5-rework/`, and "the ticket does not proceed until it is [fixed]" — it already proceeded; and `new ticket` is defined only for **non-blocking** findings. The reviewer must write a row whose severity says blocking and whose disposition says both `—` and a spawned id. Round 2 fixed the illegal *transition* and left the illegal *severity label*: the contradiction moved rather than closed. | `review-protocol.md:325`, `:329-332` vs `:262-263`, `:265-266`; `tickets-README.md:401-402`, `:403-417` | Once step 6b has run, record it as **non-blocking** with a new-ticket disposition — severity governs whether *this* ticket ships, and it already has. Keep blocking only for the pre-6b path (or drop it entirely). |
| R3 | non-blocking | `spec-unclear` | folded | **N2 residual: the "why not" grounds are still unscoped.** Three grounds are offered with no qualifier, including "the reconciliation belongs to a ticket that owns the document". A reviewer facing a genuine disagreement about a document another ticket owns has one paragraph saying blocking and another saying write a note and finish. N2's own suggested fix — scope the grounds to the non-blocking case — was never applied; the fix record argued the default change made it moot, which it does only for the lag case. | `review-protocol.md:334-337` vs `:325-327` | Apply N2's original fix: scope the grounds explicitly to the non-blocking case. |
| R4 | non-blocking | `spec-unclear` | folded | Step 4's hand-off says findings are "**dispositioned** before the ticket moves", but blocking findings are never dispositioned. | `review-protocol.md:160-161` vs `:262-263` | Read "classified" or "given a severity". |
| R5 | non-blocking | `spec-unclear` | folded | **Out-of-reach × genuine-disagreement precedence is unstated.** Both clauses match a disagreement about a document in another team's repository, and they give opposite answers. Structure implies the blocking exception wins; amended decision 3's own rationale ("blocking here would strand the ticket on an edit the reviewer cannot make") argues the reverse. | `review-protocol.md:321-322` vs `:325-327` | One clause naming which test runs first. Moot if the carve-out is removed. |
| R6 | non-blocking | `spec-unclear` | folded | **The severity split turns on a distinction the payload gives no test for.** "Documentation lag" vs "genuinely disagree about what is *correct*" is the whole rule, and both are triggered by the same fact — the branch made the document false. Compare 4a.1's crisp one-liner. | `review-protocol.md:325-327` | A usable test is one sentence: if steps 2–4 already accepted the branch's behaviour as correct, it is a lag. Moot if the carve-out is removed. |
| R7 | non-blocking | `docs-gap` | folded | **Manual and CHANGELOG imply the in-reach case takes no disposition**, contrasting "recorded when it is within reach" against "dispositioned … when it is not" — while every non-blocking finding takes exactly one disposition, the in-reach one included. Otherwise both files match the payload and neither restates the full rule. | `lifecycle.adoc:129-131`, `CHANGELOG.md:24-26` vs `tickets-README.md:406-407` and `review-protocol.md:320` | Say "recorded as a disposition" in both, or drop the contrast. |
| R8 | non-blocking | `design` | noted | **The recorded disposition precedes the edit it asserts.** Classification happens at step 4/5, the in-reach edit at step 7; if step 7 then shows the document was right, the ticket is already in `6-done/` with a table asserting an inline fix that never happened. Defensible — the F1 fix deliberately split classification from mechanics — but worth a clause. | `review-protocol.md:161`, `:302-303` vs `:319-320` | Optional clause, or accept. |

**Disposition summary:** 2 blocking (R1, R2 — round-3 rework scope) · 5 folded into that pass (R3, R4, R5, R6, R7) · 1 noted (R8) · 0 new tickets. The 9 readability suggestions against untouched prose are **noted**.

cost: estimated S, actual XL — revised up from L. Three review rounds, each finding its blocking
defect inside the previous round's own fix text, on a ticket graded S at refinement.

### Rework fix record — round 3 (commit `b33b6ad`)

Implements decision 3's **second amendment** (user sign-off, recorded in History): the blocking
carve-out is **removed**, not re-argued. Line references are to
`skill/resources/review-protocol.md` at `b33b6ad`.

- **R1 — fixed by removal.** Step 7 no longer claims any §5 warrant, because it no longer defines
  a blocking case: `:316-319` now reads "classified at step 5 on §5's ordinary terms. **This step
  defines no severity of its own**". The one genuinely blocking situation — the document is right
  and the branch is wrong — is pointed at §5's **"ships wrong behaviour"** (`:329-331`), which is
  a real category (`review-protocol.md:260`, `tickets-README.md:398`) carrying **no** citation
  format to fill, unlike "contradicts a locked decision". Verified by reading rules §7 rather
  than assuming, which is the check round 2 skipped.
- **R2 — fixed by removal.** No finding is labelled blocking by this step, so nothing is routed
  from a blocking severity to a disposition the rules reserve for non-blocking findings. A
  reconciliation too large to perform in the review is now an ordinary non-blocking finding with a
  follow-up-ticket disposition (`:325-326`).
- **R3 — fixed by removal.** With no blocking case in the step, the "why not" grounds
  (`:333-336`) cannot discharge one. N2's original concern is closed by construction rather than
  by the qualifier it asked for.
- **R4 — fixed.** Step 4 now reads "classified before the ticket moves" (`:161`); blocking
  findings take no disposition, so the old wording was wrong for exactly the case it mattered.
- **R5, R6 — moot.** Both existed only to disambiguate the carve-out. There is no
  out-of-reach-versus-disagreement precedence question and no lag-versus-disagreement test to
  supply, because severity no longer turns on either distinction.
- **R7 — fixed.** `lifecycle.adoc:129-131` and `CHANGELOG.md:24-28` no longer contrast "recorded"
  against "dispositioned"; both now say the finding takes a severity and a disposition like any
  other, and both state that the step adds no severity rule of its own.
- **R8 — noted, unchanged**, as recorded at review.

**Cases walked end to end before committing**, since three rounds have shown that reading the new
prose in isolation is what misses the defect:

| case | route | legal? |
|---|---|---|
| stale doc, in reach | step 4 → §5 non-blocking → 6b → step 7 reconciles, edit recorded | yes |
| stale doc, out of reach | step 4 → §5 non-blocking, reach limit recorded → 6b | yes |
| reconciliation too large | §5 non-blocking, follow-up-ticket disposition | yes |
| document right, branch wrong | §5 "ships wrong behaviour" → blocking → 6a → `5-rework/` | yes — `4-in-review` → `5-rework` is a declared transition |
| first noticed at step 7 | §5's ordinary terms; non-blocking is compatible with the ticket already being in `6-done/` | yes |

**One boundary deliberately left as it was found.** A *blocking* finding first discovered at
step 7, 8 or 9 — after step 6b has moved the ticket to the terminal `6-done/` — has no defined
route in this protocol, and never did: step 8 has the same shape, framing everything it turns up
as a disposition. This round's text does **not** assert a route for it (that was R2's error); it
defers to §5 and says nothing more. Closing that hole properly means editing the generic protocol
for every finding type, not just this one, which is outside this ticket's scope. Recorded here so
the next reviewer sees it was considered rather than missed.

**Verified after the fix:** four commands green; five acceptance checks re-verified (no
renumbering; class table byte-identical to `main`; checklist 11=11; installed `pickle-test`
payload greps `governing` = 8; `DESIGN.md`/`AGENTS.md` unfalsified — `DESIGN.md:53`'s "blocking vs
non-blocking" is if anything more true now that the step defers to §5). §5's blocking bullet
confirmed **byte-identical to `main`** by `diff`, this time with the category genuinely absent
rather than relocated. `fixed inline`/`folded` still absent from the payload.

---

## Scoped re-review (round 4)

Reviewer independence: **delegated** (round-3 author = orchestrating reviewer). Scope: verify
R1–R8 and audit `git diff 6d3aaee b33b6ad` as fresh prose. Round 3 net-**removed** text, so the
brief targeted what removal orphans rather than what addition contradicts. Every finding
re-verified by hand.

**Commands:** four green at `b33b6ad`. **Five acceptance checks:** all met. **Foreign-workspace
test:** passes. `fixed inline`/`folded` absent. §5 blocking bullet still byte-identical to `main`.
**Step 4b:** 14 suggestions, 14 quotes verified verbatim, 0 discarded; 3 target reworked text and
one of them (A8) is a real defect independently matching F4's area — folded below.

**Prior findings:** R1 resolved · **R2 partial — the illegal disposition is gone and an illegal
transition is back (F1)** · R3 resolved (but F2) · R4 resolved · R5, R6 moot as claimed · R7 fixed
but the replacement wording is wrong (F4) · R8 unchanged by design.

### Round-4 findings

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | blocking | `spec-unclear` | — | **Step 7 asserts a route the state machine forbids, and the round-3 fix record's explicit denial of exactly this is false.** The text says a document-right/branch-wrong finding "takes the **ordinary route**". The ordinary route for a blocking finding is step 6a → `5-rework/`, reachable only from `4-in-review/`. Step 7 runs *after* step 6, so on the 6b path the ticket is in `6-done/` — terminal, with no outbound transition declared. Round 3 also deleted both previously-defined routes for this case and replaced them with a phrase pointing at a forbidden move. My round-3 fix record claims "This round's text does **not** assert a route for it … it defers to §5 and says nothing more" — contradicted by the words "takes the ordinary route", and its case-table row verifies a transition from a status the ticket is not in at step 7. **Fourth iteration of one pattern:** round 2 shipped a legal transition with an illegal severity label; round 3 shipped a legal severity label with an illegal transition. | `review-protocol.md:329-331` vs `:264-266`; `internal/flow/brine.go:129-150` (no `From: "6-done"`), comment `:38`; round-3 fix record | Say nothing about routing. The branch-wrong case is §5's business, found in steps 2–3; step 7 mentioning it has now failed twice. State what *kind* of defect it is and stop. |
| F2 | non-blocking | `spec-unclear` | folded | **A fourth ground escapes the exhaustive grounds list.** "A reconciliation too large to make here becomes a follow-up ticket" is offered at `:325`, while `:333-335` says "The legitimate grounds **are**" and names three, and a review leaving the document otherwise "has not finished". Size is also grammatically buried inside the *out-of-reach* sentence, so it is unclear whether it applies in reach — the case it exists for. | `review-protocol.md:325` vs `:333-335` | Move size into the grounds list, or drop the clause. |
| F3 | non-blocking | `spec-unclear` | folded | **Bare "§5" is ambiguous and the two §5s differ.** Four bare uses, all introduced by this branch; every pre-existing use says "**the rules** §5". The protocol's own §5 blocking list has four items (including 4a.1); the rules' has three. Step 7's whole warrant is now "§5's existing categories already settle the case" — which list is load-bearing is unstated. Neither list is declared closed (only the *dispositions* are exhaustive), so "settle" leans on a complement reading the text never authorises. | `review-protocol.md:317-318`, `:325`, `:330` vs `:24`, `:234`, `:268`, `:323`, `:347`; `:260-261` vs `tickets-README.md:398` | Qualify to "the rules §5" throughout, and replace the sweeping "settle the case" with the checkable claim: a stale governing document matches none of its blocking categories. |
| F4 | non-blocking | `docs-gap` | folded | **R7's replacement contradicts the same page seven lines up.** Manual and CHANGELOG now say the finding "takes a severity **and a disposition** like any other finding" — but a blocking finding takes none, as `lifecycle.adoc:111-112` itself says. True only on the non-blocking branch, which the same sentence declines to guarantee. Compounded by a clipped subject introduced the same round: "**One** the branch made false is classed …" (independently flagged by the step-4b reviewer as A8, quote verified). | `lifecycle.adoc:128-130`, `CHANGELOG.md:24` vs `lifecycle.adoc:111-112`, `tickets-README.md:401` | "handled like any other finding"; restore the subject to "A governing document the branch made false". |
| F5 | non-blocking | `stale-xref` | fixed inline | **The plan's own Task 1 is falsified by its twice-amended decision 3.** Task 1's last bullet still directs the implementer to "state the severity rule (decision 3) … in one sentence each", which the amended decision now forbids. Bookkeeping only; no ticket. | plan Task 1, line 192 of the ticket, vs amended decision 3 | Corrected in this review; noted in the fix record. |

**Disposition summary:** 1 blocking (F1) · 3 folded into the round-4 pass (F2, F3, F4) · 1 fixed
inline (F5) · 0 new tickets. The 11 readability suggestions against untouched prose are **noted**.

cost: estimated S, actual XL (unchanged from round 3 — already at the top of the scale).

### Rework fix record — round 4 (commit `42b67b4`)

Line references to `skill/resources/review-protocol.md` at `42b67b4`.

- **F1 — fixed by deletion, not rephrasing.** The document-right/branch-wrong sentence is gone.
  Step 7 (`:316-337`) now states only the reconciliation duty for a document lagging the branch;
  it asserts no route for any other case. The branch-being-wrong case is ordinary rules §5
  territory found (if at all) during steps 2–3, before step 6 runs, so step 7 saying nothing
  about it is accurate rather than silent-on-a-gap. **One boundary knowingly left open**, same as
  round 3 recorded: a *blocking* finding of any kind, first surfacing after step 6b, has no
  defined route in the generic protocol. That is pre-existing and out of this ticket's scope;
  this round changes nothing about it, and, unlike round 3, does not claim otherwise.
- **F2 — fixed.** "too large" is now a fourth item in the grounds list (`:330-333`) rather than
  buried in the out-of-reach sentence, and applies regardless of reach.
- **F3 — fixed.** Every bare `§5` in the section now reads "the rules §5" (`:317`, `:319`,
  `:325`, `:327`, `:333`). "Already settle the case" replaced with the checkable claim at
  `:318-320`: a document merely lagging the branch matches none of the rules §5's three blocking
  categories (`tickets-README.md:398`) — golden path, wrong behaviour, or a *ticket's* locked
  decision.
- **F4 — fixed.** `lifecycle.adoc` and `CHANGELOG.md` now say the finding is "handled like any
  other finding" rather than claiming it always "takes a severity and a disposition" (a
  blocking finding takes none). Clipped subject restored to "A governing document the branch
  made false".
- **F5 — already fixed** in this ticket's own plan during round-4 review recording.

**Verified after the fix:** four commands green; five acceptance checks re-verified (no
renumbering; class table byte-identical; checklist 11=11; installed `pickle-test` payload greps
`governing` = 8; `DESIGN.md`/`AGENTS.md` unfalsified). `§5`'s blocking bullet still byte-identical
to `main`. `fixed inline`/`folded` still absent. Every case re-walked once more: stale/in-reach,
stale/out-of-reach, too-large, first-noticed-here — all legal and unchanged in outcome; the
branch-wrong case is simply no longer described by this step, which is the fix.

---

**Root cause — fourth occurrence, and it has narrowed to one sentence.** Rounds 1–3 each had the
blocking finding inside the *blocking carve-out*; round 3 deleted the carve-out and the blocking
finding moved into the single surviving sentence that still mentions the blocking case at all.
The invariant across four rounds: **every blocking finding has been in whatever sentence tries to
tell the reviewer what to do when the branch and the document disagree.** The non-blocking
default has never once been at fault. The remedy the evidence points at is not a better sentence
— three have failed — but no sentence: step 7 states the reconciliation duty and stops, and the
branch-wrong case is left entirely to §5, where steps 2–3 already find it and where it needs no
mention here to work.

---

**Root cause — third occurrence of one pattern.** Every round's blocking finding has been in the
**blocking carve-out**, never in the non-blocking default:

- round 1 (F1): blocking severity minted at step 7, after step 6 already moved the ticket;
- round 2 (N1, N2): the disposition recommendation attached to the case where the reviewer cannot
  act, and the "why not" grounds could discharge the blocking case;
- round 3 (R1, R2): the carve-out's §5 warrant does not exist, and a blocking finding is routed to
  a disposition the rules reserve for non-blocking ones.

The non-blocking default has survived all three rounds untouched. The evidence says the carve-out
itself is the defect: **step 7 should not define severity at all.** §5 already owns severity, and a
branch that genuinely ships wrong behaviour is already blocking under "ships wrong behaviour"
without step 7 minting anything. Removing the carve-out makes R1, R2, R5 and R6 vanish rather than
be patched, needs no new §5 category, and leaves amended decision 3's first two bullets exactly as
the user signed them off. It does require deleting that decision's **third** bullet — the one R1
falsifies — which is a design change and therefore the human's call.


---

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
- 2026-08-26 — plan amended inline: decision 3 replaced with user sign-off. In-reach reconciliation is now non-blocking and done during the review (the rules §5 disposition for prose this branch made false); blocking is reserved for a genuine branch-vs-document disagreement, which §5's existing "contradicts a locked decision" already carries. Reason: two review rounds (F1, then N1/N2) showed the original in-reach → blocking split could not be stated without contradicting either the disposition vocabulary or its own "why not" grounds. Closes N5's severity-list drift as a side effect.
- 2026-08-25 — REWORK → IN REVIEW: N1/N2 fixed under amended decision 3
- 2026-08-25 — IN REVIEW → REWORK: 2 blocking (R1, R2): blocking carve-out has no §5 warrant and routes to a forbidden disposition
- 2026-08-26 — plan amended inline (second time): decision 3's blocking carve-out removed with user sign-off. Step 7 now states a duty and its mechanics and defines no severity; §5's existing categories settle every case (a stale governing document is non-blocking under them; a branch that is actually wrong is blocking under "ships wrong behaviour"). Reason: all three review rounds put their blocking findings inside the carve-out and none in the non-blocking default, and R1 showed the carve-out's §5 warrant did not exist — rules §7 scopes "locked decision" to a numbered item in a ticket's Implementation Plan, which no governing document has. Removing it makes R1, R2, R5 and R6 moot rather than patched.
- 2026-08-25 — REWORK → IN REVIEW: R1/R2 fixed by removing the blocking carve-out
- 2026-08-25 — IN REVIEW → REWORK: 1 blocking (F1): step 7 asserts a route the state machine forbids
- 2026-08-25 — REWORK → IN REVIEW: F1 fixed by deleting the branch-wrong sentence
