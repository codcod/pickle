---
id: T-040
title: board audit: validate ticket frontmatter (duplicate keys, self-referencing depends-on, TEMPLATE drift)
project: pickle
depends-on: []
spawned-by: [T-027, T-028, T-033]
impact: medium-high
complexity: low
cost: M
---

# T-040 — board audit: validate ticket frontmatter (duplicate keys, self-referencing depends-on, TEMPLATE drift)

## Description

**Epic — merged from T-027, T-028 and T-033 by the 2026-07-26 board triage.** All three are in
`tickets/7-dropped/` with their full analysis and line references; read them for detail.

Three gaps in what `pickle board audit` considers a valid ticket. All three land in
`internal/audit/audit.go`, all three are "the audit is the only component that sees *every*
ticket however it was authored" — which matters because the flow explicitly permits agents and
humans to write ticket files directly (`pickle ticket new` is a convenience, not a gatekeeper).
One file, one table-driven test, one review.

### Absorbed scope

| from | check | substance |
|---|---|---|
| T-033 | duplicate frontmatter keys | `ticket.ParseFrontmatter` (`internal/ticket/ticket.go:105-123`) assigns into a `map[string]string`, so a duplicate key **silently overwrites** — last wins — and a ticket with two `impact:` or two `project:` lines audits clean. A duplicate key is malformed however it arrived: a hand-edit, a bad merge resolution leaving two `depends-on:` lines, or a future command. |
| T-027 | self-referencing `depends-on` | The existence loop never checks whether a ticket lists **itself**. `T-042` with `depends-on: [T-042]` audits clean, then silently self-blocks: the pickup gate demands the dependency be in `6-done/`, which it can never be while in development. The failure surfaces as a confusing "dependency not done" error about the ticket itself, at pickup, instead of a frontmatter error at audit time. One condition in the existing loop. |
| T-028 | TEMPLATE.md drift | `audit.requiredKeys` (`internal/audit/audit.go:23`) and `skill/resources/TEMPLATE.md` must agree on the frontmatter key set, and **nothing enforces it**. The only guard, `TestScaffoldSectionsMatchTemplate` (`internal/ticket/ticket_test.go:146-162`), compares `## ` headings and is blind to frontmatter. A key added to the audit while TEMPLATE keeps advertising the old set makes every hand-authored ticket fail audit — **in the user's project, not in this repo's tests**. T-024 walked this tightrope by hand. |

### Correction carried over from the T-030 review (finding N3, 2026-07-26)

T-027's refinement note implied `internal/audit` holds a duplicate `T-\d+` regex to unify. **It
does not** — `internal/audit/audit.go` contains no regex and does not import `regexp`; its only
shape-adjacent checks are `t.Front["id"] != t.ID` (`:52`) and the existence lookups (`:67`,
`:80`). So the self-reference check **adds the first external caller** of `ticket.ValidID`
(`internal/ticket/ticket.go:146-175`), which today has no consumer outside its own package.
Slightly larger than "swap the regex for the helper" — worth knowing before estimating.

The `T-\d+` shape *is* still literally duplicated in `filenameRE` (`internal/ticket/ticket.go:95`)
and `board.rowRE` (`internal/board/board.go:29`). Composing all three from one fragment is
optional and this ticket's call; if it is deferred, it belongs with **T-042** (duplicated
internals) rather than here.

### Folded in from the T-036 pickup gate (2026-07-26) — a fourth check

**The status directories themselves are never validated.** `tickets/3-in-development/` does not
exist in this repo right now, and `board audit` reports 0 errors. Nor is it a local accident:
`git ls-files tickets/` returns **no `.gitkeep` files at all**, so none of the seven that
`install.go:311-319` creates is tracked. `4-in-review/` and `5-rework/` survive only as untracked
local leftovers — **a fresh clone of this repo would be missing three of its seven status
directories.**

Two independent defects, both in this epic's "the audit is the only component that sees every
ticket" theme:

1. **`audit.Audit` (`internal/audit/audit.go:27`) has no directory-existence check**, and cannot
   acquire one incidentally: it consumes `ticket.LoadAll`, which at
   `internal/ticket/ticket.go:365-376` deliberately `continue`s past a `ReadDir` error with the
   comment *"absent (vanished-empty) dir is not an error"* — because git does not track empty
   directories. So the absence is swallowed one layer below the audit. Any check must either look
   at the directories directly or have `LoadAll` distinguish "empty" from "absent". Note the
   swallowing is load-bearing for the WIP pre-check in `internal/move/move.go:92-99`; do not
   simply make it an error.
