---
id: T-092
title: finalize a ticket after its PR merges: a step 10 for the review protocol
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: low
cost: S
---

# T-092 — finalize a ticket after its PR merges: a step 10 for the review protocol

## Outcome

After this ships, an agent given "PR #NNN merged" (or "finalize T-NNN") for a child-project ticket
follows one procedure that verifies the merge independently, appends the merge History line with
the real SHA, and — as a **mandatory gate, not an optional step** — runs `pickle board audit`
before any bookkeeping is committed, instead of the current gap: `resources/review-protocol.md`
step 9 stops the instant the MR is opened and never says what happens once the human reports back.

## Description

`resources/review-protocol.md`'s step 9 ends at *"push and create the merge request … merging is
always the human's"*. Nothing in the skill then tells the agent what to do once the human reports
the merge happened — the procedure simply stops, and everything that follows (recording the real
merge SHA, regenerating the board, cleaning up the branch) is left to be improvised from memory
every time.

That gap is not hypothetical: this ticket is filed directly from a session that had to improvise
it for T-081 (`tickets/6-done/T-081-…md`), and improvising it **caught a real, already-shipped
bug**. T-081's own review had committed its bookkeeping with explicit pathspecs (rules §0) naming
the file's *new* path (`6-done/`) and the regenerated board, but not the *old* path
(`4-in-review/`) the ticket-move rename had deleted — `git add <new path>` succeeds without ever
naming the delete, so nothing objected. The stale copy stayed tracked in `HEAD`, `board audit`
stayed clean because it audits the **worktree**, not `HEAD`, and the corruption surfaced only two
steps later, when merging the PR restored the stale file into the worktree and `board audit`
finally reported `duplicate id`. Fixed live, in commit `7e3dbb2`, only because the post-merge
sequence happened to run `board audit` before pushing anything. (The staging half of that same
incident is filed separately as **T-091** — this ticket is the other half: making the post-merge
check that caught it a named, mandatory step instead of something an agent happens to think of.)

**Proposed step 10, `resources/review-protocol.md`** (only applicable to a child-project ticket —
a bookkeeping-only ticket has nothing to merge):

1. **Verify the merge independently** rather than trust the report at face value —
   `gh pr view --json state,mergedAt,mergeCommit,mergedBy` (or the host's equivalent) — and record
   the real merge commit SHA and the actual strategy used (merge commit / squash / rebase —
   inspect parent count or commit list; don't assume the requested strategy was honoured).
2. **Pull the base branch** (`git checkout <base> && git pull --ff-only`).
3. **Append the merge History line** now that the real SHA/PR ref is known —
   `merged to <base> (<MR ref>, <short sha>, <url>); <strategy>, N commits kept` (rules §1's
   shape).
4. **`pickle board sync` then `pickle board audit` — a hard gate.** Do not proceed to commit
   anything if audit is not clean; this is the step T-081's incident shows is not optional.
5. **Commit + push the bookkeeping** on the base branch (explicit pathspecs, the `board: T-NNN
   merged` form — rules §0).
6. **Delete the feature branch**, locally and on the remote — its last legitimate use was the
   merge.
7. **Re-run the child's configured build/test/lint/docs commands** on the merged base as a live
   sanity check beyond CI.
8. **Conclude any impact-sweep deferral** the review left open pending "the merge landing" (T-081's
   review deferred exactly one, against T-085, for precisely this reason).
9. **Report + suggest the next ticket** (today's step 9 tail, relocated here since it currently
   fires before the merge exists).

**New `SKILL.md` trigger**, in "When to use": *"the PR/MR for T-NNN merged" / "finalize T-NNN"* →
*Procedure: finalize a merged ticket* (step 10 of the review protocol, run standalone since the
review itself already concluded at step 6).

**Deliberately out of scope, named as a candidate follow-on rather than folded in:** steps 3–5 are
exactly the class of "prose an agent must get right from memory" that T-081 spent its own diff
turning into mechanically-checked data. A `pickle ticket merged <id> --pr <ref> --sha <sha>`
subcommand that appends the line, syncs, and audits atomically — mirroring how `ticket move`
already bundles move + History + board-regen — would remove the chance of hand-writing step 3
wrong, or forgetting step 4. Whether to build that command now or leave step 10 as an agent
procedure over existing primitives (`board sync`, `board audit`, a hand-written History line) is a
refinement decision, not decided here.

Soft coupling: **T-091** (the incomplete-bookkeeping-commit footgun) is the other half of the same
incident — that ticket fixes/considers the *staging* mistake at its source (`move.Result` naming
both paths); this ticket makes the *post-merge audit* that already catches it mandatory. Neither
blocks the other; either alone is an improvement, together they close the incident from both ends.

Soft coupling: **T-093** (a release procedure for the skill) reads a ticket's merge History line
to build its changelog-completeness sweep — a cleanly recorded, verified merge line (this
ticket's step 3) is exactly the ground truth that sweep depends on. Not a hard dependency: T-093
can read whatever merge lines already exist; this ticket just makes new ones more trustworthy.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-12 — created (TO DO). source: chat design discussion following T-081's finalize-and-
  publish sequence, where the gap this ticket closes was hit live (see Description). Graded
  `medium`/`low`/`S` against the backlog: a docs/protocol-only change (one new review-protocol
  step, one new SKILL.md trigger), no code, no new config surface — comparable to T-091's
  `medium`/`low`/`S-M`, the other half of the same incident, slightly cheaper since this ticket
  adds no new type. Impact `medium`, not `low`: the gap it closes already produced one real,
  shipped-then-caught bug, and the fix it proposes is a mandatory gate, not a nice-to-have
