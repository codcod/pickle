---
id: T-035
title: repair the corrupted T-021 board row (unescaped pipe in title)
project: pickle
depends-on: []
spawned-by: [T-030]
impact: low
complexity: low
cost: S
---

# T-035 — repair the corrupted T-021 board row (unescaped pipe in title)

## Description

This repo's own `tickets/BOARD.md` carries a live instance of the unescaped-pipe bug. Found by the
T-030 applicability gate (2026-07-25) while checking that T-014's `|` case still reproduces — it
does, and we shipped one into our own data:

```
tickets/BOARD.md:60
| T-021 | project add|remove leave the AGENTS.md marker block stale | medium | low | S | [] |
```

Seven cells in a six-column table. `./pickle board audit` reports **0 error(s)** on it, because the
row check only locates the id. Rendered, the row shows a spurious column and the grades shift left.

### It is not cosmetic: the row silently misplaces every new row added after it

Measured while filing this ticket. `insertIntoBoard` decides where a new TO DO/READY row goes by
reading each existing row's impact as `cells[3]` of `strings.Split(lines[i], "|")`
(`internal/board/board.go:282-286`) and inserting before the first row whose
`impactRank[rowImpact] < newRank` (`:287`). The extra pipe shifts that index, so this row's apparent
impact is the string `"remove leave the AGENTS.md marker block stale"`, whose `impactRank` is the
zero value **0** — lower than every legal impact (`low` is 1, `internal/board/board.go:17-20`).

So the corrupt row acts as a wall: **every newly added row is inserted immediately before T-021**,
whatever its impact. Observed live when this ticket and T-034 were created — T-034 (`medium`) landed
there harmlessly, and T-035 (`low`) landed inside the *medium* block, four rows above where it
belonged, and had to be moved by hand. Any future `ticket new` hits the same wall until this is
repaired.

Consequence for the repair's form, which the escaping decision must account for: escaping as `\|`
fixes the *rendered* table but **not** this parse — `strings.Split(lines[i], "|")` is not
escape-aware, so `cells[3]` stays wrong and the insert point stays broken. Only **replacing** the
character (or making the row parse escape-aware, which is code and therefore T-014/T-034's) actually
restores ordering. Flag this to T-014 when it is refined.

This is a **data repair, not a code change** — which is why it is its own ticket rather than a
drive-by edit. The repair is one line, but the *form* of the repair is a decision that belongs with
the escaping rule:

- If **T-014** escapes cells as `\|`, this row should end up byte-identical to what
  `board.renderRow` would now emit for that title, so the file matches the renderer.
- If T-014 instead **replaces** `|` (e.g. with `/`), the repair should match that.
- Repairing it *before* T-014 decides risks a second, different-looking fix later.

The title itself is legitimate — `project add|remove` names two subcommands and reads naturally — so
the answer is not "rename T-021".

### Scope

- Repair `tickets/BOARD.md:60` so the row has exactly six cells, consistent with whatever escaping
  rule **T-014** lands.
- Check the rest of the real board and every ticket title for the same hazard (only this one row is
  known corrupt today — verify at implementation time, since titles keep arriving).
- Optionally add the row to a fixture so the repair is regression-covered rather than a one-off.

Out of scope: the escaping rule itself (**T-014**), the audit check that would have caught it
(**T-034**), and the input-boundary rejection that stops newlines but deliberately **not** pipes
(**T-030**).

> **DROPPED 2026-07-26 — read this before assuming the row is fixed.** The rendering half was
> repaired inline (`| T-021 | project add\|remove … |`) and this ticket dropped as not worth the
> overhead. That is correct as far as it goes, but the **insert-point wall documented above survives
> the escape**: `strings.Split` is not escape-aware, so `cells[3]` is still garbage and every new TO
> DO row is still inserted in front of T-021. **T-034** owns that residual. Do not read the drop as
> "the T-021 row is now fully sound".

### Couplings

`spawned-by: [T-030]` — found by T-030's applicability gate.

Soft couplings (no `depends-on`, no ordering enforced):

- **T-014** — owns the escaping rule this repair must match. Strong argument for landing T-014 first,
  or for doing this repair *as part of* T-014's acceptance test (a legitimate outcome of refining
  this ticket is "fold into T-014 and drop"). Raise that with the user at refinement.
- **T-034** — the audit check that would flag this row. Should land *after* the repair, or its
  regression gate against the real `tickets/` starts red.
- **T-030** — does not fix this: it rejects newlines in titles at the input boundary, and
  deliberately leaves `|` to the render boundary.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-25 — created (TO DO). source: pickle ticket new
- 2026-07-26 — TO DO → DROPPED: board row repaired inline during triage; ticket overhead exceeded the one-character fix
- 2026-07-26 — correction to the drop note above (append-only, so the original line stands): the
  inline repair escaped the title to `project add\|remove`, which fixes the **rendered** table but
  **not** the functional half this ticket documented. `insertIntoBoard` splits rows with
  `strings.Split(lines[i], "|")` (`internal/board/board.go:282`), which is not escape-aware, so the
  escaped row still yields 8 fields and `cells[3]` is still the title fragment
  `"remove leave the AGENTS.md marker block stale"` → `impactRank` **0**. Re-measured against the
  repaired board on 2026-07-26. The insert-point wall therefore survives the drop and every new TO
  DO row is still inserted in front of T-021. That residual is **T-034**'s, which documents this
  exact analysis — the drop is sound *because* T-034 owns it, not because the bug is gone.
