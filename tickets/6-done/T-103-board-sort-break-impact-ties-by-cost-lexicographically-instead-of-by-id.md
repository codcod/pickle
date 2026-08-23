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

`board.Sort` (`internal/board/board.go:241`) orders TO DO/READY by impact descending and breaks
ties by **id** — i.e. by filing order, which carries no priority information at all. Impact is a
four-value ordinal, so ties are the common case, not the exception. Exact depths move constantly
and are not worth pinning: the group stood at 21 TO DO with 7 sharing `medium` and 7 sharing
`low-medium` when this was filed (2026-08-14), and at 8 TO DO with 4 and 2 by 2026-08-23. The
*shape* is what persists — within any such group the board's order is the order the tickets
happened to be filed in.

The change is to compare `cost` **beneath impact** — a `costRank` map (`S<M<L<XL`, with the
adjacent-pair ranges the rules allow slotting between) applied ascending, so cheap wins a tie.
It takes two insertion points in the existing comparator plus one new helper, `famCostRank`
(see the Implementation Plan; an early scoping of "~4 lines at a single insertion point" was
wrong and is corrected there). Nothing else changes: no config surface, no new frontmatter, no
second source of truth, and decision **D1** in `Sort`'s doc comment (`board.go:233`,
"deterministic, no hand-curated order") stays intact — this is still a pure function of the
ticket files. T-059's `family:` contiguity rule sits above the tiebreak and is preserved (proved
by construction and by test at refinement — see Implementation Plan decision 3).

### Provenance, and the honest case against it

This is the **one surviving idea** from T-063 (dropped 2026-08-01, which proposed a
value-per-cost *ratio* ordering). T-063 measured the alternatives: the ratio cut tied pairs
34 → 19, while the lexicographic tiebreak cuts them 34 → **10**, can never invert impact the way
a ratio does, and is invariant under any monotone renumbering of the ordinals — the defect that
killed the ratio (renumbering `cost` on an equally defensible scale moved 11 of 18 rows).

T-056 work area 5 recorded the idea with a trigger: file it only **if an `impact` recalibration
pass leaves the `medium` group ≥5 deep**. Two recalibration passes had been run (`NOTES.md`,
2026-08-01 and 2026-08-03) and the group stood 7 deep on 2026-08-14, so the trigger fired and
this is that ticket. The group has since fallen below that threshold; the trigger governed
*whether to file*, not whether to build what was filed, so it is spent either way — the case for
building now rests on the measured two-row effect above, not on the trigger.

**T-063's fatal finding still applies and is why this is graded `low`.** The queue anyone picks
from is READY, not TO DO — and across all **294 revisions** of `tickets/BOARD.md`, READY has
held 0 rows in 205 of them, 1 row in 65, 2 rows in 22 and 3 rows in 2. It has never held more
than three. Re-ordering the TO DO list improves a *reading* surface, not a *pickup* queue.
That is a real but narrow win, which is exactly what `low` means. Measured directly at the
2026-08-23 refinement by rendering the live tree with a throwaway prototype: the change moves
**exactly two rows** (T-103 and T-102 each swap with one neighbour). That is the honest size of
the win — it confirms `low` rather than undermining it, and anyone who expects more from this
ticket should drop it instead.

### Why the fix needs two insertion points, not one

A first attempt at this ticket (2026-08-23) scoped the tiebreak to `Sort`'s final `a.Num < b.Num`
fallback alone and was abandoned mid-implementation when a test proved that fallback is
unreachable for ordinary tickets. The reasoning is load-bearing for the plan below, so it is
recorded here rather than left as a lesson someone has to re-learn.

`Sort` has two places that resolve to id order: **(a)** different families/loose tickets tied at
the same family-level rank, ordered by `familyKey` ascending; **(b)** members of the *same*
family tied on their own impact, falling through to `a.Num < b.Num`. Branch (b) looks like "the
tiebreak", but `familyKey` returns a **loose ticket's own id**, so two distinct loose tickets
always have `fa != fb` and are decided at (a) — a string comparison of ids — without ever
reaching (b). Branch (b) fires only between two members sharing one real `family:` umbrella.
Since the overwhelming majority of tickets are loose, **(a) is the common case**: a cost check
installed only at (b) is dead code for exactly the tickets this ticket exists to reorder.
Verified twice — once by the failing attempt, once at refinement by prototype (below).

