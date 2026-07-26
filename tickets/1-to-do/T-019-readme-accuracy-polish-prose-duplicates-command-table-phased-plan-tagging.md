---
id: T-019
title: README accuracy polish (prose duplicates command table, phased-plan tagging)
project: pickle
depends-on: []
spawned-by: []
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
   accepted no-op (`internal/cli/install.go:27`), as `README.md:198-199` admits a few sections
   later; word it so the two passages agree (e.g. note that agent breadth beyond Claude Code is
   still pending, tracked by T-009/T-010).

2. **Phased-plan tagging is inconsistent.** In `README.md:322-331` only P5 carries
   `**[done: T-011]**`, yet P1 (T-002/T-003) and P3 (T-007/T-008) are fully delivered and
   untagged. Tag them. P2 is a deliberate exception: its "skill install (Claude Code + **Zed/Pi**)"
   clause (`README.md:326-328`) is still outstanding (T-009/T-010), so either split that clause
   out to P4 so P2 can be tagged truthfully, or leave P2 untagged with a one-line note saying
   why. Decide during refinement.

3. **A factual error about `board audit`'s severity** (added by the T-024 review's whole-tree
   sweep, finding N11). `README.md:316` claims the `ticket move` pickup gate "is intentionally
   stricter than `board audit`, which only warns." That is wrong: `internal/audit/audit.go:150`
   **errors** when an in-development ticket's dependency is not in `6-done/`; only the
   done-but-no-`MERGED`-line case warns (`:152-154`). The gate is stricter in *substance* (it
   demands a `MERGED` History line), not because the audit merely warns. Reword to say what the
   extra strictness actually is.

4. **`PLAN.md`'s `ticket new` synopsis is stale** (added by the T-030 review's whole-tree sweep,
   finding N5 — folded in here rather than spawned as its own ticket, per the T-036 concern about
   unbounded review spawn). `PLAN.md:227` reads
   `pickle ticket new "<title>" --project <name> [--impact … --complexity … --cost …]` — no
   `--spawned-by` (added by T-024) and no input contract (added by T-030). It does not *contradict*
   the current behaviour, it is an incomplete flag list in a roadmap document, which is why it was
   non-blocking; but it is now the **only** place in the tree whose `ticket new` synopsis disagrees
   with `README.md:274`, `internal/cli/ticket.go:35` and `pickle help`. Either sync it or add a
   "superseded by README" note at the head of that section. Note this widens the ticket beyond
   `README.md` despite the title — decide at refinement whether to retitle or split.

> **Line refs re-anchored again 2026-07-25** (T-018 re-review): T-018's rework added ~13 lines to
> the Configuration and `## pickle upgrade` sections, shifting everything below. Items 1/2 cited
> `:189-190`, `:307-318`, `:296-298` before this pass. Note T-018's own re-review left a **false**
> statement at `README.md:102-104` for its rework to fix — do not also fix it here.
>
> **Line refs re-anchored 2026-07-24** (T-006 scoped re-review): T-006's rework inserted the
> `## pickle upgrade` / `## pickle uninstall` sections at `README.md:185-227`, shifting everything
> below. Items 1/2 originally cited `:180-181`, `:246-257`, `:250-252`; the refs above are the
> post-rework equivalents. `:87-90` and `:75-83` (item 1's main targets) were above the insertion
> and are unchanged. Item 1's scope now also covers the new sections' own phrasing only insofar as
> it must not re-duplicate the table.
>
> **Re-anchored again 2026-07-25** (T-018 review): T-018 expanded the Configuration section
> (+8 lines) and the `## pickle upgrade` section (+7), shifting both of this ticket's targets
> down. `:182-183` → `:189-190`; `:292-303` → `:307-318`. Item 1's other refs (`:75-83`,
> `:87-90`) sit above the first insertion and are unchanged. Note T-018 also rewrote the
> Configuration paragraph itself, so item 1's "prose duplicates the table" complaint should be
> re-checked against the new wording before acting on it.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-24 — created (TO DO). source: pickle ticket new
- 2026-07-25 — re-anchored by the T-018 re-review: items 1/2 line refs shifted by T-018's README edits; noted that `README.md:102-104` belongs to T-018's rework
- 2026-07-25 — scope extended: item 3 (README.md:316 misstates board audit's severity), from the T-024 review's whole-tree sweep (finding N11)
