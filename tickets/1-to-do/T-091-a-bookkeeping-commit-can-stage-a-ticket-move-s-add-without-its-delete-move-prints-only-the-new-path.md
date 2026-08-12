---
id: T-091
title: a bookkeeping commit can stage a ticket move's add without its delete: move prints only the new path
project: pickle
depends-on: []
spawned-by: [T-081]
impact: medium
complexity: low
cost: S-M
---

# T-091 — a bookkeeping commit can stage a ticket move's add without its delete: move prints only the new path

## Outcome

After this ships, `pickle ticket move` tells the operator **both** paths a bookkeeping commit has
to stage — the file it wrote and the one it removed — so a commit made with the explicit pathspecs
rules §0 mandates can no longer land the add without the delete, leaving a duplicate ticket id in
git history that the worktree-based `board audit` cannot see.

## Description

Rules §0 requires bookkeeping commits to use **explicit pathspecs** (`git add <paths>`, never
`git add -A`/`.`), so that deliberately-untracked material is never swept in. `pickle ticket move`
is a **rename**: it writes the ticket at its new status path and removes the old one
(`internal/move/move.go:131,134`). But `move.Result` carries only the new path
(`Path string // new path, relative to root`, `move.go:27`) and the CLI prints only that
(`internal/cli/ticket.go:66`: `moved T-081: IN REVIEW → DONE  (tickets/6-done/T-081-…md)`).

So the operator is handed exactly one of the two paths a complete rename needs, and must supply
the other from memory. Nothing then objects: `git add` with the paths you *did* name succeeds,
and the commit lands with the add and without the delete.

**Deliberately not the cause: any awkwardness in git.** `git add <old path>` stages a deletion
perfectly well when the file is gone from the worktree but still in the index — verified on git
2.55.0: exit 0, `D a/f.md` staged. (An earlier draft of this ticket claimed it refuses; that
reading came from a `fatal: pathspec … did not match any files` produced by naming a path already
removed with `git rm`, which is a different situation.) The fix therefore needs no new git idiom
and no change to §0's explicit-pathspec rule: **naming the old path is sufficient, and the only
reason it goes unnamed is that the tool never prints it.**

**Observed, 2026-08-11 (T-081's own review — this ticket's `spawned-by`).** The reviewer moved
T-081 to `6-done/` with `pickle ticket move`, then committed with explicit pathspecs naming the
added file and the board, but not the deleted `tickets/4-in-review/T-081-…md`. The commit
(`ba01f41`) kept the old copy tracked. Nothing complained: the **worktree** was correct, so
`pickle board audit` reported `90 tickets, 0 error(s), 0 warning(s)`. The corruption existed only
in git history, and surfaced two steps later, when merging PR #30 restored the stale file into the
worktree and audit finally reported `duplicate id — also at 4-in-review/T-081-…md`. A fresh clone
would have shown the same. Fixed in `7e3dbb2`.

**The underlying gap is that `board audit` audits the worktree, not the commit.** Every invariant
it checks — unique ids, status-matches-history, the board matching the tickets — can hold in the
working tree while being broken in `HEAD`. That is fine for the flow's normal use, and this ticket
does not propose making the audit git-aware in general; it proposes closing the one hole the
flow's own instructions open, at the cheapest point.

Candidate shapes, for refinement to choose between (they are not exclusive, and the first alone
may be enough):

1. **Make the move name both paths.** Add `OldPath` to `move.Result` and print it — and, better,
   print the ready-to-paste line, since one plain `git add` covers the rename:
   `git add tickets/6-done/T-…md tickets/4-in-review/T-…md tickets/BOARD.md`. Smallest change,
   needs no new git idiom (see above), and it lands the fix at the moment the mistake is made.
   `pickle ticket new` has the same shape at lower stakes (it prints one path and also rewrites
   the board), so refinement should decide whether both commands grow the same line.
2. **Have the `pre-commit` hook catch it.** T-057's hook already inspects `git diff --cached` for
   `tickets/` paths (`internal/hook/hook.go:347–383`) to enforce branch placement, so the staged
   file list is already in its hands: a staged ticket add whose id also exists at another status
   path in `HEAD`, with no corresponding staged delete, is a mechanical check on data it has
   already read. Refinement should weigh this against the hook's deliberate narrowness.
3. **An `--staged`/`HEAD` audit mode** — named for completeness and probably the wrong trade:
   it duplicates the tree walk against a git object source for one failure mode.

Soft coupling: **T-082** (pre-push hook refusing a feature-branch push carrying `tickets/` paths)
is the same *family* — git hygiene around bookkeeping — but a different failure: T-082 catches
bookkeeping on the **wrong branch**, this catches an **incomplete** bookkeeping commit on the
right one. If both land, shape 2 above and T-082's guard are two checks in one hook and should be
built to share its staged-path plumbing; neither blocks the other.

Soft coupling: **T-071** (harden the PATH probe) and **T-050** (pi staging gate) both touch hook
behaviour; if shape 2 wins, sequence with them rather than merging.

Soft coupling: **T-092** (detect an unfinalized merge) is the other half of the same incident that
spawned this ticket — that ticket makes the omission *detectable* after the fact (generalising the
DONE-without-a-merge-line audit check, and running `board audit` in CI, which nothing does today);
this ticket fixes the staging mistake at its source (`move` naming both paths). Neither blocks the
other.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-12 — created (TO DO). source: pickle ticket new — filed from T-081's post-merge
  bookkeeping (`spawned-by: [T-081]`), where the failure was hit for real: `ba01f41` staged a
  ticket move's add without its delete, `board audit` stayed clean because it reads the worktree,
  and the duplicate id only surfaced when the merge of PR #30 restored the stale file. Graded
  against the backlog: impact `medium` (silent, and it corrupts git history rather than the
  worktree, so it is found late — but it is recoverable in one commit and has been seen once),
  complexity `low`, cost `S-M`, on the assumption refinement picks shape 1; shape 2 would raise
  both. Adjacent T-082 (`medium`/`medium`/`M`) is the same family but a distinct failure, and is
  deliberately not absorbed here