**Family contiguity is preserved by construction**, which is why (a) can safely be touched at
all. The value compared at (a) is `famCostRank`, which resolves through the umbrella; every
member of a family therefore yields the *same* rank as its umbrella, so the new comparison can
reorder whole families relative to each other but can never separate a member from its umbrella.
T-059's guarantee survives as an invariant of the helper's definition, not as an accident.

**Prototype verification (2026-08-23 refinement).** The plan below was implemented end-to-end as
a throwaway prototype, exercised, and then reverted — nothing was committed. Results: all four
cases pass (loose-vs-loose reorders by cost; same-family members reorder by cost under their
umbrella; impact+cost both tied still falls back to id ascending; a cheap loose ticket racing an
expensive family leaves that family contiguous). The **entire existing suite passed unedited**
(`go test ./...`, all packages green), confirming the standing claim that no existing test needs
changing. Rendering the live tree with the prototype changed exactly two board rows. This plan is
therefore known-executable, not merely plausible — the failure mode that sent the first attempt
back has been closed by testing the design rather than asserting it.

### Soft couplings

- **T-056** (dropped 2026-08-14) — its work area 5 carried this idea and its trigger.
- **T-063** (dropped) — the hearing that produced the measurements above; read its DROPPED
  banner before refining, including the errors it marks in its own text.
- **T-042** (done 2026-08-23) — also touched `internal/board`; the "sequence, do not run
  concurrently" caution is spent, its code is unmerged but its dev work is finished.
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
   fallback.** `Sort` (`internal/board/board.go:241-269`) has two such places: (a) different
   families/loose tickets tied at the same family-level rank (`ra == rb`), ordered by `familyKey`
   ascending (`fa < fb`); (b) members of the *same* family tied on their own impact, falling
   through to `a.Num < b.Num`. **Both need the cost tiebreak, not only (b)** — the Description's
   *"Why the fix needs two insertion points"* section carries the full argument and the evidence;
   do not re-scope this to (b) alone without reading it first.
2. **`costRank` mirrors `impactRank`'s existing shape exactly** (`internal/board/board.go:22-26`):
   a package-level `map[string]int` covering all seven legal `cost` values
   (`ticket.LegalCost`, verified at refinement: `S, S-M, M, M-L, L, L-XL, XL`), ranked `1..7`
   ascending. The comparison direction is the mirror image of `impactRank`'s: `impactRank` compares
   `ri > rj` (higher impact wins); `costRank` compares `ci < cj` (cheaper wins) — call this out
   in the doc comment so the inverted direction reads as deliberate, not a copy-paste mistake.
