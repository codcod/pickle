---
id: T-085
title: make the per-ticket record aggregable: finding class, plan-amendment line, cost actual, provenance class
project: pickle
depends-on: []
spawned-by: []
impact: medium
complexity: low
cost: S-M
---

# T-085 — make the per-ticket record aggregable: finding class, plan-amendment line, cost actual, provenance class

## Description

**Outcome.** After this ships, the question *"what does brine keep getting wrong?"* is answered
by counting, not by reading 36 tickets. Each review records **which kind** of defect each finding
was, each implementation records **whether the plan it executed was wrong**, and each ticket
records **what its estimate cost in reality** — so a recurring class can be promoted to a
`board audit` lint instead of being re-discovered by every reviewer.

*(Outcome-first per T-083's finding, which this ticket would otherwise violate.)*

### The problem, measured

brine captures more per-ticket evidence than most flows: dated History transitions, a findings
table with severity + disposition + evidence, and a merge line. **The corpus is large and it is
unaggregable.** Counted across `6-done/` (36 tickets, all 36 carrying a `## Review`):

| available signal | state today |
|---|---|
| **≈165 dispositioned findings** | classifiable by *severity* and *disposition*, and by nothing else — there is no field saying what **kind** of defect it was |
| rework rate | **9 of 36** done tickets took a blocking finding; never aggregated |
| review yield | a `Disposition summary` line exists in only **23 of 36** — the one pre-aggregated number is present 64% of the time |
| plan-defect rate | the `plan amended inline` History line exists in **exactly one ticket** (T-049, which invented it unilaterally) |
| spawn rate | measured **once**, by hand, R=0.625 over 8 reviews — and it killed T-045 on a pre-registered threshold |
| grade rationale | the two `impact` recalibration passes wrote **11 justifications into `NOTES.md` and zero into tickets** (T-083's measurement) |

The consequence is visible in this repo's own record: **T-045, T-063 and T-064 each proposed new
machinery while the data that would have settled them sat unread** (`NOTES.md:95-101`). The
failure was never capture. It was that nothing could be counted without an archaeology pass, so
counting only happened when someone already suspected the answer.

### The four items (one theme: the per-ticket record becomes countable)

Batched deliberately — §5's *"batching is mandatory, not encouraged"*. All four are the same
change in four places: **a field that already exists in prose becomes a field that can be
grouped by.**

1. **`class` column on the findings table.** One word per row from a closed vocabulary
   (candidates, to be pinned at refinement: `stale-xref`, `docs-gap`, `test-gap`, `plan-wrong`,
   `naming`, `dead-code`, `other`). This is the item that carries the ticket: ≈165 findings is
   the largest structured dataset in the repo, and it is one column away from being diagnostic
   rather than merely countable. A class that recurs is a **lint specification** — the T-040 /
   T-067 conversion, where review labour becomes an audit check.
2. **`plan amended inline` promoted from convention to rule.** It is the only direct measurement
   of READY-gate quality. T-049 used it twice, and *both* amendments retracted a **confirmed
   design decision** — decisions the READY gate had certified. If that rate is material, rules §4
   is mis-tuned; today n=1 because only one implementer bothered to write the line.
3. **`cost` actual vs estimate**, recorded as one row in the review's implementation-audit table.
   `impact` gets re-graded in periodic passes; `cost` is set once at refinement and never
   confronted with reality, which makes it unfalsifiable — the exact defect the recalibration
   passes exist to correct for `impact`.
4. **Provenance class on the `created … source:` line.** Field-use vs self-host vs review vs
   audit vs chat. This is the **weight on every other signal**: the two field-finding triages
   (2026-07-27) produced more real defects than any self-host pass, and an n=36 single-repo
   corpus is exactly what one would overfit to. Partially present in prose today, unstructured.

### Pre-registered criterion (the point of the ticket)

Written in at filing, per the T-045 precedent — *"the first decision in the project made by a
pre-registered criterion … worth copying"* (`NOTES.md:112-115`):

> **After 8 further reviews carrying the `class` column:** if any single class accounts for
> **≥25%** of non-blocking findings, that class is promoted to a `board audit` check or a payload
> rule. If the distribution is flat — **no class above 15%** — the column is removed and this
> direction is dropped.

The failure mode is therefore "delete one column", which is what makes the bet cheap enough to
take at all.

### Deliberately out of scope

- **Backfilling the existing 165 findings.** T-025 was dropped as *"lineage archaeology with no
  consumer"*; retro-classifying a corpus nobody has queried yet is the identical mistake.
  **Prospective only.**
- **A metrics command, a retro command, or a dashboard.** T-045's measurement cost one `for`
  loop. The substrate is **T-065** (below), after which every number above is a `jq` one-liner.
- **Anything touching ordering, ranking, scoring or gating.** T-045, T-063 and T-064 are all in
  `7-dropped/` for proposing exactly that; this ticket changes what a ticket *records*, never
  how the board sorts or what it admits.
- **A second prioritisation vocabulary.** The rejection recorded at `NOTES.md:729-741` stands.

### Couplings

Soft couplings (no `depends-on:`, no ordering enforced):

- **T-065** (JSON read projection) — **the substrate.** Its projection must expose parsed
  findings and History for any of this to be queryable without re-parsing markdown; an item to
  that effect is being added to T-065. Neither blocks the other: this ticket's fields are
  useful to a human reader with `grep` on day one, and T-065 is useful without them.
- **T-083** (Outcome section) — closest neighbour and **not** a duplicate. T-083 structures the
  Description's *opening*, for a reader deciding whether to refine; this structures the *Review
  and History*, for a reader counting across tickets. Both add a per-ticket field and both
  contemplate a `board audit` warning for its absence, so they should be sequenced, not merged.
  T-083's "11 justifications in `NOTES.md`, 0 in tickets" measurement is item 4's evidence too.
- **T-081** (gate table as data) — its unit is *"a `##` section (and its non-emptiness)"*, which
  is what a presence check for the `class` column or the `Disposition summary` line would need.
  Recorded as a coupling only: the 2026-08-04 precedent refuses to credit **prospective** demand
  when grading, and T-083 already declined this same bump on that basis. **The rule cuts both
  ways — do not re-grade T-081 for this.**
- **T-036** (the four dispositions) — this adds an orthogonal axis to the table T-036 ratified;
  it must not disturb the disposition vocabulary or its default.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-07 — created (TO DO). source: chat — "how can pickle learn from itself" exploration;
  scoped down from a five-part proposal to the four capture items after the retrieval half was
  split out and the mechanism half was noted-not-filed (`NOTES.md`, 2026-08-07)
