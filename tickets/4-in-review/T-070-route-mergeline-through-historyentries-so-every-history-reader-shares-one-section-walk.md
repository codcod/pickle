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

### 0. Feature branch (mandatory)

```
cd .
git checkout main
git checkout -b feat/T-070-mergeline-historyentries
```

Root-path child. Tidy WIP commits into atomic ones before presenting (Finish, below).

### Prerequisite gate (hard)

None (`depends-on: []`). `spawned-by: [T-043]` is lineage only — T-043, T-080 and T-089 are all
in `6-done/`, so the tree this plan describes (line numbers, signatures) is the current one, not
a moving target.

### Confirmed design decisions (do not deviate without asking)

1. **`MergeLine` (item 1) becomes a thin fold over `HistoryEntries`**, exactly like
   `LastHistoryStatus`/`LastHistoryReason` already are: iterate `HistoryEntries(def, text)`,
   keep the newest entry with `Kind == HistoryMerged`, return its (already-folded) `.Text`. Its
   private `inHistory` flag, `## ` heading test, and `historyRE` match are deleted outright —
   there is nothing left for them to do. `HasMergeLine` is unchanged (already defined in terms of
   `MergeLine`).
2. **Item 2's fix is *not* "re-scan the folded text for the best arrow"** — that is the
   mechanism that produces the bug (a later, unrelated arrow in a continuation line can outrank
   the real one). Instead: `HistoryEntry` gains a `Reason` field, resolved **in the same call**
   that resolves `Target`, from the same accepting arrow in the entry's first physical line
   (`body`) — never re-derived from folded `Text`. Concretely, `transitionParts` gains a fourth
   return, `hasReason bool` (true iff a colon followed the accepting arrow on that first line;
   false for a bare `"OLD → NEW"` with no colon at all, as distinct from a colon present but an
   empty clause after it). `HistoryEntries` then:
   - sets `e.Target, e.Reason, hasReason, _ = transitionParts(def, body)` (four return values) at
     entry creation, and remembers `hasReason` for that entry (an unexported bookkeeping field on
     `HistoryEntry`, e.g. `reasonOpen bool` — it answers "does a continuation line extend this
     entry's reason clause", which nothing outside this package needs to know);
   - when folding a continuation line onto an entry whose `Kind == HistoryTransition` **and**
     `reasonOpen`, appends the continuation to `Reason` the same way it already appends to `Text`
     (`+= " " + trimmed`); when `!reasonOpen`, the continuation still folds into `Text` (unchanged
     behaviour) but **never** into `Reason` — a bare transition with a hand-written continuation
     note stays reason-less, which is what keeps
     `TestTransitionSurvivesContinuationFolding`'s "reason-less transition with a continuation
     line" case (`wantReason: ""`) passing unchanged, and is exactly what stops the R7 bug: a
     continuation's own arrow+colon can never be mistaken for this entry's reason, because
     reason-folding never re-parses the continuation for arrows at all — it only ever appends
     raw text onto an already-open clause.
   - `LastHistoryReason` simplifies to reading `e.Reason` directly off the newest `HistoryTransition`
     entry — it stops calling `transitionParts` altogether, removing the second, independent
     scan that could disagree with `Target`'s.
3. **Verified by hand against every existing pinned test in `internal/ticket/ticket_test.go`**
   before writing this plan (`TestLastHistoryReasonFoldsContinuations`,
   `TestLastHistoryStatusArrowInReason`, `TestTransitionSurvivesContinuationFolding`'s four
   cases) — decision 2 reproduces every one of their expected outputs unchanged. None of their
   assertions are to be edited; if implementation reveals a mismatch, the design (this plan), not
   the pinned test, is what is wrong, and the human is told before changing an existing
   expectation.
4. **No change to `historyKind`, `HistoryEntry.Kind`'s freeze-on-first-physical-line rule, or
   `historyRE`** (Description, "Not in scope") — only `transitionParts`'s signature (add
   `hasReason`), `HistoryEntry` (add `Reason` + the unexported `reasonOpen`), `HistoryEntries`'s
   fold step, `LastHistoryReason`, and `MergeLine` change.

### Tasks

#### Task 1 — `transitionParts` reports whether a reason clause was opened
In `internal/ticket/ticket.go` (`transitionParts`, currently returning `(target, reason string,
ok bool)`), add a fourth return `hasReason bool`, true exactly when the accepting arrow's
candidate was found via the colon branch (`j >= 0`), false via the no-colon branch. Update its
doc comment to state the new contract. It has exactly the callers listed in decision 4's scope;
update each at their own call sites in the tasks below.