3. **A new `famCostRank` mirrors the existing `famRank` exactly** (`famRank` is at
   `internal/board/board.go:291-301`; put `famCostRank` directly after it): a family's cost is
   its umbrella's cost when it has one (`byID[t.Family].Front["cost"]`), else its own — the same
   umbrella-resolution shape `famRank` already uses for impact, including the same graceful
   fallback to the ticket's own value when the umbrella can't be resolved (audit-dirty tree, or
   `byID` nil). **This resolution is what preserves T-059 contiguity**: because every member
   resolves to its umbrella's cost, the branch-(a) comparison can reorder whole families but can
   never split one — keep that property if the helper is rewritten.
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
add `famCostRank` immediately after `famRank` (`:291-301`) per decision 3. In `Sort` (`:241-269`):
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
    `TestSortIsTheOrderRenderUses`; confirm it still passes unedited);
  - **a family-contiguity case** (decision 3's invariant): a cheap loose ticket and an
    expensive family (umbrella + member) at the same family rank, asserting the family's rows
    stay adjacent whichever side of the loose ticket they land on. This is the regression guard
    for the one way the branch-(a) change could break T-059, so it is not optional.
- All four cases above were run green against a throwaway prototype at refinement (see the
  Description); they are specified here to be re-written, not assumed already covered.
- `ticketBody`/`familyBody` hardcode `cost: M` and take no cost argument, so this task needs a
  small fixture helper that accepts one (the prototype used a `protoBody(id, title, impact, cost,
  family)` shape; match the file's existing conventions when naming it).
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
in scope. Line numbers below re-verified 2026-08-23 by grepping "ties by id"/"impact descending"
(they drift with unrelated edits — trust the quoted text over the number):
- `docs/user-manual/cli-reference.adoc:944` ("deterministic ordering — TO DO/READY by descending
  impact (ties by id), everything else by id")
- `docs/user-manual/cli-reference.adoc:1331` (`pickle serve`'s ordering guarantee: "impact
  descending, ties by id")
- `skill/resources/tickets-README.md:288` (§6: "renders TO DO/READY by descending impact within
  each child's group, ties by id")
- `skill/resources/tickets-README.md:452` (§6 board section: "ordered deterministically (impact
  descending, ties by id)")
- `skill/SKILL.md:110` ("the board orders each child's TO DO/READY group deterministically from
  it (impact descending, ties by id)")

The grep confirmed these five are the complete set — no sixth site states the rule.

Each becomes "impact descending, ties by cost ascending, then by id" (or equivalent phrasing
fitted to its sentence). The two `skill/` files are payload — read `just test`'s
`payload_lint_test.go` output (already covered by the Acceptance test) to confirm the new wording
passes the foreign-workspace mechanical checks; the phrasing above names no internal file, no
repo-only path, and no ticket id, so it is already foreign-workspace-safe by inspection.

### Finish (mandatory)

1. Acceptance test green.
2. Docs updated if a stated tiebreak rule was found (see above).
3. Write a summary confirming `TestSortIsTheOrderRenderUses` and the two family-render tests
   needed zero edits — verified at refinement against a full-suite prototype run (`go test
   ./...` all green), so an edit to any of them means the implementation diverged from the plan
   and should be re-examined rather than the test adjusted.
4. Suggested commit message:
   ```
   feat(board): break impact ties by cost before falling back to id (T-103)
   ```
5. Tidy WIP commits into atomic ones (root-path child) before presenting.
6. Commit locally; do not push or open an MR without explicit user approval. Hand back with
   `pickle ticket move T-103 in-review --reason "acceptance green"`.

## Review

- [x] Reviewer independence settled (step 0): the orchestrating reviewer authored the branch in
  this same session, so steps 2–4a were **delegated** to an independently spawned sub-agent,
  briefed adversarially with no memory of writing the code. Every delegated finding below was
  then re-verified by hand before recording, per the protocol.
- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass on the ticket's changed files (step 4b) — run twice (once during
  implementation, once during this review, both times on
  `docs/user-manual/cli-reference.adoc`, `skill/SKILL.md`, `skill/resources/tickets-README.md`);
  every suggestion both times targeted pre-existing prose outside this branch's diff, so none
  applied to this ticket's scope.

**Acceptance test re-run** (independent reviewer, on `feat/T-103-sort-cost-tiebreak`), all green:
`just build`; `go test ./internal/board/... -v -run 'TestSort|TestRenderFamily'` (including the
new `TestSortBreaksImpactTiesByCost` and its 4 subtests); `just test`; `just lint`;
`just docs-check`; `./pickle board audit` (0 errors; the 2 warnings present are pre-existing on
`main`, not introduced by this branch — verified via a disposable worktree). Re-confirmed a
second time after the two inline fixes below (F2, F3), still all green.

| id | severity | class | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|---|
| F1 | non-blocking | design | noted | Illegal/missing `cost` degrades to Go's zero value (rank 0), which under the ascending cost comparison sorts *first* (cheapest of all) — the opposite of `impactRank`'s existing "degrade to worst" convention (illegal impact sorts last, rank 0 loses under `ri > rj`). Unreachable through the normal flow (`ticket new` always writes a legal cost; `board audit` flags an illegal one) — the same mitigation `impactRank` already relies on — but the asymmetry itself is real. | `internal/board/board.go` `costRank`/`impactRank`; verified directly: `costRank["bogus"]=0 < costRank["S"]=1` vs `impactRank["bogus"]=0 < impactRank["low"]=1` under opposite comparison directions | Documented in `costRank`'s doc comment (done inline, see commit `ae46708`) rather than special-cased — no behaviour change needed since the flow can't reach it. |
| F2 | non-blocking | other | fixed inline | This branch's docs edit to `skill/resources/tickets-README.md` merged two lines without rewrapping, leaving one line at 121 chars vs the paragraph's ~91–95 char wrap — a line-wrap regression this branch authored, no behaviour change. | `skill/resources/tickets-README.md:454` (before fix, via `awk` line-length check) | Rewrapped; commit `ae46708`. |
| F3 | non-blocking | stale-xref | fixed inline | Two comments this branch made incomplete: `internal/board/board_test.go`'s `TestRenderOrdering` doc comment and `TestSortIsTheOrderRenderUses`'s `t.Errorf` message both said "ties by id asc", now only true because those specific fixtures happen to also tie on cost — the general claim needed updating to name cost first. | `internal/board/board_test.go:398` (doc comment), `:671` (errorf string), pre-fix | Reworded both to "ties by cost then id asc" plus a one-clause note that the fixture also ties on cost; commit `ae46708`. |
| F4 | non-blocking | design | noted | The Description/decision-3 language ("preserved by construction", "can never separate a member from its umbrella") slightly overstates the T-059 contiguity guarantee: it holds only on an audit-clean tree. Nested families are excluded by `internal/audit/audit.go:150` ("families do not nest"), not by `famRank`/`famCostRank` themselves — a caveat `famRank` already carried before this ticket, not a new gap it introduced. | `internal/audit/audit.go:150`; `internal/board/board.go` `famRank`/`famCostRank` | None — pre-existing caveat, not this branch's defect to fix. |
| F5 | non-blocking | docs-gap | noted | No `CHANGELOG.md` entry for this user-facing ordering change. The project's own `pickle changelog check` design (`internal/changelog/changelog.go`, T-093 decision 5) deliberately treats "does this ticket need an entry" as a reader judgement call with no exemption mechanism, reconciled at release-cut rather than mandated per-ticket — judged non-blocking on that basis. | `git diff main --stat` on the feature branch shows no `CHANGELOG.md` hunk; `internal/changelog/changelog.go` header comment | A human may add an entry at release-cut if `pickle changelog check` flags T-103 as unmentioned. |

Disposition summary: 2 fixed inline (F2, F3), 3 noted (F1, F4, F5). No blocking findings, no new
tickets, no folds.

```
cost: estimated S-M, actual S-M
```

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
- 2026-08-23 — refined: plan verified by throwaway prototype (all 4 cases green, full suite green
  unedited, 2 board rows move); Description de-staled, docs line refs corrected, contiguity
  regression test added to Task 3. Grade unchanged.
- 2026-08-23 — READY → IN DEVELOPMENT: picked up. Applicability gate (fresh sub-agent): no
  findings, plan holds exactly as written.
- 2026-08-23 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-08-23 — reviewed (independent audit delegated, hand-verified): 5 findings, all
  non-blocking (2 fixed inline — a comment/wrap regression this branch introduced; 3 noted).
  cost estimated S-M, actual S-M. No blocking findings, no spawned tickets.
- 2026-08-23 — published: `main` pushed (5be888f..e48a352), branch `feat/T-103-sort-cost-tiebreak`
  pushed, PR #67 opened against `main`. Merging is the human's.
- 2026-08-23 — merged to main (PR #67, 60a715b), user-approved; branch deleted
- 2026-08-23 — IN REVIEW → DONE: reviewed: no blocking findings
