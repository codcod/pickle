---
id: T-020
title: doctor: detect AGENTS.md marker-block drift
project: pickle
depends-on: []
impact: medium
complexity: low
cost: S
---

# T-020 — doctor: detect AGENTS.md marker-block drift

## Description

Split out of **T-018** during its refinement (user decision, 2026-07-25): T-018 stops
`pickle upgrade` from destroying hand-written content, but the *detection* half is a distinct
`doctor` feature with its own design questions, so it is tracked here.

`doctor.checkMarkers` (`internal/doctor/doctor.go:111-137`) only checks that a marker **pair is
present** — never whether the installed block still matches what `markerBlock`
(`internal/install/install.go:506+`) would render today. Consequences:

- A project whose marker block was hand-edited gets a clean bill of health from `doctor`, then
  silently loses those edits on the next `pickle upgrade`. The loss is invisible **before** the
  fact (doctor says OK) and **after** it (nothing reports what changed).
- After a `pickle upgrade` that changes the payload, a project whose block was *not* re-injected
  (e.g. an install predating a `markerBlock` change) is equally undetectable.

Add a drift check: render `markerBlock(cfg)` and compare it to the block currently between
`MarkerBegin`/`MarkerEnd` in `AGENTS.md` (and `CLAUDE.md` when it is a regular file); report when
they differ.

**Open design questions for refinement** (why this is not a one-liner):

1. **Severity** — warning or error? A hand-edited block is a legitimate user choice until they
   run `upgrade`; erroring would make `doctor` fail on a working project. Probably a warning,
   but `doctor`'s exit-code contract (`0` errors / warnings tolerated) needs stating.
2. **Report granularity** — "differs" vs a real diff. A unified diff of a ~30-line block is a
   lot of `doctor` output; a summary ("3 lines differ; run `pickle upgrade` to regenerate, or
   move hand-written content outside the markers") may be better.
3. **Normalisation before comparison** — trailing whitespace and final-newline handling must not
   produce false positives; decide whether comparison is byte-exact or whitespace-normalised.
4. **Interaction with T-018's outcome.** T-018 makes `markerBlock` render commands, WIP limits
   and `branch_prefix` from `pickle.toml`, so after it lands a correctly-installed project should
   be **byte-identical** and drift becomes a meaningful signal rather than near-universal noise.
   Refine this ticket **after** T-018 lands, or the check will fire on every existing install.

Soft couplings (no hard `depends-on`): **T-018** — should land first (see #4); it also fixes the
stale marker sentence this check would otherwise flag everywhere. **T-017** (unify marker-pair
detection) — owns the `markerSpan`/`HasMarkerBlock` helper this check should reuse rather than
add a fifth copy of the marker-scanning logic. **T-013** — touches the same `injectMarker` surface.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-25 — created (TO DO). source: pickle ticket new
