---
id: T-055
title: serve: the board's at-limit WIP badge is never highlighted (.count overrides .wip-full)
project: pickle
depends-on: []
spawned-by: [T-054]
impact: low
complexity: low
cost: S
---

# T-055 — serve: the board's at-limit WIP badge is never highlighted (.count overrides .wip-full)

## Description

On the `pickle serve` board page, a child-project whose WIP count has reached its limit is
supposed to be highlighted — `.wip-full` paints the badge `--warn` and bolds it
(`internal/serve/static/styles.css:53`). It never does.

`internal/serve/templates/board.html:25` emits the badge as
`<span class="count {{if ge .Count .Limit}}wip-full{{end}}">`, and `.count`
(`styles.css:73`) declares `color: var(--muted); font-weight: 400`. The two selectors have
**equal specificity** (0,1,0) and `.count` comes later in the sheet, so it wins on source
order: an at-limit badge renders identical to a badge with room to spare. The at-limit
signal is silently dead on the page where it matters most.

The health banner's copy of the same badge (`templates/layout.html:41-42`) *is* correct —
there the span carries `wip-full` alone with no competing class, which is why the bug
survived T-053's review: the reviewer saw a working highlight and a `wip-full` rule, on the
banner, and had no reason to check the board's variant.

The fix is one line — either give the combined selector real specificity
(`.count.wip-full { … }`) or move `.wip-full` after `.count`. The former is the more honest
statement of intent and is immune to further reordering. Whichever is chosen, it wants a
regression test: the current test suite asserts nothing about the board's WIP badge, and the
bug is exactly the kind that CSS ships silently. A test can at least pin the template's
class emission and the sheet's ordering/specificity relationship.

Worth confirming during refinement: whether `.count`'s `--muted` is deliberate elsewhere
(it also styles the per-status total at `board.html:22`), so the fix does not flip the
neutral counts to `--warn` as well.

Lineage: found by the applicability audit run before T-054 was picked up (`spawned-by:
T-054`), while checking which pairs of colours actually render against each other. It is a
pre-existing T-053 defect, not caused by T-054, and was deliberately kept out of T-054's
diff to keep that ticket purely about theming. Soft coupling: T-054 edits the same
stylesheet — if T-054 lands first, expect a trivial context conflict, nothing more.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-28 — created (TO DO). source: applicability audit before T-054's pickup, finding F7
  (pre-existing T-053 defect; promoted to its own ticket rather than folded, to keep T-054's
  diff purely about theming)
