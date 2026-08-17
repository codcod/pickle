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
follow a rule that cannot apply to them, and a reader in the in-tree layout gets the full
consequence list spelled out: every command that reads tickets reports the checked-out branch's
copy, that staleness is one-directional, CI fires on every bookkeeping push, and the base
branch's history interleaves board moves with code. The user manual gains a section naming both
layouts, stating which is the default, and stating what choosing in-tree costs.

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
- The staleness is **one-directional**: it can only under-report progress, never falsely claim
  `DONE`. That makes it quiet and easy to shrug off, which is why it is worth stating.
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

1. **Every affected rule becomes conditional on the recorded layout; none is deleted.** The
   base-branch bookkeeping rule stays true, mandatory and fully stated for the in-tree layout.
   What changes is that it is marked as not applying to the umbrella layout, where no feature
   branch can fork the board.
2. **The foreign-workspace test binds every sentence added under `skill/`.** No "this repo"
   meaning pickle's own; no count or claim drawn from a corpus the reader does not have; no
   ticket id the reader is told to go and look up; and no path that resolves only in pickle's
   source tree — phrase paths relative to the skill the reader is holding.
3. **`payload_lint_test.go` staying green is necessary but not sufficient.** It matches four
   mechanical shapes and cannot judge what a sentence means; the judgement remains the author's.
4. **Nothing in the enforcement family is deleted or deprecated.** The guards from T-057, T-072,
   T-082 and T-100 remain correct and load-bearing in the in-tree layout. The documentation
   explains that they are inert in the umbrella layout, so an operator is never left wondering
   why a guard never fires.
5. **The manual states the default explicitly and gives the in-tree consequence list in full.**
   That list is: every reading command reports the checked-out branch's copy; the staleness is
   one-directional (it can only under-report progress, never falsely claim `DONE`); CI fires on
   every bookkeeping push, with a `paths-ignore` entry for the ticket tree recommended as the
   mitigation; and the base branch's history interleaves board moves with code.
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

1. the staleness is **one-directional** — it can only under-report progress, never falsely claim
   `DONE`, which is what makes it quiet and easy to shrug off;
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
   decision 5 — per-branch reads, one-directional staleness, CI-on-every-bookkeeping-push with
   the `paths-ignore` mitigation, and interleaved history.
6. **The guards' applicability is documented:** the `pickle hooks` section states both that the
   guards matter in in-tree and that they are inert in umbrella.

### Docs update (mandatory when user-facing)

This ticket *is* the docs update. Both surfaces change: the **payload** under `skill/` (which
ships to other projects and is bound by decision 2) and the **user manual** under
`docs/user-manual/`. The new conceptual chapter must be registered in the manual's `include::`
tree, or `just docs-check` will not see it.

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

<!-- empty until IN REVIEW -->

## History

- 2026-08-17 — created (TO DO). source: pickle ticket new
- 2026-08-17 — TO DO → READY: refined: 6 confirmed decisions, 6 tasks, hard-depends on T-108 being merged
- 2026-08-17 — plan amended inline: Task 3 also corrects SKILL.md's "registers the first child-project" claim, folded from T-108 review finding F5
- 2026-08-17 — plan amended inline: Task 4 now cites the `pickle hooks` section by heading instead of by line number — T-108's merge shifts 385/487 to 445/547 (T-108 review 2 impact sweep)
- 2026-08-17 — READY → IN DEVELOPMENT: picked up
- 2026-08-17 — plan amended inline: pickup applicability audit, 10 findings, all non-blocking, 7 dispositioned inline. Acceptance check 2 was inverted (`rg -L` is `--follow`, not `--files-without-match`) and passed vacuously, so it would have rubber-stamped a no-op — corrected. Task 5 rescoped: T-108 already shipped both layout names, the default and consequence 1 in `[#install-layout]`, so only three consequences remain, added to `concepts/project-structure.adoc`'s existing in-tree section with no new file and no `include::` change. Task 2 gained `review-protocol.md:55`; Task 3 now names SKILL.md `:77` and `:271-272` explicitly; Task 6 widened to the stale "single-repo default" phrasing at `tickets-README.md:83`/`:119`. Checks 3 and 4 repaired (recursive search plus asserted phrases; two known-good "sibling extension files" hits recorded). Noted and closed: Task 3's second half confirmed required (`SKILL.md:56` contradicts `install.go:179`), no plan item forces a foreign-workspace violation, and T-067 means new xrefs need checking by eye.
- 2026-08-17 — READY → IN DEVELOPMENT: picked up
