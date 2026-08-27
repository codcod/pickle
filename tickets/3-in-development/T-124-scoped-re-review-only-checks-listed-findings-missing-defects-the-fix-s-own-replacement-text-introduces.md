---
id: T-124
title: scoped re-review only checks listed findings, missing defects the fix's own replacement text introduces
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: low
cost: M
---

# T-124 — scoped re-review only checks listed findings, missing defects the fix's own replacement text introduces

## Outcome

After this ships, a scoped re-review checks the fix's own diff for new defects — not only
whether the listed findings are closed. A rework fix that introduces a fresh false statement,
contradiction, or gap in the text/code it just wrote is caught in the same round instead of
surfacing as a new blocking finding one or more rounds later.

## Description

`review-protocol.md` § *1. Load context* (mirrored in `tickets-README.md` § *5. Findings — severity,
then disposition*, blocking bullet) scopes a scoped re-review to: *"only the findings listed there
need re-verification; do not re-audit the whole feature from
scratch."* Taken literally, that mandate covers whether each **listed** finding is now closed —
it gives the reviewer no instruction to look at the **new** text or code the fix pass just wrote
to close it. In practice, that is exactly where the next round's blocking findings keep turning
up:

- **T-109** — round 2 found R1-R3 in surfaces round 1's fix hadn't reached; round 3 found S1
  explicitly in "the *replacement* prose from round 2 … including two of review 2's own inline
  fixes"; round 4 found T1 in "the replacement for S1". Four rounds, each finding sitting in the
  round before's own fix.
- **T-067** — rework round 2 found "2 new blocking … N1 invariant guards the pattern not the
  scanner, N2 failure message prescribes a fix that breaks the build" on the just-landed fix.
- **T-098** — a re-review found "F2 blocking (N3's fix left the provenance glosses contradicting
  their own tie-break)" — the fix *for* N3 created F2.
- **T-018** — a re-review found "new false statement + missed third copy" plus a "rework record
  overclaim" — new false statements introduced by the rework pass itself.
- **T-013** — "new blocking B4 (advisory payload diff can abort the whole upgrade)" surfaced only
  after B1-B3 were fixed.
- **T-122** — round 2 found "R1/R2 blocking: F2 residual on blockquote/list prefixes, F4 exit not
  admitted by the skip" — the first fix didn't fully close its own claim.

Six of the eight tickets that were ever reworked more than once show this same shape: the
second-or-later round's blocking finding lives in text the *previous round's fix* wrote, not in
behaviour the reviewer was told to re-check. The rework procedure itself only asks the fixer to
"fix only the listed findings" (`resources/tickets-README.md` rework procedure) — it says nothing
about auditing the new lines that fix produces, and the scoped re-review that follows has no
mandate to either.

