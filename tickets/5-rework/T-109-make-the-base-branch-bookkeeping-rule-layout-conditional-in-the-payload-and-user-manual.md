---
id: T-109
title: make the base-branch bookkeeping rule layout-conditional in the payload and user manual
project: pickle
depends-on: [T-108]
spawned-by: []
impact: high
complexity: medium
cost: L
---

# T-109 — make the base-branch bookkeeping rule layout-conditional in the payload and user manual

## Outcome

After this ships, the flow's rules stop stating the base-branch bookkeeping requirement as
universal law. A reader operating in the default umbrella layout is no longer instructed to
follow a rule that, for their **child repositories**, cannot apply — while the rule itself is
restated correctly as binding whichever repository holds `tickets/`, not a layout name, so the
overarching project is never mistakenly told it is exempt too. A reader in the in-tree layout
gets the full consequence list spelled out: every command that reads tickets reports the
checked-out branch's copy, that a stale worktree can never falsely show `DONE` but can show
other statuses out of order, CI fires on every bookkeeping push, and the base branch's history
interleaves board moves with code. The user manual gains a section naming both layouts, stating
which is the default, and stating what choosing in-tree costs.

## Description

The payload states the base-branch bookkeeping rule as though it governs every project:

```
skill/resources/tickets-README.md:52   "...is committed on the base branch of the overarching
                                        project**, never on..."
skill/resources/review-protocol.md:38  "Bookkeeping is committed on the base branch, not..."
```

Both are unconditional, and both are among the most emphatic statements the flow makes. In the
**umbrella layout — the primary and default one** — they are vacuous: the board lives in the
overarching project, the child-projects are separate repositories that know nothing about it,
and no feature branch cut in a child can fork the board. There is no stale-worktree hazard to
guard against, so a reader is being handed a rule whose entire justification is absent from
their setup. In the **in-tree layout** the rule is real, load-bearing, and currently
under-explained: it states what to do without stating what goes wrong if you do not.

This is the same defect class **T-022** already corrected once ("payload states commit policy,
branch/ticket prefixes and WIP limits unconditionally"), which makes it useful precedent for
both the shape of the fix and the review bar.

**The consequences to document are broader than the stale board UI.** Choosing in-tree also
means:

- Every reading command — `serve`, `board audit`, `board state --json` — reports the state of
  the checked-out branch's copy of `tickets/`, not the project's true state.
- A stale worktree can never show `DONE` for a ticket that is not — `6-done` is terminal, so a
  ticket that reaches it stays there. That half is quiet and easy to shrug off, which is why it
  is worth stating; it is not the whole picture, since a ticket sent backward (`in review` to
  `rework`, `ready` dropped) shows through a stale copy as further along than it now is.
- CI fires on every bookkeeping push. This project's own `.github/workflows/ci.yml` runs
  `on: push: branches: [main]`, so a run of bookkeeping commits triggers a full CI run per
  commit for zero code change. Recommending a `paths-ignore` entry for the ticket tree belongs
  in the manual as part of choosing this layout.
- The base branch's history interleaves board moves with code, so changelog and release tooling
  must filter.

**Constraint that governs every sentence.** Everything under `skill/` is read by projects that
are not this one, in workspaces where this repository does not exist, so the foreign-workspace
test in `AGENTS.md` binds: no "this repo" meaning ours, no counts drawn from a corpus the reader
does not have, no ticket id the reader is told to go and look up, and no path that only resolves
in pickle's own source tree. `payload_lint_test.go` enforces the mechanical part of that at
`just test`, but it matches four rules and cannot judge meaning, so the judgement stays with the
author.

**Why this is a separate ticket from T-108.** Documentation may only describe behaviour that
exists. Writing "in the umbrella layout rule §0 does not apply" before the `layout` key ships
would document unshipped behaviour — the same reason `docs/proposals/post-merge-done-move.adoc`
was deliberately kept outside the manual's `include::` tree. Hence the hard `depends-on`.

**Soft couplings (not `depends-on`):**

- **T-022** (done) — the precedent: unconditional payload prose treated as a defect.
- **T-057, T-072, T-082, T-100** (all done) — the enforcement family whose applicability this
  ticket documents. Their code is correct in the in-tree layout and inert in the umbrella one;
  the hook documentation should say so rather than leaving an operator to wonder why a guard
  never fires. Nothing here deletes or deprecates them.
- **T-046** (done) — precedent for layout-conditional behaviour being explained rather than
  silently skipped.

## Implementation Plan

### 0. Feature branch (mandatory)

The target child is `pickle` at `path = "."` (a root-path child):

```
git checkout main
git checkout -b feat/T-109-layout-conditional-rules
```

Local WIP commits encouraged; **publish-gated** — no push and no merge request without explicit
user approval, and merging is always the human's. Root-path child, so interactive-rebase the WIP
commits into atomic, correctly typed commits and **keep that history** rather than squashing
(rules §0). Before pushing, verify the remote base is not behind the local base — `git fetch
origin main && git diff --name-only origin/main...HEAD | grep '^tickets/'` must print nothing,
or push `origin main` first.

Because this ticket edits `skill/`, note that `.agents/skills/brine` is a symlink to it: editing
either path changes what `pickle install` ships.

### Prerequisite gate (hard)

**T-108 must be in `6-done/` and merged to `main`.** This is a hard dependency, recorded in
`depends-on:`. The reason is not convenience: this ticket documents the `layout` key, the
`--in-tree` flag and the `serve` banner, and documenting behaviour that has not shipped is the
defect that kept `docs/proposals/post-merge-done-move.adoc` out of the manual's `include::`
tree. Do not start until T-108's branch is merged, not merely approved.

### Confirmed design decisions (do not deviate without asking)

1. **Every affected rule states which repository it binds — whichever holds `tickets/` — rather
   than naming a layout as though the rule itself were conditional; none is deleted.** The
   base-branch bookkeeping rule stays true, mandatory and fully stated. Under `in-tree` that
   repository is the sole child, so the rule governs its branches as a matter of course. Under
   `umbrella` it is the overarching project, not any child — so a **child**'s feature branch can
   never fork the board, but the overarching project's own branches remain bound exactly as an
   in-tree child's would.
2. **The foreign-workspace test binds every sentence added under `skill/`.** No "this repo"
   meaning pickle's own; no count or claim drawn from a corpus the reader does not have; no
   ticket id the reader is told to go and look up; and no path that resolves only in pickle's
   source tree — phrase paths relative to the skill the reader is holding.
3. **`payload_lint_test.go` staying green is necessary but not sufficient.** It matches four
   mechanical shapes and cannot judge what a sentence means; the judgement remains the author's.
4. **Nothing in the enforcement family is deleted or deprecated.** The guards from T-057, T-072,
   T-082 and T-100 remain correct and load-bearing wherever `tickets/` lives — every child's repo
   under `in-tree`, the overarching project under `umbrella`. The documentation explains that
   they are inert **inside every child repository under `umbrella`** (no `tickets/` there to
   guard against), not that the rule or the guards are inert for the layout as a whole, so an
   operator is never left wondering why a guard never fires in a child, nor mistakenly assumes
   the same of the overarching project.
5. **The manual states the default explicitly and gives the in-tree consequence list in full.**
   That list is: every reading command reports the checked-out branch's copy; a stale worktree
   can never show `DONE` for a ticket that is not (`6-done` is terminal), though a ticket sent
   backward (`in review` to `rework`, `ready` dropped) can show through a stale copy as further
   along than it now is; CI fires on every bookkeeping push, with a `paths-ignore` entry for the
   ticket tree recommended as the mitigation; and the base branch's history interleaves board
   moves with code.
6. **Terminology is fixed to `umbrella` and `in-tree`, matching T-108's config values exactly.**
   The word "sibling" is not used for either layout: it was used during design for two opposite
   arrangements and would arrive in the manual already ambiguous.

### Tasks

#### Task 1 — the flow rules (`skill/resources/tickets-README.md`)
Make §0's bookkeeping rule conditional (the unconditional statement is at line 52, with related
mentions at 109 and 121–122). State the in-tree rule in full, and state plainly that in the
umbrella layout the board is not in any child's repository, so the rule does not apply.

