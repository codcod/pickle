---
id: T-105
title: pickle project decisions: query a child's confirmed design decisions from a specified, audit-enforced format
project: pickle
depends-on: []
spawned-by: []
impact: low
complexity: medium
cost: M
---

# T-105 — pickle project decisions: query a child's confirmed design decisions from a specified, audit-enforced format

## Outcome

Running `pickle project decisions <child>` prints every confirmed design decision recorded
against that child's tickets, already in citable `T-NNN decision N` form, filterable by status
and topic. Today the same answer needs a hand-written `awk` that has to re-solve two parsing
traps every time; after this, decisions are a queryable projection of the ticket corpus rather
than prose only a bespoke script can reach.

## Description

A child's design decisions accumulate in each ticket's `### Confirmed design decisions` section
and are currently reachable only by ad-hoc `grep`/`awk`. Measured on this repo at filing time:
**397 decisions across 104 tickets** (382 of them in `6-done/`). The convention works well enough
that decisions are cited across tickets in running prose — `T-057 decision 3` (the fail-open
contract) is cited 7 times, and one grep recovers a three-hop inheritance chain
`T-057 d3 → T-082 d2 → T-100 d10`. What is missing is a supported way to *ask*.

**The ask is determinism, not just convenience.** The ad-hoc scripts written while exploring this
are fragile in two ways that a shipped command must not be:

1. **`project:` must be frontmatter-scoped.** A plain `grep '^project:'` also matches ticket body
   text — this repo has a `project: nope | medium | medium | M |` inside a fixture table — so the
   child filter silently mis-attributes a decision.
2. **The decisions section must be bounded.** An extractor that stops only at `### ` bleeds
   numbered items in from `## Tasks` or `## History` and reports them as decisions.

**The proposed route to determinism is to specify the format that already exists, not to invent
one.** Measured: **367 of 397 decisions (92%) already match `N. **<statement>.** <rationale>`** —
a numbered item whose first bold run is the decision statement and whose remainder is the
rationale. So the work is to (a) write that grammar down in the ticket template, (b) have the
audit enforce it so the corpus stays parseable, and (c) let the command read only what the audit
guarantees. That is materially different from bolting a tolerant parser onto free prose, which is
the defect shape T-097 fixed (a permissive scan reporting confidently and wrongly).

**Build on T-065, which shipped while this was being explored.** `board state --json` already
emits a versioned projection (`schema: 1`) with a per-ticket frontmatter walk, including `project`
and `status`. Decisions should be projected as a schema bump on that existing structure rather
than parsed a second time — that inherits the correct frontmatter scoping for free and keeps one
reader over the ticket corpus. The human-readable command is then a surface over the projection.

**Open questions for refinement** (not pinned at filing):

- **Namespace.** Filed as `pickle project decisions` per the request. Worth challenging at
  refinement: today `project` is registry operations (`add`/`list`/`remove`), and decisions are
  per-ticket data aggregated by child, so `pickle ticket decisions --project <name>` is arguably
  the truer noun. Decide once, deliberately.
- **Audit severity and the 30 non-conforming decisions.** Backfilling is refused by precedent
  (T-025, archaeology with no consumer), so enforcement should almost certainly be prospective
  and a *warning*, with non-conforming items still listed but marked unstructured. Refinement must
  pin whether the audit gains a row at all.
- **Payload impact.** Specifying the grammar edits `TEMPLATE.md`, which ships to projects that are
  not pickle — the wording must pass the foreign-workspace test in `AGENTS.md`.

**Scope fence — deliberately out of v1.** Distinguishing *locked* decisions from ticket-local ones
is excluded: it is a judgement, not a property of the text, and the only mechanical proxy
(citation counting) is fragile — an ad-hoc citation scan missed 6 of 29 real citations on
possessives, bold and backticks. The rationale and the pre-registered trigger for revisiting are
in `NOTES.md` § *"ADR exploration (2026-08-15) — explored, nothing filed; the convention already
works"*. Citation validation (auditing that a cited `T-NNN decision N` resolves) is likewise out.

**Soft couplings, no hard `depends-on`:** T-065 (shipped; this extends its projection and schema),
T-066 (a new command needs a `cli-reference.adoc` entry — that ticket is already about closing
CLI-surface documentation gaps), T-102 (frontmatter writer, adjacent ticket-parsing surface).

**Grading rationale.** `impact: low` deliberately, consistent with T-065's own re-grade: the need
is served today by a one-off script, the measured cross-ticket citation rate is 7.6%, and the
2026-08-04 precedent refuses to credit prospective demand. Refinement should challenge this grade
rather than inherit it.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-08-16 — created (TO DO). source: chat: requested directly after an exploration that had
  recommended against a decisions command; the human overrode that recommendation, and two of the
  three objections lapsed in the meantime — T-065 shipped (so the command no longer pre-empts it)
  and the format was measured 92% regular (so the parser need not be tolerant). Exploration record
  and the objections that still stand: `NOTES.md` § "ADR exploration (2026-08-15)"
