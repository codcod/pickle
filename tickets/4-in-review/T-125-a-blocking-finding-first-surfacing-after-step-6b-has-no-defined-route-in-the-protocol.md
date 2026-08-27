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

<!-- empty until IN REVIEW -->

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
