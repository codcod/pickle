---
id: T-059
title: family: group tickets under an umbrella ticket id for curated pickup order
project: pickle
depends-on: []
spawned-by: [T-045]
impact: medium
complexity: medium
cost: M
---

# T-059 — family: group tickets under an umbrella ticket id for curated pickup order

## Description

Spawned from exploration under **T-045** (2026-07-28), alongside **T-058** (per-child
`ticket_prefix`) — same conversation, two independent schema questions about picking the next
refinement candidate from a large backlog.

**Problem:** the board only orders TO DO/READY by `impact` (`internal/board/board.go:240`); at
scale that collapses into wide ties. Measured on unity's live board (91 tickets, children
`rick` + `snowball`): 28 of 62 TO DO tickets tied at `impact: medium` — selection is genuinely
undetermined by the current schema. Pickle's own backlog (17 tickets) does not yet show this
symptom, but unity's does, today.

**Rejected alternative — reusable `category:` scope key** (like a conventional-commit scope,
e.g. `framework`, `cli`): tested by hand-clustering unity's real 62 TO DO tickets. Produced
~11 categories, mean cluster 4.4 (usable at unity's scale — but pickle's own 17-ticket backlog
clustered to mean ~2, so `category:` should **not** be built speculatively; it only pays off
past a certain backlog size). Three real failure modes surfaced:

- a dominant, still-undifferentiated bucket (`framework` = 12 of 62);
- 4 singleton categories that add vocabulary without grouping anything;
- ~6 genuinely multi-category tickets (e.g. one ticket that is both "verify" and "docs"),
  forcing either an arbitrary single-value pick or a `categories: []` list that breaks
  "group by" as a clean partition.

Deriving categories from title prefixes also does not work on this corpus — unity's titles use
free prose dashes (`Framework-audit P3 —`, `Eval harness:`), not a controlled vocabulary.

**Chosen approach — `family:` as a ticket id (epic-as-ticket-id).** A ticket sets
`family: T-NNN` pointing at an umbrella ticket. The umbrella is an **ordinary ticket** — no new
entity, no second board, no new lifecycle — and the mechanism reuses the existing
`spawned-by`-style lineage validation (`internal/audit/audit.go:71-83` already checks lineage
ids exist without gating pickup; the same pattern applies to `family:`).

**Validated against real evidence, not hypothetical need:** unity's `tickets/NOTES.md` (228
lines) already hand-maintains exactly this concept, unsupported by tooling — 5 named,
goal-bearing, finite families recorded today: "T-122's five follow-ups" (T-180–T-184), "the
2026-07-24 field review family" (T-164/165/166/167/172/113 — NOTES.md's own words: "one root
cause… refine as a family"), "the rename three" (T-186 → T-187 → T-188, with a stated landing
order), "the x- commands" (T-147/148/150/153, already partly expressed via
`depends-on: [T-155]`), and "Is this tree green? family order". NOTES.md names the missing
feature explicitly: *"Hand annotations / curated ordering that survive `pickle board sync`
(this file is the workaround). Still parked — never filed."*

**Category vs. family are different jobs, not competing solutions to the same one:** `category`
answers "show me everything touching docs" (reusable scope, browse/filter); `family` answers
"what do I refine next" (goal-bearing, finite, ordered) — the second matches what unity's humans
actually built by hand. `category:` is deliberately **not** filed here — only worth revisiting
if browsing a large backlog is still painful once `family:` and curated order (below) exist.

**Scope for refinement:** board/serve should be able to group TO DO/READY rows by `family:` for
display. Curated pickup order *within* a family is a related but separate concern, already
scoped in **T-056** (make the serve dashboard writable) — note as a soft coupling, not a hard
`depends-on:`. `family:` does not replace impact ordering; it supplements it once a backlog is
large enough for impact to tie widely.

## Implementation Plan

**Feature branch (in `pickle`):** `feat/T-059-family-umbrella-grouping`, cut from `main`.

