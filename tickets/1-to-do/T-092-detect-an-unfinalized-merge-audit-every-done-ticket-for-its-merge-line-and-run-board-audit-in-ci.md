---
id: T-092
title: detect an unfinalized merge: audit every DONE ticket for its merge line, and run board audit in CI
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: low
cost: S
---

# T-092 — detect an unfinalized merge: audit every DONE ticket for its merge line, and run board audit in CI

## Outcome

After this ships, forgetting to finalize a merged ticket is **detected instead of merely
documented**: `pickle board audit` warns for any ticket in `6-done/` with no `merged to <base>`
History line — today that check exists but fires only when the done ticket happens to be some
*other* ticket's `depends-on` target — and CI runs `board audit` on push, which today **nothing
does**: this board is audited only when a human remembers to run it locally.

## Description

**Corrected baseline (the original filing of this ticket got this wrong).** This ticket was first
filed claiming *"nothing in the skill tells the agent what to do once the human reports the merge
happened."* That is false. `resources/tickets-README.md` §4 already prescribes it:

> When the human reports a merge, append a dated `merged to <base>` History line (ideally with a
> commit reference alongside the MR ref, per §1) to the done ticket — the board's `merged` cell
> renders from that line — and run `pickle board sync`.

So the append-the-line and regenerate-the-board steps are **already flow law**, and a new "step 10"
restating them in `review-protocol.md` would put two copies of the same instruction in the payload
— exactly the drift T-081 spent a whole ticket removing for gate definitions. The original filing
oversold a narrow gap by mis-stating the baseline; this amendment fixes that and re-aims the
ticket at what is genuinely missing.

**What is actually missing is enforcement, not instruction.** §4 says *what* to do; nothing checks
whether it was done:

1. **The merged-line check exists but is scoped too narrowly.** `internal/audit/audit.go:239`
   already emits `dependency %s is DONE but has no 'MERGED' History line` — but only for a ticket
   reached as some other ticket's `depends-on` target. A done ticket nobody depends on is never
   checked, so the common case (finalize forgotten, nothing downstream points at it) is invisible.
2. **Nothing runs `board audit` automatically.** Verified at filing: `grep -rn "board audit"
   .github/workflows/` returns nothing. The board is audited only when a human remembers to, on a
   laptop.

Those two together are why the T-081 incident (`tickets/6-done/T-081-…md`) went as far as it did:
a bookkeeping commit staged a ticket move's new path plus the regenerated board, but never the old
path the rename deleted, so the stale copy stayed tracked in `HEAD`; `board audit` stayed clean
because it audits the **worktree**, not `HEAD`; and the corruption surfaced only when merging the
PR restored the stale file and a *hand-run* audit finally reported `duplicate id` (fixed live in
`7e3dbb2`). It was caught by luck of ordering, and CI would never have caught it at all.

**Proposed change (data and tooling, not prose):**

- Generalize the `audit.go:239` check: warn for **every** ticket in `6-done/` whose History has no
  `merged to <base>` line — not only dependency targets. Whether this is a warning or an error,
  and whether bookkeeping-only tickets (which have no branch to merge) need an exemption, are
  refinement decisions; an exemption almost certainly is needed, since a docs/board-only ticket
  legitimately never merges anything.
- Run `pickle board audit` in CI on push/PR, so the board is checked by something other than a
  human's memory.
- Optionally, extend §4's existing sentence with the two genuinely-uncovered residual steps
  (delete the feature branch; conclude any impact-sweep deferral the review left pending the
  merge) — as a clause on the rule that already exists, **not** as a new protocol section.

**Deliberately dropped from the original filing**, recorded so refinement does not re-add them:

- *A nine-step "step 10" finalize procedure in `review-protocol.md`.* The remedy for "an agent
  forgot a step" should not be "write the agent another step to remember" — that is precisely the
  lesson T-081 shipped one day earlier (encode the requirement as data, audit it). Detection also
  needs no trigger: a documented procedure only runs if the human remembers to say "finalize
  T-NNN", which is the same failure mode as the forgotten step.
- *"Verify the merge independently — don't trust the report at face value."* Calling `gh pr view`
  is worth doing because it is where the merge SHA and the actual merge strategy come from, not
  because the human might misreport a merge. Keep the call, drop the framing.
- *"Re-run the child's build/test/lint/docs on the merged base."* Duplicates CI on a PR that was
  already green and mergeable; the merge itself is the only new variable, and a semantic conflict
  that survives a green CI is rare enough not to justify a standing step.
- *Unconditional branch deletion.* Stated as "its last legitimate use was the merge", which is
  wrong while a revert is still plausible. If kept at all (above), it belongs as a soft clause.

**Candidate follow-on, not folded in:** a `pickle ticket merged <id> --pr <ref> --sha <sha>`
subcommand that appends the History line and syncs atomically, mirroring how `ticket move` already
bundles move + History + board-regen. Note this is now *less* pressing than at first filing:
detection catches the forgotten line regardless of whether writing it is automated.

Soft coupling: **T-091** (a bookkeeping commit can stage a move's add without its delete) is the
other half of the same incident — it fixes the staging mistake at its source, while this ticket
makes the omission detectable afterwards. Neither blocks the other; either alone is an
improvement.

Soft coupling: **T-093** (reconcile merged tickets against the changelog) consumes exactly the
`merged to <base>` lines this ticket makes reliable — a done ticket silently missing its line is a
ticket T-093's sweep cannot see. Not a hard dependency.

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
- 2026-08-12 — **re-aimed and retitled** (was: *finalize a ticket after its PR merges: a step 10
  for the review protocol*) after the filing session challenged its own two new tickets. Two
  substantive corrections: (1) the Description's central claim was **factually false** — rules §4
  already prescribes appending the `merged to <base>` line and running `board sync`, so the
  proposed step 10 would have duplicated existing flow law; (2) the remedy is flipped from prose
  (a nine-step protocol section) to data/tooling (generalize the `audit.go:239` merged-line check
  beyond dependency targets; run `board audit` in CI, which nothing does today), because
  prescribing another step for an agent to remember is the failure mode T-081 had just shipped a
  fix for. Four weak steps dropped with reasons recorded in the Description. Grade unchanged at
  `medium`/`low`/`S`: scope narrowed from nine prose steps to two mechanical checks, but the work
  moves from docs into `internal/audit` + a CI job, so cost holds
