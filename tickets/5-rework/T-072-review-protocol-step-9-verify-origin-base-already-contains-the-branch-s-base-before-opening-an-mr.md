---
id: T-072
title: review protocol step 9: verify origin/<base> already contains the branch's base before opening an MR
project: pickle
depends-on: []
spawned-by: [T-068]
impact: medium
complexity: low
cost: S
---

# T-072 — review protocol step 9: verify origin/<base> already contains the branch's base before opening an MR

## Description

Rules §0 splits every change in two — code on the child's `feat/T-NNN-<slug>` branch, ticket and
board bookkeeping on the base branch — because a squash-merge of a branch carrying bookkeeping
folds or drops it and leaves `BOARD.md` disagreeing with the tickets it indexes. T-057 shipped a
`pre-commit` hook that enforces it, and T-068 made that hook report when it is inert.

**The hook can only see what you stage, so it is structurally blind to the same failure arriving
one step later — at publish time.** Measured during T-068's own publish (2026-08-06):

- Bookkeeping had been committed correctly on `main` throughout: six `docs(tickets):` commits, and
  `git diff origin/main..main -- . ':!tickets'` was empty. The hook never fired, correctly.
- But `main` had not been **pushed**, so `origin/main` was six commits behind — and the feature
  branch's own base (`aeb0bb4`, the *move to in-development* commit) was one of those six.
- Opening the MR at that moment would therefore have carried **two bookkeeping commits**
  (`bcd82c3` refine-to-READY, `aeb0bb4` move-to-in-development) and **~573 lines of `tickets/`
  churn** into the MR — measured with `git log --oneline origin/main..<branch>` and
  `git diff --stat`. A squash-merge would then have folded ticket bookkeeping into the code
  commit: exactly the §0 outcome, reached through a step nothing checks.
- The repair is trivial once seen — fast-forward the base first (`git push origin main`), after
  which the MR reduced to one commit, 15 files, zero `tickets/` paths — but nothing in the
  protocol tells you to look, and the symptom is invisible in the local repo: every local check
  (`git status`, `board audit`, the hook) is green.

This is the third distinct appearance of the same hazard — T-053/T-054 (bookkeeping committed on
the branch), T-022 (a branch cut before the bookkeeping landed shows a stale ticket, review
finding F6), and now the publish-time variant — which is the pattern that earned the hook in the
first place.

### Fourth appearance — and the first time it actually landed (T-073, 2026-08-07)

T-068's was caught *before* the push. **T-073's was not.** It reached `origin/main` and is now
permanent history, which moves this ticket from "prevents a near-miss" to "prevents a defect the
project has already shipped once".

