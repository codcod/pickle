---
id: T-103
title: board.Sort: break impact ties by cost lexicographically instead of by id
project: pickle
depends-on: []
spawned-by: [T-056]
impact: low
complexity: low
cost: S
---

# T-103 — board.Sort: break impact ties by cost lexicographically instead of by id

## Outcome

After this ships, two tickets with the same impact are ordered cheapest-first on the board
instead of oldest-first, so the TO DO group's largest tie stops being an arbitrary list.

## Description

`board.Sort` (`internal/board/board.go:240`) orders TO DO/READY by impact descending and breaks
ties by **id** — i.e. by filing order, which carries no priority information at all. Impact is a
four-value ordinal over a backlog that is now 21 tickets deep in TO DO, so ties are the common
case, not the exception: **7 tickets share `medium` and 7 share `low-medium`** as of
2026-08-14. Within each of those groups the board's order is the order they happened to be
filed in.

The change is to compare `cost` **lexicographically beneath impact** — a `costRank` map
(`S<M<L<XL`, with the adjacent-pair ranges the rules allow slotting between) and roughly four
lines in the existing comparator, ascending so cheap wins a tie. Nothing else changes: no config
surface, no new frontmatter, no second source of truth, and decision **D1** at `board.go:231`
("deterministic, no hand-curated order") stays intact — this is still a pure function of the
ticket files. T-059's `family:` contiguity rule sits above the tiebreak and is unaffected.

### Provenance, and the honest case against it

This is the **one surviving idea** from T-063 (dropped 2026-08-01, which proposed a
value-per-cost *ratio* ordering). T-063 measured the alternatives: the ratio cut tied pairs
34 → 19, while the lexicographic tiebreak cuts them 34 → **10**, can never invert impact the way
a ratio does, and is invariant under any monotone renumbering of the ordinals — the defect that
killed the ratio (renumbering `cost` on an equally defensible scale moved 11 of 18 rows).

T-056 work area 5 recorded the idea with a trigger: file it only **if an `impact` recalibration
pass leaves the `medium` group ≥5 deep**. Two recalibration passes have been run (`NOTES.md`,
2026-08-01 and 2026-08-03) and the group is 7 deep, so the trigger has fired and this is that
ticket.

**T-063's fatal finding still applies and is why this is graded `low`.** The queue anyone picks
from is READY, not TO DO — and across all **294 revisions** of `tickets/BOARD.md`, READY has
held 0 rows in 205 of them, 1 row in 65, 2 rows in 22 and 3 rows in 2. It has never held more
than three. Re-ordering a 21-row TO DO list improves a *reading* surface, not a *pickup* queue.
That is a real but narrow win, which is exactly what `low` means. If refinement finds the change
costs more than the ~4 lines claimed here, dropping it is the right answer.

### Soft couplings

- **T-056** (dropped 2026-08-14) — its work area 5 carried this idea and its trigger.
- **T-063** (dropped) — the hearing that produced the measurements above; read its DROPPED
  banner before refining, including the errors it marks in its own text.
- **T-042** — also touches `internal/board`; sequence, do not run concurrently.
- **T-059** (done) — `family:` grouping in the same comparator.
- **T-045** (dropped) — the `user-visible:` axis, the other proposed answer to wide impact ties.

## Implementation Plan

### 0. Feature branch (mandatory)

```
cd .
git checkout main
git checkout -b feat/T-103-sort-cost-tiebreak
```

Root-path child. Tidy WIP commits before presenting (Finish, below).

### Prerequisite gate (hard)

None (`depends-on: []`). **T-042 also touches `internal/board`** (verified: still `2-ready/`,
not picked up) — the soft coupling's "sequence, do not run concurrently" is a note for whichever
ticket is picked up second, not a blocker for refining or picking up either one first; recorded
here so it is not missed at pickup time.

### Confirmed design decisions (do not deviate without asking)

