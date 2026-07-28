---
id: T-060
title: pickle ticket renumber: retroactively migrate a child to a new ticket_prefix
project: pickle
depends-on: []
spawned-by: [T-058]
impact: low-medium
complexity: high
cost: L
---

# T-060 — pickle ticket renumber: retroactively migrate a child to a new ticket_prefix

## Description

Split out of **T-058** at refinement (2026-07-28). T-058 adds an optional per-child
`ticket_prefix` (default `T`) and a **per-child counter**, but deliberately does **not**
retroactively rename a child's existing tickets — a new workspace sets its prefix before filing
anything, and T-058's audit invariant (a ticket's id prefix must match its project's configured
prefix) simply goes red if you change a populated child's prefix without migrating. This ticket
is that migration.

**Hard dependency on T-058 (pending user sign-off before it goes in `depends-on:`).** Renumbering
is meaningless without the prefix schema T-058 introduces; T-060 must not enter development until
T-058 is done and merged.

`pickle ticket renumber <child> <NEW-PREFIX>` must, atomically:

- rename every `T-NNN-<slug>.md` in that child to `<NEW>-MMM-<slug>.md` (per D1 counters,
  numbers are re-sequenced per prefix, so the map is not a straight prefix swap);
- rewrite the `id:` frontmatter, and **every `depends-on:` / `spawned-by:` reference to a renamed
  id across all tickets in all children** (references cross children — rules §3);
- rewrite `## History` lines that mention the old id;
- regenerate the board;
- emit an **old→new mapping table** (the same artifact the unity/pickle onboarding migration
  already produced) — because two things are unfixable and must be recorded, not edited: commit
  subjects already in git that cite `(T-NNN)`, and branch names already pushed.

**Explicitly must not touch `tickets/NOTES.md` prose** (tooling is forbidden to rewrite the
hand-written planning notes) — the mapping table is what lets a human update those by hand.

Open question for refinement: whether this is a real command or, if the cost proves too high, a
documented manual procedure + the mapping table only. Cost graded `L` on the assumption it is a
real command; complexity `high` for the cross-child reference rewrite and the atomic-or-refuse
guarantee.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-28 — created (TO DO). source: pickle ticket new