#### Task 2 — the review protocol (`skill/resources/review-protocol.md`)
The boxed note at lines 38–44 tells every reviewer to read the ticket from the base branch. Make
it conditional on the in-tree layout and explain the failure it prevents (a branch cut before
the bookkeeping commit shows a stale ticket), so the instruction carries its own justification.

The same unconditional instruction recurs at **line 55** ("**as it exists on the base branch**",
in the locate-the-ticket step), outside the boxed note's range. Both occurrences are in scope.

#### Task 3 — `skill/SKILL.md`
Update the places that assert the rule unconditionally, keeping SKILL.md's summary register
rather than duplicating the long-form explanation. There are **two**, and only the first contains
the literal phrase "base branch":

- **line 77** — the commit-policy bullet ("on the base branch — never on a feature branch").
- **lines 271–272** — the review procedure's origin-base check (`origin/<base>...HEAD` must carry
  no `tickets/` path), which is likewise in-tree-only and is easy to declare done after editing
  only line 77.

Also correct the *Install & register* paragraph (lines 54–56), which still states that
`pickle install` "registers the first child-project": once the layout is recorded, a plain
`install` registers **no** child and `pickle project add` registers the first one, while
`--in-tree` registers the sole child at `.`. (Folded here from T-108's review, finding F5 —
payload edits belong in this ticket rather than in a branch whose plan does not name `skill/`.)

#### Task 4 — hook documentation
`docs/user-manual/cli-reference.adoc`'s `== pickle hooks` section, including its
`=== What the guards do and do not catch` subsection: state that the guards are meaningful
in the in-tree layout and inert in the umbrella layout, and why. (Cited by heading rather than
line number: T-108 inserted its `[#install-layout]` section above both, moving them from 385/487
to 445/547, and a merge of anything else will move them again.)

#### Task 5 — the conceptual manual chapter
**Rescoped: most of this already shipped in T-108.** `docs/user-manual/cli-reference.adoc`'s
`[#install-layout]` section already names both layouts, marks `umbrella (default)`, states when
in-tree is the right choice ("it makes tickets visible to anyone who clones the code"), and gives
the *first* of decision 5's four consequences (every reading command reports the checked-out
branch's copy). `concepts/project-structure.adoc`'s `== The root child` section already carries
the in-tree base-branch rule and the hooks. Re-deriving any of that as a new chapter would
duplicate shipped prose.

What is genuinely missing is **three of decision 5's four consequences**:

1. a stale worktree can never show `DONE` for a ticket that is not (`6-done` is terminal), which
   is what makes it quiet and easy to shrug off — though a ticket sent backward can show through
   a stale copy as further along than it now is;
2. **CI fires on every bookkeeping push**, with a `paths-ignore` entry for the ticket tree as the
   recommended mitigation;
3. the base branch's **history interleaves board moves with code**, so changelog and release
   tooling must filter.

Add these to `concepts/project-structure.adoc`'s existing in-tree section (`== The root child`),
cross-referencing `<<install-layout>>` rather than restating it. **No new file and no
`include::` change**, since the host chapter is already registered.

#### Task 6 — sweep for residual unconditional phrasing
Re-read every payload file that mentions the base branch and confirm each occurrence is either
inside a layout-conditional block or genuinely layout-independent.

