---
id: T-059
title: family: group tickets under an umbrella ticket id for curated pickup order
project: pickle
depends-on: []
spawned-by: [T-045]
impact: medium-high
complexity: medium
cost: M
---

# T-059 — family: group tickets under an umbrella ticket id for curated pickup order

## Description

Spawned from exploration under **T-045** (2026-07-28), alongside **T-058** (per-child
`ticket_prefix`) — same conversation, two independent schema questions about picking the next
refinement candidate from a large backlog.

**Problem:** the board only orders TO DO/READY by `impact` (`internal/board/board.go:240`); at
scale that collapses into wide ties. Measured on unity's live board (91 tickets, children
`rick` + `snowball`): 28 of 62 TO DO tickets tied at `impact: medium` — selection is genuinely
undetermined by the current schema. Pickle's own backlog (17 tickets) does not yet show this
symptom, but unity's does, today.

**Rejected alternative — reusable `category:` scope key** (like a conventional-commit scope,
e.g. `framework`, `cli`): tested by hand-clustering unity's real 62 TO DO tickets. Produced
~11 categories, mean cluster 4.4 (usable at unity's scale — but pickle's own 17-ticket backlog
clustered to mean ~2, so `category:` should **not** be built speculatively; it only pays off
past a certain backlog size). Three real failure modes surfaced:

- a dominant, still-undifferentiated bucket (`framework` = 12 of 62);
- 4 singleton categories that add vocabulary without grouping anything;
- ~6 genuinely multi-category tickets (e.g. one ticket that is both "verify" and "docs"),
  forcing either an arbitrary single-value pick or a `categories: []` list that breaks
  "group by" as a clean partition.

Deriving categories from title prefixes also does not work on this corpus — unity's titles use
free prose dashes (`Framework-audit P3 —`, `Eval harness:`), not a controlled vocabulary.

**Chosen approach — `family:` as a ticket id (epic-as-ticket-id).** A ticket sets
`family: T-NNN` pointing at an umbrella ticket. The umbrella is an **ordinary ticket** — no new
entity, no second board, no new lifecycle — and the mechanism reuses the existing
`spawned-by`-style lineage validation (`internal/audit/audit.go:71-83` already checks lineage
ids exist without gating pickup; the same pattern applies to `family:`).

**Validated against real evidence, not hypothetical need:** unity's `tickets/NOTES.md` (228
lines) already hand-maintains exactly this concept, unsupported by tooling — 5 named,
goal-bearing, finite families recorded today: "T-122's five follow-ups" (T-180–T-184), "the
2026-07-24 field review family" (T-164/165/166/167/172/113 — NOTES.md's own words: "one root
cause… refine as a family"), "the rename three" (T-186 → T-187 → T-188, with a stated landing
order), "the x- commands" (T-147/148/150/153, already partly expressed via
`depends-on: [T-155]`), and "Is this tree green? family order". NOTES.md names the missing
feature explicitly: *"Hand annotations / curated ordering that survive `pickle board sync`
(this file is the workaround). Still parked — never filed."*

**Category vs. family are different jobs, not competing solutions to the same one:** `category`
answers "show me everything touching docs" (reusable scope, browse/filter); `family` answers
"what do I refine next" (goal-bearing, finite, ordered) — the second matches what unity's humans
actually built by hand. `category:` is deliberately **not** filed here — only worth revisiting
if browsing a large backlog is still painful once `family:` and curated order (below) exist.

**Scope for refinement:** board/serve should be able to group TO DO/READY rows by `family:` for
display. Curated pickup order *within* a family is a related but separate concern, already
scoped in **T-056** (make the serve dashboard writable) — note as a soft coupling, not a hard
`depends-on:`. `family:` does not replace impact ordering; it supplements it once a backlog is
large enough for impact to tie widely.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-28 — created (TO DO). source: pickle ticket new
