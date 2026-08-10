---
id: T-070
title: route MergeLine through HistoryEntries so every ## History reader shares one section walk
project: pickle
depends-on: []
spawned-by: [T-043]
impact: low-medium
complexity: low
cost: S
---

# T-070 — route MergeLine through HistoryEntries so every ## History reader shares one section walk

## Outcome

After this ships, `MergeLine` reads a ticket's `## History` through the same shared section walk every other History reader already uses, so a merge line with a folded continuation line is parsed correctly and a future change to that walk can't silently miss this fourth caller.

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

### Item 2 — resolve a transition's target and reason in one pass (T-043 review, R7)

Folded here by T-043's **scoped re-review** (finding R7, disposition `folded`): T-043's rework left
a deliberate asymmetry between the two halves of a transition.

- The **target** is frozen on the entry's *first physical line*, together with `Kind`
  (`HistoryEntry.Target`) — that is exactly what makes the two agree by construction, and it must
  stay that way.
- The **reason** is read from the *folded* text by `LastHistoryReason`, because folding a wrapped
  reason back together is the whole point of `HistoryEntries`.

Nothing checks that the two came from the *same* arrow. For a hand-authored entry whose
continuation line contains a second arrow, they need not:

```
- 2026-08-06 — TO DO → READY
  and later IN REVIEW → DONE: some clause
```

yields `LastHistoryStatus` = `"READY"` (first line) with `LastHistoryReason` = `"some clause"`
(the continuation's arrow). No entry in the tree looks like this and `pickle ticket move` cannot
write one, which is why it is `low` and folded rather than filed on its own.

**Shape of the fix:** have `HistoryEntries` resolve target *and* reason in the same pass it already
classifies `Kind` — e.g. store the reason on the entry too, taking it from the folded text but
anchored at the arrow the target was found at — so the pair provably belongs to one transition and
`LastHistoryReason` stops re-scanning. That lands naturally with item 1: both are "one pass, one
source of truth" for the `## History` readers.

### Why both items are one ticket

They are the same theme — the `## History` reader family sharing one path — and they touch the same
twenty lines of `internal/ticket/ticket.go`. Splitting them would mean two reviews of one function.

**Impact was `low` deliberately, and T-089 raised it to `low-medium`.** The original reasoning: a
conventional merge line (`merged to main (abc1234)`) is far too short to wrap, so no ticket in the
tree is affected today and `HasMergeLine`'s gate verdict is unchanged either way (a truncated line
is still non-empty). The value was the collapse to one walk, plus removing the one reader that can
still disagree with the other three.

T-089 invalidated the "far too short to wrap" premise: the merge line's recommended form now
carries a commit reference including a full commit URL, which measures ~113–125 runes before the
board's `yes — ` prefix (GitHub ~113, GitLab subgroup paths ~123). That is past the width tickets
in this tree wrap prose at, so a human following the current convention is now *likely* to wrap a
merge line across two physical lines — the exact input this ticket fixes, and one that today
silently truncates the board's `merged` cell. The defect is no longer hypothetical, which is the
whole of the re-grade; scope, complexity and cost are unchanged.

A `board audit` / `board sync` before-and-after on the real tree is the guard, exactly as T-043 D8
used it — and a wrapped merge line in the recommended T-089 form is now the obvious fixture to add.

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
- 2026-08-06 — patched by T-043's scoped re-review: gains **item 2** (finding R7, disposition
  `folded`) — a transition's target is frozen on the entry's first physical line while its reason
  is read from the folded text, and nothing checks that the two came from the same arrow. Same
  theme, same twenty lines, so it is an item here rather than a second ticket
- 2026-08-09 — impact low → low-medium; T-089's review impact sweep invalidated the "a merge line
  is far too short to wrap" premise — the recommended form now includes a full commit URL, so the
  wrapped-line defect this ticket fixes is reachable in normal use
- 2026-08-10 — patched by T-080's review impact sweep (step 8): **premise intact, and the fix got
  slightly smaller.** T-080 gave `MergeLine`/`HasMergeLine` a leading `def *flow.Definition`
  parameter (they call `historyKind`, which needs the status vocabulary), so the signature this
  ticket's fix requires — routing `MergeLine` through `HistoryEntries(def, text)` — is already in
  place; this ticket no longer has to introduce it. Everything this ticket is about is unchanged:
  `MergeLine` still carries its own `inHistory` flag, its own `## ` heading test, its own
  `historyRE` match and its own newest-wins loop, and it still does not fold continuation lines.
  Anchors re-anchored to the post-T-080 tree: `MergeLine` `ticket.go:391` → `:385` (it had
  already drifted to `:432` before T-080, which then removed ~47 lines above it); the board's
  DONE `merged` cell `board.go:178` → `:184` (likewise already at `:194` pre-T-080). Both were
  therefore stale *before* this branch as well — the numbers above are the ones that will be
  correct once T-080 merges. Scope and grade unchanged (low-medium/low/S)
