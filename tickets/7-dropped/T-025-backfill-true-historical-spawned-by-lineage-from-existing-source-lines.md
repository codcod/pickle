---
id: T-025
title: backfill true historical spawned-by lineage from existing source: lines
project: pickle
depends-on: [T-024]
spawned-by: []
impact: low
complexity: low
cost: S
---

# T-025 — backfill true historical spawned-by lineage from existing source: lines

## Description

Once T-024 ships the `spawned-by:` frontmatter field, every existing ticket will carry the
safe default `spawned-by: []` (T-024 deliberately performs only a **uniform** backfill — no
lineage archaeology). This ticket does the follow-up: **populate the true historical parent(s)**
for the tickets whose provenance is already recorded, replacing `[]` with the real id(s).

The authoritative source is each ticket's own first `## History` line
(`created (TO DO). source: …`). Where that line names a parent review/ticket, set `spawned-by`
to it. Known mappings from the current board (verify at implementation time — the set may grow
before this is picked up):

| ticket | evidence (`source:` line, verbatim intent) | `spawned-by` |
|---|---|---|
| T-012 | "T-001 review (non-blocking findings N1–N3)" | `[T-001]` |
| T-013 | "T-004 review (non-blocking findings)" | `[T-004]` |
| T-014 | "T-007 review (non-blocking findings)" | `[T-007]` |
| T-015 | "T-008 review (non-blocking findings …)" | `[T-008]` |
| T-021 | Description: "Non-blocking finding 7 from the T-018 review" | `[T-018]` |
| T-022 | Description: "Non-blocking finding 8 from the T-018 review" | `[T-018]` |
| T-023 | Description: "Non-blocking finding 9 from the T-018 review" | `[T-018]` |

Note T-021/T-022/T-023's parent is stated in the **Description** (their `source:` line is the
generic "pickle ticket new"); refinement must decide whether to trust Description prose or
restrict the backfill to tickets whose `source:` **line itself** names the parent (T-012–T-015).
Everything not evidenced stays `[]`.

**Post-T-024 update (2026-07-25).** T-024 has shipped and the set has already grown — the table
above is a floor, not a ceiling. Tickets born *since* the field existed already carry real
values and need **no** backfill: `T-029` and `T-030` were filed with
`--spawned-by "T-024"` by T-024's own review, and `T-031`/`T-032` with `--spawned-by "T-029"` by
T-029's (2026-07-25) — the born-with-lineage set keeps growing as reviews spawn follow-ups, which is
the mechanism this ticket is compensating for the absence of. Tickets created in the gap — filed before the
field existed but after this ticket was written — still need entries, and their evidence is
their `source:` line: `T-027` ("refinement of T-024"), `T-028` ("applicability audit of T-024"),
and **this ticket itself** ("T-024 refinement") — all `[T-024]`. Re-derive the full set at
implementation time rather than trusting either list.

This is pure data (ticket-file frontmatter) — **no code changes**. Value is modest: it makes
`spawned-by` queries accurate for the pre-feature archive and demonstrates the field with real
non-empty values. Purely cosmetic on the DONE/DROPPED archive, more useful on the live backlog.

### Couplings

**Spawned by T-024** (this ticket exists only because T-024 adds the field). Hard
`depends-on: [T-024]` — there is nothing to populate until the field and its audit validation
exist; entering development before T-024 is merged would fail `pickle board audit` (dangling /
unknown-key). Once T-024 ships, this ticket's own `spawned-by:` should be set to `[T-024]`.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-25 — created (TO DO). source: T-024 refinement (deferred real-lineage backfill)
- 2026-07-25 — impact sweep after T-024 was reviewed: candidate set extended (T-025/T-027/T-028
  need `[T-024]`; T-029/T-030 already carry it from creation)
- 2026-07-25 — impact sweep after T-029 was reviewed: no new backfill candidates (T-031/T-032 were
  born with `spawned-by: [T-029]`); note added that the born-with-lineage set keeps growing
- 2026-07-26 — TO DO → DROPPED: lineage archaeology with no consumer; source: History lines already carry provenance
