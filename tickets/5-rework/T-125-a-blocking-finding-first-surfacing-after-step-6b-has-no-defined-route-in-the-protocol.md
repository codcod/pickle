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
moved to `6-done/` has a defined, legal next step instead of a dead end: the finding is filed as
its own ticket, `spawned-by` the concluded one, with a dated `## History` line on the concluded
ticket recording the filed id. The reviewer stops having to invent an answer at the point where the state machine
offers none, and steps 8 and 9 stop implying that a blocking finding can be "dispositioned" like a
non-blocking one.

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

**Couplings.** Sibling of T-124 (which widened what a *scoped re-review* must check). Both came
from the same review series and both touch `resources/review-protocol.md`, but are independent:
T-124 was about re-review scope, this one is about step ordering. T-124 is already merged to
`6-done/`, so there is no rebase risk left to track — noted resolved at pickup (applicability gate).

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .
git checkout main
git checkout -b feat/T-125-late-blocking-finding-route
```

Root-path child (`project: pickle`), so tidy WIP commits into atomic ones before presenting
(Finish, below) and default to keeping that history rather than squashing.

### Prerequisite gate (hard)

None. No `depends-on:`. Verified at refinement that no other to-do/ready/in-development/
in-review/rework ticket touches `resources/review-protocol.md` or `resources/tickets-README.md`,
so no rebase risk.

### Confirmed design decisions (do not deviate without asking)

1. **The fix adds a route, not a severity rule, and touches the fewest sentences that make the
   route reachable.** T-123's own history (five review rounds, all four blocking-finding clusters
   inside whatever sentence tried to say what happens to a blocking case) is the direct warning:
   every extra clause is another chance to ship a plausible, unexecutable rule. The new text says
   only "this got filed as its own ticket instead" and stops.
2. **Structural home: a third bold lead-in paragraph, `6c`, inside the existing `## 6. Move the
   ticket` section — a sibling of `6a`/`6b`, not a new `## ` heading.** Review addenda are keyed
   to `## ` step numbers (the protocol's intro); adding a heading here would risk exactly the
   renumbering hazard T-123's decision 1 guarded against, even though `6c` itself introduces no
   heading. Step 7 gets **no edit** — its post-T-123 text already asserts no route or severity for
   anything, and stays correct once 6c exists elsewhere. Steps 8 and 9 each gain one clause
   redirecting a *blocking* finding about the ticket this same review pass just concluded to 6c,
   since their current wording ("takes a disposition", never mentions a finding at all) only fits
   a non-blocking one.
3. **Scope is exactly the ticket this review pass just moved to `6-done/` via its own step 6b** —
   not any already-done ticket a sweep happens to touch. That is the literal shape of the gap this
   ticket names ("first surfacing after step 6b"); a wider claim is a different problem and stays
   out of scope.
4. **The concluded ticket gets a dated `## History` line only — its closed `## Review` table is
   not touched.** Reopening/annotating an archived findings table to carry a route it still can't
   take is the same shape of trap the carve-out in T-123 fell into; a plain pointer line carries
   the provenance without it.
5. **The ticket is filed unconditionally, not run through the non-blocking promotion
   test.** "Would this actually be scheduled?" and the `noted`/`folded` defaults exist for
   non-blocking findings (rules §5); a blocking one has no discard option once `6-done/` has
   already run. It is graded normally (rules §3) and carries `spawned-by:` naming the concluded
   ticket — lineage only, per rules §3, never gating. If more than one such late-blocking finding
   surfaces in the same review pass, batch by theme into one ticket rather than one each, mirroring
   the flow's existing batching principle even though this route is not literally one of the four
   dispositions.
6. **`tickets-README.md` §5's Blocking bullet gets one matching sentence pointing at
   `review-protocol.md` §6c**, so the rule's two statements (protocol mechanics, rules summary)
   do not drift apart the way T-123's Description warned against for its own rule.
7. **Payload prose stays foreign-workspace-safe:** no ticket id a reader is told to look up, no
   pickle-source-only path, no first-person "this repo", no count or claim drawn from a corpus the
   reader doesn't have. `just test` runs `payload_lint_test.go`, which checks the mechanical part
   of this.

### Tasks

#### Task 1 — add step 6c
In `skill/resources/review-protocol.md`, inside `## 6. Move the ticket` (`:309`), immediately
after the `6b` paragraph (`:318-320`), add a third bold lead-in paragraph:

> **6c. A blocking finding first identified at step 7, 8 or 9 — after 6b has already moved the
> ticket to the terminal `6-done/`.** `6-done/` declares no outbound transition, so the finding
> cannot take the `5-rework/` route above, and it is not dispositioned either — the rules §5's
> four dispositions, `new ticket` included, are defined only for non-blocking findings. **It is
> filed as its own ticket instead**: `pickle ticket new … --spawned-by "T-NNN"` (this skill's
> `resources/TEMPLATE.md`, graded per the rules §3) — the same command the `new ticket`
> disposition uses, but not that disposition, since a blocking finding is never dispositioned.
> Append a dated `## History` line to the concluded ticket recording the filed id — the archive
> stays terminal, but the pointer is not silent. Steps 7, 8 and 9 below take this route for a
> blocking finding about the ticket that just moved; their own text otherwise covers the
> non-blocking case.
>
> **Correction during implementation (not part of the original refined plan):** the first draft
> of this paragraph called the spawned ticket a "follow-up ticket" to avoid colliding with the
> rules §5 `new ticket` disposition token — but "follow-up ticket" turned out to already be this
> same file's own established prose name for what that disposition produces (`:284`, `:360`).
> Renamed to "filed as its own ticket" throughout, which collides with neither.

Do not renumber anything else in the file.

#### Task 2 — redirect step 8's blocking case
In the same file's `## 8.` section (`:356-362`), the closing sentence currently reads "anything it
turns up that is not a patch takes a disposition per the rules §5, and a new ticket needs the
promotion test." Add one sentence after it: a blocking finding about the ticket that just
concluded, rather than about a dependent ticket, takes step 6c's route instead of a disposition.

#### Task 3 — redirect step 9's blocking case
In the same file's `## 9.` section (`:364-367`), add one leading bullet before "Summarize: …":
a blocking finding noticed only now — writing the summary is often when the whole change is
finally seen at once — takes step 6c's route before anything below is presented.

#### Task 4 — matching line in the rules
In `skill/resources/tickets-README.md` §5's Blocking bullet (`:398-404`), append one sentence:
first identified after the ticket has already moved to `6-done/`, this route no longer exists, so
the finding is filed as its own ticket instead (not the `new ticket` disposition — a blocking
finding is never dispositioned), `spawned-by` the concluded one, with a dated `## History` line on
the concluded ticket recording the filed id (`resources/review-protocol.md` §6c).

#### Task 5 — the manual
In `docs/user-manual/concepts/lifecycle.adoc`, in the *Reviews: severity, then disposition*
section, after the paragraph on reconciling governing documents (around `:130-136`), add a short
paragraph: a blocking finding discovered only after a review already moved the ticket to
`6-done/` — while updating references, sweeping dependent tickets, or writing the summary — has no
route back into that ticket, since the status is terminal; it is filed as its own ticket instead
(not the `new ticket` disposition), `spawned-by:` the concluded one, with a `## History` line on
the concluded ticket recording the filed id. Point at the skill's `resources/review-protocol.md`
step 6c for the full rule; do not restate the mechanics.

#### Task 6 — CHANGELOG entry
Add an `[Unreleased]` entry in `CHANGELOG.md` in the established style (bold lead sentence, then
the qualification, ticket id in trailing parens), naming `resources/review-protocol.md` (step 6c,
and the steps-8/9 redirect clauses) and `resources/tickets-README.md` §5 as the changed payload,
and noting every installed project picks the rule up on its next `pickle upgrade`.

### Acceptance test

```
just build
just test
just lint
just docs-check
```

All clean. `just test` runs `payload_lint_test.go` over the embedded payload (task 7's foreign-
workspace check) and `TestPayloadDispositionVocabulary`; `just docs-check` runs the xref tests over
the edited manual page.

Then, specifically:

1. **No step was renumbered** —
   `grep -n '^## [0-9]' skill/resources/review-protocol.md` shows `## 0.` through `## 9.` with
   `## 4a.`/`## 4b.` present, identical to `main`
   (`git show main:skill/resources/review-protocol.md | grep -n '^## [0-9]'`); `6c` appears only
   as a bold lead-in inside `## 6.`, not as its own heading.
2. **The class vocabulary and disposition list are untouched** —
   `git diff main -- skill/resources/review-protocol.md skill/resources/tickets-README.md` shows
   no added, removed or re-ordered row inside either closed-vocabulary table.
3. **Every case walked end to end, before committing** (T-123's round-3 practice, adopted here
   because the same file broke on plausible-but-wrong prose four times):

   | case | route | legal? |
   |---|---|---|
   | blocking, found at step 4 (before move) | 6a → `5-rework/` | yes — unaffected |
   | non-blocking governing-doc finding, found at step 7 | step 5 disposition, per existing text | yes — unaffected |
   | blocking finding about a dependent ticket, found at step 8 | patch the dependent per existing text | yes — unaffected |
   | blocking finding about the just-concluded ticket, found at step 7, 8, or 9 | 6c → filed as its own ticket, `spawned-by`, History line on the concluded ticket | yes — no transition out of `6-done/` asserted |

4. **The rule ships, not just exists locally** — install into a throwaway dir per `AGENTS.md`'s
   self-modify policy:
   ```
   D=$(mktemp -d) && cp pickle "$D/pickle-test" && cd "$D" && ./pickle-test install --in-tree --project demo
   grep -c '6c' .agents/skills/brine/resources/review-protocol.md   # expect >= 1
   ```
5. **Foreign-workspace test on the new prose** — re-read the added sentences against `AGENTS.md`'s
   four shapes; none names a ticket id to look up, a pickle-source-only path, a first-person
   "this repo", or an invisible corpus count.

### Docs update (mandatory when user-facing)

User-facing: `docs/user-manual/concepts/lifecycle.adoc` gains the short paragraph (Task 5) and
`CHANGELOG.md` gains an `[Unreleased]` entry (Task 6). Run the docs-readability reviewer over both
changed files during review (protocol step 4b).

### Finish (mandatory)

1. Acceptance test green, including the four specific checks above.
2. Docs updated (Tasks 5 and 6) and `just docs-check` clean.
3. Write a summary: the files touched, the exact 6c text as shipped, confirmation that no step was
   renumbered and neither closed vocabulary changed, and the result of walking the case table
   above against the shipped text.
4. Suggested commit message:
   ```
   feat(skill): route a late-surfacing blocking finding to its own ticket (T-125)
   ```
5. Tidy WIP commits into atomic ones (root-path child) before presenting.
6. Commit locally; do not push or open an MR without explicit user approval. Hand back with
   `pickle ticket move T-125 in-review --reason "acceptance green"`.

## Review

### Checklist

- [x] Reviewer independence settled (step 0): **delegated** — the reviewing agent authored the branch in this same session, so audits 2–4a ran in a freshly spawned, adversarially briefed sub-agent with no memory of writing the code; classification, severity, dispositions and the move stayed with the orchestrating reviewer, and the delegated findings were re-verified by hand before entering the table below
- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4) — every case walked by hand against the shipped text, independently of the ticket's own case table
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass on the ticket's changed `.adoc`/`.md` files — run over all four (`review-protocol.md`, `tickets-README.md`, `lifecycle.adoc`, `CHANGELOG.md`); 12 suggestions, all against prose this branch never touched, 0 targeting the branch's own new text (step 4b)
- [x] Findings recorded with severity, class and disposition (step 5)
- [x] Ticket moved to `tickets/5-rework/`; `## History` appended (step 6a)
- [ ] Other references / governing documents — not applicable; this ticket touches no governing document
- [x] Remaining-tickets impact sweep done (step 8) — no ticket in `2-ready/`/`1-to-do/` lists T-125 in `depends-on:` or references it
- [ ] Summary + commit message presented for approval (step 9) — not applicable on a rework verdict; nothing is published