**Widened:** the sweep also covers phrasing that names the *old* default, which no "base branch"
grep would find. `skill/resources/tickets-README.md:83` ("In the **single-repo default**
(`path = "."`, one child at the overarching root)") and `:119` ("which is the single-repo default
above") both assert a default that T-108 reversed: `umbrella` is now the default and `path = "."`
is the explicitly-selected exception. Rename these to the recorded layout name (`in-tree`) and
drop the "default" claim.

### Acceptance test

Run from the repository root on the feature branch.

1. `just build && just test && just lint && just docs-check` — all clean. `just test` includes
   `payload_lint_test.go`, which enforces the mechanical half of decision 2.
2. **No payload file still states the rule unconditionally:**

   ```
   rg -l 'base branch' skill/ | xargs rg --files-without-match 'in-tree'
   ```

   prints nothing — every payload file that mentions the base branch also names the layout
   condition. **Use `--files-without-match`, not `-L`:** in ripgrep `-L` is `--follow`, so the
   check as originally written printed nothing *before* any work and would have rubber-stamped a
   no-op. Today the corrected form prints all three payload files
   (`review-protocol.md`, `tickets-README.md`, `SKILL.md`); after the edits it must print nothing.

   **The payload is `skill/` *and* `agents/`** (`assets.go`: both trees are embedded and installed),
   so run the same check over both — `rg -l 'base branch' skill/ agents/ | xargs rg
   --files-without-match 'in-tree'` — or the residual occurrence in a shipped agent scaffold is
   invisible to it (review 1, F3).
3. **Both layouts are named in the manual, including the new prose:** search **recursively** —
   `docs/user-manual/`, not the non-recursive `docs/user-manual/*.adoc` glob, which cannot see
   `concepts/` where Task 5 puts the content. Bare term presence proves nothing (`umbrella`
   already appears in three manual files), so assert the three new consequence phrases instead —
   see check 5.
4. **The rejected term does not appear as a layout name:** `rg -i 'sibling' skill/ docs/` returns
   no hit that uses it to mean a layout (decision 6). Two hits are pre-existing and
   **known-good** — `cli-reference.adoc:320` and `:365`, both "sibling extension files" (agent
   config files, not layouts). Only a *new* hit, or one of these repurposed, fails the check.
5. **The consequence list is present and complete:** across `[#install-layout]` (which already
   carries per-branch reads) and Task 5's addition, the manual names all four items from
   decision 5 — per-branch reads, the DONE-is-terminal staleness asymmetry, CI-on-every-bookkeeping-push
   with the `paths-ignore` mitigation, and interleaved history.
6. **The guards' applicability is documented:** the `pickle hooks` section states both that the
   guards matter in in-tree and that they are inert in umbrella.

### Docs update (mandatory when user-facing)

This ticket *is* the docs update. Both surfaces change: the **payload** under `skill/` and
`agents/` (which ships to other projects and is bound by decision 2) and the **user manual** under
`docs/user-manual/`. **No new file and no `include::` change** — Task 5's rescope puts the new
prose into `concepts/project-structure.adoc`, a chapter already registered, so there is nothing
new for `just docs-check` to be pointed at.

`CHANGELOG.md` **does** need an entry: the rendered `AGENTS.md` marker block now differs by
layout, which is user-visible output of `install`/`upgrade`/`project add`, and a shipped-skill
rules change has its own precedent under `[Unreleased]` → `### Changed`. `pickle changelog check`
names the ticket until it is there (review 1, F4).

### Finish (mandatory)

1. Acceptance test green; `just build && just test && just lint && just docs-check` clean.
2. Docs updated and registered in the `include::` tree.
3. Write a summary of every file touched, with the layout-conditional wording quoted for the two
   headline rules, plus anything deferred.
4. Suggested Conventional Commit message:

   ```
   docs(skill): make the base-branch bookkeeping rule layout-conditional (T-109)

   The rule is load-bearing in the in-tree layout and vacuous in the umbrella
   layout, but was stated as universal law. State the condition, explain the
   failure it prevents, and document what choosing in-tree costs.
   ```

5. Root-path child: interactive-rebase the WIP commits into atomic, correctly typed commits and
   keep that history rather than squashing.
6. Commit locally; present for approval. Do not push or open a merge request without it.

## Review

**2026-08-18 — review 1 (full).** Branch `feat/T-109-layout-conditional-rules` at `a655f42`
(3 commits, MR #54 open), read against `main`. Verdict: **blocking findings — to `5-rework/`**.
The restructuring is the right shape and every acceptance check passes as written, but the
layout-conditional claim was carried one step too far: the guards are scoped to *the repository
that holds `tickets/`*, not to the layout, and both the manual and the payload now tell an
umbrella reader they can never fire. They do — reproduced below.

- [x] Implementation audit — acceptance test re-run (all 6 checks), tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass on the changed `.adoc`/`.md` files (step 4b) — 20 suggestions returned,
      18 on prose this branch did not touch (out of scope); the two that land on branch-authored
      prose are carried into rework as polish, not findings
- [x] Findings recorded with severity, class and disposition per the rules §5 (step 5)
- [x] Ticket moved to `5-rework/`; `## History` appended (step 6)
- [x] Other references updated; board regenerated by the move (step 7)
- [x] Remaining-tickets impact sweep done (step 8)
- [x] Summary + commit message presented for approval; publish gate re-opens after rework (step 9)

**Verified green.** `just build && just test && just lint && just docs-check` all clean on the
branch. Acceptance checks 1–6 all pass verbatim, including the corrected `--files-without-match`
form of check 2 (prints nothing) and check 4 (only the two known-good "sibling extension files"
hits). `pickle doctor -v`: 0 errors, 0 warnings — the hand-edited `AGENTS.md` marker block
round-trips against the new conditional `MarkerBlock` render, and both rendered variants were
inspected in throwaway installs (`pickle-test`): umbrella (no child, and with a nested child) and
in-tree via the golden file.

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | blocking | plan-wrong | — | The manual states twice that the hooks **cannot fire** under `umbrella`. They can, and in the one repository an umbrella user installs them in. The guard's predicate is *this repo contains `tickets/`* **and** *HEAD (or the push destination) matches a registered child's `branch_prefix`* (`internal/hook/hook.go`, `ticketsPrefix` — "tickets/ is outside this repository — nothing to guard" — plus `onFeatureBranch`; `prepush.go:128,145` repeats both). In the umbrella **overarching** repo both hold, and that repo is where `pickle hooks install` is naturally run. So "they will simply never fire" and "do not read the silence as a broken hook or a missing install" can send a reader who *did* hit the refusal looking in the wrong place, or teach `--no-verify` as the cure. What is false is decision 4's premise — "they are inert in the umbrella layout" — hence `plan-wrong` rather than `docs-gap`. | Reproduced with a throwaway `pickle-test`: umbrella install + nested child + `hooks install`, then `git checkout -b feat/T-001-demo && git add tickets` → `pickle hooks run pre-commit` exits **1** with the full "refusing to commit ticket bookkeeping on a feature branch" rejection, with `layout = "umbrella"` in `pickle.toml`. Claims: `cli-reference.adoc:455-465` (the NOTE) and `:596-601` (the bullet). | Re-frame both from *layout* to *repository*: the guards fire in whichever repo holds `tickets/` when HEAD looks like a registered child's feature branch. Under `umbrella` that means they are inert **inside the child repositories** (no `tickets/` there), while in the overarching repo they still catch bookkeeping riding on one of *its* feature branches — which is a real hazard there too, since a squash-merge folds or drops it exactly as in-tree. Keep the "silence is not a fault" reassurance, but attach it to the child repos. |
| F2 | blocking | correctness | — | The same over-reach in the payload, in stronger words. `SKILL.md`: "the board is in a different repository from every child, so no feature branch can fork it, and **the hooks have nothing to catch**". `tickets-README.md` §0: "**Under `umbrella` this bullet and all of its sub-bullets do not apply** … no feature branch can fork it, and **bookkeeping may be committed whenever it is ready**". Both reason only about *child* branches and then generalise to *any* branch. An umbrella overarching repo that uses feature branches of its own (docs/NOTES.md work, a PR-based board repo) has the identical fold-or-drop hazard, and the installed guard refuses that commit — so the payload now sanctions the very commit the shipped hook rejects. | `skill/SKILL.md:81-84`; `skill/resources/tickets-README.md:96-103` (and the umbrella clause of `:56-58`, "Every rule below that names the base branch belongs to `in-tree`"). Same reproduction as F1. | Qualify the subject: no **child's** feature branch can fork the board. Then state the residual rule that survives in both layouts — bookkeeping must not ride on a feature branch **of the repository that holds `tickets/`** — and let the layout decide only its *reach* (in-tree: the same branches carry code, so the collision is routine; umbrella: only if the overarching repo itself uses feature branches). That keeps §0's promise honest without restoring the universal-law framing. |
| F3 | blocking | docs-gap | — | Task 6 ("re-read **every payload file** that mentions the base branch") is not met: `agents/pi/extensions/pickle-guardrails.ts` still states the rule unconditionally, as one of "brine's non-negotiable git rules" — "*where commits land*: code on the feature branch, ticket/board bookkeeping on the base branch — is deliberately NOT mirrored here. It is enforced by `pickle hooks install`". That file is payload (`assets.go`: `//go:embed all:skill all:agents`, installed to `.pi/extensions/`), so every `--agent pi` project reads it. Acceptance check 2 cannot see it because it greps `skill/` only. | `agents/pi/extensions/pickle-guardrails.ts:12-17`; `assets.go:19`; check 2's `rg -l 'base branch' skill/`. | Add the same condition F1/F2 settle on to that header comment (one clause), and widen acceptance check 2 to `skill/ agents/` — done in the plan's acceptance test as part of this review (see the amended check 2). |
| F4 | blocking | docs-gap | — | No `CHANGELOG.md` entry, for a change that is user-visible twice over: the rendered `AGENTS.md` marker block now differs by layout (output of `install`, `upgrade` and `project add`), and the shipped skill's rules changed. `pickle changelog check` flags exactly this. The plan's docs step named both doc surfaces and omitted the changelog, so nothing prompted it. Precedent is settled in both directions: a shipped-skill prose change got an `### Changed` entry (T-106), and a docs-only ticket with no user-facing change recorded the decision *not* to add one (T-019 decision 9) — here neither happened. | `pickle changelog check` → "1 candidate(s) shipped but not named in \"Unreleased\": T-109"; `git diff main...HEAD -- CHANGELOG.md` empty. | One `[Unreleased]` → `### Changed` entry, appended (file convention: landing order), naming the layout-conditional marker block and the layout-conditional rules, symptom first. |
| F5 | blocking | plan-wrong | — | The new "staleness is one-directional" paragraph states an invariant that does not hold as written: "a stale worktree can only *under*-report progress". Only the `DONE` half is invariant (`6-done` is terminal — `internal/flow/brine.go:130-146` gives it no outgoing transition). The flow has four **backward/abort** transitions (`2-ready → 1-to-do`, `3-in-development → 2-ready`, `4-in-review → 5-rework`, and `→ 7-dropped` from four states), so a stale copy can equally show a status *further along* than the truth: `IN REVIEW` for a ticket now in `REWORK`, `READY` for one since dropped. The paragraph's own purpose is to be relied on, and decision 5 mandates the wording, so the fix needs a recorded deviation. | `docs/user-manual/concepts/project-structure.adoc:198-206`; `internal/flow/brine.go:130-146`; decision 5 bullet 2 and the Description's second consequence. | Keep the load-bearing claim — a stale read can never show `DONE` for a ticket that is not done, because `6-done` is terminal — and drop or qualify the broader "only under-reports": a backward move or a drop lets a stale copy over-report too. The actionable advice ("check which branch you are on") is unchanged. Amend decision 5 and the Description in the same pass, with a `plan amended inline` History line. |
| N1 | non-blocking | stale-xref | noted | The pre-T-108 default survives outside the payload, which Task 6's widened sweep covered only inside it: `cli-reference.adoc:395` "A child at `.` (the single-repo default) is exempt", `concepts/project-structure.adoc:127` "not only the single-repo case below", and `internal/vcs/vcs.go:114` "the single-repo default, however it was spelled". The first is an outright false claim about the default; the other two are the terminology decision 6 retired. Pre-existing (T-108 made them false, not this branch), so the inline bar in the rules §5 rules out fixing them here. | `rg -n 'single-repo' docs/ internal/ skill/` after the branch's own fix at `tickets-README.md:83`/`:119`. | Three one-line edits. Cheap enough to fold into the rework pass if the user widens its scope; otherwise it batches with T-108 review F8 (the two layout errors with no remedy clause) into one post-layout tidy-up ticket. |
| N2 | non-blocking | other | noted | The branch shipped four files no task named — `internal/install/install.go` (the conditional `MarkerBlock` bullet), `AGENTS.md`, `internal/install/testdata/markerblock.golden`, `internal/install/hooks_test.go` — with no `plan amended inline` History line recording the extension. The work itself is correct and was necessary (leaving the marker block unconditional would have contradicted the payload it renders alongside), the hand-edit of `AGENTS.md` follows the self-modify policy, and the new test asserts *both* directions. Only the traceability is missing: a reader of the plan cannot tell why Go code changed under a docs ticket. | `git diff --stat main...HEAD` vs Tasks 1-6, which name only `skill/` and `docs/user-manual/`; `## History` has three `plan amended inline` lines, none covering this. | Nothing to undo. Record the extension in the rework pass's History line, so the plan and the diff agree. |
| N3 | non-blocking | stale-xref | fixed inline | The plan's "Docs update" step still required "The new conceptual chapter must be registered in the manual's `include::` tree", which the pickup amendment had already retired ("**No new file and no `include::` change**"). The plan therefore contradicted itself about its own deliverable, and a re-reader could read the docs step as an unmet obligation. | The ticket's own `### Docs update` vs Task 5 as amended 2026-08-17. | Rewritten to match Task 5 (no new file, host chapter already registered) and extended with F4's changelog obligation. |
| N4 | non-blocking | test-gap | fixed inline | Acceptance check 2 greps `skill/` only, so it certifies "no payload file states the rule unconditionally" while being structurally blind to half the payload (`agents/`) — the half F3 found. | Check 2 as written vs `assets.go:19` (`//go:embed all:skill all:agents`). | Widened in the plan to `rg -l 'base branch' skill/ agents/ \| xargs rg --files-without-match 'in-tree'`, so the re-review's own run would have caught F3. |

**Disposition summary:** 5 blocking (F1-F5, no disposition — they are the rework scope), 4
non-blocking: 2 `fixed inline` (N3, N4 — both in this ticket's own plan), 2 `noted` (N1, N2). No
follow-up ticket minted: N1 is the only promotion candidate and it batches naturally with T-108
review F8 rather than alone.

cost: estimated L, actual L — the delivered scope matches the estimate; the rework is prose in
files already open, plus one changelog entry.

**Docs-readability suggestions carried into rework** (polish, not findings; the other 18 landed on
prose this branch did not touch): in `review-protocol.md` §1, the inserted umbrella sentence leaves
"…read it directly. Read it in full — …" reading as two commands in a row — "Then read it in full:"
resolves it; in §9, the `in-tree` push check reads more easily with the "(§0 explains the three-dot
choice…)" aside promoted to its own sentence — keep the `(§0)` cross-reference on the `pre-push`
clause when doing so.

**Impact sweep (step 8):** no ticket in `1-to-do/` or `2-ready/` lists T-109 in `depends-on:` or
depends on its assumptions. T-050 is the only open ticket that edits
`agents/pi/extensions/pickle-guardrails.ts`, and it rewrites the rule-1 *verdict*, not the header
comment F3 touches — no collision, no patch needed. T-071 (hook PATH probe) is unaffected: F1/F2
change prose, not the guard predicate.

**2026-08-18 — review 2 (scoped re-review).** Branch `feat/T-109-layout-conditional-rules` at
`6421c95`, read against `main`. Scope: the five blocking findings of review 1 only. Verdict:
**blocking findings — back to `5-rework/`**. F1, F3 and F5 are fixed and verified; F4's mechanical
half is fixed but its content misdescribes what shipped; **F2's correction reached two of the four
surfaces that carry the claim** — the rendered marker block and the review protocol still state it
the old way, and a test now locks the marker block's wording in place.

- [x] Blocking findings re-verified individually (step 2, scoped — no full re-audit)
- [x] Acceptance test re-run: all 6 checks, including check 2 over the widened `skill/ agents/`
- [x] `just build && just test && just lint && just docs-check` clean (19 packages ok)
- [x] Docs-readability pass on the rework's changed `.adoc`/`.md` (step 4b) — 26 suggestions, 24 on
      prose neither pass authored (out of scope); 1 applied, 1 rejected with reason (below)
- [x] Findings recorded with severity, class and disposition per the rules §5 (step 5)
- [x] Ticket moved to `5-rework/`; `## History` appended (step 6)
- [x] Impact sweep re-checked — unchanged from review 1, nothing depends on T-109 (step 8)

**Review-1 findings, re-verified one by one:**

| finding | verdict | evidence |
|---|---|---|
| F1 (manual claims hooks cannot fire under umbrella) | **fixed** | `cli-reference.adoc:456-473` NOTE and `:604-612` bullet now scope the guards to *the repository holding `tickets/`*, name the in-tree and umbrella cases separately, and keep the "silence is not a fault" reassurance attached to child repos. Reproduced both directions again: umbrella overarching repo on `feat/T-001-demo` → `pre-commit` exits 1; the same project's child repo → exits 0. |
| F2 (payload over-reach) | **partially fixed — see R1, R3** | `SKILL.md:79-87` and `tickets-README.md:53-58, 99-112` are correct now. But `install.go:1101-1104` (rendered marker block) and `review-protocol.md:49` still carry the original claim verbatim. |
| F3 (pi guardrail scaffold) | **fixed** | `pickle-guardrails.ts:13-20` now names the binding repository and both layouts; widened check 2 (`rg -l 'base branch' skill/ agents/ \| xargs rg --files-without-match 'in-tree'`) prints nothing. |
| F4 (no CHANGELOG entry) | **entry added, content wrong — see R2** | `pickle changelog check` → "no candidates — every shipped ticket is mentioned". The entry itself misstates two things. |
| F5 (one-directional staleness overclaim) | **fixed** | `project-structure.adoc:199-210` now leads with the invariant that holds (`DONE` is terminal, so a stale read can never show it falsely) and states the backward-move case (`in review` → `rework`, `ready` dropped) as over-reporting. Decision 5, the Outcome, the Description and Task 5 were amended to match, with a `plan amended inline` History line. |

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| R1 | blocking | correctness | — | F2's fix skipped the surface with the widest reach. `MarkerBlock`'s umbrella branch still renders "The board lives in the overarching project, a different repository from every child, so **no feature branch can fork it**: ticket and board bookkeeping is committed there, **whenever it is ready**" — the two phrases review 1 quoted as false, now corrected everywhere except here. This is the block injected into every umbrella project's `AGENTS.md`/`CLAUDE.md`, i.e. what every agent session reads before touching the flow. Worse, `hooks_test.go` **locks the defect in**: the umbrella case asserts `want: "no feature branch can"` and `unwanted: "bookkeeping is committed on the base branch"`, so a corrected block fails the test. The code comment above the string, and the test's own doc comment ("a guard that never fires"), repeat decision 4's disproved premise. | Reproduced in one throwaway workspace: `pickle-test install` + `project add web child` renders the bullet above with `layout = "umbrella"`; `hooks install` + `git add tickets` on `feat/T-001-demo` in that same repo → `refusing to commit ticket bookkeeping on a feature branch`. Sources: `internal/install/install.go:1094-1104`, `internal/install/hooks_test.go:206-240`. | Render the same repo-scoped framing the payload now uses: under `umbrella`, no *child*'s feature branch can fork the board, and the rule binds the overarching project's own branches. Then invert the test — assert the umbrella block *states* the rule with its repository named, and drop the `unwanted` entry that currently forbids it. Fix the two comments in the same pass, since both still assert the umbrella-inert premise. |
| R2 | blocking | docs-gap | — | The new `CHANGELOG.md` entry misdescribes the change twice, and one of the two contradicts F5's fix in the very same commit. It says the manual documents "**one-directional staleness**" — the exact claim F5 established is false and which `project-structure.adoc` no longer makes. It also says the rule "in both the rendered marker block **and** the shipped skill, now states which repository it binds", which is not true of the marker block while R1 stands. A changelog is read at release time by people who cannot check it against the diff. | `CHANGELOG.md:44-52` vs `project-structure.adoc:199-210` (F5's fix) and `install.go:1101-1104` (R1). Both landed in `6421c95`. | Replace "one-directional staleness" with the shipped claim (a stale read can never show `DONE` falsely, though a backward move can over-report). Re-check the marker-block clause once R1 lands — it becomes true then, so the sentence needs no second edit if R1 is fixed first. |
| R3 | blocking | correctness | — | The third surface F2's fix missed, and the one that matters most for this flow's own operation: `review-protocol.md`'s box still tells every reviewer "Under the default `umbrella` layout **none of that arises**: … the worktree copy of the ticket is the only copy and reading it directly is correct." It reasons only about *the child's* branch checkout. But the reviewer reads the ticket from, and commits this review's own bookkeeping to, the **overarching** repository — the one that holds `tickets/` — and if that repository is itself on a feature branch, the stale-ticket hazard the box exists to prevent applies in full. The corrected `tickets-README.md` §0 now says exactly this; the protocol has not caught up, so the two payload files disagree. | `skill/resources/review-protocol.md:49-51` vs `skill/resources/tickets-README.md:105-112`. Step 1's own bullet (`:62-67`) carries the same absolute umbrella clause. | Keep "read it directly" as the umbrella default — it is right in the common case — but qualify it the way §0 now does: the copy you read is the overarching project's worktree, so if *that* repository is on a feature branch, read from its base branch instead. Update the step-1 bullet in the same pass. |
| R4 | non-blocking | docs-gap | fixed inline | The reframed hooks NOTE said "installing the hooks there is harmless" of a child repository, implying `pickle hooks install` can arm one. It cannot: `hookRoot()` resolves the target upward to the directory holding `pickle.toml`, so it always arms the board's repository whatever directory you run it from. Reproduced: running it from `child/` installed into the *overarching* `.git/hooks`. | `internal/cli/hooks.go:57-68`; reproduction in the umbrella throwaway workspace. | Fixed in `2d0ee7a`: the NOTE now names the repository `hooks install` arms and frames child-repo inertness as what a hand-placed shim would do. |
| R5 | non-blocking | spec-unclear | fixed inline | `tickets-README.md` §0's corrected bullet ended "the sub-bullets below apply to it exactly as they would to an in-tree child" — over-carrying in the opposite direction from F2. The merge-style preference sub-bullet is in-tree's alone: it exists because one history there carries both the code and the bookkeeping, which an umbrella board's repository does not. | `skill/resources/tickets-README.md` §0, the bullet as delivered in `6421c95`, vs the merge-preference sub-bullet directly beneath it. | Fixed in `2d0ee7a`: names the enforcement sub-bullets that do carry (hooks, origin-base check, stale-read hazard) and exempts the merge-style one. The sentence was also split in two, applying the one in-scope docs-readability suggestion. |

**Disposition summary:** 3 blocking (R1, R2, R3 — the rework scope; all three are one root cause,
F2's correction not reaching every surface that carries the claim, plus the changelog written from
the pre-fix wording), 2 non-blocking both `fixed inline` (R4, R5, in commit `2d0ee7a`). No
follow-up ticket minted; N1 and N2 from review 1 stand as recorded.

cost: estimated L, actual L — unchanged; the remaining work is three prose surfaces and one test
inversion.

**Docs-readability (step 4b).** 26 suggestions; 24 landed on prose neither the original pass nor
the rework authored, so they are out of scope for this ticket. Applied: the split of §0's
"That is not the same as the rule being inert everywhere" sentence (folded into R5's fix).
Rejected with reason: rewriting F5's "That half is a reassuring property and a hazard at once" to
"That is both reassuring and hazardous" — it reads better but drops the "half" antecedent that
F5's fix introduced precisely to separate the invariant claim from the backward-move case.

**Impact sweep (step 8).** Unchanged from review 1: no ticket in `1-to-do/` or `2-ready/` lists
T-109 in `depends-on:` or leans on its assumptions. T-050 still edits
`agents/pi/extensions/pickle-guardrails.ts` but touches the rule-1 verdict, not the header comment
F3 corrected — no collision. R1 adds `internal/install/install.go` and
`internal/install/hooks_test.go` to the rework's file set; no open ticket names either.

**2026-08-18 — review 3 (scoped re-review).** Branch `feat/T-109-layout-conditional-rules` at
`c21e4f3`, read against `main`. Scope: R1, R2, R3. Verdict: **one blocking finding — back to
`5-rework/`**. All three fixes landed and are structurally correct; R1's replacement prose then
made the mirror-image claim of the one it repaired, and it did so on four surfaces at once.

- [x] R1/R2/R3 re-verified individually (step 2, scoped)
- [x] Acceptance test re-run: all 6 checks pass; `pickle doctor` 0/0; `changelog check` clean
- [x] `just build && just test && just lint && just docs-check` clean (19 packages)
- [x] Findings recorded with severity, class and disposition per the rules §5 (step 5)
- [x] Ticket moved to `5-rework/`; `## History` appended (step 6)
- [x] Impact sweep re-checked (step 8) — unchanged; nothing depends on T-109
- [ ] Docs-readability pass — **conscious skip**: the rework's prose deltas are the four sentences
      under audit in S1 and are about to be rewritten; a readability pass on wording that is
      changing again would be discarded unread

**R1/R2/R3, re-verified:**

| finding | verdict | evidence |
|---|---|---|
| R1 (marker block + test lock) | **structurally fixed; new claim defective — see S1** | The umbrella render now says "no **child's** feature branch can fork it" and states the rule binds this project's base branch; `hooks_test.go`'s umbrella case asserts the corrected wording and rejects both old phrases (`want`: `no **child's** feature`, `bookkeeping is still committed on`, `cut in the overarching project itself would`; `unwanted`: `no feature branch can fork it`, `whenever it is ready`). Both doc comments corrected. `TestMarkerBlockGolden` still passes untouched — correctly, since only the umbrella branch changed and the golden's fixture (`alpha` at `.`) resolves in-tree. |
| R2 (CHANGELOG contradicted F5) | **fixed** | `CHANGELOG.md:50-53` now carries the shipped claim ("a stale worktree can never show `DONE` for a ticket that is not, though a ticket sent backward can show through it as further along") and no longer says "one-directional". The marker-block clause in the same entry became true when R1 landed, as review 2 predicted, so it needed no second edit. |
| R3 (review protocol vs §0) | **fixed** | `review-protocol.md:49-57` and the step-1 bullet at `:69-74` now distinguish the child's checkout (never the hazard) from the overarching project's own worktree (the hazard, if it is on a feature branch), matching `tickets-README.md` §0. Correctly hedged as "rare … but not impossible". |

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| S1 | blocking | correctness | — | R1's replacement prose over-claims the guard's *coverage* in the umbrella overarching repo, the mirror image of the defect it fixed. A branch is a "feature branch" to the guards only when it matches **a registered child's `branch_prefix`** (`onFeatureBranch`, shared by `hook.go` and `prepush.go:145`). Under `umbrella` no registered prefix describes the *overarching* project's own branches — a child's `feat/` says nothing about how the board's repository names its docs branches — so the guard's protection there is a coincidence of naming, not a guarantee. Four surfaces now assert it flatly, and the failure direction is worse than F1's: F1 sent a reader hunting for a fault that did not exist, S1 invites reliance on protection that is not there. The docs never state the prefix dependency in the hooks section at all (only `serve`'s banner spells it out, `cli-reference.adoc:1306`). | Reproduced in one umbrella throwaway (`branch_prefix = "feat/"`, child `web` at `child`): overarching repo on `feat/T-001-demo` + `git add tickets` → `hooks run pre-commit` exits **1**; same repo, same staged paths, on `docs/layout-notes` → exits **0**, guard silent. Claims: `internal/install/install.go` umbrella render ("and `pickle hooks install` run here still catches that"), `docs/user-manual/cli-reference.adoc:475` ("the guards will fire exactly as they would in-tree") and `:612` ("the guards fire on it identically"), `skill/SKILL.md:87` ("the hooks, if installed there, still catch it"), `skill/resources/tickets-README.md:112` ("every enforcement sub-bullet below — the hooks, the origin-base check, the stale-read hazard — applies to it exactly as it would to an in-tree child"). | One hedging clause per surface, not a rewrite: the guards catch it **when the overarching project's branch name matches a registered child's `branch_prefix`**, which under `umbrella` is not guaranteed — so the base-branch discipline there rests on the operator, with the hook as a bonus rather than a backstop. The `tickets-README.md` line needs one extra distinction: the *manual* origin-base check (`git diff … origin/<base>...HEAD`) has no naming dependency and does carry over unchanged; only the two hooks do. Two of the four surfaces (`cli-reference.adoc:475`, and the `:612` bullet) were authored by **this review's own inline fixes in `2d0ee7a`** — recorded here rather than quietly repaired, since the same standard applies to a reviewer's edit as to the branch's. While in the marker-block string, also unsplit the `` `pickle hooks\n  install` `` code span: CommonMark renders the break as a space so it is harmless, but no neighbouring bullet splits one. |

**Disposition summary:** 1 blocking (S1 — the rework scope), 0 non-blocking. No follow-up ticket
minted. Review 1's N1/N2 and review 2's R4/R5 stand as recorded.

cost: estimated L, actual XL — three review rounds, each finding the previous fix true in substance
and over-stated at one more surface. The estimate was not wrong about the *work*; it was wrong that
"make a rule conditional" is a prose edit, when the rule's condition turned out to be a claim about
a guard's predicate that only a reproduction could settle.

**Impact sweep (step 8).** Unchanged: nothing in `1-to-do/` or `2-ready/` lists T-109 in
`depends-on:` or leans on its assumptions; T-050 still does not collide with the guardrail header.
S1's fix touches only files already in this ticket's set.

**Note for the next pass — the pattern to break.** Three rounds have each corrected a claim about
the guards and then made a fresh unconditional claim one clause over. The reliable fix is to state
the predicate once, in the one place that owns it (`cli-reference.adoc`'s "what the guards do and do
not catch"), and have the other three surfaces point at it instead of restating it — the same
single-source discipline the rules already use for the four dispositions and the `class` vocabulary.
## History

- 2026-08-17 — created (TO DO). source: pickle ticket new
- 2026-08-17 — TO DO → READY: refined: 6 confirmed decisions, 6 tasks, hard-depends on T-108 being merged
- 2026-08-17 — plan amended inline: Task 3 also corrects SKILL.md's "registers the first child-project" claim, folded from T-108 review finding F5
- 2026-08-17 — plan amended inline: Task 4 now cites the `pickle hooks` section by heading instead of by line number — T-108's merge shifts 385/487 to 445/547 (T-108 review 2 impact sweep)
- 2026-08-17 — plan amended inline: pickup applicability audit, 10 findings, all non-blocking, 7 dispositioned inline. Acceptance check 2 was inverted (`rg -L` is `--follow`, not `--files-without-match`) and passed vacuously, so it would have rubber-stamped a no-op — corrected. Task 5 rescoped: T-108 already shipped both layout names, the default and consequence 1 in `[#install-layout]`, so only three consequences remain, added to `concepts/project-structure.adoc`'s existing in-tree section with no new file and no `include::` change. Task 2 gained `review-protocol.md:55`; Task 3 now names SKILL.md `:77` and `:271-272` explicitly; Task 6 widened to the stale "single-repo default" phrasing at `tickets-README.md:83`/`:119`. Checks 3 and 4 repaired (recursive search plus asserted phrases; two known-good "sibling extension files" hits recorded). Noted and closed: Task 3's second half confirmed required (`SKILL.md:56` contradicts `install.go:179`), no plan item forces a foreign-workspace violation, and T-067 means new xrefs need checking by eye.
- 2026-08-17 — READY → IN DEVELOPMENT: picked up
- 2026-08-17 — IN DEVELOPMENT → IN REVIEW: acceptance green: build/test/lint/docs-check clean, all 6 checks pass
- 2026-08-17 — publish approved by user: pushed `feat/T-109-layout-conditional-rules`, opened MR #54. Base `main` was pushed first — the origin-base check fired on the two unpushed `board:` commits, the exact leak this ticket documents. Awaiting review; merging is the human's.
- 2026-08-18 — plan amended inline: review 1 fixed two plan defects it found (rules §5 `fixed inline`) — the Docs-update step still demanded a new chapter registered in the `include::` tree, which Task 5's rescope had retired, and it gained F4's `CHANGELOG.md` obligation; acceptance check 2 widened from `skill/` to `skill/ agents/`, the blind spot that hid F3's residual unconditional statement in the shipped pi guardrail scaffold.
- 2026-08-18 — IN REVIEW → REWORK: review 1: 5 blocking findings — the guards are repo-scoped, not layout-scoped, so the umbrella 'never fires' claims are false (F1/F2); residual unconditional statement in the pi guardrail payload (F3); no CHANGELOG entry (F4); the one-directional-staleness invariant overclaims (F5)
- 2026-08-18 — plan amended inline: rework fixed F1–F5 on `feat/T-109-layout-conditional-rules` (commit `6421c95`) and, per F1/F2/F5's authority to correct a false confirmed decision (rules §5 `plan-wrong`), rewrote decisions 1, 4 and 5 plus the Outcome, Description, Task 5 body and acceptance checks 1 and 5 to match: the base-branch rule binds whichever repository holds `tickets/` rather than being conditional on a layout name (decisions 1, 4), and the staleness claim is now DONE-is-terminal (never falsely `DONE`) rather than strictly one-directional, since a backward move or a drop lets a stale copy over-report (decision 5). Also applied, as polish (not a finding): two docs-readability suggestions on `review-protocol.md` §1/§9 that review 1 flagged as landing on this branch's own prose. `just build && just test && just lint && just docs-check` clean; all 6 acceptance checks re-verified.
- 2026-08-18 — REWORK → IN REVIEW: fixes complete, ready for scoped re-review
- 2026-08-18 — review 2 (scoped): F1, F3, F5 verified fixed; F4's entry added but its content misstates what shipped (R2); F2's correction reached the two payload docs but not the rendered marker block (R1, with a test that locks the old wording) nor the review protocol (R3). Two non-blocking findings fixed inline in `2d0ee7a`: the repository `hooks install` actually arms (R4) and which §0 sub-bullets carry to an umbrella board repo (R5).
- 2026-08-18 — IN REVIEW → REWORK: review 2 (scoped): F1/F3/F5 fixed; F2's correction missed the rendered marker block (R1) and the review protocol (R3), and the new CHANGELOG entry misstates what shipped (R2)
- 2026-08-18 — rework: fixed R1–R3 on `feat/T-109-layout-conditional-rules` (commit `c21e4f3`). R1: MarkerBlock's umbrella render and its hooks_test.go assertions both stated the old 'inert under umbrella, full stop' claim — the test even forbade the corrected wording — fixed and inverted, reproduced against a throwaway install. R2: the CHANGELOG entry's 'one-directional staleness' phrase (self-contradicting F5's own fix in the same commit) replaced with the shipped DONE-is-terminal claim. R3: review-protocol.md's box and step-1 bullet now qualify 'read it directly' under umbrella to the overarching project's own worktree, matching tickets-README.md §0. `just build && just test && just lint && just docs-check` clean; all 6 acceptance checks, `pickle doctor` and `changelog check` re-verified.
- 2026-08-18 — REWORK → IN REVIEW: R1-R3 fixed, ready for scoped re-review
- 2026-08-18 — review 3 (scoped): R1, R2, R3 all verified fixed. One new blocking finding (S1): R1 replacement prose, plus review 2 own inline fixes, now over-claim the guard coverage in the umbrella overarching repo — a branch is a feature branch to the guards only when it matches a registered child branch_prefix, and under umbrella no configured prefix describes the board repository own branches. Reproduced: same staged tickets/ paths refused on feat/T-001-demo, waved through on docs/layout-notes.
- 2026-08-18 — IN REVIEW → REWORK: review 3 (scoped): R1-R3 verified fixed; S1 — the replacement prose over-claims guard coverage in the umbrella overarching repo (no configured branch_prefix describes its own branches), on four surfaces including two of review 2's own inline fixes
