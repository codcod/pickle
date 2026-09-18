---
id: T-130
title: validate/implement: resolve ticket status from base branch, wire in pickle doctor
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: low
cost: S
---

# T-130 — validate/implement: resolve ticket status from base branch, wire in pickle doctor

## Outcome

After this ships, a brand-new session told "validate ticket T-NNN" or "implement ticket T-NNN"
under `layout = "in-tree"` locates the ticket's real current status from the base branch by
construction — never by assuming a status directory or trusting a checked-out feature branch's
stale worktree copy — and is pointed at `pickle doctor -v` as a mandatory pre-flight step that
already knows how to name the exact mismatch (T-128) and its fix (rebase onto base).

## Description

T-108 documented the in-tree "mirror-image hazard" (bookkeeping lands on base, so a feature
branch cut earlier carries a stale ticket file) and T-128 built the mechanical detection for it
in `pickle doctor` (`checkStaleTicketBranch`, `internal/doctor/doctor.go:427`) — it already
diffs a feature branch's own ticket against its base-branch copy and warns naming both
directories and the fix. It did not alleviate the field problem it was built for.

**Why not, concretely.** Neither `skill/resources/review-protocol.md` nor `skill/SKILL.md`
(inline "Procedure: implement a ticket" / "Procedure: validate a ticket") ever tells the agent
to run `pickle doctor` — grepped both, zero hits. The correct, already-shipped check is purely
opt-in, and a fresh session with no memory of T-108/T-128 has no reason to opt in. Separately,
`review-protocol.md` step 1 reads *"Locate the ticket: `tickets/4-in-review/T-NNN-*.md`"* — it
presumes the status directory the agent is supposed to already know is current, which is
exactly the fact a stale worktree gets wrong. The correct base-branch-read instruction exists,
but only as a blockquote aside *before* numbered step 0 ("reviewer independence"), easy to skim
past when an agent is pattern-matching on numbered steps rather than reading every aside.

**Fix, two parts, doc-only (no new detection logic — T-128 already built the right one):**

1. **Make ticket lookup branch-agnostic in the procedure text.** Replace the presumed-directory
   "Locate the ticket" instruction (`review-protocol.md` step 1, and the equivalent implicit
   assumption in `SKILL.md`'s "Procedure: implement a ticket" step 1) with a lookup that resolves
   the ticket's actual current path from base under `layout = "in-tree"` — the same
   `git ls-tree -r --name-only <base> -- tickets` scan `checkStaleTicketBranch` already performs
   internally (`internal/doctor/doctor.go:471`), given as a runnable command in the doc rather
   than an aside to remember.
2. **Promote `pickle doctor -v` to a mandatory step, correctly sequenced.** Add it as an explicit
   numbered step — same procedural weight as "reviewer independence" — in both the "implement a
   ticket" and "validate a ticket" procedures, run *after* the ticket's feature branch is checked
   out (the hazard only exists once `HEAD` is on `feat/T-NNN-*`; running it before checkout would
   silently pass). Any stale-ticket-branch warning must be resolved (per its own suggested fix —
   rebase onto base) before the procedure proceeds.

**Soft couplings (not `depends-on`, both already `6-done/`):**

- **T-128** — supplies `checkStaleTicketBranch`, the mechanism this ticket wires in; this ticket
  changes no Go code, only where the procedures point.
- **T-108** — the mirror-image hazard this whole chain (T-108 → T-128 → this ticket) traces back
  to; its `tickets-README.md` §0 prose is unaffected, only the two trigger procedures change.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-09-18 — created (TO DO). source: field-use: observed across in-tree brine installs — a
  brand-new session told "validate ticket T-NNN" reads ticket status from the checked-out
  feature branch's stale worktree copy instead of base, reporting an already-moved ticket as
  still in its prior status; T-128 built the correct detection but nothing in the trigger
  procedures invokes it, and `review-protocol.md`'s own step 1 presumes the status directory
  instead of resolving it from base.
