---
id: T-041
title: keep the AGENTS.md marker block fresh and detect drift
project: pickle
depends-on: []
spawned-by: [T-020, T-021]
impact: high
complexity: medium
cost: M
---

# T-041 — keep the AGENTS.md marker block fresh and detect drift

## Description

**Epic — merged from T-020 and T-021 by the 2026-07-26 board triage.** Both are in
`tickets/7-dropped/` with their full reproductions; read them for detail.

The `AGENTS.md`/`CLAUDE.md` marker block is generated from `pickle.toml` and is what every agent
actually reads. Nothing keeps it in step with the config, and nothing detects that it has fallen
out of step. These are the **write half** and the **detect half** of one defect, and splitting
them guarantees two half-fixes: a freshness fix with no way to verify it, or a detector with
nothing to make the block correct.

### Absorbed scope

| from | half | substance |
|---|---|---|
| T-021 | write | `pickle project add` and `pickle project remove` mutate `pickle.toml` (`internal/cli/project.go:95`, `:137` — both call `cfg.Save("")`) and return. Neither calls `injectMarker`, so the block still describes the *previous* set of children. Reproduced: after `pickle project add web sub` with WIP 5/5, `AGENTS.md` still listed only `demo`, with `demo`'s commands and `≤ 1` limits. An agent reading that refuses legitimate work on a project it is not told about. |
| T-020 | detect | `doctor.checkMarkers` (`internal/doctor/doctor.go:111-137`) only checks that a marker **pair is present** — never whether the installed block matches what `markerBlock` (`internal/install/install.go:506+`) would render today. So a hand-edited block gets a clean bill of health and then silently loses those edits on the next `pickle upgrade`; and a block that predates a payload change is equally undetectable. |

### Why the severity grew

T-018 changed the stakes. The block used to inline only the child **name list**; it now renders
each child's **commands, branch prefix and WIP limits** (`markerBlock`,
`internal/install/install.go:516+`). A stale block is no longer a cosmetic name omission — it
publishes wrong build commands and wrong WIP limits as authoritative instructions.

T-020 was split out of T-018 during refinement (user decision, 2026-07-25) precisely because
detection is a distinct feature with its own design questions: what counts as drift when the user
*intended* the edit, and how `doctor` should report it without crying wolf on every legitimate
customisation. That question is now this epic's to settle, and it is the same question the write
half raises from the other side — re-injecting on `project add` must not clobber intentional
hand-edits.

### Cross-references

- **T-018** (done) established that `upgrade` must not silently discard the marker body; its
  surgical-edit approach is the model for re-injecting without destroying user content.
- **T-022** (still standalone) fixes the *skill payload* stating commit policy, branch prefix and
  WIP limits unconditionally — the same contradiction seen from the other surface. When both land,
  the payload defers to the block and the block is finally trustworthy; they are worth sequencing
  together but do not share code.
- **T-036**'s note-and-close valve and **T-044**'s generated-board design (superseded T-039's
  parse-back gate, 2026-07-26) are unrelated in code but share the "never silently destroy user
  content" principle.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-26 — created (TO DO). source: board triage — epic merged from T-020 and T-021, both
  moved to 7-dropped/ as absorbed
