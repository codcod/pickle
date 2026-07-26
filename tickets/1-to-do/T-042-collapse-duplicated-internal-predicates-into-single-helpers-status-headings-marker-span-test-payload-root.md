---
id: T-042
title: collapse duplicated internal predicates into single helpers (status headings, marker span, test payload root)
project: pickle
depends-on: []
spawned-by: [T-015, T-017, T-032]
impact: low
complexity: low
cost: M
---

# T-042 — collapse duplicated internal predicates into single helpers (status headings, marker span, test payload root)

## Description

**Epic — merged from T-015, T-017 and T-032 by the 2026-07-26 board triage.** All three are in
`tickets/7-dropped/` with their full site inventories; read them for detail.

Three instances of the same defect shape: **one fact written in four or five places, and the
copies have begun to disagree**. Different layers, identical review criteria — extract one
helper, route every caller through it, prove behaviour is unchanged. Reviewing them separately
means making the same "is this the right choke point?" judgement three times.

They are ordered below by severity, because only the first has a **measured behavioural
divergence**; the other two are latent.

### 1. Marker-pair detection — four copies, and the newest one diverges (T-017)

| site | predicate |
|---|---|
| `internal/install/install.go:244` (T-006 dry-run) | `Contains(MarkerBegin) && Contains(MarkerEnd)` — **no ordering check** |
| `internal/install/install.go:475-480` (`stripMarker`) | `bi < 0 \|\| ei < bi` → skip |
| `internal/install/install.go:429-431` (`injectMarker`) | `bi >= 0 && ei > bi` |
| `internal/doctor/doctor.go:139-149` (`hasMarkerBlock`) | `bi >= 0 && ei > bi` |

**Reproduced during the T-006 review** on an `AGENTS.md` containing `<!-- pickle:end -->` *before*
`<!-- pickle:begin -->`: `uninstall --dry-run` reports `- AGENTS.md (marker, dry-run)` while the
real `uninstall` reports `= AGENTS.md (no marker)`. A dry-run that disagrees with the real run is
the one property a dry-run must never violate. Extract `markerSpan(text) (start, end int, ok bool)`
in `internal/install`, route all three callers through it, and export a thin
`install.HasMarkerBlock(path)` for `internal/doctor` (which already imports `install`).

T-017 also carried a second item: **dry-run labels don't match the real run's labels** for the
skill dir. Same theme — the preview and the act must agree.

### 2. Board status-heading matching — ~~four copies~~ **resolved by T-044** (T-015)

**Patched 2026-07-26 by T-044's review impact sweep.** T-044 deleted three of the four copies
(`board.ParseCells`, `sync.matchStatus`, `board.sectionSpan`) along with the whole parse-back
machinery; the heading-match loop now exists exactly once, in the read-only `board.Parse`
(drift summary only). No duplication remains — this item is **dropped from scope**.

T-015's second item (the `TestSyncTerminalMembership` / D3 carry-over test gap) is likewise
obsolete: sync no longer carries cells over at all — terminal cells are derived from ticket
History at render time and every DONE/DROPPED ticket is always rendered (T-044 D3/D4). The
epic's remaining scope is items 1 (marker span) and 3 (test payload root).

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
- **T-040** may compose a shared `T-\d+` fragment from `filenameRE` (`internal/ticket/ticket.go:95`)
  and `board.rowRE` (`internal/board/board.go:29`). If T-040 defers that, it belongs here.
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
