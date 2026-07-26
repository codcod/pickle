---
id: T-028
title: guard TEMPLATE.md frontmatter against audit requiredKeys
project: pickle
depends-on: []
spawned-by: []
impact: low
complexity: low
cost: S
---

# T-028 — guard TEMPLATE.md frontmatter against audit requiredKeys

## Description

> **ABSORBED into T-040 (board triage, 2026-07-26) — this ticket is closed, its work is not.**
> Everything below stands as the record: the analysis, measurements and line references
> are still the authoritative detail for this part of T-040's scope. Do not re-file it;
> do not implement from here. T-040 is the refinable, reviewable unit.

Two artefacts must agree on the set of frontmatter keys a ticket carries: `audit.requiredKeys`
(`internal/audit/audit.go:23`), which errors when a key is missing, and
`skill/resources/TEMPLATE.md`, the authoring guide shipped into every installed project.
**Nothing enforces that agreement.** The only TEMPLATE drift guard is
`TestScaffoldSectionsMatchTemplate` (`internal/ticket/ticket_test.go:146-162`), which compares
`## ` headings via `SectionHeadings` and is blind to frontmatter; the sole other reference is an
existence check at `internal/install/install_test.go:37`.

So a required key can be added to the audit while TEMPLATE.md keeps advertising the old
frontmatter — every ticket hand-authored from the template then fails `pickle board audit`, and
the failure surfaces in the *user's* project, not in this repo's tests. T-024 (`spawned-by:`)
is the first change to walk this tightrope; it kept the two in step by hand.

### Scope

A test asserting `ParseFrontmatter(TEMPLATE.md)`'s key set is a **superset** of
`audit.requiredKeys` (superset, not equality — the template may legitimately illustrate optional
keys). Natural home is `internal/audit` (it owns `requiredKeys`; note the constant is currently
unexported, so either move the test there or export a `RequiredKeys` accessor). Skip cleanly
when TEMPLATE.md is absent, matching the existing guard's `t.Skipf`.

Worth also considering the same guard for `ticket.Scaffold`'s emitted frontmatter — though
`TestScaffoldIsAuditClean` (`internal/ticket/ticket_test.go:125`) already covers it with a
hand-maintained key list, which has the same drift problem one level down.

### Couplings

Soft coupling to **T-024**, whose applicability audit surfaced this gap (finding N8). No hard
dependency: the guard can land before or after, though after T-024 it starts life green with
`spawned-by` already in both places.

## Implementation Plan

<!-- empty until refined; must meet the READY gate before moving to 2-ready/ -->

## Review

<!-- empty until IN REVIEW -->

## History

- 2026-07-25 — created (TO DO). source: applicability audit of T-024 (finding N8, non-blocking)
- 2026-07-26 — TO DO → DROPPED: absorbed into T-040 (board triage merge); content preserved here as the record
