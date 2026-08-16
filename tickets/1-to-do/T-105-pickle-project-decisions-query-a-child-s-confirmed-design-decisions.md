---
id: T-105
title: "pickle project decisions: query a child's confirmed design decisions"
project: pickle
depends-on: []
spawned-by: []
impact: low
complexity: low-medium
cost: S
---

# T-105 — pickle project decisions: query a child's confirmed design decisions

## Outcome

Running `pickle project decisions <child>` prints every confirmed design decision recorded against
that child's tickets, already in citable `<ID> decision <N>` form, filterable by status and topic.
Today the same answer needs a hand-written `awk` that re-solves two parsing traps every time and
that gets the child filter wrong on any workspace whose ticket prefix is not `T-`.

## Description

A child's design decisions accumulate in each ticket's `### Confirmed design decisions` subsection
and are currently reachable only by ad-hoc `grep`/`awk`. This ticket adds a supported read-only
query. It is a **human-facing tool**, invoked ad hoc like `pickle doctor` — no procedure in the
brine flow calls it, and none is changed to call it (see *Scope fence*).

**All figures below are measurements taken at filing (2026-08-16) and will drift.** They are
recorded as the evidence the ticket was filed on, not as maintained facts; re-measure with the
command itself once it exists. Measured across two independent workspaces:

| corpus | tickets | decisions | `N. **statement.** rationale` form |
| --- | --- | --- | --- |
| this repo | 105 | 397 | 367 (92%) |
| an unrelated 3-child brine workspace | 95 | 138 | 120 (86%) |

The second row is the load-bearing one: the statement/rationale shape is an **emergent convention
that already holds in a workspace which has never seen this ticket**, so the command can key on it
without first imposing it. Raising the remaining ~10% is deliberately a separate concern (T-106).

### Determinism — reuse the existing readers, do not write a third one

The ad-hoc scripts are fragile in three ways a shipped command must not be. All three are already
solved by tested code in this repo:

1. **`project:` must be frontmatter-scoped.** A plain `grep '^project:'` also matches ticket body
   text — this repo has a `project: nope | medium | medium | M |` inside a fixture table. Use
   `ticket.LoadAll` + `ticket.ParseFrontmatter`.
2. **The decisions subsection must be bounded**, or numbered items bleed in from `## Tasks` and
   `## History`. `ticket.SectionBody` bounds `## ` sections and `ticket.SubsectionMissing` already
   scans `### ` subsections — but returns only a bool, so the one genuinely new primitive here is
   `SubsectionBody(text, section, stem) (string, bool)`, factored out of the scan
   `SubsectionMissing` already performs. That factoring is aligned with T-042, not in tension with it.
3. **Ticket ids must never be pattern-matched as `T-NNN`.** Children set their own `ticket_prefix`;
   the reference workspace above uses `RICK-` and `SNOW-` alongside a `T-` child. Ids come from the
   loaded ticket.

**What to project, following the rule `internal/state/review.go` already established** (T-065): it
projects only closed-vocabulary fields and never the multi-sentence prose columns. Applied here —
project the **ticket id**, the **decision number**, and the **statement** (the leading bold run,
emphasis-stripped per T-065 decision 11); do **not** project the rationale prose that follows. A
numbered item with no leading bold run is emitted with the structured flag false and its raw first
line — **never inferred**. That is what makes the command deterministic without requiring the
corpus to change first.

### Surface

- `pickle project decisions <child> [--status <dir>] [--grep <regex>] [--json]`, read-only.
- **Exit semantics matter here** and are the subject of the acceptance test: an **unregistered**
  child is an error; a **registered child with no matching decisions is exit 0 with an empty
  result**. The prototype conflated these by leaking `grep`'s exit 1, and all three empty cases in
  the acceptance test below are legitimate answers, not failures. This repo already treats exit
  codes as contract (T-057/T-082 fail-open, T-071 exit-1 semantics).
- Human-readable output must be pasteable as a citation; `--json` is the machine form.

### Open questions for refinement (not pinned at filing)

- **Namespace.** Filed as `pickle project decisions` per the request. Worth one deliberate
  challenge at refinement: `project` is registry operations today (`add`/`list`/`remove`), so
  `pickle ticket decisions --project <name>` is arguably the truer noun.
- **Whether `--json` here is standalone or a `board state --json` schema bump.** Deferred on
  purpose: T-065 shipped `schema: 1`, and folding decisions in is a cheap later decision that
  should not gate this command.

### Scope fence — deliberately out

- **Changing `TEMPLATE.md` and adding an audit row** → T-106. This command must work on the corpus
  as it stands, which is why it reports non-conforming items rather than requiring conformance.
- **Locked-vs-ticket-local classification.** It is a judgement, not a property of the text, and the
  only mechanical proxy (citation counting) is fragile — a citation scan missed 6 of 29 real
  citations here on possessives, bold and backticks, and the reference workspace also uses a comma
  variant (`RICK-156, decision 1`). Rationale and the pre-registered trigger for revisiting are in
  `NOTES.md` § *"ADR exploration (2026-08-15) — explored, nothing filed; the convention already
  works"*.
- **Reading citations at all.** The command emits citable output; it does not parse citations.
- **Wiring the command into any brine procedure.** It has no procedural consumer by design.

### Acceptance test — six cases, two workspaces

A multi-child workspace is required to test the child filter at all; this repo has one child and
cannot exercise it. The six cases below were run against the prototype and are the expected
outcomes:

1. topic filter recovers a contract chain — 2 rows, each citable
2. topic + `--status 6-done` narrows to shipped work — 1 row
3. `--status 1-to-do` — **0 rows, exit 0** (decisions are written at refinement, so TO DO has none)
4. multi-child workspace, one child + topic — matches only that child's tickets
5. child whose done tickets recorded no decisions — **0 rows, exit 0**
6. registered child with no tickets at all — **0 rows, exit 0**; unregistered name — **error**

### Soft couplings (no hard `depends-on`)

T-065 (shipped; its `internal/state` prose-parsing precedent is the pattern to follow), T-106 (the
format spec this ticket deliberately does not do), T-066 (a new command needs a
`cli-reference.adoc` entry), T-042 (the `SubsectionBody` factoring).

### Grading rationale

`impact: low` deliberately, consistent with T-065's own re-grade: the need is served today by a
one-off script, the measured cross-ticket citation rate is 7.6%, and the 2026-08-04 precedent
refuses to credit prospective demand. `cost: S` because the parsing infrastructure already exists —
the new code is one small helper, a numbered-item scan, a CLI surface, tests and docs. Refinement
should challenge both rather than inherit them.

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
- 2026-08-16 — scope narrowed in TO DO after an adversarial pass, and retitled ("from a specified,
  audit-enforced format" dropped). The pass found one blocking defect: the ticket bundled four
  separable changes, and its only cross-ticket consumer (`SKILL.md` applicability gate) needs the
  *locked* decisions this ticket explicitly excludes — so as filed it would have shipped a command
  no procedure calls **and** a format burden it did not need. Split: the `TEMPLATE.md` grammar and
  the audit row moved to T-106; the `board state --json` schema bump deferred; the command now
  reads the corpus as it stands. Re-graded `complexity: medium → low-medium`, `cost: M → S` on the
  finding that `ticket.LoadAll`/`ParseFrontmatter`/`SectionBody` already exist. Also corrected: the
  filing figures are now marked as at-filing measurements, per the warning `internal/state/review.go`
  records against hard-coding counts that go stale
