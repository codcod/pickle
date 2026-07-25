---
id: T-021
title: project add|remove leave the AGENTS.md marker block stale
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: low
cost: S
---

# T-021 — project add|remove leave the AGENTS.md marker block stale

## Description

Non-blocking finding 7 from the T-018 review.

`pickle project add` and `pickle project remove` mutate `pickle.toml` (`internal/cli/project.go:95`
and `:137` both call `cfg.Save("")`) and then return. Neither calls `injectMarker`, so the
`AGENTS.md`/`CLAUDE.md` marker block still describes the *previous* set of children. `doctor` does
not help: `checkMarkers` (`internal/doctor/doctor.go:111-137`) tests only for the presence of the
delimiters, never for freshness.

The staleness pre-dates T-018, but T-018 changed its severity. Before, the block inlined only the
child **name list**; now it renders each child's **commands, branch prefix and WIP limits** too
(`markerBlock`, `internal/install/install.go:516+`). Reproduced during the review: after
`pickle project add web sub` with WIP 5/5, `AGENTS.md` still listed only `demo`, with `demo`'s
commands and `≤ 1` limits. An agent reading that will refuse legitimate work on a project
configured at 5, and will run a build command that does not exist for the child it is working on.
The marker block is the agent's primary instruction file, so wrong facts there get acted on.

Two candidate fixes, not yet decided:

1. **Re-inject on mutation** — call `injectMarker(root/AGENTS.md, …, markerBlock(cfg), …)` (and the
   `CLAUDE.md` variant, under the same regular-file guard `Upgrade` uses) at the end of
   `runProjectAdd`/`runProjectRemove`. ~6 lines, makes the block correct by construction, and
   matches the claim already in `markerBlock`'s doc comment (`install.go:509-514`) that
   regenerating it "cannot silently drop project-specific facts".
2. **Detect the drift** instead, in `doctor` — but that is **T-020**'s subject, and detection
   without correction still leaves the user to run `upgrade` by hand.

They are complementary rather than exclusive; decide during refinement.

Soft coupling: **T-020** (doctor marker-drift detection — refine the two together, they share the
comparison logic) and **T-018**, which introduced the amplification and deliberately scoped drift
out via its decision 9.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-25 — created (TO DO). source: pickle ticket new