1. **Scope is exactly the final id-based fallback — not the `fa != fb` cross-family/loose
   ordering branch.** `Sort` (`internal/board/board.go:241-266`, re-verified at refinement) has
   two places that currently resolve to id order: (a) different families/loose tickets tied at
   the same family-level rank, ordered by `familyKey` ascending; (b) members of the *same*
   family (or two loose tickets) tied on their own impact, falling through to `a.Num < b.Num`.
   The Description's own text ("T-059's `family:` contiguity rule sits above the tiebreak and is
   unaffected") settles this: only (b) gets the cost tiebreak. (a) stays exactly as T-059 shipped
   it — changing it would be a second, un-asked-for design decision about family-group ordering,
   not the "~4 lines beneath impact" this ticket describes.
2. **`costRank` mirrors `impactRank`'s existing shape exactly** (`internal/board/board.go:23-26`):
   a package-level `map[string]int` covering all seven legal `cost` values
   (`ticket.LegalCost`, verified at refinement: `S, S-M, M, M-L, L, L-XL, XL`), ranked `1..7`
   ascending. The comparison direction is the mirror image of `impactRank`'s: `impactRank` compares
   `ri > rj` (higher impact wins); `costRank` compares `ci < cj` (cheaper wins) — call this out
   in the doc comment so the inverted direction reads as deliberate, not a copy-paste mistake.
3. **An illegal/absent `cost` degrades the same way an illegal/absent `impact` already does**:
   a Go zero-value map lookup (rank `0`), no panic, no special-cased fallback — consistent with
   `impactRank`/`famRank`'s existing graceful-degradation convention, not a new failure mode to
   design. (In practice unreachable through the normal flow: `pickle ticket new` always writes a
   legal `cost`, and `board audit` flags an illegal one.)
4. **The final tiebreak stays `a.Num < b.Num`, now three levels down** (family rank, then own
   impact, then cost, then id) — still fully deterministic (D1), still a pure function of the
   ticket files, still no new frontmatter or config surface.

### Tasks

#### Task 1 — add `costRank` and the tiebreak
In `internal/board/board.go`, add `costRank` near `impactRank` (`:23-26`) per decision 2. In
`Sort` (`:241-266`), insert the cost comparison immediately after the existing own-impact
comparison (`if ri, rj := impactRank[a.Front["impact"]], impactRank[b.Front["impact"]]; ri != rj
{ return ri > rj }`, currently the last check before `return a.Num < b.Num`), per decisions 1 and 3.

#### Task 2 — update the doc comment
`Sort`'s doc comment (`:231-235`) currently reads "TO DO/READY by descending impact (tie id
asc)". Update it to name the full chain: descending impact, then ascending cost, then ascending
id — and note (per decision 1) that the family-group-level tiebreak (different families at the
same rank, ordered by `familyKey`) is unchanged by this ticket.

#### Task 3 — tests
- Add `TestSortBreaksImpactTiesByCost` to `internal/board/board_test.go`: three `1-to-do`
  tickets sharing one impact value with distinct costs filed in an order that would mislead if
  id-order were still used (e.g. the highest-id ticket has the cheapest cost), asserting the
  cheapest sorts first. Add a second case (or subtest) proving cost *and* impact both tying still
  falls back to id ascending (mirrors `ticketBody`'s existing uniform-`cost: M` tickets — already
  exercised by `TestSortIsTheOrderRenderUses`, confirm it still passes unedited).
- Re-run `TestRenderFamilyGrouping`/`TestRenderFamilySinksToUmbrellaImpact` unedited — verified at
  refinement neither has an impact tie whose members share cost in a way this change could
  disturb (`TestRenderFamilySinksToUmbrellaImpact`'s three tickets have three distinct impacts;
  `familyBody`/`ticketBody` both hardcode `cost: M`, so `TestSortIsTheOrderRenderUses`'s
  medium-impact tie (T-002/T-005) stays a same-cost tie and keeps resolving to id order —
  confirmed by hand at refinement, not merely assumed).

### Acceptance test

```
just build
go test ./internal/board/... -v -run 'TestSort|TestRenderFamily'
just test
just lint
just docs-check
./pickle board audit
```
All clean; `./pickle board audit` against this repo's own live tree additionally confirms the new
ordering renders without error over the real ~21-deep TO DO backlog the Description measured
against.

### Docs update (mandatory when user-facing)