### Commands

`just build` · `just test` · `just lint` · `just docs-check` — **all four green**, re-run independently by the delegated reviewer at commit `471fcbd` and again by the orchestrating reviewer after the F2 inline fix at `0fa89d3`. `just test` includes `payload_lint_test.go` and `go test . -run TestPayloadSpeaksToAForeignReader -v -count=1` (uncached), both green. Throwaway install as `pickle-test` (`AGENTS.md`'s self-modify policy): `grep -c '6c' .agents/skills/brine/resources/review-protocol.md` = 3 (≥ 1 required).

### Findings

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | blocking | correctness | — | **Confirmed design decision 5's batching clause never shipped.** Decision 5 says a review pass surfacing more than one late-blocking finding about the concluded ticket must batch them by theme into one filed ticket, mirroring the flow's existing batching principle. No occurrence of "batch"/"theme" exists anywhere in the shipped 6c text, the step 8/9 redirect clauses, or the `tickets-README.md` §5 addition — only the two pre-existing, unrelated uses (for the rules §5 `new ticket` disposition) survive untouched. A reviewer following only the shipped rule with two unrelated late-blocking findings in one pass has no batching instruction and would file two, contradicting the plan's own confirmed decision — the "contradicts a locked decision" ground the rules' own severity test names as blocking. | plan decision 5 (T-125 decision 5) vs `grep -n batch skill/resources/review-protocol.md skill/resources/tickets-README.md` (only pre-existing `review-protocol.md:285` and `tickets-README.md:321`, both about the `new ticket` disposition, neither touched by this branch) | Add one clause to 6c (or the redirect text) stating that more than one such finding in the same review pass is filed as one ticket, batched by theme, per the same principle rules §5 already states for the `new ticket` disposition. |
| F2 | non-blocking | other | fixed inline | Dangling modifier: "First identified after the ticket has already moved to `6-done/`, this route no longer exists…" reads as modifying "this route" (which cannot be "identified"), when it means the *finding*. Cosmetic, no behaviour change. | `skill/resources/tickets-README.md` (pre-fix, committed `471fcbd`) | Fixed by this review at `0fa89d3`: reworded to "A blocking finding first identified after the ticket has already moved to `6-done/` cannot take this route — it is filed as its own ticket instead…", giving the participial phrase an unambiguous subject. |

**Disposition summary:** 1 blocking (F1 — defines the rework scope) · 1 fixed inline (F2) · 0 new tickets. The delegated audit also proposed a second non-blocking finding (F3, a corroborating docs-readability claim against the new `lifecycle.adoc` paragraph) that did not survive hand-verification: the orchestrating reviewer's own `docs_readability` run returned zero suggestions against any of this branch's new prose in `lifecycle.adoc` or `CHANGELOG.md` (its 12 suggestions all target paragraphs this branch never touched) — discarded as unverified rather than recorded, per step 0's "delegation buys independence, not accuracy."

cost: estimated S, actual S — one round so far; F1 is a small, well-scoped addition (one clause) and does not itself change that estimate.

### Rework fix record — round 1 (commit `4cf190a`)

- **F1 — fixed.** Added one sentence to 6c, immediately after the History-line instruction and
  before the "steps 7, 8 and 9 below take this route" close: "More than one such finding in the
  same review pass is **batched by theme** into a single filed ticket — one ticket carrying
  several findings rather than one per finding — the same principle the rules §5 `new ticket`
  disposition already applies." (`skill/resources/review-protocol.md:330-333`). No new clause
  added to the step 8/9 redirects or to `tickets-README.md` §5 — 6c is the rule's single home, so
  batching lives there once rather than being restated at every call site, consistent with
  decision 1's "fewest sentences" discipline and decision 6's "one statement, not two that can
  drift."

**Verified after the fix:** `just build`, `just test` (including `payload_lint_test.go` and an
uncached `go test . -run TestPayloadSpeaksToAForeignReader -v -count=1`), `just lint`,
`just docs-check` all green at `4cf190a`. No step renumbered; both closed vocabularies
(`class`, disposition) still byte-identical to `main`. Throwaway install as `pickle-test`:
`grep -c '6c'` = 3 (≥ 1); the batching sentence itself greps present in the installed payload.
Every case from the review's own walk-through re-checked against the new sentence — it fires
only inside 6c's existing scope (a blocking finding about the ticket this same review pass just
concluded), so it cannot reach the pre-6b, non-blocking, or dependent-ticket cases the review
confirmed unaffected. Re-read the replacement sentence once more before handing back: it does not
name or imply a route out of `6-done/`, and "filed ticket" is used, not "follow-up ticket" or
"new ticket", preserving the vocabulary separation the review's F2 fix and the ticket's own
second implementation-time correction both depended on.

