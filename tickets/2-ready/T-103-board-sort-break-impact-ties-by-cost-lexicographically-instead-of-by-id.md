---
id: T-103
title: board.Sort: break impact ties by cost lexicographically instead of by id
project: pickle
depends-on: []
spawned-by: [T-056]
impact: low
complexity: low
cost: S-M
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

### Re-refinement note (2026-08-23): Decision 1 is wrong, needs a different fix shape

Picked up 2026-08-23. The applicability-gate audit (fresh sub-agent) found the code-level
assumptions still held (impactRank/costRank shape, `ticket.LegalCost`'s 7 values, `Sort`'s two
branches, the three existing test fixtures, WIP 0/1) and only non-blocking notes: T-042 has
since moved to `6-done/` so the "don't run concurrently" coupling is moot; a handful of line
citations in this ticket drifted a few lines from unrelated intervening commits, though every
quoted anchor *text* still matched verbatim; and the Description's 2026-08-14 backlog snapshot
(21 TO DO / 7 medium / 7 low-medium) is stale — current board is 8 TO DO / 4 medium / 2
low-medium, below T-056's original trigger — dispositioned non-blocking/note-and-close since the
change's value doesn't depend on current tie depth.

Implementation itself then surfaced a **blocking** problem the audit didn't catch, because it's
about behavior, not assumptions: **the confirmed Decision 1 is built on a false premise.**
Decision 1 says the cost tiebreak belongs only in the final `a.Num < b.Num` fallback (`Sort`'s
branch "b": two members of the *same* family tied on their own impact), and that the `fa != fb`
branch (different families/loose tickets tied at the same family rank) is untouched. But trace
`familyKey`: a loose ticket's `familyKey` is **its own id**. Two distinct loose tickets therefore
*always* have `fa != fb` — they can never reach branch "b" at all, because that branch only
fires once `fa == fb`, true only for two members sharing one real `family:` umbrella. So a
loose-vs-loose impact tie — the common case, and the exact shape of the 7-deep `medium` /
2-deep `low-medium` groups this ticket exists to fix — is decided earlier, at `if fa != fb {
return fa < fb }`, a **string** comparison of ids, never reaching the fallback Decision 1 scoped
the change to. Confirmed empirically: adding the cost check only to branch "b" left a
loose-vs-loose test (three same-impact loose tickets, highest id cheapest) still sorting by id.

