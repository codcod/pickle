---
id: T-036
title: review protocol spawns unbounded follow-up tickets; add inline-fix, note-and-close and backlog-cap valves
project: pickle
depends-on: []
spawned-by: []
impact: high
complexity: medium
cost: M
---

# T-036 — review protocol spawns unbounded follow-up tickets; add inline-fix, note-and-close and backlog-cap valves

## Description

The shipped review protocol is an unbounded ticket generator. `resources/review-protocol.md`
§5 requires that **every** non-blocking finding become a new `1-to-do/` ticket ("Spawn a **new
ticket** … for each"), and §0 forbids the alternative ("never inline drift"). The only escape
hatch is line 116 — "trivial, purely-cosmetic spec typos in the workflow scaffolding itself".
So a reviewer's only legal move on any substantive polish observation is to mint a ticket, and
a competent reviewer can always find polish observations.

Measured on this repo's own lineage (`spawned-by:`), the resulting branching factor is
**R ≈ 2.3** follow-up tickets per reviewed ticket:

| reviewed | spawned |
|---|---|
| T-024 | T-029, T-030 |
| T-029 | T-031, T-032 |
| T-030 | T-033, T-034, T-035 |

Any R > 1 diverges: the backlog grows without bound no matter how fast tickets are completed.
Supporting evidence that the flow has no closing valve at all — after 35 tickets and 59
commits, `7-dropped/` is **empty**, and 19 of the 22 open TO DO tickets are internal hygiene
(titles containing *harden / consolidate / unify / polish / backfill / repair / guard*) rather
than user-visible work.

This is a **product bug, not a repo-hygiene problem**. `pickle install` ships this protocol, so
every project that adopts the flow inherits R > 1 and the same divergence. This repo only hit
it first because it self-hosts. The fix therefore belongs in the embedded skill payload
(`skill/resources/`), not merely in this repo's board.

Four valves to add, all in the payload:

1. **Inline-fix path.** A non-blocking finding in a file the ticket already touches, below a
   stated size bar, is fixed on the current branch instead of spawning a ticket. Requires
   relaxing the §0 "never inline drift" absolute and stating the bar precisely enough that two
   reviewers agree on it.
2. **Note-and-close as the default disposition.** A non-blocking finding is recorded in the
   reviewed ticket's `## Review` table and closed there. Promotion to a ticket becomes the
   exception, justified by "would actually be scheduled", not the default.
3. **Backlog cap.** WIP limits today bound `3-in-development/` and `4-in-review/` but leave
   `1-to-do/` unbounded, so the pressure escapes exactly where nobody feels it. Add a
   configurable per-child TO DO cap (`pickle.toml`), enforced by `pickle board audit` and by
   `pickle ticket new`; at cap, filing requires dropping something.
4. **A `user-visible:` grade axis.** `impact` has collapsed as a signal — no `high` since T-018,
   everything since is medium/low — because it measures "is this a real defect" rather than
   "would a user notice". A separate boolean axis lets the board sort hygiene below
   user-visible work.

Valves 1–2 are protocol text (`skill/resources/review-protocol.md`, `resources/tickets-README.md`).
Valves 3–4 are CLI + schema changes (config, `board audit`, `ticket new`, `TEMPLATE.md`,
`BOARD.md` skeleton) and may warrant splitting into their own ticket at refinement — decide
then, and prefer one ticket over four unless the plan genuinely does not fit.

Soft couplings (no hard `depends-on:`): **T-016** adds a Step 4b to the same protocol file and
**T-022** rewrites payload conditionals in the same tree — sequence to avoid edit collisions.
**T-025** (backfill historical `spawned-by:` lineage) would make the R metric measurable
retroactively and is the natural companion for a "spawn rate" audit warning.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-26 — created (TO DO). source: chat — board growth analysis; measured R ≈ 2.3
  spawned tickets per reviewed ticket across T-024 → T-029/T-030 → T-031/T-032/T-033/T-034/T-035,
  with `7-dropped/` empty after 35 tickets