#### Task 2 — `HistoryEntry` gains `Reason`; `HistoryEntries` resolves it in the same pass as `Target`
In `internal/ticket/ticket.go`:
- Add `Reason string` (exported, doc comment mirroring `Target`'s) and an unexported
  `reasonOpen bool` to `HistoryEntry` (`:176-188`).
- In `HistoryEntries` (`:204-234`), at entry creation: when `e.Kind == HistoryTransition`, call
  `target, reason, hasReason, _ := transitionParts(def, body)` and set `e.Target = target`,
  `e.Reason = reason`, `e.reasonOpen = hasReason`.
- In the continuation-folding branch, after `last.Text += " " + trimmed`, add: if
  `last.Kind == HistoryTransition && last.reasonOpen`, also `last.Reason += " " + trimmed`.

#### Task 3 — `LastHistoryReason` reads `Reason` directly; drop its own `transitionParts` call
In `internal/ticket/ticket.go` (`LastHistoryReason`, `:367-379`), replace the loop body with:
skip non-`HistoryTransition` entries; otherwise `reason = e.Reason`. Update the doc comment: it no
longer re-derives from folded `Text`, so the two paragraphs explaining that re-derivation are
replaced with a short note pointing at `HistoryEntries`/`transitionParts` as the single source now.

#### Task 4 — `MergeLine` becomes a fold over `HistoryEntries`
In `internal/ticket/ticket.go` (`MergeLine`, `:385-402`), replace the whole body with the loop
from decision 1. Delete the now-unused local `inHistory` variable and its own `historyRE` call
site (the package-level `historyRE` itself stays — `HistoryEntries` still needs it).

#### Task 5 — tests
In `internal/ticket/ticket_test.go`:
- `TestMergeLineFoldsContinuations` — the Description's own literal repro
  (`MergeLine("## History\n- 2026-08-06 — merged to main\n  (abc1234) after review\n")`), asserting
  the full unwrapped string, plus a second case using T-089's recommended form (a full commit URL
  wrapped across two lines) as the realistic fixture the re-grade cites.
- `TestLastHistoryReasonAnchoredAtSameArrowAsTarget` (the R7 regression) — the Description's exact
  doc (`"TO DO → READY\n  and later IN REVIEW → DONE: some clause"`), asserting
  `LastHistoryStatus == "READY"` **and** `LastHistoryReason == ""` (not `"some clause"`).
- Run the full existing `internal/ticket` suite and confirm every pinned test from decision 3
  still passes with **zero literal changes**.

#### Task 6 — real-tree guard (Description's own suggested check)
Add a wrapped merge line in T-089's recommended form to one convenient `6-done/` ticket's
`## History` in a **scratch/local-only check**, not a committed change: run `pickle board audit`
and `pickle board sync` before and after, confirm the `merged` cell now shows the full unwrapped
line instead of a truncated one, then revert the scratch edit before committing (mirrors T-043's
own D8 use of this same guard).

### Acceptance test

```
just build
go test ./internal/ticket/... -v -run 'TestMergeLine|TestLastHistoryReason|TestLastHistoryStatus|TestTransitionSurvivesContinuationFolding|TestHistoryEntries'
just test
just lint
just docs-check
./pickle board audit
```
All clean; the targeted `-run` output confirms both new tests pass and every pre-existing pinned
test's expectations are unchanged (decision 3). Task 6's manual real-tree check done and reverted.

### Docs update (mandatory when user-facing)

No user-facing surface — internal parsing fix only; no CLI/output shape changes beyond correctness
(a previously-truncated `merged` cell now renders in full, which is the bug fix itself, not a new
feature to document).

### Finish (mandatory)

1. Acceptance test green.
2. No docs to update.
3. Write a summary: confirm no pinned test literal needed changing (decision 3's bet paid off),
   and that `MergeLine`'s private section walk is gone entirely (item 1) alongside the R7 fix
   (item 2).
4. Suggested commit message:
   ```
   fix(ticket): resolve History target and reason from one arrow, fold MergeLine (T-070)
   ```
5. Tidy WIP commits into atomic ones (root-path child) before presenting.
6. Commit locally; do not push or open an MR without explicit user approval. Hand back with
   `pickle ticket move T-070 in-review --reason "acceptance green"`.

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
- 2026-08-20 — refined: re-anchored line numbers against the current tree (`MergeLine`
  `ticket.go:385`, `board.go:185`, `serve/view.go:216` — both call sites had drifted again).
  Designed and hand-verified the item-2 fix (`transitionParts` gains a `hasReason` return;
  `HistoryEntry` gains `Reason` + an unexported `reasonOpen`) against every pinned test in
  `ticket_test.go`, including the exact case (`TestTransitionSurvivesContinuationFolding`'s
  "reason-less transition with a continuation line") that a naive "fold reason like Text" fix
  would have silently broken — confirmed the chosen design reproduces all of them with zero
  literal changes. Grade unchanged. TO DO → READY: implementation plan complete.
- 2026-08-22 — TO DO → READY: plan complete
- 2026-08-22 — READY → IN DEVELOPMENT: picked up
- 2026-08-22 — plan amended inline: Task 6 assumed the real-tree guard would be a scratch,
  reverted-before-commit edit because no ticket in the tree was expected to already carry a
  wrapped merge line. Hand-verified against the actual tree (old binary vs. fixed binary, same
  ticket files) that this was wrong: several real `6-done/` tickets (e.g. T-108) already have a
  merge line wrapped across a continuation line, and the pre-fix `MergeLine` was silently
  truncating their board `merged` cell today — the defect T-089 flagged as reachable was already
  live, not merely likely. `pickle board sync` with the fixed binary regenerates ~16 DONE rows to
  their full, correct text; `board audit` on the feature branch reports `tickets/BOARD.md` as
  stale until that sync is committed. Per this repo's own commit-landing rule, that regeneration
  is board bookkeeping and cannot land on `feat/T-070-mergeline-historyentries` (the pre-commit
  hook refuses it) — it is committed here, on the base branch, once this ticket's code has merged
  to `main`, as an ordinary `pickle board sync` bookkeeping commit. Code, tests, build, lint and
  docs-check are green on the feature branch now; only that post-merge sync remains
- 2026-08-22 — IN DEVELOPMENT → IN REVIEW: acceptance green (build/test/lint/docs-check clean; board sync deferred to post-merge, see amendment)
