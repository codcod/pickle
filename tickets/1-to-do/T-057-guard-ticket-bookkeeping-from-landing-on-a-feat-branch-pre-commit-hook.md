---
id: T-057
title: guard ticket bookkeeping from landing on a feat/ branch (pre-commit hook)
project: pickle
depends-on: []
spawned-by: [T-054]
impact: medium-high
complexity: medium
cost: M
---

# T-057 — guard ticket bookkeeping from landing on a feat/ branch (pre-commit hook)

## Description

The flow's commit policy splits every change in two: **code** goes on the child-project's
`feat/T-NNN-<slug>` branch, **bookkeeping** (`tickets/`, `BOARD.md`) is committed on the base
branch. Nothing enforces that split, and it has now been violated three times in this repo:

| when | what | repair |
|---|---|---|
| T-053 | two bookkeeping commits rode the branch; the squash-merge ate them | `59dc0fd docs(tickets): restore T-053 bookkeeping after the squash merge` |
| T-054 | the in-review move committed on the branch | caught as review finding **Q1**; branch reset, commit cherry-picked to `main` |
| T-054 again | `T-056`'s ticket file committed on the branch **while closing the review that flagged Q1** | noticed only because `pickle board audit` reported 55 tickets instead of 56 |

Three occurrences, one of them immediately after the same mistake was written up as a finding,
is not inattention — it is a missing guardrail. The failure is silent at the moment it happens
(`git commit` succeeds, the board still renders) and only surfaces later, either as a wrong
ticket count or as bookkeeping destroyed by a squash-merge.

### This is not a self-hosting quirk — it affects default installs

The obvious dismissal is "pickle self-hosts, so the overarching repo *is* the child; a normal
install keeps `tickets/` in a different repo from the code, where this cannot happen." That is
wrong. `pickle install` registers the first child with **`--path` defaulting to `.`**
(`internal/install/install.go:95`; documented at `cli-reference.adoc:69`), so the **default
install is single-repo** — `tickets/` and the code share one tree and one branch namespace.
Every default installation has this hazard; pickle's own repo is simply where it has been
observed, because it is the one being worked in daily.

### Design question the refinement must settle: hook or guardrail?

Pickle already ships a guard for git behaviour — the pi extension
`agents/pi/extensions/pickle-guardrails.ts`, embedded in the payload and installed to
`.pi/extensions/`. It enforces staging discipline, a publish gate and a self-modify rule. A
fourth rule there would be cheap and consistent with existing machinery.

**But a pi extension only guards a pi session.** All three violations above were made by an
agent shelling out to `git` outside any such hook. A `pre-commit` hook guards the repository
regardless of who or what is committing — agent, human, or script — which is the property
actually needed here. The likely answer is *both*: the hook as the real enforcement, a
guardrail rule as the fast, in-session explanation. Refinement must decide, and must confront
what makes hooks awkward:

- `.git/hooks/` is not version-controlled, so the hook has to be **installed** — `core.hooksPath`
  pointing at a tracked directory is the usual answer, but it is global to the repo and would
  collide with a project that already sets it (Husky, pre-commit.com, Lefthook).
- `pickle install` writing into a user's git config is materially more invasive than anything
  it does today, so it likely needs to be **opt-in** (`--hooks`) and removed by
  `pickle uninstall`.
- The rule needs an **escape hatch** (`--no-verify` already exists, but the message should name
  the legitimate case) and must not fire on the base branch, on a detached HEAD, during a
  rebase/merge, or in a repo where `tickets/` is genuinely part of the product — which is
  exactly pickle's own case for `skill/`, though not for `tickets/`.
- The branch-name test must come from config (`branch_prefix` in `pickle.toml`, default
  `feat/`), not a hardcoded literal.

A cheaper, near-zero-risk subset worth pricing separately during refinement: have
**`pickle board audit` fail** when the current branch matches `branch_prefix` and `tickets/`
has staged or committed changes not on the base. That reuses machinery that already exists and
already runs, needs no git config, and `DESIGN.md` §7 already anticipates wiring audit into
"CI + a pre-commit hint".

### Soft couplings

- **T-050** — the pi guardrail's verdict semantics (hard block vs confirm). If this ticket adds
  a fourth rule there, it inherits whatever T-050 decides; a hard block for a rule this
  mechanical is likely right, but T-050 is the ticket that reasons about that choice. No hard
  dependency; whichever lands first, the other adapts.
- **T-046** — same self-host-awareness family, different subsystem (doctor/upgrade). Worth
  reading together, not merging.
- Lineage: `spawned-by: T-054`, whose review raised the violation as finding Q1 and then
  reproduced it.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-28 — created (TO DO). source: chat — user request for a pre-commit hook rejecting
  `tickets/` paths on a `feat/` branch, after the T-054 review caught the violation (Q1) and
  it then recurred while that same review was being closed