This changes documented, cross-project behaviour (every installed project's board renders with
this ordering, not just this repo's own), so all five places that currently say "ties by id" are
in scope, verified at refinement by grepping "ties by id"/"impact descending":
- `docs/user-manual/cli-reference.adoc:928` ("deterministic ordering — TO DO/READY by descending
  impact (ties by id), everything else by id")
- `docs/user-manual/cli-reference.adoc:1311` (`pickle serve`'s ordering guarantee: "impact
  descending, ties by id")
- `skill/resources/tickets-README.md:287` (§6: "renders TO DO/READY by descending impact within
  each child's group, ties by id")
- `skill/resources/tickets-README.md:451` (§6 board section: "ordered deterministically (impact
  descending, ties by id)")
- `skill/SKILL.md:110` ("the board orders each child's TO DO/READY group deterministically from
  it (impact descending, ties by id)")

Each becomes "impact descending, ties by cost ascending, then by id" (or equivalent phrasing
fitted to its sentence). The two `skill/` files are payload — read `just test`'s
`payload_lint_test.go` output (already covered by the Acceptance test) to confirm the new wording
passes the foreign-workspace mechanical checks; the phrasing above names no internal file, no
repo-only path, and no ticket id, so it is already foreign-workspace-safe by inspection.

### Finish (mandatory)

1. Acceptance test green.
2. Docs updated if a stated tiebreak rule was found (see above).
3. Write a summary confirming `TestSortIsTheOrderRenderUses` and the two family-render tests
   needed zero edits, per decision-time verification.
4. Suggested commit message:
   ```
   feat(board): break impact ties by cost before falling back to id (T-103)
   ```
5. Tidy WIP commits into atomic ones (root-path child) before presenting.
6. Commit locally; do not push or open an MR without explicit user approval. Hand back with
   `pickle ticket move T-103 in-review --reason "acceptance green"`.

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-14 — created (TO DO). source: chat: refinement split of T-056 (dropped the same day) —
  the surviving residue of its work area 5, filed because the trigger it recorded (an `impact`
  recalibration leaving the `medium` group ≥5 deep) has fired at 7
- 2026-08-20 — refined: confirmed T-042 (also touching `internal/board`, still `2-ready/`) is a
  sequencing note only, not a blocker. Scoped the fix to exactly the final id-based fallback,
  leaving T-059's family-group-level ordering (the `familyKey` branch) untouched, per the
  Description's own text. Verified by hand that both existing family/impact-tie tests
  (`TestSortIsTheOrderRenderUses`, `TestRenderFamilySinksToUmbrellaImpact`) need zero edits—their
  fixtures either have no impact tie or tie on a uniform `cost: M`. Found the id-tiebreak
  wording is stated in five places, not documented as internal-only: the manual (twice) and the
  shipped `skill/` payload (three times, including `SKILL.md` and `tickets-README.md` §6) — all
  five now in the Docs update task, since this changes behaviour every installed project sees,
  not only this repo's own board. Grade unchanged. TO DO → READY: implementation plan complete.
- 2026-08-22 — TO DO → READY: plan complete
- 2026-08-23 — READY → IN DEVELOPMENT: picked up. Applicability gate (fresh sub-agent):
  code-level assumptions (impactRank/costRank shape, ticket.LegalCost's 7 values, Sort's two
  branches, the three existing test fixtures, WIP 0/1) all re-verified true; T-042 has since
  moved to 6-done/ so the "sequence, don't run concurrently" coupling is moot; several line
  citations in the plan drifted a few lines from unrelated intervening commits but every quoted
  anchor text still matches verbatim, so tasks remain executable as written (re-anchor line
  numbers during implementation). One finding — the Description's 2026-08-14 backlog snapshot
  (21 TO DO / 7 medium / 7 low-medium) is stale; current board shows 8 TO DO / 4 medium / 2
  low-medium, below T-056's original ≥5 trigger — dispositioned non-blocking/note-and-close: the
  change is a pure readability/determinism improvement independent of current tie depth, and the
  ticket is already graded low reflecting exactly that modest value. Proceeding with the plan
  unmodified.
- 2026-08-23 — READY → IN DEVELOPMENT: picked up
