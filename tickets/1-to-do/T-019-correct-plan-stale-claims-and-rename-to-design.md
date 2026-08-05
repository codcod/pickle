---
id: T-019
title: correct PLAN.md's five stale claims and rename it to DESIGN.md
project: pickle
depends-on: []
spawned-by: []
impact: low-medium
complexity: low
cost: S
---

# T-019 — correct PLAN.md's five stale claims and rename it to DESIGN.md

## Description

**Re-scoped 2026-08-04 from "README accuracy polish" to `PLAN.md`.** This ticket had already
become a `PLAN.md` ticket without being renamed: T-047 superseded its items 1 and 3 (README
rewritten to a shop window, command reference moved to `docs/user-manual/`) and commit `f7b0a0a`
mooted item 2, leaving only the `PLAN.md` synopsis. Its own note said *"consider retitling to
reflect the PLAN.md-only scope"*. Reading `PLAN.md` end-to-end on 2026-08-04 found four further
stale claims alongside it, so the ticket is now the whole `PLAN.md` correction plus the rename.
The superseded README items and their line-ref re-anchoring notes are gone from this Description
(they described text that no longer exists); the trail is in `## History`.

`PLAN.md` is the origin design doc — *"kept for the why (rationale and locked decisions)"*, with
an explicit precedence rule (*"Where this doc and a ticket disagree, the ticket wins"*). That
mitigates but does not remove the hazard: §3 presents **"six locked decisions"** as binding
architecture, and one of them is now false. This is the T-041 / T-022 defect class — stale text
that still reads as authoritative — which is why the impact moves `low` → `low-medium`. It stays
below T-022's `medium-high` precisely because of the "ticket wins" rule and the historical
framing, which `AGENTS.md`'s marker block and the skill payload do not have.

### The five stale claims

1. **§3.2 asserts one global id namespace — a "locked decision" that was later unlocked.** It
   reads *"One ticket-id namespace… a single global sequence, `T-NNN`, `max()+1` across all status
   dirs regardless of child-project"*. Shipped behaviour is **per-child counters**:
   `ticket_prefix` in `pickle.toml` (`internal/config/config.go:82`, legal shape at `:32`), and
   `tickets-README.md:111-124` — *"per-child counters"*, *"numbers are only unique within a
   prefix"*, so an id must always be written qualified. **The most serious of the five**: an agent
   designing multi-child work could act on it. Correct it and note that single-global is now the
   degenerate case (children leaving the prefix unset share the legacy `T` counter).
2. **§4 says the board is a copied skeleton.** The payload table's `resources/BOARD.md` row calls
   it *"copied into the project as `tickets/BOARD.md` (the only instance-data copy)"*.
   `internal/install/install.go:544-546` refutes it in a comment: *"a generated artifact (T-044):
   **there is no skeleton to copy**"* — `writeBoard` seeds a fresh render. Overturned by T-044.
3. **§7 says the board stays hand-maintainable.** Its `board sync` bullet: *"the board stays
   hand-maintainable but this rescues it"*. The T-044 decision is the opposite — generated,
   **never** hand-edited, which is the whole reason `NOTES.md` exists for prose. Same overturn as
   item 2; both are the pre-T-044 design still described as current.
4. **§5, §6 and §7 say `install` is interactive — and now contradict §13.4 of the same file.**
   §5: *"`pickle install` prompts (or takes flags)"*; §6: *"gathers these interactively"*; §7:
   *"prompt for name + path"*. No interactive prompt exists anywhere in `internal/install` or
   `internal/cli/install.go` — flags only. §13.4 was rewritten on 2026-08-04 to record exactly
   that, so the file now disagrees with itself. Fixing §13.4 alone sharpened the contradiction
   instead of resolving it; this item closes it.
5. **§7's `ticket new` synopsis is an incomplete flag list** *(the original item 4, from the T-030
   review's sweep, finding N5 — folded in rather than spawned, per the T-036 concern about
   unbounded review spawn)*. `PLAN.md:227` reads
   `pickle ticket new "<title>" --project <name> [--impact … --complexity … --cost …]` — missing
   `--spawned-by` (T-024) and `--family` (T-059), and no input contract (T-030). It does not
   *contradict* behaviour, it is incomplete, which is why it was non-blocking; but it is the only
   `ticket new` synopsis in the tree disagreeing with
   `docs/user-manual/cli-reference.adoc`, `internal/cli/ticket.go` and `pickle help`.

### The rename, and what to prune

