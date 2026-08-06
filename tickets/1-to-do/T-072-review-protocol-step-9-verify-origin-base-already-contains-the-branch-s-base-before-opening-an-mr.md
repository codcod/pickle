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
   (`pre-push`) could refuse a push whose range contains `tickets/` paths on a feature branch — but
   note the failure here was the *absence* of a push, not a bad one, so `pre-push` on the feature
   branch would not have caught it. A `pickle board audit`-style publish check
   (`pickle publish-check`?) is the other shape. Both are materially bigger than a protocol line
   and would need their own ticket; refinement should **scope this one to the prose** and record
   whether a mechanical follow-up is worth filing.
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

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

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
