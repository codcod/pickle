---
id: T-019
title: README accuracy polish (prose duplicates command table, phased-plan tagging)
project: pickle
depends-on: []
impact: low
complexity: low
cost: S
---

# T-019 — README accuracy polish (prose duplicates command table, phased-plan tagging)

## Description

Non-blocking documentation polish surfaced by the T-006 review's whole-tree sweep (step 4a.2).
Pure `README.md` accuracy/redundancy work — no behaviour change. (The *blocking* docs gaps found
by that review — the undocumented `uninstall --dry-run` flag, the undocumented
"uninstall keeps `pickle.toml`" contract, and the stale `pickle.toml` normalisation sentence —
are fixed in T-006's own rework pass, not here.)

1. **The status prose duplicates the table it sits under.** `README.md:87-90` re-lists all eight
   command→ticket mappings that the `[done: T-NNN]` tags at `README.md:75-83` already carry, so
   every future command change needs two edits and they will drift. Replace with a single
   sentence and let the table's tags be the single source of truth. While there: "the full
   command surface is implemented" is true of *commands* only — `install --agent` is still an
   accepted no-op (`internal/cli/install.go:27`), as `README.md:182-183` admits a few sections
   later; word it so the two passages agree (e.g. note that agent breadth beyond Claude Code is
   still pending, tracked by T-009/T-010).

2. **Phased-plan tagging is inconsistent.** In `README.md:292-303` only P5 carries
   `**[done: T-011]**`, yet P1 (T-002/T-003) and P3 (T-007/T-008) are fully delivered and
   untagged. Tag them. P2 is a deliberate exception: its "skill install (Claude Code + **Zed/Pi**)"
   clause (`README.md:296-298`) is still outstanding (T-009/T-010), so either split that clause
   out to P4 so P2 can be tagged truthfully, or leave P2 untagged with a one-line note saying
   why. Decide during refinement.

> **Line refs re-anchored 2026-07-24** (T-006 scoped re-review): T-006's rework inserted the
> `## pickle upgrade` / `## pickle uninstall` sections at `README.md:185-227`, shifting everything
> below. Items 1/2 originally cited `:180-181`, `:246-257`, `:250-252`; the refs above are the
> post-rework equivalents. `:87-90` and `:75-83` (item 1's main targets) were above the insertion
> and are unchanged. Item 1's scope now also covers the new sections' own phrasing only insofar as
> it must not re-duplicate the table.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-24 — created (TO DO). source: pickle ticket new