**Proposed remedy for the next refinement pass:** add a `famCostRank` mirroring the existing
`famRank` (family's cost when it has one via its umbrella, else its own), and insert a cost
comparison right after `ra != rb` but before `fa != fb` — so two different families/loose
tickets tied at the same family rank break by (representative) cost before falling back to
`familyKey`/id, in addition to keeping branch "b"'s own-impact-tie cost check for members within
one family. T-059 contiguity is unaffected either way (this only reorders *among* families/loose
tickets at a tied rank, never splits one apart), and the three existing family tests stay green
regardless (their fixtures hardcode uniform `cost: M`). This is materially more than "~4 lines in
the existing comparator" — it touches the branch Decision 1 said to leave alone and adds one new
helper — so re-refinement should replace Decision 1 and Task 1 with this shape rather than patch
around it.

The Implementation Plan below has been rewritten accordingly (decisions 1/3/4/5, Tasks 1–3), and
`cost` re-graded `S` → `S-M`: two insertion points and one new helper function is more than the
original "~4 lines" scope, though still a small, mechanical, pattern-following change — impact
and complexity are unchanged.

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

None (`depends-on: []`). **T-042 also touched `internal/board`**, but is now `6-done/`
(re-verified at the 2026-08-23 re-refinement) — the soft coupling's "sequence, do not run
concurrently" no longer applies; no prerequisite blocks picking this up.

### Confirmed design decisions (do not deviate without asking)

1. **Scope is both places `Sort` currently resolves to id order — not just the final
   fallback.** `Sort` (`internal/board/board.go:241-269`, re-verified at the 2026-08-23
   re-refinement) has two such places: (a) different families/loose tickets tied at the same
   family-level rank (`ra == rb`), ordered by `familyKey` ascending (`fa < fb`); (b) members of
   the *same* family tied on their own impact, falling through to `a.Num < b.Num`. **Both need
   the cost tiebreak, not only (b).** A loose ticket's `familyKey` is its own id
   (`familyKey`, `:298-305`), so two distinct loose tickets always have `fa != fb` and can
   *never* reach branch (b) — confirmed empirically at re-refinement: adding the cost check only
   to (b) left three same-impact loose tickets (highest id cheapest) still sorting by id. Since
   the vast majority of tickets carry no `family:` (including the 7-deep `medium` / 2-deep
   `low-medium` groups this ticket exists to fix), (a) is the common case, not (b) — both must
   change or the fix is a no-op for ordinary tickets. T-059's contiguity guarantee is still
   unaffected: this only reorders *among* families/loose tickets already tied at one rank, never
   splits a family apart.
2. **`costRank` mirrors `impactRank`'s existing shape exactly** (`internal/board/board.go:22-26`):
   a package-level `map[string]int` covering all seven legal `cost` values
   (`ticket.LegalCost`, verified at refinement: `S, S-M, M, M-L, L, L-XL, XL`), ranked `1..7`
   ascending. The comparison direction is the mirror image of `impactRank`'s: `impactRank` compares
   `ri > rj` (higher impact wins); `costRank` compares `ci < cj` (cheaper wins) — call this out
   in the doc comment so the inverted direction reads as deliberate, not a copy-paste mistake.
3. **A new `famCostRank` mirrors the existing `famRank` exactly** (`internal/board/board.go:308-316`):
   a family's cost is its umbrella's cost when it has one (`byID[t.Family].Front["cost"]`), else
   its own — the same umbrella-resolution shape `famRank` already uses for impact, including the
   same graceful fallback to the ticket's own value when the umbrella can't be resolved
   (audit-dirty tree, or `byID` nil).
4. **An illegal/absent `cost` degrades the same way an illegal/absent `impact` already does**:
   a Go zero-value map lookup (rank `0`), no panic, no special-cased fallback — consistent with
   `impactRank`/`famRank`'s existing graceful-degradation convention, not a new failure mode to
   design. (In practice unreachable through the normal flow: `pickle ticket new` always writes a
   legal `cost`, and `board audit` flags an illegal one.)
5. **The final tiebreak stays `a.Num < b.Num`**, now reached only after: family rank, family
   cost, familyKey, own impact, own cost — still fully deterministic (D1), still a pure function
   of the ticket files, still no new frontmatter or config surface.

### Tasks

#### Task 1 — add `costRank`, `famCostRank`, and both tiebreaks
In `internal/board/board.go`, add `costRank` near `impactRank` (`:22-26`) per decision 2, and
add `famCostRank` near `famRank` (`:308-316`) per decision 3. In `Sort` (`:241-269`):
- insert a `famCostRank` comparison immediately after `if ra != rb { return ra > rb }` and
  before `if fa != fb { ... }`, per decisions 1 and 4: `if ca, cb :=
  famCostRank(a, byID), famCostRank(b, byID); ca != cb { return ca < cb }`;
- insert the plain `costRank` comparison immediately after the existing own-impact comparison
  (`if ri, rj := impactRank[a.Front["impact"]], impactRank[b.Front["impact"]]; ri != rj { return
  ri > rj }`, currently the last check before `return a.Num < b.Num`), per decisions 1 and 4.

#### Task 2 — update the doc comment
`Sort`'s doc comment (`:231-238`) currently reads "TO DO/READY by descending impact (tie id
asc)". Update it to name the full chain: descending impact, then ascending cost (at whichever
level the tie occurs — family rank or own impact), then ascending id. Note that this now touches
both branches Decision 1 identifies, unlike the ticket's original (wrong) assumption that only
the final fallback needed it — T-059's family-contiguity guarantee itself is still unaffected.

#### Task 3 — tests
- Add `TestSortBreaksImpactTiesByCost` to `internal/board/board_test.go`, covering **both**
  branches from decision 1:
  - loose-vs-loose (branch (a)): three `1-to-do` tickets sharing one impact value with distinct
    costs, filed so the highest-id ticket has the cheapest cost, asserting the cheapest sorts
    first — this is the case the original plan missed and must not regress;
  - same-family (branch (b)): two members of one family sharing their own impact but distinct
    costs, asserting the cheaper member sorts first within the family;
  - a case where impact *and* cost both tie, asserting the fallback to id ascending still holds
    (mirrors `ticketBody`'s existing uniform-`cost: M` tickets — already exercised by
    `TestSortIsTheOrderRenderUses`; confirm it still passes unedited).
- Re-run `TestRenderFamilyGrouping`/`TestRenderFamilySinksToUmbrellaImpact` unedited — verified at
  refinement neither has an impact tie whose members/umbrellas share cost in a way this change
  could disturb (`TestRenderFamilySinksToUmbrellaImpact`'s three tickets have three distinct
  impacts; `familyBody`/`ticketBody` both hardcode `cost: M`, so `TestSortIsTheOrderRenderUses`'s
  medium-impact tie (T-002/T-005, both loose) stays a same-cost tie and keeps resolving to id
  order — confirmed by hand at refinement, not merely assumed, and now doubly relevant since that
  tie is decided by the newly-changed branch (a), not (b)).

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
ordering renders without error over the real live TO DO backlog (whatever depth it holds at
pickup time — the Description's specific snapshot counts have already gone stale once, see the
Re-refinement note, so the acceptance test deliberately makes no numeric claim of its own).

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
- 2026-08-23 — READY → IN DEVELOPMENT: picked up. Applicability gate (fresh sub-agent) found
  only non-blocking items — see Description for the full note.
- 2026-08-23 — IN DEVELOPMENT → READY: re-refine — Decision 1 invalidated during coding, plan
  rewritten, cost re-graded S → S-M; see Description for the finding and remedy.