**Prerequisite gate.** No hard `depends-on:`. Soft coupling only: **T-056** (writable serve
dashboard) owns *curated pickup order within a family* — this ticket delivers only deterministic
grouping + ordering, not hand-curated order. Do not wait on T-056.

### Confirmed decisions (locked at refinement)

1. **`family:` is a single umbrella id**, not a list — `family: T-NNN`. A ticket belongs to at
   most one family; grouping stays a clean partition. (Rejected list form for the same
   multi-membership reason the Description rejected multi-`category:`.)
2. **Same-child only.** The umbrella and every member share one child-project. The board groups
   per child, so a cross-child family could not render as one group; the audit enforces this
   (unlike `depends-on:`/`spawned-by:`, which may cross children).
3. **No nesting.** An umbrella is an ordinary ticket that itself sets no `family:`. The audit
   errors if a `family:` target is itself a member of a family — keeps the partition flat.
4. **`family:` gates nothing** — pure lineage-style provenance, exactly like `spawned-by:`. It
   never blocks a move; `move.go` is untouched.
5. **`family:` is optional, NOT a required frontmatter key.** It is deliberately left out of
   `audit.requiredKeys` so the existing backlog stays green without a migration. `Scaffold`
   emits the `family:` line **only when `--family` is given** — a no-family ticket is
   byte-identical to today's scaffold (contrast `spawned-by: []`, which always renders; the
   asymmetry is justified because `family` is single-valued and optional).
6. **Ordering:** a family's rank in a TO DO/READY table = **its umbrella ticket's impact**
   (looked up across the whole tree, since the umbrella may live in another status). Loose
   tickets interleave by their own impact (a loose ticket is its own singleton family). Within
   one family: umbrella first, then members by impact desc (tie by id asc). Families/loose sort
   against each other by rank desc, tie by umbrella-id asc — fully deterministic, no new grade
   frontmatter. If the umbrella is missing (audit-dirty tree), fall back to the ticket's own
   impact so the board still renders.

### Tasks

1. **Model + parse — `internal/ticket/ticket.go`.**
   - Add `Family string` to `Ticket` (doc it: single umbrella id; lineage only, never a gate;
     same-child; `""` when none).
   - In `LoadAll`, set `Family: strings.TrimSpace(fm["family"])`.
   - `Scaffold(id, title, project, impact, complexity, cost string, spawnedBy []string, family string)`:
     add the `family` param; emit `family: <id>\n` on its own line **immediately after
     `spawned-by:`** only when `family != ""`; omit the line entirely when empty. Add/extend a
     test asserting (a) empty family ⇒ no `family:` line and byte-identical to the pre-change
     scaffold, (b) `--family T-050` ⇒ `family: T-050` sits right after the `spawned-by:` line.

2. **Authoring flag — `internal/cli/ticket.go` + `internal/cli/cli.go`.**
   - Add `--family` to `ticket new` (`fs.String("family", "", …)`). Shape-check a non-empty
     value with `ticket.ValidID`; reject with `errf("--family: %q is not a valid ticket id", v)`.
     Existence stays the audit's job (mirror the `--spawned-by` comment at ticket.go:103-105).
   - Pass it through to `Scaffold`.
   - Update `ticketNewUsage` and the `cli.go` usage block (lines ~96-99) to document
     `[--family T-NNN]` and one line: `--family groups this ticket under an umbrella (never
     gates pickup; same child)`.

3. **Audit — `internal/audit/audit.go`.** In the per-ticket loop, after the `spawned-by`
   block, validate `family` **only when non-empty** (it is not in `requiredKeys`):
   - self-reference: `t.Front["family"] == t.ID` ⇒ `errf("%s: family lists itself", ref)`.
   - existence: target not in `byID` ⇒ `errf("%s: family %s does not exist", ref, fam)`.
   - same-child: target exists but `target.Project() != t.Project()` ⇒
     `errf("%s: family %s is in a different child-project", ref, fam)`.
   - no nesting: target exists and `target.Front["family"] != ""` ⇒
     `errf("%s: family %s is itself a family member (no nesting)", ref, fam)`.
   - Comment it like the `spawned-by` block: family gates nothing; these are existence/shape
     invariants only.

