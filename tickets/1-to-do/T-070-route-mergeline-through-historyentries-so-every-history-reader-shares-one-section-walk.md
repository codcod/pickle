---
id: T-070
title: route MergeLine through HistoryEntries so every ## History reader shares one section walk
project: pickle
depends-on: []
spawned-by: [T-043]
impact: low
complexity: low
cost: S
---

# T-070 — route MergeLine through HistoryEntries so every ## History reader shares one section walk

## Description

Filed by the **T-043 review** (finding R2, disposition `new ticket`). T-043 routed
`LastHistoryStatus` and `LastHistoryReason` through `HistoryEntries` so the readers of a ticket's
`## History` could not disagree about where an entry ends. It left the fourth reader behind.

`MergeLine` (`internal/ticket/ticket.go:391`) still carries its own copy of the section walk: its
own `inHistory` flag, its own `## ` heading test, its own `historyRE` match, its own
newest-wins loop. Two consequences:

1. **The duplication itself.** The section-walk now exists in exactly two places instead of one —
   `HistoryEntries` and `MergeLine` — and `MergeLine` is the copy nobody will remember to update.
   This is the pattern this repo files tickets about (see **T-042**, and T-043's own item 6, which
   collapsed three independent `LastIndex(body, "→")` re-derivations into one `transitionTarget`).
2. **It does not fold continuation lines.** `HistoryEntries` folds an indented follow-on line back
   into the entry above; `MergeLine` reads only the first physical line. So a wrapped merge line is
   silently truncated, and the board's DONE `merged` cell (`internal/board/board.go:178`) plus
   `serve`'s ticket view (`internal/serve/view.go:148`) show the truncation.

Verified on both sides of T-043's branch — this is pre-existing, not a T-043 regression:

```
MergeLine("## History\n- 2026-08-06 — merged to main\n  (abc1234) after review\n")
  = "merged to main"        // want "merged to main (abc1234) after review"
```

**Impact is `low` deliberately.** A conventional merge line (`merged to main (abc1234)`) is far too
short to wrap, so no ticket in the tree is affected today and `HasMergeLine`'s gate verdict is
unchanged either way (a truncated line is still non-empty). The value is the collapse to one walk,
plus removing the one reader that can still disagree with the other three. A `board audit` /
`board sync` before-and-after on the real tree is the guard, exactly as T-043 D8 used it.

### Not in scope

- **T-043's R1** (transition classification vs. continuation folding) is being fixed **in T-043's
  own rework pass**, not here. If that pass changes where the target is derived from, land it
  first and follow its shape.
- `HistoryEntries`' own contract, `historyKind`'s freeze-on-first-physical-line rule, and
  `historyRE` are untouched.
- **T-042** owns the *other* duplication cluster (status headings, marker span, test payload root).
  Different files; no hard dependency, and there is no reason to run them together.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-06 — created (TO DO). source: T-043 review finding R2 (disposition `new ticket`) — the
  last `## History` reader that still walks the section itself, and the only one that does not fold
  continuation lines. Filed narrow on purpose: R1 (transition classification) went to T-043's
  rework instead, so this ticket is the `MergeLine` unification alone
