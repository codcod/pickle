---
id: T-131
title: rework: resolve ticket status from base branch, wire in pickle doctor
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: low
cost: S
---

# T-131 — rework: resolve ticket status from base branch, wire in pickle doctor

## Outcome

After this ships, a brand-new session told "rework ticket T-NNN" under `layout = "in-tree"`
resolves the ticket's real current status from the base branch mechanically — via the same
unconditional, directory-agnostic lookup and mandatory `pickle doctor` pre-flight T-130 wired
into *implement* and *validate* — instead of the agent reading its feature branch's possibly
stale worktree copy and wrongly stopping with "must be in `5-rework/`" when the review verdict
that put it there landed on the base branch, on a branch cut earlier.

## Description

T-130 fixed this exact mirror-image hazard (T-108/T-128) for two of the three trigger
procedures — `resources/review-protocol.md` step 1 and `SKILL.md`'s inline "Procedure: implement
a ticket" — by resolving the ticket's path from base unconditionally
(`git ls-tree -r --name-only <base> -- tickets | grep -- "/T-NNN-"`, then `git show`) and adding a
mandatory `pickle doctor` pre-flight once the ticket's own feature branch is checked out. It did
not touch `resources/procedure-rework.md`, whose own step 1 still reads: *"The ticket must be in
`5-rework/` — if not, stop and explain."* — a worktree read of whatever status the checked-out
`feat/T-NNN-*` branch happens to carry, with no base-branch resolution and no `pickle doctor`
step, the same gap T-130's Description called out in `SKILL.md`'s implement procedure before that
ticket fixed it.

**Confirmed in the field.** An in-tree install (a foreign workspace, not this repo) had T-047
reviewed to a `5-rework/` verdict — bookkeeping committed on the base branch, per this project's
own "where commits land" rule (mirrored in every in-tree install). The agent, still on
`feat/T-047-*` (cut while the ticket was in `3-in-development/`), was then told "rework ticket
T-047", read its own worktree copy per `procedure-rework.md` step 1, saw `3-in-development/`, and
stopped — exactly the failure mode T-108/T-128/T-130 already diagnose and detect, on the one
trigger procedure T-130 left unpatched.

**Fix, same shape as T-130, doc-only:**

1. `resources/procedure-rework.md` step 1: replace "the ticket must be in `5-rework/` — if not,
   stop and explain" with the unconditional base-branch lookup (same `git ls-tree`/`git show`
   recipe T-130 put in `review-protocol.md` step 1) — a ticket "rework ticket T-NNN" is invoked
   on is legitimately read from base, since the branch was necessarily cut before the review
   verdict that moved it to `5-rework/` landed there. Stop-and-explain is still correct, just
   gated on the *base-branch* status, not the worktree's.
2. Add the same `pickle doctor` pre-flight T-130 put in `review-protocol.md`'s new `## 0a.` and
   `SKILL.md`'s implement preamble — `procedure-rework.md` gains an equivalent step before step 1
   (or a preamble sentence, mirroring whichever of T-130's two patterns fits this file's shape),
   run once `feat/T-NNN-*` is checked out, any stale-ticket-branch warning resolved (rebase onto
   base) before proceeding.
3. `SKILL.md`'s "Procedure: rework a ticket" section (currently just "Read
   `resources/procedure-rework.md` and follow it.") needs no independent edit if the base-branch
   resolution and `pickle doctor` step live entirely inside `procedure-rework.md` — confirm during
   refinement whether T-130's split (short-form clause in `SKILL.md` *and* the full step in
   `review-protocol.md`) is needed here too, or whether the single-file procedure makes the
   `SKILL.md` pointer sufficient as-is.

**Soft couplings (not `depends-on`, both already `6-done/`):**

- **T-130** — the precedent this ticket copies verbatim for a third trigger procedure; same
  detection (`checkStaleTicketBranch`, T-128), same doc-only shape.
- **T-128** — supplies the `pickle doctor` mechanism this ticket wires into
  `procedure-rework.md`; no Go code changes here either.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-20 — created (TO DO). source: field-use: an in-tree install had a review verdict move
  T-047 to `5-rework/` via base-branch bookkeeping; a fresh session told "rework ticket T-047"
  read its stale feature-branch worktree copy (still `3-in-development/`) per
  `procedure-rework.md` step 1's unconditional worktree read, and wrongly stopped — the same
  mirror-image hazard T-130 fixed for *implement* and *validate*, left open on *rework*.