4. **Board display — `internal/board/board.go`.**
   - `SectionColumns`: append `"family"` to the `TO DO`/`READY` column list (last column, per
     the refinement preview).
   - `cellFor`: `case "family": return t.Front["family"]` (empty for umbrella/loose; passes
     through `sanitizeCell` like every cell).
   - Family-aware ordering. Add `func FamilyRank(all []*ticket.Ticket) map[string]int` returning
     each ticket id → the `impactRank` of its family key (umbrella's impact, or own impact when
     loose or umbrella-missing). Change `Sort` to
     `func Sort(group []*ticket.Ticket, name string, byID map[string]*ticket.Ticket)`; only the
     `TO DO`/`READY` branch uses `byID`. Comparator for those sections, in order:
     famRank desc → famID asc (`family` value or own id) → umbrella-before-member
     (`t.Front["family"] == ""`) → own impact desc → id asc. Other sections keep id asc and
     ignore `byID`.
   - `Render`: build `byID := map[string]*ticket.Ticket{}` from all `tickets` once, pass to
     every `Sort` call.

5. **Serve — `internal/serve/view.go` + templates.** `board.Sort` is the single ordering rule
   the dashboard already reuses (view.go:84), so grouping is inherited once the signature
   changes.
   - `buildBoard`: build the same `byID` map from all tickets, pass to `board.Sort`.
   - `Entry`: add `Family string`; set `Family: t.Front["family"]` in `newEntry`.
   - `TicketView`: add `Members []string` (tickets whose `family` names this id) — the reverse
     edge, mirroring the existing `Spawned`/`Blocks` build in `buildTicket`.
   - `internal/serve/templates/board.html`: render a `family <id>` edge span next to the
     existing `depends on` edge when `.Family` is set.
   - `internal/serve/templates/ticket.html`: add a `family` `<dt>/<dd>` row (link the id) and a
     `members` row from `.Members`.

6. **Docs.**
   - `.claude/skills/ticket-flow/resources/TEMPLATE.md`: add a `family:` frontmatter line after
     `spawned-by:` — `family:                     # optional single umbrella id (same child); groups pickup order; NEVER gates`.
   - `.claude/skills/ticket-flow/resources/tickets-README.md` §3 (IDs, priority, dependencies,
     and lineage): add a **Families** bullet after the Lineage bullet — umbrella-as-ordinary-ticket,
     single id, same child, flat (no nesting), never gates, and that TO DO/READY group + rank by
     the umbrella's impact.
   - `README.md`: does NOT enumerate frontmatter fields (grep confirmed empty) — skip.

### Applicability-gate amendments (2026-07-28, non-blocking)

Fresh-agent audit confirmed the plan CLEAN in design; four mechanical additions the tasks
missed:

- **Task 1** — updating `Scaffold`'s signature breaks **5 call sites**, not 1. Besides
  `internal/cli/ticket.go:132`, pass the new trailing family arg (`""` when none) at the test
  helpers `internal/move/move_test.go:42`, `internal/sync/sync_test.go:36`, and the existing
  `internal/ticket/ticket_test.go` calls (~235/265/294).
- **Task 4/5** — changing `Sort`'s signature also breaks the in-package test caller
  `internal/board/board_test.go:475` (`Sort(group, "TO DO")`) — give it the new `byID` arg.
- **Task 4** — collapse the `FamilyRank` helper vs. `byID`-into-`Sort` overlap into **one**
  lookup mechanism: pass `byID map[string]*ticket.Ticket` into `Sort` and resolve the family
  key inline (drop the separate `FamilyRank` export).