2. **The `.gitkeep` scaffold is not preserved in this repo.** Whatever the audit learns to detect,
   the seven files should be tracked here — this is the same class of defect as T-028 above:
   a guarantee `install` makes to users that this repo, self-hosting the flow, does not keep.

Deliberately *not* urgent: `internal/move/move.go:124` runs `os.MkdirAll` before the rename, so
every move re-creates what it needs and the flow self-heals. This is a "fresh clone looks broken
and the audit lies about it" defect, not a functional one.

### Folded in from the 2026-07-27 field-finding triage — a fifth check

**History lines have a documented shape that nothing enforces.**
`skill/resources/TEMPLATE.md:116-124` requires *"One line per status transition, dated YYYY-MM-DD,
in the form `OLD → NEW: one-clause reason`"*, with a merge recorded as `merged to <base> (<MR
ref>)`. The audit never checks it, and the flow permits hand-authored tickets — so a migrated or
hand-edited ticket can carry a merge note that is a whole paragraph on one physical line.

Measured in the field (2026-07-27, migrating an 84-ticket hand-rolled flow into a fresh
`pickle install` workspace): one `6-done/` ticket's merge line was **~1,900 characters** — pipeline
ids, job names, fast-forward reasoning — and `board audit` reported it clean.

Why it belongs in this epic and not in **T-049**:

- Same file (`internal/audit/audit.go`), same table-driven test, same theme — *the audit is the
  only component that sees every ticket however it was authored*.
- T-049 caps the **rendered cell**, which makes the board legible while leaving the ticket
  malformed forever. The two are complementary, and the split is deliberate: **truncation must not
  become the way malformed history is hidden.**

Shape notes for refinement:

- The parse already exists and is per-line: `historyRE`
  (`internal/ticket/ticket.go:104`) plus `mergedRE` (`:106`). A check needs no new scanner — it
  needs a length/clause judgement over the bodies `historyRE` already yields.
- It must be a **warning, not an error**: an over-long reason breaks no invariant, and every
  existing ticket in a migrated workspace would fail on import. Compare this epic's other four,
  which are genuine malformations.
- Decide at refinement whether "one clause" is checkable at all, or whether the practical check is
  length only. Do not ship a heuristic that flags this repo's own legitimate multi-clause
  transitions (e.g. T-036's `review clean; 6 non-blocking, all dispositioned`).

### Cross-references

- **T-049** — the render half of the same field finding (cap board cell width at `sanitizeCell`).
  Different file (`internal/board/board.go`); no ordering enforced, neither blocks the other.
- **T-044** (which superseded T-039, 2026-07-26) replaces the audit's board cross-check with a
  staleness check in the same file — the old plan's row-shape checks are gone. Still the same
  `internal/audit/audit.go` + test table: sequence to avoid edit collisions.
- **T-045** (not T-036 — corrected 2026-07-26 when the valves were split out) proposes a TO DO cap
  enforced by the audit; if it lands first, this epic inherits its test scaffolding. Note T-045 is
  measurement-gated and may well be dropped, so do not plan around it.
- **`pickle doctor`** (`internal/doctor/doctor.go:79`) checks only the skill dir's `SKILL.md` and
  `resources/tickets-README.md` — nothing under `tickets/`. If the directory check belongs in
  `doctor` rather than `audit`, that is this ticket's call to make at refinement.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-26 — created (TO DO). source: board triage — epic merged from T-027, T-028 and T-033,
  all three moved to 7-dropped/ as absorbed
- 2026-07-26 — a fourth check folded in from the **T-036 pickup applicability gate** (rather than
  spawned as its own ticket): status directories are never validated. `3-in-development/` is
  currently absent and `board audit` reports 0 errors; no `.gitkeep` is tracked anywhere under
  `tickets/`, so a fresh clone lacks three status dirs. Fits this epic's existing theme and file
  (`internal/audit/audit.go`), and the absence is swallowed by `ticket.LoadAll`'s deliberate
  vanished-empty-dir `continue`, which the WIP pre-check depends on — so it needs this epic's
  judgement, not a standalone ticket. Also corrected a reference this epic inherited: the TO DO cap
  moved from T-036 to T-045 at T-036's refinement.
- 2026-07-27 — a fifth check folded in from the field-finding triage: History lines have a
  TEMPLATE-documented shape the audit never enforces (a ~1,900-character merge line audited clean).
  Render half filed separately as T-049.
