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

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-17 — created (TO DO). source: pickle ticket new
