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

### 2. Board status-heading matching — four copies (T-015)

"Match a `## ` heading to its status display name, longest-name-first so a prefix cannot shadow a
longer one" exists in `board.Parse`, `board.ParseCells`, `sync.matchStatus`, and a variant in
`board.sectionSpan` (`board.go:~329`). Extract `board.MatchStatusHeading(line) string` (returning
`""` for a non-status heading) and route all four through it; the shared helper can memoise the
sorted status-name slice instead of rebuilding it per call.

T-015's second item is a **test gap, not duplication**: `TestSyncTerminalMembership`
(`internal/sync`) covers only the first half of decision D3 ("a DONE ticket not on the board is
not re-added"); the second half ("a DONE ticket under the **wrong** section is relocated to DONE
with its `merged` cell") is unwritten. Writing it requires deciding whether a `merged` cell should
survive relocation *from a wrong section* — it currently does not, because `board.ParseCells` keys
carry-over cells by the columns of the section the row was found in. Likely acceptable (a misfiled
row's cells were never DONE-shaped); encode whichever way in the test.

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
- **T-039** touches `internal/board` and `internal/sync` heavily (render, sync rebuild). Item 2
  edits `board.Parse`/`ParseCells`/`sectionSpan` and item 1 of T-015's test gap edits
  `internal/sync` — **high collision risk**. Sequence after T-039, or coordinate carefully.
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
