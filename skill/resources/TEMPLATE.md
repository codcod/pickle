---
id: T-NNN
title: <short title>
project: <child-project name>   # which registered child-project this ticket targets
                                # (a name from `pickle project list` / pickle.toml)
depends-on: []              # hard dependencies only (other T-NNN ids, any child-project); [] if none
impact: critical|high|medium|low
complexity: high|medium|low
cost: S|M|L|XL               # implementation effort — see tickets/README.md §3
# unrefined tickets may use adjacent-pair ranges (low-medium, medium-high, S-M, M-L, L-XL);
# refinement collapses them to a single value
---

# T-NNN — <title>

## Description

<The current spec of the feature, in prose. Kept up to date as understanding evolves —
this is what the ticket *is*, in its latest form. Note soft couplings to other tickets
here (cross-reference by id) — hard dependencies go in `depends-on:` frontmatter instead.
The `project:` frontmatter names which child-project this feature targets; a cross-child
soft coupling (e.g. a frontend feature that pairs with a backend one) is noted here, a hard
one goes in `depends-on:`.>

## Implementation Plan

<Empty while in TO DO. Filled in during exploration/refinement; must satisfy the READY
gate (tickets/README.md §4) before the ticket moves to `2-ready/`. Structure:>

### 0. Feature branch (mandatory)

Before any change, create a feature branch **inside the target child-project's repo**
(the `project:` frontmatter names it; its path is in pickle.toml):

```
cd <child-project path>
git checkout main            # or the agreed base
git checkout -b feat/T-NNN-<slug>
```

Do all work on this branch, committing locally as you go (WIP commits are encouraged —
crash-safety and diffable rework rounds). **Never push the child-project or open a merge
request without explicit user approval** (the project's commit policy): end with a summary
and a suggested commit message (see Finish, below); once the user approves, finalize the
branch (squash to the approved commit or keep the history — the user chooses), push, and
open the merge request — **merging is always the human's.**

> If this ticket depends on an un-merged branch (in any child-project), **stop and tell the
> human** rather than building on top of it.

### Prerequisite gate (hard)

<Any preconditions that must hold before starting — merged branches, clean tree, other tickets
in `6-done/` and merged in their child (should already be reflected in `depends-on:`). Delete
if none.>

### Confirmed design decisions (do not deviate without asking)

<Numbered, unambiguous decisions the implementer must honour. Pull any project-wide decisions
from the project's own docs / `AGENTS.md`.>

### Tasks

#### Task 1 — <name>
<What to build, where, and how. Reference exact paths in the target child-project.>

#### Task 2 — <name>
<…>

### Acceptance test

<Concrete, runnable checks that prove the ticket is done. Use the child-project's configured
build/test/lint/docs commands and the specific behaviour to exercise, with expected results.
A reviewer must be able to re-run these verbatim.>

### Docs update (mandatory when user-facing)

<Which docs to add/update and where to register them. Delete if the ticket ships no
user-facing surface.>

### Finish (mandatory)

1. Acceptance test green; the child-project's build/validate commands clean.
2. Docs updated and registered.
3. Write a **summary** of everything done (files touched, decisions made, anything deferred).
4. Suggest a **Conventional Commit message** for the human: `<type>(<scope>): <description>`
   (Conventional Commits; `<scope>` is optional — per spec, omit the parens entirely rather
   than filling them with a placeholder like `all` when the change is genuinely broad and
   has no single scope. `<scope>` must never be the ticket id itself; the ticket id is
   appended in brackets at the end of the subject line), e.g.:

   ```
   <type>[(<scope>)]: <concise summary> (T-NNN)

   <body — what and why>
   ```

5. Commit locally on the ticket branch, but do **not** push or open a merge request without
   user approval. Present the commit message; only after approval finalize the branch
   (squash or keep history — the user chooses), push, and open the merge request (merging is
   the human's). Hand back to the user.

## Review

<Empty until IN REVIEW. Filled in by the review protocol (`review-protocol.md`): a findings
table (severity: blocking/non-blocking, description, suggestion), the verdict, notes from any
scoped re-review, and pointers to any new tickets spawned from non-blocking findings.>

## History

<!-- append-only; newest last. One line per status transition, dated YYYY-MM-DD, in the form
     `OLD → NEW: one-clause reason`. The first line is `created (TO DO). source: …`; a human
     merge is recorded as `merged to <base> (<MR ref>)`. Examples:
     - 2026-07-13 — TO DO → READY: implementation plan complete
     - 2026-07-14 — READY → IN DEVELOPMENT: picked up, branch feat/T-NNN-<slug>
     - 2026-07-15 — IN DEVELOPMENT → IN REVIEW: acceptance test green
     - 2026-07-16 — IN REVIEW → DONE: review clean; non-blocking → T-MMM
     - 2026-07-17 — merged to main (MR !12)
-->

- YYYY-MM-DD — created (TO DO). source: <chat | review of T-xxx | audit | idea>