The route in was new, and worth recording because it is the one a careful operator is *most*
likely to take: the branch was **rebased onto local `main`** before squashing, to keep history
linear. Local `main` was 4 bookkeeping commits ahead of `origin/main` (this ticket's own moves
plus an impact sweep). The rebase pulled all four into the branch's ancestry; GitHub diffed the
PR against `origin/main`, saw them, and the squash-merge folded ticket bookkeeping into the code
commit `7b33876` (PR #18). Nothing was lost — the resulting ticket text was byte-identical — but
the bookkeeping landed through a child-project code MR, which is precisely the §0 outcome.

**The proposed check was verified against this incident's real SHAs, after the fact.** At the
moment before the push (`origin/main` = `152fea8`, branch head = `850ea3c`):

```
git log --oneline origin/main..HEAD            → 5 commits (4 bookkeeping + 1 code); 1 expected
git diff --name-only origin/main...HEAD        → 7 tickets/ paths, incl. BOARD.md
```

So item 1's check is not merely plausible — it is **measured to catch the case that got through**.
Note also that it catches the *branch-carried* variant (T-053/T-054) equally, since that too
shows `tickets/` paths in the three-dot diff, whereas a `merge-base --is-ancestor` formulation
only catches the unpushed-base variant. That asymmetry should decide item 1.

A second, independent lesson from the same incident: **the guard was inert and had been saying so.**
`pickle doctor` was reporting `hooks: … was written by an older pickle (shim v1, this binary ships
v2)` and the `pickle` on `PATH` (Homebrew 0.2.2) predated the `hooks` verb entirely, so every
commit in the session printed `unknown command "hooks"` → `bookkeeping guard skipped`. That is
T-068's exact failure mode, reported correctly and never read. It changes nothing about *this*
ticket's scope — a working hook still could not have caught a rebase — but it is the reason the
session had no safety net at all, and it argues for the operator habit in item 6.

### What to add (for refinement — nothing is decided)

1. **The protocol line.** `skill/resources/review-protocol.md` step 9, in the child-project
   publish bullet: before pushing the branch and opening the MR, verify the base branch's remote
   already contains the branch's merge-base — e.g. `git merge-base --is-ancestor $(git merge-base
   origin/<base> HEAD) origin/<base>`, or the cheaper observable check
   `git log --oneline origin/<base>..HEAD` showing **only** the code commit(s), and
   `git diff --name-only origin/<base>...HEAD` naming no `tickets/` path. Push the base first if
   not. Refinement should pick **one** formulation — the protocol is prose an agent follows, so a
   single copy-pasteable check beats three alternatives.
2. **Two-dot vs three-dot is part of the trap, and worth stating.** During the same publish,
   `origin/main..branch` gave misleading answers in *both* directions as the base moved behind
   and then ahead (bookkeeping shown as additions, then as 330 deletions), while forges compute
   MR diffs from the **merge base** (`...`). Whatever check lands should be the three-dot form, and
   should say why, or the next reader will "simplify" it back.
3. **Does rules §0 also need a clause?** §0's *Where commits land* bullet documents the commit-time
   rule and names `pickle hooks install` as the local enforcement. The publish-time variant may
   belong there as a third sub-bullet (next to the single-repo and stale-ticket ones) rather than
   only in the review protocol — the protocol is *a* publisher, but a human pushing by hand hits
   the same hazard. Refinement decides: protocol only, or §0 + protocol.
4. **Could this be mechanically enforced instead of written down?** A `pickle hooks` addition
   (`pre-push`) could refuse a push whose range contains `tickets/` paths on a feature branch.
   ~~note the failure here was the *absence* of a push, not a bad one, so `pre-push` on the
   feature branch would not have caught it~~ — **this reasoning was wrong, corrected 2026-08-07
   by the T-073 incident.** The guard does not need to observe the missing *base* push: it fires
   on the **feature-branch push**, which does happen, and measures the branch against
   `origin/<base>`. That range contained `tickets/` paths in T-068's case and in T-073's
   (verified above, 7 paths), so a `pre-push` guard would have caught **both**, and could print
   the one-line repair (`git push origin <base>` first). Design notes for whoever picks it up: it
   must not fire when the ref being pushed *is* the base branch (bookkeeping there is the correct
   destination); it gets the ref and remote SHA on stdin, so it can compute the range exactly
   rather than guessing; and it needs the same fail-open semantics as the v2 `pre-commit` shim.
   A `pickle board audit`-style publish check (`pickle publish-check`?) remains the other shape.
   Either is materially bigger than a protocol line and needs its own ticket: refinement still
   **scopes this one to the prose**, but should now record the mechanical follow-up as
   **recommended** rather than merely "worth considering" — four occurrences is the promotion
   test answering itself.
6. **An operator habit the prose should name explicitly.** Both landed variants share one
   precondition: *the feature branch's base was allowed to get ahead of `origin/<base>`*. Two
   routes reach it — rebasing onto an unpushed base (T-073), or cutting the branch and then
   committing bookkeeping without pushing (T-068). A single sentence covers both: **push the base
   before you rebase onto it or open an MR from it.** Whether that belongs in the protocol, in
   §0, or in both is item 3's decision.
5. **Payload consequences.** Editing `skill/` changes what `pickle install` ships and what
   `pickle upgrade` re-installs, so the change reaches existing projects only on upgrade. Check
   whether `docs/user-manual/` describes step 9 anywhere and needs the same sentence, and whether
   the `AGENTS.md` marker block (`internal/install/install.go`, `markerBlock()`) mentions the
   commit/publish split closely enough to need a matching clause — if it does, the self-host
   mirror must be hand-edited inside this ticket's diff per the repo's self-modify policy, and
   `internal/install/testdata/markerblock.golden` regenerated.

### Soft couplings

- **T-068** — lineage (`spawned-by`); this was found while publishing it, and its History records
  the incident and the repair.
- **T-057** — shipped the commit-time guard whose blind spot this is. Its decision 2 (keep
  `board audit` git-free) is the reason a publish check is not simply bolted onto the audit.
- **T-067** — docs link/anchor validation; if item 5 turns up a manual page to change, it lands in
  the same tree T-067 will start validating.
- **T-071** — also spawned by T-068 and also small; both touch flow surfaces rather than
  `internal/config`, so they can be sequenced in either order, but neither should run concurrently
  with the other if both end up editing `skill/`.
- **T-073** — the fourth occurrence, and the one that actually landed (see the Description
  section above); its own History records the reconciliation. Not a dependency: T-073 is done and
  merged, and nothing in this ticket builds on its code.
- **T-046** (`doctor`/`upgrade` self-host-aware) and **T-068**/**T-071** (the PATH probe) — the
  "guard was inert and doctor said so" lesson above lives in their ground, not this one's. Noted
  here only so the next reader does not re-file it against this ticket.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .                     # pickle is the overarching project and its own child
git push origin main     # ← this ticket's own subject: land the base before branching
git checkout main
git checkout -b feat/T-072-publish-time-base-check
```