> **SHA note.** This record cites `4cf190a`; a rebase onto `main` immediately after the fix
> rewrote it to `84810d5`. The old object still resolves but is no longer an ancestor of the
> branch. Content verified byte-identical apart from the SHA line. Recorded per §1's
> broken-record clause rather than silently corrected, and noted as R5 below — the rework
> procedure says not to tidy during rework precisely because it rewrites these SHAs, and
> rebasing to stay current is a tidy.

---

## Scoped re-review (round 2)

Reviewer independence: **delegated** — the orchestrating reviewer wrote the round-1 rework in this
same session. Scope per §1: verify F1, plus read the diff that closed it. That diff is **two**
commits, not one: `84810d5` (the F1 rework fix) and `c3dd85c` (the F2 inline fix, written *during*
round 1's review after the delegated audit had already finished, so no reviewer had ever read it).
Range audited: `5d336e3..84810d5`.

**Commands:** `just build` · `just test` · `just lint` · `just docs-check` — all four green at
`84810d5`, re-run independently. `TestPayloadSpeaksToAForeignReader` PASS (uncached). Throwaway
install as `pickle-test`: `grep -c '6c'` = 3, batching sentence present in the installed payload.
**Step 4b:** 13 suggestions, **0** against either commit's new text — all target prose this branch
never touched; **noted**.

**F1 — resolved.** The batching sentence reproduces decision 5's substance, and placing it only at
6c is defensible: steps 8 and 9 both *redirect* to 6c rather than restating the route, so a reader
arriving by either path lands on it. **F2 — resolved in substance, but its replacement text
introduced R1** (below), which is the precise hazard §1 names: a fix's own replacement text is the
one part of the branch nothing has audited.

### Round-2 findings

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| R1 | non-blocking | other | fixed inline | The F2 inline fix left two em-dashes reading as a matched pair, so the trailing "`spawned-by` the concluded one, with a dated `## History` line…" attached to "cannot take this route" instead of to "is filed as its own ticket". Introduced by the previous round's own fix. | `tickets-README.md` §5 Blocking bullet at `c3dd85c` | Fixed at `cc13850`: recast the aside as a parenthetical so the trailing clauses attach to the right verb. |
| R2 | non-blocking | spec-unclear | fixed inline | 6c claimed batching was "the same principle the rules §5 `new ticket` disposition already applies", but §5's rule is "One new ticket per *theme*" (several tickets when themes differ) while 6c said "a single filed ticket" regardless — a false identity claim, and the sentence's two halves disagreed for the 2-themes case. | `review-protocol.md:330-333` vs `tickets-README.md:440-441` | Fixed at `cc13850`: dropped "a single filed ticket", so by-theme genuinely *is* §5's mechanism and the claim became true. |
| R3 | non-blocking | spec-unclear | fixed inline | Batching timing was undefined for the multi-step case, which is 6c's primary case: a reviewer filing at step 8 and finding a second at step 9 had no stated rule. | `review-protocol.md:330-333` vs `:377-378`, `:383-384` | Fixed at `cc13850` by adding a join clause — **which introduced V1 below.** |
| R4 | non-blocking | docs-gap | fixed inline | `tickets-README.md` §5 restated 6c's mechanics inline (filed ticket, `spawned-by`, History line) but omitted batching, so a rules-only reader would file one per finding — asymmetric with the same file's non-blocking path, which does state batching. | `tickets-README.md:405-409`, 0 occurrences of "batch" | Fixed at `cc13850`: added "and batched by theme when one pass turns up more than one". |
| R5 | non-blocking | stale-xref | noted | The round-1 fix record cites `4cf190a`, which a post-rework rebase rewrote to `84810d5`; the old object resolves but is not on the branch. Content verified byte-identical. §1 makes a broken fix record a finding of its own, never blocking. | fix record vs `git merge-base --is-ancestor 4cf190a HEAD` → false | Recorded in the SHA note above. The deeper fix is procedural, not textual: the rework procedure says not to tidy during rework *because* it rewrites these SHAs — and rebasing to stay current is a tidy. |

### Verification of the inline fixes — second delegated pass

The four `fixed inline` repairs above were themselves unaudited replacement text, and a
non-blocking verdict means no rework round would ever read them. Rather than ship them unread
— or inflate their severity to force an audit — they were committed at `cc13850` and handed to a
second, independently briefed reviewer before concluding. That pass found:

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| V1 | **blocking** | correctness | — | **The clause added to close R3 resurrects the exact contradiction R2 was raised for.** "a second found later in the same pass joins the ticket already filed rather than starting another" is **unconditional**: for a second finding of a *different* theme, "batched by theme" (and the rules §5's "One new ticket per *theme*", and decision 5's "batch by theme") say file a second ticket, while the join clause says do not. Two answers for one real case, and it re-falsifies the "exactly as the rules §5 batches" claim that R2's fix had just made true. Contradicts locked decision 5. | `review-protocol.md:330-333` vs `tickets-README.md:440-441`; T-125 decision 5 | Scope the clause to the same theme: "…a second **of the same theme** found later in the same pass joins the ticket already filed rather than starting another." |
| V2 | non-blocking | spec-unclear | folded | The History instruction is singular ("recording the filed id") and sits *before* the batching sentence, so it is silent on both batched cases: two themes → two ids, and join-an-existing → id already recorded. Coherent only by charitable default. | `review-protocol.md:329` vs `:330-333` | Folded into the V1 rework pass, which rewrites that sentence anyway: state one line per filed id, and none for a finding joining an already-recorded ticket. |

**Disposition summary:** 1 blocking (V1 — defines the round-2 rework scope) · 4 fixed inline (R1–R4) · 1 folded (V2) · 1 noted (R5) · 0 new tickets. The 13 step-4b suggestions, all against untouched prose, are **noted**.

cost: estimated S, actual M — revised up from S. Two review rounds, the second finding a blocking
defect inside the first round's own fix text, on a one-clause change.

### Root cause — fifth occurrence, and this time inside a review's own inline fix

Every blocking finding in this ticket family has been in **whatever sentence tries to tell the
reviewer what to do when there is more than one way to handle a blocking case** — four times in
T-123's carve-out, and now once here, in the batching clause. The new datum is *where* it was
introduced: not by the implementer, and not by a rework pass, but by the **reviewer, fixing a
non-blocking finding inline during the review itself**. R3 was a genuine gap; the clause written
to close it re-opened R2. The pattern is not carelessness, it is that each fix is written with the
finding it closes in view and the neighbouring rule out of view.

Two things follow, and only the first is this ticket's business:

1. **The V1 fix must be checked against the by-theme rule and §5 together, not against R3 alone.**
   The suggested two-word scope ("of the same theme") does that; the rework pass should re-read
   the whole sentence as one rule afterwards, not just the words it changed.
2. **A review that fixes findings inline has no audit step of its own.** This review invented one
   (the second delegated pass above) and it is what caught V1 — an unaudited inline fix would
   otherwise have shipped a rule contradicting the rules file. Whether that should become part of
   the generic protocol is a real question, but it is a change to the review protocol for *every*
   finding type, which is exactly the scope this ticket already declined once. Recorded here, not
   acted on.

### A note on this review's own routing — evidence for 6c's boundary

V1 is a blocking finding that surfaced **late in this ticket's own review**, after the delegated
audits had finished, while recording findings at step 5. Step 6 had not yet run, so the ordinary
route was available and nothing had to be invented: 6a, to `5-rework/`. That is not a 6c case, and
it should not be — 6c begins only once 6b has moved a ticket into the terminal status. This
ticket's own review therefore exercised the exact boundary the ticket draws, from the pre-6b side,
and found the existing machinery sufficient there. It is the cheapest available evidence that 6c's
scope is drawn in the right place rather than one step too wide.

## History

- 2026-08-26 — created (TO DO). source: pickle ticket new
- 2026-08-27 — TO DO → READY: plan complete
- 2026-08-27 — plan amended inline: pickup applicability gate (independent sub-agent audit) found
  0 blocking, 5 non-blocking findings. F1 fixed: the Couplings note's T-124 land-order hedge was
  stale (T-124 already merged) — reworded to note it resolved. F2 fixed: Task 5's manual sentence
  now says "the skill's `resources/review-protocol.md`", matching the adjacent paragraph's own
  phrasing. F3 fixed: every occurrence of the 6c mechanism was renamed from "new ticket" to
  "follow-up ticket" throughout the plan (Outcome, decision 1, Tasks 1/4/5, the acceptance-test
  case table, the commit message) so it can never be misread as the rules §5 `new ticket`
  disposition — the exact vocabulary-overlap shape that cost T-123 five rounds. F4 (6c is silent
  on the follow-up ticket's branch/merge-base relative to the concluded ticket's own branch) and
  F5 (T-123, a comparably-sized prose-only edit to the same file, also graded cost S and finished
  actual XL) noted — both out of this ticket's scope, recorded for the record.
- 2026-08-27 — READY → IN DEVELOPMENT: picked up; applicability gate clean, F1-F3 amended inline, F4/F5 noted
- 2026-08-27 — plan amended inline: F3's own fix (renaming the 6c mechanism from "new ticket" to
  "follow-up ticket") turned out to collide with a different pre-existing term — this same file
  already uses "follow-up ticket" as its established prose name for what the rules §5 `new
  ticket` disposition produces (`review-protocol.md:284`, `:360`; `tickets-README.md:321-324`).
  Renamed again, this time to "filed as its own ticket" / "the filed id", which collides with
  neither the disposition token nor its own prose name. Found by walking the acceptance test's
  case table against the shipped text before committing (task 3), not by a reviewer — caught
  during implementation itself, one layer earlier than the review that has caught this exact
  vocabulary-overlap shape in T-123 four times.
- 2026-08-27 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-08-27 — IN REVIEW → REWORK: 1 blocking finding (F1): decision 5's batching clause never shipped
- 2026-08-27 — REWORK → IN REVIEW: F1 fixed: batching clause added to 6c
- 2026-08-27 — IN REVIEW → REWORK: 1 blocking (V1): the R3 join clause contradicts batch-by-theme