- **Task 6** — edit the **canonical `skill/resources/`** files (what the drift test reads via
  `../../skill/resources/…`), not the `.claude/skills/ticket-flow/` symlink view.

### Acceptance test

Self-host policy forbids running `install`/`board sync` against this repo, so acceptance is Go
unit tests (the child's `just test`) plus one throwaway-dir smoke with a copied binary.

1. **Unit tests — `just test` green**, including new cases:
   - `internal/ticket`: `Scaffold` empty-family omits the line / `--family` places it after
     `spawned-by`; `LoadAll` parses `Family`.
   - `internal/audit`: table-driven cases for each of the four family errors (self, missing,
     cross-child, nesting) and a clean same-child family (no error).
   - `internal/board`: a fixture with an umbrella (impact `high`) + two members (impact
     `medium`) + a loose ticket (impact `high`) asserts the rendered `TO DO` table has a
     `family` column and rows ordered: loose-high and the family adjacent by umbrella rank,
     umbrella before its members, members by impact desc then id asc.
   - `internal/serve`: `buildTicket` populates `Members` for an umbrella.
2. **Throwaway-dir smoke** (copy the freshly built binary out, never run WIP against this repo):
   ```
   just build
   D=$(mktemp -d) && cp pickle "$D/pk" && cd "$D"
   ./pk install --project demo .
   ./pk ticket new "umbrella epic" --project demo --impact high
   ./pk ticket new "step one" --project demo --impact medium --family T-001
   ./pk ticket new "step two" --project demo --impact medium --family T-001
   ./pk ticket new "loose" --project demo --impact high
   ./pk board sync
   grep -q '| family |' tickets/BOARD.md          # column present
   ./pk board audit                                # clean
   # negative: point a family at a missing / cross-nothing id by hand-edit, expect audit error
   ```
   Assert the TO DO table lists T-001 immediately above T-002/T-003 (umbrella first, members
   adjacent) and that `board audit` is clean; then hand-edit one member's `family:` to a
   non-existent id and confirm `board audit` reports `family … does not exist`.

### Docs step

`just docs-check` green after the TEMPLATE/README/rules edits.

### Finish step

Run `just build`, `just test`, `just lint`, `just docs-check` until all green. Write the
implementation summary. Prepare the child commit message
`feat(cli): group tickets under a family umbrella for board ordering (T-059)` (local commit on
the ticket branch; **no push / no MR without user approval** per the commit policy). Then
`pickle ticket move T-059 in-review --reason "acceptance green"`.

## Implementation summary

Delivered on `feat/T-059-family-umbrella-grouping` (commit `68627d1`). `family:` is an optional
single umbrella id, same-child, lineage-only (gates nothing) — grouping + deterministic
ordering only; curated intra-family order stays with T-056.

- **Model** (`internal/ticket/ticket.go`): `Ticket.Family` parsed from frontmatter; `Scaffold`
  gained a `family` param and emits the line **only when set** (a no-family scaffold is
  byte-identical to before, so the existing backlog needs no migration — `family` stays out of
  the audit's `requiredKeys`).
- **Authoring** (`internal/cli/ticket.go`, `cli.go`): `ticket new --family T-NNN`, shape-checked
  at creation via `ticket.ValidID`, existence deferred to the audit (same split as
  `--spawned-by`).
- **Audit** (`internal/audit/audit.go`): validates `family` only when set — must exist, be
  same-child, not self, and not nest (umbrella must not itself be a member).
- **Board** (`internal/board/board.go`): new `family` column on TO DO/READY; `Sort` now takes a
  whole-tree `byID` map and orders by famRank (umbrella's impact) → famID → umbrella-first →
  own impact → id, keeping families contiguous. Falls back to own impact if the umbrella is
  missing so an audit-dirty tree still renders.
- **Serve** (`internal/serve/view.go`, templates): inherits ordering via the shared `board.Sort`;
  `Entry.Family` + `TicketView.Members` reverse edge; board card + ticket page show the edges.
- **Docs**: `skill/resources/TEMPLATE.md` frontmatter line, `tickets-README.md` §3 Families
  bullet. README does not enumerate frontmatter fields (skipped).

Acceptance: `just test/build/lint/docs-check` all green (new tests in ticket/cli/audit/board/serve);
throwaway-dir smoke confirmed the column, umbrella-first ordering, clean audit, and the
`family … does not exist` error on a dangling id.

## Review

Reviewed 2026-07-28 on `feat/T-059-family-umbrella-grouping` (impl commit `68627d1`).

**Checklist**

- [x] Implementation audit — acceptance test re-run, tasks & criteria verified (step 2)
- [x] Quality audit (step 3)
- [x] Consistency audit (step 4)
- [x] Documentation audit — coverage, whole-tree sweep, docs build clean (step 4a)
- [x] Docs-readability pass — skipped: no docs-readability reviewer configured in this session (step 4b)
- [x] Findings recorded with severity + disposition (step 5)
- [x] Ticket moved + `## History` appended (step 6)
- [x] Board regenerated by the move (step 7)
- [x] Remaining-tickets impact sweep done — no ticket references T-059 (step 8)

**Implementation audit — all met.** Every task landed in the named files: model +
`Scaffold` param (`internal/ticket/ticket.go`), `--family` flag with shape-check
(`internal/cli/ticket.go`, `cli.go`), four family invariants (`internal/audit/audit.go`),
board `family` column + family-contiguous `Sort` (`internal/board/board.go`), serve
`Entry.Family`/`TicketView.Members` + template edges, and the canonical `skill/resources/`
docs. All five `Scaffold` call sites + the in-package `Sort` test caller were updated
(applicability-gate amendments applied). `just build/test/lint/docs-check` all green.
Throwaway-dir smoke re-run verbatim: `family` column present, ordering `T-001` (umbrella,
high) → `T-002`/`T-003` (members, adjacent) → `T-004` (loose, high), `board audit` clean,
and a hand-dangled `family:` produces `family T-999 does not exist`.

**Findings**

| id | severity | disposition | description | evidence | suggestion |
|---|---|---|---|---|---|
| F1 | non-blocking | fixed inline | Stray blank line inside the `cellFor` switch after the `family` case — cosmetic, this-branch-authored, no behaviour change. | `internal/board/board.go` (was L176-177) | Removed; committed `style(board): drop stray blank line…` on the feature branch. |
| F2 | non-blocking | noted | Family/loose grouping tie-break sorts on the id **string** (`fa < fb`), whereas the rest of the board ties by numeric `Num`. Harmless while ids are fixed-width `%03d`-padded (`T-001`..`T-999`, lexical == numeric), but would diverge at 4-digit ids (`T-1000` sorts before `T-999`). | `internal/board/board.go` `Sort` comparator | Latent only; if pickle ever passes 999 tickets in a child, compare on `Num`/`SplitID` there. Not scheduled. |

**Disposition summary:** 2 non-blocking, 0 blocking — F1 fixed inline, F2 noted. No
follow-up ticket (neither passes the promotion test). Verdict: **DONE**.

## History

- 2026-07-28 — created (TO DO). source: pickle ticket new
- 2026-07-28 — refined → READY. plan written; impact re-graded medium-high→medium.
- 2026-07-28 — applicability gate (fresh agent): CLEAN design, AMEND — 4 non-blocking
  mechanical additions applied inline to the plan (extra Scaffold/Sort call sites, single
  lookup mechanism, skill/resources docs path).
- 2026-07-28 — TO DO → READY: plan complete; impact re-graded medium-high->medium (single value)
- 2026-07-28 — READY → IN DEVELOPMENT: picked up; applicability gate clean
- 2026-07-28 — IN DEVELOPMENT → IN REVIEW: acceptance green
- 2026-07-28 — IN REVIEW → DONE: review clean; 2 non-blocking (1 fixed inline, 1 noted)
- 2026-07-28 — merged to main (#7)
