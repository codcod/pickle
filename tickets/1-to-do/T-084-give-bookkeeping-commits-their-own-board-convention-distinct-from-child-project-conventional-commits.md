---
id: T-084
title: give bookkeeping commits their own board: convention, distinct from child-project Conventional Commits
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: medium
cost: M
---

# T-084 — give bookkeeping commits their own board: convention, distinct from child-project Conventional Commits

## Description

Measured on this repo's own history (2026-08-07 audit): 188 of 301 commits are overarching
bookkeeping (ticket/board mutations), and virtually all of them carry the scope `(tickets)` —
`docs(tickets): …`, `chore(tickets): …`. Checked against the real diffs: the scope is *accurate*
(187 of 188 touch only `tickets/`), so it isn't a mislabeling bug. It's a category-fit problem:
Conventional Commits' `type(scope)` grammar names **what area of the product changed** — a
bookkeeping commit isn't a product change, it's a **state transition of a ticket**. Forcing every
such commit through that grammar either produces an uninformative scope (`tickets`, always) or an
artificial one (`docs`/`skill`/`agents`/`other`, picked by which directory happened to be touched)
that still misdescribes what the commit actually is.

The symmetric half of the same investigation: this repo's child-project commits (the actual
`pickle` product changes, on `feat/T-NNN-<slug>` branches) are squash-merged today, which is fine
for a child registered at a nested path (it has, or could have, its own separate repo whose log
is naturally clean) but is actively costly for `pickle`'s own setup, where the child **is** the
repo root (`path = "."` in `pickle.toml`). Thought experiment that surfaced it: if `pickle` were
developed as a nested child inside a separate `pickle-meta` repo, the meta repo's log would show
`board: T-NNN …`-style bookkeeping (fine, expected) and `pickle`'s own repo would independently
show a clean, granular `feat`/`fix`/`ci`/`test`/`refactor` history — because bookkeeping would
never have been mixed into it to begin with. Since there is no second repo here, squashing the
child's branch on merge is the only thing destroying that granularity: `git log --invert-grep
--grep '^board:'` on `main` should recover exactly what that hypothetical standalone `pickle`
repo's log would look like, and today it can't, because every ticket's internal commit sequence
(whatever `feat`/`fix`/`ci` structure it had) is flattened into one squash commit carrying one
type/scope for the whole ticket.

**Decisions reached in discussion (2026-08-07), to be locked in refinement:**

1. **Bookkeeping commits stop using Conventional-Commit `type(scope)` entirely.** New flat
   format: `board: T-NNN[, T-MMM …] <verb phrase>`, state moves phrased with an arrow (`picked
   up → in development`), content-only edits as a plain clause, multiple tickets touched in one
   sitting listing all ids. The ticket id leads the subject (it's the actual subject of a
   bookkeeping commit, unlike a code commit where the id is a trailing cross-reference).
2. **Granularity — fold only when adjacent.** A content-only annotation (no board move) folds
   into an *adjacent* same-sitting bookkeeping commit for the same ticket, when one exists (no
   branch switch in between); otherwise it stays its own commit. Checked against T-072's real
   9-commit sequence: this rule is a strict improvement but has narrow effect in practice, because
   most of that sequence already brackets a real branch switch or a distinct trigger invocation
   (pickup/review/rework/re-review), which must stay individually committed — collapsing those
   would reintroduce the exact uncommitted-bookkeeping-crosses-a-branch-switch hazard `pickle
   hooks install` (T-057) and the origin-base check (T-072) exist to prevent. A deliberately
   bigger lever (putting the feature branch in its own `git worktree`, decoupling bookkeeping
   commit timing from branch-switch safety entirely) was considered and set aside for a possible
   follow-up ticket rather than folded into this one.
3. **Root-path child (`path = "."`) defaults to preserving individual commits on merge**
   ("rebase and merge", not "squash and merge") instead of today's squash default. A child
   registered at a nested path is unaffected — squashing there doesn't cost the same thing, since
   its own history isn't sharing a log with bookkeeping. Consequence: the WIP commits that
   survive onto `main` must themselves be well-formed — the **Finish** step gains a tidy-up
   sub-step (interactive rebase into a small number of atomic, correctly typed/scoped commits)
   before the commit sequence is presented for approval, replacing today's "write one commit
   message" with "curate the commit sequence that will actually land".

**Open for refinement, not yet decided:**

- Is decision 3 pure agent/operator guidance (prose in the skill payload + this repo's
  `AGENTS.md`), or does it need a `pickle.toml` surface (e.g. a per-child `merge_strategy` key)
  so tooling (`doctor`, `pickle serve`) could one day see or enforce it? Leaning prose-only for
  this ticket, mirroring how T-072 stayed prose-only and spawned T-082 for the mechanical half —
  a mechanical enforcement of decision 3 (if wanted) would be a similar follow-up, not this
  ticket's job.
- Exact wording/placement in `resources/tickets-README.md` (rules §0 and the Finish-adjacent
  text), `resources/review-protocol.md` (step 9), `resources/TEMPLATE.md` (Finish section), and
  this repo's own `AGENTS.md` ("Branch & commit" / "Commit policy" / "Where commits land").
- Whether `board:` commits need any mention in `resources/review-protocol.md`'s reviewer-facing
  checklist (a reviewer reading bookkeeping history to reconstruct what happened before a review).

**Soft couplings:** T-057 (the `pre-commit` hook guarding bookkeeping-on-feature-branch) and
T-072/T-082 (the origin-base check and its proposed `pre-push` guard) all protect the same
invariant this ticket's decision 2 leans on — none of them change, and the three-dot
`origin/<base>...HEAD` check stays valid unchanged whether the child's merge is squash or
rebase-and-merge. T-022 and T-036 are the precedent for filing rather than hand-editing a
payload-prose change to `skill/`.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-07 — created (TO DO). source: pickle ticket new