Rename `PLAN.md` → **`DESIGN.md`**. The file spends its first fifteen lines explaining it is *not*
a plan; a name needing a preamble to disclaim it is the wrong name, and it calls itself an
*"Origin design doc"*. **Not** into `docs/` — that tree is the published AsciiDoc manual, and this
is internal rationale.

Candidates to prune while there, since they duplicate a live source that will drift again:

- **§11 (migration/dogfooding) and §12 (phased build plan)** — pure delivered history. All five
  phases shipped; the board is the record. §12 is also what the removed progress preamble kept
  re-stating.
- **§4–§7's command and payload descriptions** — largely superseded by
  `docs/user-manual/cli-reference.adoc`. Prune to rationale, or replace with a pointer.

Keep what does not rot: **§1** (goal), **§2** (split judgment from mechanics — the principle the
whole tool rests on), **§3** (multi-project model, once item 1 is corrected), **§10**
(vocabulary), **§13** (resolved questions, rewritten 2026-08-04).

**Hazard for the prune:** `T-057` cites **`PLAN.md:241`** (*"Wire it into CI + a pre-commit
hint"*) as evidence its pre-commit hook was anticipated. That sentence is inside §7. Either keep
it or update T-057's reference in the same commit — do not silently delete a line another open
ticket argues from.

### Reference updates the rename requires

`rg -n "PLAN\.md"` before finishing. Live references to update:

- `tickets/1-to-do/T-019-*.md` — this file (5 occurrences pre-rewrite).
- `tickets/1-to-do/T-057-*.md:71` — `PLAN.md:241`; re-anchor the line number too, since pruning
  will move it.
- `tickets/NOTES.md:152` — the 2026-08-03 recalibration row citing *"the stale `PLAN.md:227`
  synopsis"*.

**Do not touch `tickets/6-done/T-047-*.md`** (3 occurrences) — `6-done/` is a permanent archive
(rules §3: *"tickets are never deleted"*). Its references become historical, which is correct;
rewriting a done ticket to match a later rename would falsify the record. If a breadcrumb is
wanted, a one-line note in `DESIGN.md`'s preamble that it was formerly `PLAN.md` is enough.

1. **The status prose duplicates the table it sits under.** `README.md:87-90` re-lists all eight
   command→ticket mappings that the `[done: T-NNN]` tags at `README.md:75-83` already carry, so
   every future command change needs two edits and they will drift. Replace with a single
   sentence and let the table's tags be the single source of truth. ~~While there: "the full
   command surface is implemented" is true of *commands* only — `install --agent` is still an
   accepted no-op…~~ *(obsoleted 2026-07-26 by T-009: `--agent claude,opencode,pi` is fully
   implemented and documented; the two passages no longer disagree. Only the
   prose-duplicates-table complaint remains.)*

2. **Phased-plan tagging is inconsistent.** In `README.md` (phased-plan section) P5 carries
   `**[done: T-011]**` and T-009 tagged P4, yet P1 (T-002/T-003) and P3 (T-007/T-008) are fully
   delivered and untagged. Tag them. *(Updated 2026-07-26 by the T-009 review: P4 is now done
   and tagged, so P2's outstanding "Zed/Pi" clause argument is weaker — the agent breadth
   shipped in T-009. Re-check whether P2 can simply be tagged now; line refs shifted again by
   T-009's README edits.)*

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

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-24 — created (TO DO). source: pickle ticket new
- 2026-07-25 — re-anchored by the T-018 re-review: items 1/2 line refs shifted by T-018's README edits; noted that `README.md:102-104` belongs to T-018's rework
- 2026-07-25 — scope extended: item 3 (README.md:316 misstates board audit's severity), from the T-024 review's whole-tree sweep (finding N11)
- 2026-07-26 — patched by the T-009 review (impact sweep): item 1's "--agent is a no-op" clause and item 2's "Zed/Pi outstanding" premise are obsolete (T-009 shipped the agent breadth and tagged P4); line refs below the install section shifted again
- 2026-07-26 — scope shrunk by T-047: items 1 and 3 superseded (README rewritten, command reference moved to docs/ with item 3's error fixed), item 2 mooted by commit f7b0a0a (phased-plan section deleted); only item 4 (PLAN.md synopsis) remains
- 2026-08-04 — re-scoped and retitled: README items 1-3 were already dead (T-047, f7b0a0a), so the ticket becomes the whole PLAN.md correction — four further stale claims found reading it end-to-end (one an overturned "locked decision") plus the surviving synopsis item — and the rename to DESIGN.md; impact low -> low-medium; superseded README line-ref notes dropped from the Description
