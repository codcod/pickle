---
id: T-108
title: pickle install --in-tree: record the board layout in config, and warn when an in-tree board is read from a feature branch
project: pickle
depends-on: []
spawned-by: []
impact: high
complexity: medium
cost: M
---

# T-108 — pickle install --in-tree: record the board layout in config, and warn when an in-tree board is read from a feature branch

## Outcome

After this ships, `pickle install` no longer infers where the board lives: the umbrella layout
is the default, `--in-tree` explicitly selects the layout where the board sits inside its sole
child's own repository, and the choice is recorded as `layout` in `pickle.toml`. `pickle doctor`
reports an error when the recorded layout contradicts the registered children, and `pickle
upgrade` back-fills the key for existing projects by inference rather than requiring a migration
command. `pickle serve` states which layout it is reading and warns when an in-tree board is
being read from a non-base branch — the one situation in which the board UI can show a ticket
status that is silently out of date.

## Description

Ticket status is a single-valued fact about a project, but it is stored as a file's location
inside a git working tree. Git's purpose is to let branches disagree about file locations, so
any layout where the board lives inside a repository that gets feature branches will fork the
board, and no git mechanism can say which fork is authoritative — that is a convention, not a
property.

**pickle's primary layout does not have this problem.** In the umbrella layout the overarching
project holds `tickets/` and `pickle.toml`, and the child-projects are separate repositories
registered by path; children know nothing about the umbrella, by design. Cutting a feature
branch in a child cannot fork the umbrella's board, because the board is not in that repository.
This is the layout `internal/vcs`'s `ChildState`/`Advice` already serve, with their "nested git
repository that this repository does not ignore" guidance.

**The in-tree layout is the exception, and it accepts a known price.** When the overarching root
and the sole child are the same repository (`path = "."`, described in-code as "the single-repo
default"), the board is inside the branching medium. Tickets become visible to anyone who clones
the code, which is genuinely valuable — but bookkeeping lives on the base branch only, so any
feature branch cut before the latest bookkeeping commit carries a stale copy, and every pickle
command that reads tickets reports that stale copy without saying so.

**This is demonstrated, not theoretical.** Running `pickle serve` against a worktree checked out
at `8b4caa6` — the real tip of T-065's feature branch, after `main` had already moved that ticket
to `6-done/` and merged it — rendered:

```
GET /t/T-065  ->  <dt>status</dt><dd>IN DEVELOPMENT</dd>
GET /         ->  T-065 under the "IN DEVELOPMENT 1/1" lane
```

while `main`'s own `serve` showed `DONE` at the same instant. Identical markup, identical route,
opposite answers, and nothing on the page distinguishes them. The drift is one-directional:
because bookkeeping only lands on the base branch, a stale worktree can only under-report
progress, never falsely claim `DONE`.

**Why the layout must be recorded rather than detected.** Both facts needed to warn accurately —
which layout this is, and what the base branch is called — are currently unavailable. There is
no `layout` key at all, and the base branch is *guessed* from a hardcoded list at
`internal/hook/prepush.go:206` (`for _, name := range []string{"main", "master"}`), so a project
based on `develop` or `trunk` would be silently mis-served by any check built on that guess.
Recording the layout at install time, when the human knows the answer, replaces both guesses
with one stated fact.

**Why `--in-tree` and not `--sibling`.** In that layout nothing is a sibling: the tickets are
*inside* the code repository. "Sibling" was also used during design for the opposite arrangement
(a board directory beside the code), so the term would arrive in the manual already ambiguous.
`--in-tree` names where the tickets live.

**Scope boundary.** The warning ships on `pickle serve`, which is where the incorrect status was
actually observed. `board state --json` is deliberately left unchanged here so this ticket raises
no wire-format question against T-065's versioned envelope; extending it is a follow-up if
wanted.

**Supersedes T-107**, which proposed printing the checked-out branch name derived from the
`main`/`master` guess. This ticket does the same job from a recorded fact instead, so T-107 is
dropped rather than kept alongside it.

**Soft couplings (not `depends-on`):**

- **T-109** — the payload and manual rewrite that makes the base-branch rule layout-conditional.
  It hard-depends on this ticket, because documentation may only describe behaviour that exists.
- **T-065** (done) — supplied `internal/state` and the `health` shape; its envelope is what this
  ticket deliberately does not modify.
- **T-057, T-072, T-082, T-100** (all done) — the base-branch enforcement family. They remain
  correct and load-bearing in the in-tree layout, and are vacuous in the umbrella layout; this
  ticket records the fact that distinguishes those two cases, and T-109 documents it.
- **T-046** (done) — made `doctor` self-host-aware, so `doctor` is the established home for a
  layout-conditional check.
- **`docs/proposals/post-merge-done-move.adoc`** — unaffected and still live: the in-tree layout
  persists, so the `4-in-review` to `6-done` timing question it raises is untouched.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-17 — created (TO DO). source: pickle ticket new
