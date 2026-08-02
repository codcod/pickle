---
id: T-042
title: collapse duplicated internal predicates into single helpers (skill-dir dry-run labels, test payload root)
project: pickle
depends-on: []
spawned-by: [T-015, T-017, T-032, T-041]
impact: low
complexity: low
cost: S
---

# T-042 — collapse duplicated internal predicates into single helpers (skill-dir dry-run labels, test payload root)

## Description

**Epic — merged from T-015, T-017 and T-032 by the 2026-07-26 board triage.** All three are in
`tickets/7-dropped/` with their full site inventories; read them for detail.

Three instances of the same defect shape: **one fact written in four or five places, and the
copies have begun to disagree**. Different layers, identical review criteria — extract one
helper, route every caller through it, prove behaviour is unchanged. Reviewing them separately
means making the same "is this the right choke point?" judgement three times.

They are ordered below by severity, because only the first has a **measured behavioural
divergence**; the other two are latent.

### 1. Marker-pair detection — ~~four copies, and the newest one diverges~~ **marker-span half resolved by T-041; the skill-dir dry-run-label sub-item remains** (T-017)

**Resolved by T-041 (2026-08-01).** T-041 needed to *read* the marker block's body (not just
detect its presence) for its drift check, which required exactly the extraction this item asked
for. It added `markerSpan(text) (bi, ei int, ok bool)` in `internal/install/install.go` — ordering
is part of the predicate, so a dry-run and the real run can no longer disagree about whether a
reversed `<!-- pickle:end -->`…`<!-- pickle:begin -->` pair is a marker block at all (T-041 pins
this with `TestUninstallDryRunAgreesOnReversedMarkers`) — and routed every in-package copy
(`injectMarker`, `stripMarker`, the `uninstall --dry-run` check) through it, plus an exported
`install.InstalledMarkerBody(path)` that `internal/doctor` now calls instead of its own
`hasMarkerBlock`. The **measured divergence this item was filed for is gone.**

T-017's *second* item is **not** touched by T-041 and remains this epic's scope: **dry-run labels
don't match the real run's labels** for the skill dir. `internal/install/install.go`: the dry-run
branch reports `res.removed(SkillDir + " (dry-run)")` (`:372`) while the real run reports
`(symlink)` (`:379`) or a bare `SkillDir + "/"` (`:384`) depending on which branch actually ran —
so a dry-run preview can name a different outcome than the real run produces. Same theme as the
marker-pair fix: the preview and the act must agree, this time on the *label*, not the *predicate*.

### 2. Board status-heading matching — ~~four copies~~ **resolved by T-044** (T-015)

**Patched 2026-07-26 by T-044's review impact sweep.** T-044 deleted three of the four copies
(`board.ParseCells`, `sync.matchStatus`, `board.sectionSpan`) along with the whole parse-back
machinery; the heading-match loop now exists exactly once, in the read-only `board.Parse`
(drift summary only). No duplication remains — this item is **dropped from scope**.

T-015's second item (the `TestSyncTerminalMembership` / D3 carry-over test gap) is likewise
obsolete: sync no longer carries cells over at all — terminal cells are derived from ticket
History at render time and every DONE/DROPPED ticket is always rendered (T-044 D3/D4). The
epic's remaining scope is item 1's skill-dir dry-run-label sub-item (its marker-span half was
resolved by T-041, 2026-08-01 — see above) and item 3 (test payload root).

### 3. Test payload root — five copies, four of them CWD-relative (T-032)

| site | idiom |
|---|---|
| `internal/install/install_test.go:15` | `payloadRoot()` returning `filepath.Join("..", "..")` |
| `internal/doctor/doctor_test.go:14` | the same function duplicated verbatim |
| `internal/move/move_test.go:20` | inlined `os.DirFS(filepath.Join("..", ".."))` |
| `internal/sync/sync_test.go:21` | inlined `os.DirFS(filepath.Join("..", ".."))` |
| `internal/cli/cli_test.go:19` (T-029) | package-level `repoRoot`, absolute, computed in `TestMain` |

The first four break if the test process's CWD ever moves — which **T-043**'s `TestMain` sandbox
does deliberately. Unify on the absolute, CWD-independent form.

### Sequencing

- **T-043** (test harness + coverage) touches `internal/cli/cli_test.go` and the same test files
  as item 3. Land one before the other, not concurrently; item 3 is arguably T-043's prerequisite,
  though not a hard `depends-on`.
- **T-044** (which superseded T-039, 2026-07-26) rewrote `internal/board` and `internal/sync`
  heavily (single renderer, sync becomes regenerate) and **deleted** the helpers item 2 would
  have unified (`ParseCells`, `sectionSpan`, `subgroupSpan`, sync's `matchStatus`) — the sweep
  after T-044's review dropped item 2 from scope (see above). The former collision risk with
  T-044 is gone; only the T-043 sequencing note (item 3) remains live.
- **T-040 deferred the `T-\d+` unification here — it is now this epic's, unconditionally**
  (T-040 decision D6, 2026-08-02: its self-reference check needed no shape check, so it touched no
  regex). The shape is spelled out three times: `idRE` and `filenameRE`
  (`internal/ticket/ticket.go:100-110`) and `board.rowRE` (`internal/board/board.go:51`);
  `ticket.go`'s comment now names T-042 as the owner.
- **T-013 item 6** ("project-root resolution is triplicated" across the setup commands) is the same
  defect shape and a natural fourth item, but T-013 stays standalone; fold it in only if T-013 is
  not picked up first.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-26 — created (TO DO). source: board triage — epic merged from T-015, T-017 and T-032,
  all three moved to 7-dropped/ as absorbed
- 2026-07-26 — patched by T-044's review impact sweep: item 2 (status-heading duplication +
  D3 carry-over test gap) dropped from scope — its targets were deleted by the generated-board
  rewrite; remaining scope is items 1 and 3
- 2026-08-01 — patched by T-041 (Finish step 3): item 1's marker-span half resolved — T-041
  extracted `markerSpan`/`InstalledMarkerBody` and routed every caller through them while
  building its own drift check; item 1's skill-dir dry-run-label sub-item (T-017) was not in
  T-041's scope and remains. Re-titled and re-graded (impact/complexity unchanged, cost M → S)
  to reflect the smaller remaining surface: item 1's label sub-item + item 3 (test payload root)
- 2026-08-03 — patched by T-040's review impact sweep (finding N8): T-040 deferred the `T-\d+`
  regex unification here by decision D6, so the conditional cross-reference ("if T-040 defers
  that") is now settled fact and its two stale line references were refreshed
  (`ticket.go:95` → `:100-110`, `board.go:29` → `:51`). Scope grows by one small item; not
  re-graded (still low/low/S).
