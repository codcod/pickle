---
id: T-017
title: unify marker-pair detection + dry-run fidelity
project: pickle
depends-on: []
impact: medium
complexity: low
cost: S
---

# T-017 — unify marker-pair detection + dry-run fidelity

## Description

Non-blocking follow-ups from the T-006 review. `pickle uninstall --dry-run` works on the golden
path, but its preview is computed by a *second, divergent* copy of the marker-pair predicate, so
it can disagree with the real run — the one property a dry-run must never violate.

1. **Marker-pair detection now exists in four places, and the newest copy diverges.**
   - `internal/install/install.go:244` (T-006's dry-run branch):
     `strings.Contains(MarkerBegin) && strings.Contains(MarkerEnd)` — **no ordering check**
   - `internal/install/install.go:475-480` (`stripMarker`): `bi < 0 || ei < bi` → skip
   - `internal/install/install.go:429-431` (`injectMarker`): `bi >= 0 && ei > bi`
   - `internal/doctor/doctor.go:139-149` (`hasMarkerBlock`): `bi >= 0 && ei > bi`

   **Reproduced during the T-006 review** on an `AGENTS.md` containing `<!-- pickle:end -->`
   *before* `<!-- pickle:begin -->`: `uninstall --dry-run` reports
   `- AGENTS.md (marker, dry-run)` while the real `uninstall` reports `= AGENTS.md (no marker)`.
   Extract one helper (e.g. `func markerSpan(text string) (start, end int, ok bool)`) in
   `internal/install`, route `injectMarker`, `stripMarker`, and the dry-run branch through it,
   and export a thin `install.HasMarkerBlock(path)` for `internal/doctor` (which already imports
   `install`) — collapsing four predicates into one. Same de-duplication class as T-015, but a
   disjoint layer (markers, not board status headings).

2. **Dry-run labels don't match the real run's labels.** For the skill dir, dry-run always emits
   `SkillDir + " (dry-run)"` (`install.go:178-179`) while the real run distinguishes
   `" (symlink)"` from `"/"` (`install.go:186-191`). Make the preview report the same
   classification it will act on, so `--dry-run` output is diffable against the real run.

3. **Dead branch in `stripMarker`.** `install.go:466-468` records `rel + " (absent)"`, but the
   only caller (`uninstallMarkerFile`, `install.go:251`) reaches it *after* a successful
   `os.Lstat` (`install.go:223-226`) that already returns early on absence with **no** `Result`
   entry. So the `(absent)` line can never print, and a missing `AGENTS.md` produces no `=` line
   at all — unlike every other skip. Drop the dead branch, or (better) have
   `uninstallMarkerFile` record the absence so the summary stays uniform.

Add a table test over marker-pair shapes (well-formed / reversed / begin-only / end-only /
absent) asserting `--dry-run` and the real run agree on every input. No user-facing behaviour
change on the golden path.

Soft coupling: **T-013** touches the same functions — its item 1 rewrites `injectMarker`'s
separator logic (which item 1 here re-routes through `markerSpan`), and its items 2/8 rewrite the
same `Result` summary labels that item 2 here changes, while its item 7 would add a field to
`Result`. Sequence the two tickets or keep the edits disjoint.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-24 — created (TO DO). source: pickle ticket new