**Proposed change.** Widen the scoped re-review's stated mandate — from *"only the findings listed
there need re-verification"* to *"the findings listed there, plus the diff that closed them"*: read
the fix's own diff and check the new text or code it introduces for the same defect classes,
blocking or not, before concluding. This stays bounded (the fix diff, not "re-audit the whole
feature") while closing the blind spot that let a fix's own replacement text ship unread.

### What refinement confirmed

- **The rule lives on more than the two surfaces the Description first named.** Verified against
  the current tree, the wording to change is: `resources/review-protocol.md` § *1. Load context*
  (the scoped re-review bullet — the authoritative statement), its restatement in that file's
  § *6. Move the ticket* (6a), the reviewer checklist at the end of the same file,
  `resources/tickets-README.md` § *5*'s blocking bullet, `SKILL.md`'s rework and validate
  procedures, and `docs/user-manual/concepts/lifecycle.adoc`'s review treatment. They restate one
  rule and must move together, or the payload ships two policies.
- **A de-facto anchor already exists; the payload just never names it.** *(Corrected by the
  applicability audit at pickup — the earlier version of this bullet claimed "nothing today anchors
  the fix diff", which is false in practice.)* Reworked tickets already head each round's fix
  record `### Rework fix record — round N (commit <sha>)`, and re-reviews already anchor against a
  SHA. What is missing is that no shipped document *asks* for it, so it is convention rather than
  duty and an older or hurried round simply omits it. Decision 3 therefore **normalises the shape
  already in use** instead of minting a second one.
- **No new severity or disposition machinery is needed.** A defect found in the fix diff is an
  ordinary finding: the rules §5 already routes it (blocking → another rework round; non-blocking →
  one of the four dispositions). The widened mandate changes *what is read*, never *what a finding
  costs*.
- **Re-graded `cost: S` → `M`.** Impact and complexity stand. The two nearest neighbours — T-122
  and T-123, both prose-only payload edits graded S — came in at actual M and XL respectively, in
  both cases because the shipped sentences needed several review rounds rather than because the
  edit was large. This ticket touches six surfaces instead of their two or three, so S is not
  defensible for it either.
- **Evidence stays out of the payload.** The six tickets cited above argue for the change; they are
  not shipped text. A reader of an installed skill cannot resolve those ids, and
  `payload_lint_test.go` rejects both that shape and the corpus counts beside it.

### Decisions taken at refinement (user-confirmed)

The five open questions this ticket carried are settled and written as numbered decisions in the
plan: the mandate widens to **the listed findings plus the diff that closed them** (1); a round
reads **its own round's fix diff only** (2); the **fix pass records the range** and the reviewer
derives it only as a fallback (3); the **fixer re-reads its own replacement text** before handing
back (4); findings from the diff take **ordinary §5 severity and disposition** (5); and the
reviewer checklist is **extended, never lengthened** (6).

**Not in scope.** `NOTES.md § "T-109 partial merge"` records a distinct, still-unfiled idea —
verifying a branch's pushed tip matches the ticket's own Review section before reporting a push
as done. That is a publish-verification gap, not a review-scope gap; this ticket does not
subsume it. T-112 (review-protocol bias-mitigation for a same-session implementer reviewing their
own ticket) is also a different axis — reviewer independence, not re-review scope — and does not
overlap this one. **Widening the state machine** is out of scope too: this ticket adds no
transition and changes no status; a blocking finding in the fix diff routes through the existing
`4-in-review/` → `5-rework/` move like any other.

### Couplings

T-125 (a blocking finding first surfacing after step 6b) edits the same file,
`skill/resources/review-protocol.md`, in different sections (§6b/§8 versus this ticket's §1/§6a).
They are independent and can land in either order; whichever lands second rebases. Both were
surfaced by the same review series.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .
git checkout main
git checkout -b feat/T-124-re-review-reads-the-fix-diff
```

Root-path child (`project: pickle`), so tidy WIP commits into atomic ones before presenting
(Finish, below) and default to keeping that history rather than squashing.

### Prerequisite gate (hard)

None. No `depends-on:`. The one soft coupling (T-125, same file, different sections) imposes no
ordering — if it lands first, rebase before starting.

### Confirmed design decisions (do not deviate without asking)

1. **The scoped re-review's mandate becomes "the listed findings, plus the diff that closed
   them".** Both halves are stated together everywhere the rule appears: the bound (*not* a
   re-audit of the whole feature) is as load-bearing as the widening, and dropping it would trade
   this ticket's blind spot for an unbounded re-review that nobody would run.
2. **A round reads its own round's fix diff — not every rework diff since the first review.** Round
   N audits the text round N's fix wrote. Each round therefore covers its predecessor's new text,
   so by induction every line the branch gained after the first review is read exactly once, at the
   round that can still act on it cheaply.
3. **The fix pass records what it committed, in the shape already in use.** *(Rewritten at the
   pickup applicability gate; the original is quoted in the amendment note under Tasks.)* The
   rework pass captures the branch tip **before its first fix commit** — not "the commit the
   previous review audited", which is a different commit whenever that review authored inline
   fixes after auditing — and records the round's commits in the ticket's `## Review` under the
   heading the flow already uses: `### Rework fix record — round N (commit <sha>)`, or
   `(commits <sha>..<sha>)` when the round produced several. Three edges the rule states rather
   than leaves to the reader:
   - **Record after the tidy, not before.** A root-path child rebases its WIP commits into atomic
     ones (Finish step 5), which rewrites the SHAs; a range recorded earlier can point at commits
     that no longer exist.
   - **A round with no commits says so** — `no commits this round — <why>` (a ticket-text-only fix,
     a deferred commit) — so the re-reviewer stops looking instead of hunting for a diff that was
     never made.
   - **The reviewer's fallback covers absent *and* unresolvable.** If the record is missing or its
     SHAs do not resolve, the re-reviewer reconstructs the range from the branch log rather than
     skipping the audit. The missing or broken record is itself worth a finding, but never a
     blocker on its own.
4. **The fixer re-reads its own replacement text before handing back.** One clause in the rework
   procedure, added to steps it already has — no new numbered step. The re-review remains the
   guarantee; this only moves the cheapest catch one pass earlier, to the agent that still has the
   change in context.
5. **No new severity, disposition or class.** A defect found in the fix diff is an ordinary finding:
   the rules §5 routes it (blocking → another rework round via the existing move; non-blocking → one
   of the four dispositions) and `review-protocol.md` §5's closed class vocabulary names it. Nothing
   in this ticket adds, renames or re-orders a value in that vocabulary, and no step gains a
   severity rule of its own.
6. **The reviewer checklist is extended, not lengthened.** The existing implementation-audit line
   gains a conditional clause; no new `- [ ]` row is added. A checklist that grows a row per rule
   stops being read.
7. **The payload states the rule on its own reasoning.** No ticket ids, no counts drawn from a
   corpus the reader cannot see, no path that resolves only in pickle's own tree — AGENTS.md's
   foreign-workspace test, whose mechanical half `payload_lint_test.go` enforces in `just test`.
8. **No step is renumbered.** Project review addenda key on this protocol's step numbers, so the
   rule is written into the steps that already exist (§1, §6a) rather than as a new one.

### Tasks

> Line numbers below were read from `main` at refinement time and are given only as a locator.
> Find each target by its heading or quoted text; if T-125 lands first, the numbers in
> `review-protocol.md` shift. Task 2's `:263-267` is **already** stale in a way that matters: it
> now lands on §5's blocking bullet, and §6a is at `:296-300` — locate that one by its quoted text
> alone.

> **Amended at the pickup applicability gate (2026-08-26), before any move or branch.** An
> independent audit found one blocking defect in decision 3 and five improvements; all were folded
> in while the ticket was still in `2-ready/`. The blocking one: decision 3 originally read *"the
> rework pass captures the branch tip (the commit the previous review audited)"* and asserted
> *"nothing today anchors the diff that fixed them"* — the parenthetical is false whenever a review
> authored inline fixes after auditing, and the assertion is false in practice, a
> `### Rework fix record — round N (commit <sha>)` convention already being in use. Decision 3 and
> the "What refinement confirmed" bullet above were rewritten accordingly, and tasks 2, 5 and 6
> below now normalise the existing shape rather than introducing `round N fix: <sha>..<sha>`.

#### Task 1 — widen the authoritative statement (`review-protocol.md` §1)
In `skill/resources/review-protocol.md`, the scoped re-review bullet under `## 1. Load context`
(`:119-121`), currently:

> - If this is a **scoped re-review** (the ticket was previously in `tickets/5-rework/`), read the
>   existing `## Review` section first — only the findings listed there need re-verification; do
>   not re-audit the whole feature from scratch.

Rewrite it to carry three things (decisions 1-3, 5), in the file's own voice:

- the re-verification scope is **the listed findings plus the diff that closed them** — the new
  text or code the fix pass wrote is read for defects of the same classes, whether or not any
  listed finding names that ground;
- **how to obtain the diff**: the range recorded by the fix pass (decision 3), with the branch-log
  fallback in one clause when it is absent;
- the **bound**, kept explicit: this is the current round's fix diff, still not a re-audit of the
  whole feature.

One short paragraph or a two-item nested list — not more; §1 is a loading step, and the routing of
whatever the read turns up already belongs to §5.

#### Task 2 — mirror the widened scope in §6a
In the same file, `## 6. Move the ticket`, 6a (`:263-267`) currently ends *"a **scoped re-review**
verifies just those findings and concludes via 6a/6b again."* Extend that clause so it names the
fix diff too (decision 1), and add — in the same sentence or one short one after it — the fix
pass's duty from decision 3: it records the commits it produced in `## Review`, under that
section's rework fix record. Keep it to a clause; §1 is the authority and 6a is the pointer.

Locate 6a by that quoted sentence: the section is at `:294`, its 6a paragraph at `:296-300`, and
the plan's original `:263-267` locator points at §5 instead.

#### Task 3 — extend the reviewer checklist line
In the same file's checklist block (the `### Checklist …` heading at the end of the file), extend the
existing implementation-audit row (`:390`) with a conditional clause and add no row (decision 6),
e.g.:

`- [ ] Implementation audit — acceptance test re-run, tasks & criteria verified; on a scoped
re-review, the fix diff read for new defects (steps 1, 2)`

#### Task 4 — mirror in the rules (`tickets-README.md` §5)
In `skill/resources/tickets-README.md`, the **Blocking** bullet of `## 5. Findings — severity,
then disposition` (`:398-402`) currently parenthesises *"(verify the findings are resolved — do not
re-audit the whole feature from scratch)"*. Widen that parenthetical the same way — findings plus
the fix diff, bound retained — in one clause, without restating §1's mechanics. This is the
mirrored sentence the Description names: it must not be left behind.

#### Task 5 — the rework procedure and the validate summary (`SKILL.md`)
In `skill/SKILL.md`, *Procedure: rework a ticket* (`:248-260`):

- **step 3** (`:255-256`, "fix only the listed findings") — add the capture from decision 3: before
  the first fix commit, record the branch tip, since that is the commit the re-review diffs
  against;
- **step 5** (`:258`, "Record what was fixed against each finding") — add that the record names the
  round's commits in the `### Rework fix record — round N (commit <sha>)` shape (decision 3,
  including its zero-commit form and the record-after-tidy ordering), and the self-read from
  decision 4: re-read the replacement text the fix wrote before handing back, since the next
  round's finding otherwise lands there.

Also extend *Procedure: validate a ticket*'s summary clause (`:273`, "scoped re-review after the
fix") so it does not contradict the widened scope — four or five words.

#### Task 6 — give the recorded range a documented home (`TEMPLATE.md`)
In `skill/resources/TEMPLATE.md`, the `## Review` guidance (`:148-155`) already lists what the
section carries, including "notes from any scoped re-review". Name the rework fix record in that
list — heading shape and the commit(s) it carries (decision 3) — in one clause, so the convention
already in use has a documented home instead of being re-invented each round. Use a plausible hex
SHA in any example.

#### Task 7 — the manual
In `docs/user-manual/concepts/lifecycle.adoc`, the *Reviews: severity, then disposition* section
(`:84-86`) says the scoped re-review "verifies them". Add the fix diff in one clause, pointing at
the protocol as the authority. Do not restate the mechanics — the payload owns them and the two
must not drift into two policies.

#### Task 8 — CHANGELOG entry
Add an `[Unreleased]` entry to `CHANGELOG.md` in the established style (bold lead sentence, then the
qualification, ticket id in trailing parens), naming the changed payload files
(`resources/review-protocol.md`, `resources/tickets-README.md`, `SKILL.md`,
`resources/TEMPLATE.md`) and noting that every installed project picks the rule up on its next
`pickle upgrade`.

### Acceptance test

```
just build
just test
just lint
just docs-check
```

All clean. `just test` is load-bearing, not a formality: it runs `payload_lint_test.go` over the
embedded payload, which mechanically enforces decision 7 for every `skill/` file this ticket edits;
`just docs-check` runs the xref tests over the edited manual page.

Then, specifically:

1. **The rule is stated on every surface that carries it, and none is left behind** —
   `git diff main --stat` lists exactly `skill/resources/review-protocol.md`,
   `skill/resources/tickets-README.md`, `skill/SKILL.md`, `skill/resources/TEMPLATE.md`,
   `docs/user-manual/concepts/lifecycle.adoc` and `CHANGELOG.md`; and
   `grep -rn -B1 -A1 're-audit the whole feature' skill/` shows every remaining occurrence sitting
   in a sentence that also names the fix diff (the `-B1 -A1` matters — the phrase wraps across
   lines, so a bare `grep -rn` cannot show what this check asserts).
2. **No step was renumbered** (decision 8) —
   `grep -n '^## [0-9]' skill/resources/review-protocol.md` matches
   `git show main:skill/resources/review-protocol.md | grep -n '^## [0-9]'` on every number
   (`## 0.` through `## 9.`, with `## 4a.`/`## 4b.`).
3. **The checklist gained no row** (decision 6) —
   `grep -c '^- \[ \]' skill/resources/review-protocol.md` equals
   `git show main:skill/resources/review-protocol.md | grep -c '^- \[ \]'`, and the
   implementation-audit row now mentions the fix diff.
4. **The class vocabulary is untouched** (decision 5) —
   `git diff main -- skill/resources/review-protocol.md` shows no added, removed or re-ordered row
   in the closed-vocabulary table under §5's "class" sub-heading.
5. **The rule ships, not just exists locally** — install into a throwaway dir per `AGENTS.md`'s
   self-modify policy and read it out of the installed payload:
   ```
   D=$(mktemp -d) && cp pickle "$D/pickle-test" && cd "$D" && ./pickle-test install --in-tree --project demo
   grep -rn 'fix diff\|fix pass' .agents/skills/brine/   # expect hits in review-protocol.md, tickets-README.md and SKILL.md
   ```
6. **The rule is executable against this very branch** — this ticket's own review applies it: if a
   rework round happens, the fix pass records its commit range in `## Review` and the re-review
   reads that diff, saying so in its notes. A round that cannot follow its own rule is a finding.

### Docs update (mandatory when user-facing)

Two surfaces, both in tasks above: `docs/user-manual/concepts/lifecycle.adoc` (task 7 — the manual's
review treatment, one clause, payload remains the authority) and `CHANGELOG.md` (task 8). No
CLI surface changes, so `docs/user-manual/reference/` needs no edit; `just docs-check` must stay
clean.

### Finish (mandatory)

1. Acceptance test green; `just build`, `just test`, `just lint`, `just docs-check` clean.
2. Docs updated (manual + CHANGELOG) and `just docs-check` re-run after them.
3. Write a summary: files touched, the wording landed on each surface, anything deferred.
4. Suggest a Conventional Commit message, e.g.:

   ```
   feat(skill): scoped re-review reads the fix's own diff (T-124)

   <body — what and why>
   ```

5. Root-path child: interactive-rebase the WIP commits into a small number of atomic, correctly
   typed/scoped commits before presenting them.
6. Commit locally on the ticket branch; do not push or open a merge request without user approval.
   Present the commit message; after approval, keep the tidied history (root-path default), verify
   the remote base is not behind (`git fetch origin main && git diff --name-only origin/main...HEAD
   | grep '^tickets/'` prints nothing — this project is `layout = "in-tree"`), then push and open
   the merge request. Merging is the human's.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-25 — created (TO DO). source: chat: measured the rework-cause distribution across
  `tickets/6-done/`'s and `tickets/5-rework/`'s `## Review` findings tables at the user's
  request (59 blocking, 227 non-blocking rows); the scoped re-review rule's "only the listed
  findings" wording is the recurring shape behind repeat-round rework (T-109 x4, T-067/T-018 x3,
  T-098/T-013/T-122 repeat rounds)
- 2026-08-26 — TO DO → READY: plan complete
- 2026-08-27 — READY → IN DEVELOPMENT: picked up, branch feat/T-124-re-review-reads-the-fix-diff; applicability gate: 1 blocking + 5 non-blocking folded into the plan in 2-ready/
