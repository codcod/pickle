---
id: T-120
title: skill-payload summary labels describe contents, not work
project: pickle
depends-on: []
spawned-by: [T-013]
impact: low-medium
complexity: low
cost: S
---

# T-120 — skill-payload summary labels describe contents, not work

## Outcome

After this ships, the one-line summary `pickle install` and `pickle upgrade` print for the skill
directory describes what the command actually did to the tree. Today it describes only whether the
payload *file contents* matched, so it can report `= (current)` for a run that deleted a
directory, and the same `(current)` over a tree it knowingly left stale.

## Description

T-013 replaced a flat always-`+` summary with a three-way `created` / `refreshed` / `current`
label, decided by `skillPayloadDiffers` comparing the embedded payload against what is on disk.
That was a large improvement and is not in question here. But the comparison answers *"do the
payload file contents match?"*, while the label is read as *"what did this command do?"* — and
those two questions come apart in several reachable cases.

Three findings from T-013's own review rounds, all recorded there with evidence and all left
`noted` individually. They are collected here because they are one theme, and because the third
one makes the pattern hard to keep calling cosmetic:

1. **`= (current)` is printed for runs that wrote.** Both `copyPayload` and `Upgrade` write the
   payload unconditionally; the comparison chooses the wording, never whether the work happens.
   Every other `=` in the CLI means "not written", so this one reads differently from its
   neighbours. Mtimes change on a run reported as current. *(T-013 review, N4.)*

2. **`install` can print `(refreshed)` while leaving the tree stale.** `copyPayload` detects extra
   files that the payload does not contain, but never prunes them — only `upgrade` wipes. So the
   label claims an effect that path does not have. *(T-013 review, N5.)*

3. **A stale *directory* is invisible to the comparison, and the two commands then disagree.**
   The stale-entry walk skips directories, so an unseen directory leaves `changed = false`.
   Reproduced on the T-013 branch:
   - `upgrade` with a readable stale directory → prints `= … (current)`, **and removes it**. The
     label understates a run that changed the tree.
   - `install` re-run with the same directory → prints `= … (current)`, **and it survives**. The
     label overstates the cleanliness of a tree left stale.
   - Sharpest form: an **unreadable** stale directory reports `(refreshed)` (it trips the
     advisory walk's degrade-to-changed path added for T-013's B4), while a **readable** one
     reports `(current)`. Same tampering, opposite labels, decided by a permission bit.
   *(T-013 scoped re-review round 2, N13.)*

**Scope.** Decide what the label is *for* — contents-matched or work-done — and make it mean that
consistently across both commands, then align the vocabulary. Marking an unseen directory as a
change fixes (3) mechanically, but the ticket should settle (1) and (2) deliberately rather than
patching each symptom: possible answers include pruning in `copyPayload` so `install` and
`upgrade` converge, or reporting the two axes separately.

**Soft coupling, not a dependency.** T-013 is the ticket that introduced this vocabulary; nothing
here is blocked on it, and the code involved is small and self-contained. Note also that the
label vocabulary is documented nowhere in `docs/` (T-013 review, N11) — whatever this ticket
settles is worth a line in the manual, which would close N11 at the same time.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-24 — created (TO DO). source: review: T-013's review rounds. Batches three findings on
  one theme — N4 and N5 (round 1, both `noted`) promoted by this row per rules §5, plus N13
  (scoped re-review round 2), whose readable-vs-unreadable label asymmetry is what tipped the
  theme past the promotion test