The `git push origin main` is not boilerplate here: implementing this ticket while committing
its own bookkeeping on an unpushed `main` would reproduce the exact defect it closes. Push the
base before branching, and again before publishing.

Local WIP commits as work progresses; publish only after user approval (finalize, push, open
the MR — merging is the human's), per the project's configured commit policy.

### Prerequisite gate

None. T-073 is done **and merged** (`7b33876`, PR #18), but nothing here builds on it — listed
only because its incident is this ticket's primary evidence.

### Confirmed design decisions (do not deviate without asking)

1. **This ticket ships prose only.** No Go code, no new command, no hook. The mechanical
   `pre-push` guard is explicitly *out of scope* and becomes its own ticket (see the Finish
   step) — Description item 4 now recommends filing it, which is a recommendation, not this
   ticket's job.
2. **One normative check, and it is the three-dot path check.** The protocol gets exactly one
   copy-pasteable command, not a menu:
   ```
   git diff --name-only origin/<base>...HEAD | grep '^tickets/' && echo "STOP: push origin <base> first"
   ```
   Chosen over the `merge-base --is-ancestor` formulation because it is **strictly more
   general**: it catches both the unpushed-base variant (T-068, T-073) *and* the
   bookkeeping-committed-on-the-branch variant (T-053/T-054), whereas `--is-ancestor` catches
   only the first. It also measures the symptom that actually matters ("the MR contains no
   `tickets/` paths") rather than a proxy for its cause.
3. **Three-dot, and the prose says why.** `...` is the merge-base form forges use to compute an
   MR diff; `..` gave actively misleading answers in both directions during T-068's publish. The
   sentence explaining this is **required**, not optional — without it the next reader
   "simplifies" it back to `..` (Description item 2).
4. **Both surfaces: rules §0 *and* the review protocol.** §0 gets the invariant (a human pushing
   by hand never reads the review protocol); the protocol gets the operational step. Resolves
   Description item 3 in favour of "§0 + protocol".
5. **No `AGENTS.md` marker-block change, therefore no golden regeneration and no self-host
   hand-edit.** Verified: `MarkerBlock()` (`internal/install/install.go:903-908`) compresses §0's
   *Where commits land* to five lines naming `pickle hooks install`; it already omits §0's two
   existing sub-bullets, so omitting a third is consistent, not a gap. This keeps the ticket S
   and the diff free of `markerblock.golden`.
6. **No `docs/user-manual/` change.** Verified by sweep: the only publish-adjacent hits are
   `your-first-project.adoc:89,101` (both about *approval*, not push mechanics) and
   `concepts/agent-session-workflow.adoc:73` (a table cell naming the step, no mechanics).
   Resolves Description item 5's manual question in the negative.

### Tasks

#### Task 1 — rules §0 gains the publish-time sub-bullet

`skill/resources/tickets-README.md`, §0 *Where commits land* (currently lines 50–67). Insert a
new sub-bullet **immediately after the `pickle hooks install` one (ends line 64)** and before
the mirror-image stale-ticket bullet (line 65) — that position is deliberate: it reads as "and
here is what that hook structurally cannot see".

Content: the hook is commit-time and cannot see the same failure arriving at publish time; the
invariant is that the branch's base must already be on `origin/<base>` before you rebase onto it
or open an MR from it; the one-line check from decision 2; the repair (`git push origin <base>`).
Keep it to ~5 lines to match the density of its siblings.

#### Task 2 — review protocol step 9 gains the operational step

`skill/resources/review-protocol.md`, step 9's **Child-project (build target)** bullet
(currently lines 197–206). The check goes **after** "Only after the user approves all
attributes:" and **before** "push, and create the merge request" — it is a pre-push gate, and
placing it before the approval sentence would wrongly imply it runs before the human is asked.

State the check verbatim (decision 2), the three-dot rationale in one clause (decision 3), and
the repair. Cross-reference §0 rather than restating the rationale twice.

#### Task 3 — the checklist line

Same file, the *Checklist (paste into the ticket's `## Review` section)* block at the end: the
step-9 checklist item currently reads `Summary + child-project commit message & MR attributes
presented for approval …`. Extend it to name the base-is-pushed verification, so the check is
visible in the ticket's own Review section rather than only in the protocol prose.

### Acceptance test

The substantive test is that the documented check actually catches the documented failure. It is
run against a **self-contained synthetic repro**, deliberately *not* against T-073's real SHAs:
`850ea3c` (the pre-squash branch head) is no longer reachable from any ref and survives only in
reflog until `git gc` prunes it, so a test citing it would pass today and fail spuriously for a
reviewer next month. This version has no such dependency and was executed end-to-end during
refinement — the expected output below is measured, not predicted.

```sh
# 1. build + the child's configured commands
just build && just test && just lint && just docs-check
./pickle board audit          # 0 errors, 0 warnings

# 2. the check on the failing shape, then on the repaired one
D=$(mktemp -d); cd "$D"
git init -q --bare remote.git && git clone -q remote.git work && cd work
git config user.email t@t; git config user.name t
mkdir -p tickets/1-to-do && echo code > app.go && echo t > tickets/1-to-do/T-001.md
git add app.go tickets/1-to-do/T-001.md && git commit -qm base
git branch -M main && git push -q -u origin main

# bookkeeping on main, NOT pushed — the T-068 / T-073 precondition
echo moved >> tickets/1-to-do/T-001.md && git commit -qam "chore(tickets): move"
# feature branch on top of that unpushed base, carrying one code commit
git checkout -q -b feat/T-002-x && echo more >> app.go && git commit -qam "feat: x"

git diff --name-only origin/main...HEAD | grep '^tickets/'   # MUST print tickets/1-to-do/T-001.md

# the repair, then the same check again
git checkout -q main && git push -q origin main && git checkout -q feat/T-002-x
git diff --name-only origin/main...HEAD | grep '^tickets/'   # MUST print nothing (exit 1)
git diff --name-only origin/main...HEAD                      # MUST print only: app.go
```

Step 2 is the one that matters: it proves the shipped check fires on the shape that reached
`origin/main` in T-073, and goes silent once the base is pushed — which is exactly the sentence
Tasks 1 and 2 add to the payload.

### Docs update (mandatory when user-facing)

**Payload prose is the deliverable**, so the "docs" are Tasks 1–3 themselves. Per decisions 5
and 6 there is deliberately **no** `docs/user-manual/` change, **no** marker-block change and
**no** `markerblock.golden` regeneration — each verified by sweep, not assumed. Note in the
summary that `skill/` is the embedded payload, so this reaches existing projects only when they
run `pickle upgrade`.

### Finish (mandatory)

1. Acceptance test green; `just build`/`test`/`lint`/`docs-check` clean; `pickle board audit`
   clean.
2. Docs = Tasks 1–3; record the two deliberate no-ops (manual, marker block) so review can
   confirm they were decided rather than forgotten.
3. **Push `main` before publishing** — and say so in the summary. Anything else would close this
   ticket by committing its own defect.
4. Write the summary: the two payload files touched, and the verified acceptance-test output for
   step 2.
5. **Recommend filing the mechanical follow-up** (`pre-push` guard, Description item 4) as a
   separate ticket with `--spawned-by "T-072"`, unless the user has already filed it. Do not
   implement it here.
6. Suggested commit message:

   ```
   docs(skill): guard the publish-time bookkeeping leak in §0 and step 9 (T-072)

   The pre-commit hook is commit-time and cannot see bookkeeping that reaches
   an MR because the branch's base was never pushed. Adds the three-dot
   tickets/-path check to rules §0 and review-protocol step 9, with the repair.
   ```

7. Commit locally on the ticket branch. Publish only per the project's commit policy (do not
   push or open a merge request without user approval). Present the commit message; only after
   approval finalize, push, and open the MR — merging is always the human's.

## Review

**Checklist**

- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass on the ticket's changed `.adoc`/`.md` files (step 4b)
- [x] Findings recorded with severity **and** disposition; disposition summary line present (step 5)
- [ ] Ticket moved to `tickets/6-done/` or `tickets/5-rework/`; `## History` appended (step 6)
- [ ] Other references updated if needed; board regenerated by the move (step 7)
- [ ] Remaining-tickets impact sweep done (step 8)
- [ ] Summary + commit message & MR attributes presented for approval + next-ticket suggestion (step 9)

**Step 2 — Implementation audit**

- Task 1 (§0 sub-bullet): **met** — `skill/resources/tickets-README.md`, inserted immediately
  after the `pickle hooks install` bullet and before the mirror-image stale-ticket bullet, per plan.
- Task 2 (review-protocol step 9): **met** — `skill/resources/review-protocol.md`, the check lands
  after the approval sentence and before "push, and create the merge request", per plan.
- Task 3 (checklist line): **met** — same file, the step-9 checklist bullet extended.
- Confirmed decisions 1–4: **honoured** — prose-only (no Go/hook code), the three-dot path check
  verbatim, three-dot rationale stated inline, both §0 and the protocol carry it.
- Confirmed decisions 5–6: **honoured, verified not assumed** — `git diff main...HEAD --name-only`
  touches only the two named payload files; no `internal/install/testdata/markerblock.golden`,
  no `docs/user-manual/` diff.
- Acceptance test: **re-run verbatim**, self-contained synthetic repro (not the T-073 SHAs, which
  are no longer reachable from any ref). Result matches the plan exactly: the check fires
  (`tickets/1-to-do/T-001.md`) on the unpushed-base shape, and is silent (only `app.go`) after
  `git push origin main` repairs it.
- `just build && just test && just lint && just docs-check`: all green. `pickle board audit`:
  81 tickets, 0 errors, 0 warnings.

**Step 3–4 — Quality & consistency audit**

No issues. The three sites that now state or reference the check (§0, protocol step 9, the
checklist line) agree verbatim on the command and the `origin/<base>` naming; grepped the tree
for other `origin/<base>`-shaped prose and found none stale or contradictory.

**Step 4a — Documentation audit**

- Coverage: this ticket's deliverable *is* the docs; Tasks 1–3 are the coverage, confirmed above.
- Whole-tree sweep found **F1** (below): `skill/resources/TEMPLATE.md` states the identical
  publish sequence ("push, and open the merge request") at two sites (its Implementation-section
  boilerplate and its Finish section) without the base-check — and per `tickets-README.md §7` /
  `SKILL.md`, TEMPLATE.md is **the literal source every new ticket's Implementation Plan is
  authored from**, so this is the single highest-propagation gap in the tree: every ticket refined
  after this one inherits the omission by construction. By contrast, `SKILL.md:246` states the
  same sequence but explicitly defers ("In short… Follow `resources/review-protocol.md`") to the
  file that now carries the fix — correct compression, not a gap.
- `just docs-check`: clean (re-confirmed after the 4b edits, below).

**Step 4b — Docs-readability pass**

Run on the ticket's two then-changed files. Of the suggestions returned, two touched text this
ticket introduced (the §0 hook note, the step-9 Finish bullet) and were applied; the remainder
touched pre-existing prose elsewhere in the same files and were **discarded as out of scope** for
this diff — applying them would bundle unrelated drive-by edits into a payload-prose ticket.
`just docs-check` re-run clean after applying.

**Step 5 — Findings**

| id | severity | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|
| F1 | blocking | — | `skill/resources/TEMPLATE.md` restates the pre-T-072 publish sequence at two sites without the base-check, and is the literal seed text for every future ticket's own Finish section | `TEMPLATE.md:50` (Implementation-section boilerplate), `TEMPLATE.md:109` (Finish step 5) | Add the same one-line check + repair, phrased for a per-ticket Finish section (not restating §0's full rationale — cross-reference it), at both sites |

**Disposition summary:** 1 blocking (F1) — not dispositioned; routes to `tickets/5-rework/` for a
scoped fix, then a scoped re-review of F1 only. 0 non-blocking findings.

## History

- 2026-08-06 — created (TO DO). source: pickle ticket new
- 2026-08-06 — filed at the user's request immediately after T-068's merge, from a hazard measured
  during T-068's own publish: bookkeeping was correctly on `main` and the `pre-commit` guard never
  fired, yet `origin/main` was six commits behind and the branch's base was one of them — so the MR
  would have carried two `docs(tickets):` commits and ~573 lines of `tickets/` churn, which a
  squash-merge folds into the code commit. Filed rather than hand-edited because `skill/` is the
  payload embedded in the binary, so this is a user-facing change to the `pickle` child and rules
  §8 routes it through a ticket (precedent: T-022, T-036 were both payload-prose tickets). Graded
  medium/low/S: it closes the publish-side blind spot of a guarantee the flow already makes twice
  over, and the likely diff is a handful of prose lines
- 2026-08-07 — patched with a **fourth occurrence — the first to actually land**: T-073's publish
  reached `origin/main` (squash `7b33876`, PR #18) carrying four bookkeeping commits, via a route
  not previously recorded — rebasing the feature branch onto an unpushed local `main`. The
  proposed check was then verified against that incident's real SHAs (5 commits where 1 was
  expected; 7 `tickets/` paths in the three-dot diff), so item 1 is now measured rather than
  plausible. Description item 4's dismissal of a `pre-push` guard is **corrected**: the guard
  fires on the feature-branch push (which does happen) and measures against `origin/<base>`, so
  it would have caught T-068's and T-073's alike — the mechanical follow-up moves from "worth
  considering" to "recommended", still as a separate ticket. Added item 6 (the operator habit
  both variants share) and a note that the guard was inert throughout the T-073 session, which
  `pickle doctor` had been reporting and nobody read
- 2026-08-07 — **re-graded: unchanged at medium/low/S**, deliberately. Recurrence raises
  *urgency*, not impact: by §3's rubric `high` is "major capability/adoption lever", and this is
  prose preventing a consistency defect — squarely `medium`. Inflating it to force board position
  would game the grade, and it leads READY regardless. Complexity `low` / cost `S` hold:
  decisions 5–6 removed the two things that could have grown it
- 2026-08-07 — TO DO → READY: plan complete: 6 confirmed decisions, 3 tasks with exact insertion points, acceptance test executed during refinement
- 2026-08-07 — READY → IN DEVELOPMENT: picked up
- 2026-08-07 — IN DEVELOPMENT → IN REVIEW: acceptance green: synthetic repro fires on unpushed-base shape, silent after repair; build/test/lint/docs-check/board-audit clean
- 2026-08-07 — IN REVIEW → REWORK: 1 blocking (F1): TEMPLATE.md restates the pre-fix publish sequence at 2 sites, the seed text for every future ticket
